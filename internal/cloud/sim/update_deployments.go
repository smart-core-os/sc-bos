package sim

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// UpdateDeployment is the JSON representation of an update deployment: the rollout of one update
// artefact to one node. It mirrors ConfigDeployment but is node-scoped (artefacts are not).
type UpdateDeployment struct {
	ID               int64      `json:"id,string"`
	UpdateArtefactID int64      `json:"updateArtefactId,string"`
	NodeID           int64      `json:"nodeId,string"`
	Status           string     `json:"status"`
	StartTime        time.Time  `json:"startTime"`
	FinishedTime     *time.Time `json:"finishedTime,omitempty"`
	Reason           string     `json:"reason,omitempty"`
}

func toUpdateDeployment(d queries.UpdateDeployment) UpdateDeployment {
	out := UpdateDeployment{
		ID:               d.ID,
		UpdateArtefactID: d.UpdateArtefactID,
		NodeID:           d.NodeID,
		Status:           d.Status,
		StartTime:        d.StartTime,
	}
	if d.FinishedTime.Valid {
		out.FinishedTime = &d.FinishedTime.Time
	}
	if d.Reason.Valid {
		out.Reason = d.Reason.String
	}
	return out
}

func toUpdateDeployments(deployments []queries.UpdateDeployment) []UpdateDeployment {
	out := make([]UpdateDeployment, len(deployments))
	for i, d := range deployments {
		out[i] = toUpdateDeployment(d)
	}
	return out
}

type createUpdateDeploymentRequest struct {
	UpdateArtefactID int64  `json:"updateArtefactId,string"`
	NodeID           int64  `json:"nodeId,string"`
	Status           string `json:"status,omitempty"`
}

type updateUpdateDeploymentStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) listUpdateDeployments(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	beforeID, limit, err := parsePaginationDesc(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	// Optional nodeId filter.
	nodeID, err := parseID(r.URL.Query().Get("nodeId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid nodeId filter", zap.Error(err))
		return
	}

	var items []queries.UpdateDeployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		if nodeID != 0 {
			items, err = tx.ListUpdateDeploymentsByNode(r.Context(), queries.ListUpdateDeploymentsByNodeParams{
				NodeID:   nodeID,
				BeforeID: beforeID,
				Limit:    limit + 1,
			})
		} else {
			items, err = tx.ListUpdateDeployments(r.Context(), queries.ListUpdateDeploymentsParams{
				BeforeID: beforeID,
				Limit:    limit + 1,
			})
		}
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list update deployments", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[UpdateDeployment]{
		Items:         toUpdateDeployments(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createUpdateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	// Apply default status if not provided or empty; reject any explicit non-pending status.
	status := req.Status
	switch status {
	case "":
		status = statusPending
	case statusPending:
		// valid, do nothing
	default:
		writeError(w, errInvalidRequest)
		logger.Info("invalid status for creation", zap.String("status", status))
		return
	}

	var item queries.UpdateDeployment
	var conflicted bool
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		artefact, err := tx.GetUpdateArtefact(r.Context(), req.UpdateArtefactID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			return err
		}

		node, err := tx.GetNode(r.Context(), req.NodeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			return err
		}

		// The artefact must target the node's platform.
		if artefact.Platform != node.Platform {
			return errInvalidRequest
		}

		// The artefact must be generic (no site) or match the node's site.
		if artefact.SiteID.Valid && artefact.SiteID.Int64 != node.SiteID {
			return errInvalidRequest
		}

		// Conflicts are per channel (artefact kind): a BOS-image deployment and a supervisor-rpm
		// deployment may be in flight on the same node at once.
		active, err := tx.GetActiveUpdateDeploymentByNodeAndKind(r.Context(), queries.GetActiveUpdateDeploymentByNodeAndKindParams{
			NodeID: node.ID,
			Kind:   artefact.Kind,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			if active.Status == statusInProgress {
				conflicted = true
				return errUpdateDeploymentInProgress
			}
			// active is PENDING — cancel it (same kind only)
			_, err = tx.CancelPendingUpdateDeploymentsByNodeAndKind(r.Context(), queries.CancelPendingUpdateDeploymentsByNodeAndKindParams{
				NodeID: node.ID,
				Kind:   artefact.Kind,
			})
			if err != nil {
				return err
			}
		}

		item, err = tx.CreateUpdateDeployment(r.Context(), queries.CreateUpdateDeploymentParams{
			UpdateArtefactID: req.UpdateArtefactID,
			NodeID:           req.NodeID,
			Status:           status,
		})
		return err
	})
	if err != nil {
		if conflicted {
			writeError(w, errUpdateDeploymentInProgress)
			logger.Info("update deployment in progress", zap.Int64("nodeId", req.NodeID))
			return
		}
		if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid update deployment request",
				zap.Int64("updateArtefactId", req.UpdateArtefactID), zap.Int64("nodeId", req.NodeID))
			return
		}
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create update deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to create update deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toUpdateDeployment(item))
}

func (s *Server) getUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var item queries.UpdateDeployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetUpdateDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get update deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to get update deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toUpdateDeployment(item))
}

func (s *Server) updateUpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var req updateUpdateDeploymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	if !isValidStatus(req.Status) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid status", zap.String("status", req.Status))
		return
	}

	var item queries.UpdateDeployment
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.SetUpdateDeploymentStatus(r.Context(), queries.SetUpdateDeploymentStatusParams{
			ID:     id,
			Status: req.Status,
			Reason: nullString(req.Reason),
		})
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to update update deployment status", zap.Error(err))
		} else {
			logger.Debug("bad request to update update deployment status", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toUpdateDeployment(item))
}

func (s *Server) deleteUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var affected int64
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		affected, err = tx.DeleteUpdateDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete update deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to delete update deployment", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("update deployment not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
