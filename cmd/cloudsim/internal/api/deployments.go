package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

const (
	statusPending    = "PENDING"
	statusInProgress = "IN_PROGRESS"
	statusCompleted  = "COMPLETED"
	statusFailed     = "FAILED"
	statusCancelled  = "CANCELLED"
)

// Deployment is the JSON representation of a deployment.
type Deployment struct {
	ID              int64      `json:"id"`
	ConfigVersionID int64      `json:"configVersionId"`
	Status          string     `json:"status"`
	StartTime       time.Time  `json:"startTime"`
	FinishedTime    *time.Time `json:"finishedTime,omitempty"`
}

func toDeployment(d queries.Deployment) Deployment {
	out := Deployment{
		ID:              d.ID,
		ConfigVersionID: d.ConfigVersionID,
		Status:          d.Status,
		StartTime:       d.StartTime,
	}
	if d.FinishedTime.Valid {
		out.FinishedTime = &d.FinishedTime.Time
	}
	return out
}

func toDeployments(deployments []queries.Deployment) []Deployment {
	out := make([]Deployment, len(deployments))
	for i, d := range deployments {
		out[i] = toDeployment(d)
	}
	return out
}

type createDeploymentRequest struct {
	ConfigVersionID int64  `json:"configVersionId"`
	Status          string `json:"status,omitempty"`
}

type updateDeploymentStatusRequest struct {
	Status string `json:"status"`
}

var validStatuses = map[string]bool{
	"PENDING":     true,
	"IN_PROGRESS": true,
	"COMPLETED":   true,
	"FAILED":      true,
	"CANCELLED":   true,
}

func isValidStatus(status string) bool {
	return validStatuses[status]
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	// Optional filters
	nodeID, err := parseID(r.URL.Query().Get("nodeId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid nodeId filter", zap.Error(err))
		return
	}

	configVersionID, err := parseID(r.URL.Query().Get("configVersionId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid configVersionId filter", zap.Error(err))
		return
	}

	// intersection queries not supported
	if nodeID != 0 && configVersionID != 0 {
		writeError(w, errInvalidRequest)
		logger.Info("cannot filter by both nodeId and configVersionId")
		return
	}

	var items []queries.Deployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		switch {
		case nodeID != 0:
			items, err = tx.ListDeploymentsByNode(r.Context(), queries.ListDeploymentsByNodeParams{
				NodeID:  nodeID,
				AfterID: afterID,
				Limit:   limit + 1,
			})
		case configVersionID != 0:
			items, err = tx.ListDeploymentsByConfigVersion(r.Context(), queries.ListDeploymentsByConfigVersionParams{
				ConfigVersionID: configVersionID,
				AfterID:         afterID,
				Limit:           limit + 1,
			})
		default:
			items, err = tx.ListDeployments(r.Context(), queries.ListDeploymentsParams{
				AfterID: afterID,
				Limit:   limit + 1,
			})
		}
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list deployments", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[Deployment]{
		Items:         toDeployments(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	// Apply default status if not provided or empty
	status := req.Status
	switch status {
	case "":
		status = statusPending
	case statusPending:
		// valid, do nothing
	default:
		writeError(w, errInvalidRequest)
		logger.Info("invalid status for creation", zap.String("status", status))
	}

	var item queries.Deployment
	var conflicted bool
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		cv, err := tx.GetConfigVersion(r.Context(), req.ConfigVersionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			return err
		}

		active, err := tx.GetActiveDeploymentByNode(r.Context(), cv.NodeID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			if active.Status == statusInProgress {
				conflicted = true
				return errDeploymentInProgress
			}
			// active is PENDING — cancel it
			_, err = tx.CancelPendingDeploymentsByNode(r.Context(), cv.NodeID)
			if err != nil {
				return err
			}
		}

		item, err = tx.CreateDeployment(r.Context(), queries.CreateDeploymentParams{
			ConfigVersionID: req.ConfigVersionID,
			Status:          status,
		})
		return err
	})
	if err != nil {
		if conflicted {
			writeError(w, errDeploymentInProgress)
			logger.Info("deployment in progress", zap.Int64("configVersionId", req.ConfigVersionID))
			return
		}
		if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid configVersionId", zap.Int64("configVersionId", req.ConfigVersionID))
			return
		}
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to create deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toDeployment(item))
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var item queries.Deployment
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to get deployment", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toDeployment(item))
}

func (s *Server) updateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var req updateDeploymentStatusRequest
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

	var item queries.Deployment
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.UpdateDeploymentStatus(r.Context(), queries.UpdateDeploymentStatusParams{
			ID:     id,
			Status: req.Status,
		})
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to update deployment status", zap.Error(err))
		} else {
			logger.Debug("bad request to update deployment status", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toDeployment(item))
}

func (s *Server) deleteDeployment(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteDeployment(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete deployment", zap.Error(err))
		} else {
			logger.Debug("bad request to delete deployment", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("deployment not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
