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

// BinaryDeployment is the JSON representation of a binary deployment: the rollout of one binary
// artefact to one node. It mirrors ConfigDeployment but is node-scoped (artefacts are not).
type BinaryDeployment struct {
	ID               int64      `json:"id,string"`
	BinaryArtefactID int64      `json:"binaryArtefactId,string"`
	NodeID           int64      `json:"nodeId,string"`
	Status           string     `json:"status"`
	StartTime        time.Time  `json:"startTime"`
	FinishedTime     *time.Time `json:"finishedTime,omitempty"`
	Reason           string     `json:"reason,omitempty"`
}

func toBinaryDeployment(d queries.BinaryDeployment) BinaryDeployment {
	out := BinaryDeployment{
		ID:               d.ID,
		BinaryArtefactID: d.BinaryArtefactID,
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

func toBinaryDeployments(deployments []queries.BinaryDeployment) []BinaryDeployment {
	out := make([]BinaryDeployment, len(deployments))
	for i, d := range deployments {
		out[i] = toBinaryDeployment(d)
	}
	return out
}

type createBinaryDeploymentRequest struct {
	BinaryArtefactID int64  `json:"binaryArtefactId,string"`
	NodeID           int64  `json:"nodeId,string"`
	Status           string `json:"status,omitempty"`
}

type updateBinaryDeploymentStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) listBinaryDeployments(w http.ResponseWriter, r *http.Request) {
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

	var items []queries.BinaryDeployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		if nodeID != 0 {
			items, err = tx.ListBinaryDeploymentsByNode(r.Context(), queries.ListBinaryDeploymentsByNodeParams{
				NodeID:   nodeID,
				BeforeID: beforeID,
				Limit:    limit + 1,
			})
		} else {
			items, err = tx.ListBinaryDeployments(r.Context(), queries.ListBinaryDeploymentsParams{
				BeforeID: beforeID,
				Limit:    limit + 1,
			})
		}
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list binary deployments", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[BinaryDeployment]{
		Items:         toBinaryDeployments(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createBinaryDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createBinaryDeploymentRequest
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

	var item queries.BinaryDeployment
	var conflicted bool
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		artefact, err := tx.GetBinaryArtefact(r.Context(), req.BinaryArtefactID)
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

		// The artefact must target the node's platform (exact os/arch match).
		if artefact.Os != node.Os || artefact.Arch != node.Arch {
			return errInvalidRequest
		}

		// The artefact must be generic (no site) or match the node's site.
		if artefact.SiteID.Valid && artefact.SiteID.Int64 != node.SiteID {
			return errInvalidRequest
		}

		active, err := tx.GetActiveBinaryDeploymentByNode(r.Context(), node.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			if active.Status == statusInProgress {
				conflicted = true
				return errBinaryDeploymentInProgress
			}
			// active is PENDING — cancel it
			_, err = tx.CancelPendingBinaryDeploymentsByNode(r.Context(), node.ID)
			if err != nil {
				return err
			}
		}

		item, err = tx.CreateBinaryDeployment(r.Context(), queries.CreateBinaryDeploymentParams{
			BinaryArtefactID: req.BinaryArtefactID,
			NodeID:           req.NodeID,
			Status:           status,
		})
		return err
	})
	if err != nil {
		if conflicted {
			writeError(w, errBinaryDeploymentInProgress)
			logger.Info("binary deployment in progress", zap.Int64("nodeId", req.NodeID))
			return
		}
		if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid binary deployment request",
				zap.Int64("binaryArtefactId", req.BinaryArtefactID), zap.Int64("nodeId", req.NodeID))
			return
		}
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create binary deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to create binary deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toBinaryDeployment(item))
}

func (s *Server) getBinaryDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var item queries.BinaryDeployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetBinaryDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get binary deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to get binary deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toBinaryDeployment(item))
}

func (s *Server) updateBinaryDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var req updateBinaryDeploymentStatusRequest
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

	var item queries.BinaryDeployment
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.SetBinaryDeploymentStatus(r.Context(), queries.SetBinaryDeploymentStatusParams{
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
			logger.Error("failed to update binary deployment status", zap.Error(err))
		} else {
			logger.Debug("bad request to update binary deployment status", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toBinaryDeployment(item))
}

func (s *Server) deleteBinaryDeployment(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteBinaryDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete binary deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to delete binary deployment", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("binary deployment not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
