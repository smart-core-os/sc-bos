package api

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// NodeCheckIn is the JSON representation of a node check-in.
type NodeCheckIn struct {
	ID                           int64     `json:"id"`
	NodeID                       int64     `json:"nodeId"`
	CheckInTime                  time.Time `json:"checkInTime"`
	CurrentDeploymentID          *int64    `json:"currentDeploymentId,omitempty"`
	InstallingDeploymentID       *int64    `json:"installingDeploymentId,omitempty"`
	InstallingDeploymentError    string    `json:"installingDeploymentError,omitempty"`
	InstallingDeploymentAttempts *int64    `json:"installingDeploymentAttempts,omitempty"`
}

func toNodeCheckIn(c queries.NodeCheckIn) NodeCheckIn {
	out := NodeCheckIn{
		ID:          c.ID,
		NodeID:      c.NodeID,
		CheckInTime: c.CheckInTime,
	}
	if c.CurrentDeploymentID.Valid {
		out.CurrentDeploymentID = &c.CurrentDeploymentID.Int64
	}
	if c.InstallingDeploymentID.Valid {
		out.InstallingDeploymentID = &c.InstallingDeploymentID.Int64
	}
	if c.InstallingDeploymentError.Valid {
		out.InstallingDeploymentError = c.InstallingDeploymentError.String
	}
	if c.InstallingDeploymentAttempts.Valid {
		out.InstallingDeploymentAttempts = &c.InstallingDeploymentAttempts.Int64
	}
	return out
}

func toNodeCheckIns(checkIns []queries.NodeCheckIn) []NodeCheckIn {
	out := make([]NodeCheckIn, len(checkIns))
	for i, c := range checkIns {
		out[i] = toNodeCheckIn(c)
	}
	return out
}

func (s *Server) listNodeCheckIns(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := parseID(r.PathValue("nodeId"))
	if err != nil || nodeID == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("invalid nodeId", zap.Error(err))
		return
	}

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	var items []queries.NodeCheckIn
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		items, err = tx.ListNodeCheckInsByNode(r.Context(), queries.ListNodeCheckInsByNodeParams{
			NodeID:  nodeID,
			AfterID: afterID,
			Limit:   limit + 1,
		})
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list node check-ins", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[NodeCheckIn]{
		Items:         toNodeCheckIns(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) getNodeCheckIn(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := parseID(r.PathValue("nodeId"))
	if err != nil || nodeID == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("invalid nodeId", zap.Error(err))
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil || id == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var item queries.NodeCheckIn
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetNodeCheckIn(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get node check-in", zap.Error(err))
		} else {
			logger.Debug("bad request to get node check-in", zap.Error(err))
		}
		return
	}

	if item.NodeID != nodeID {
		// attempt to access check-in at the wrong URL (not under its parent node)
		writeError(w, errNotFound)
		logger.Debug("check-in does not belong to node", zap.Int64("nodeId", nodeID), zap.Int64("checkInNodeId", item.NodeID))
		return
	}

	writeJSON(w, http.StatusOK, toNodeCheckIn(item))
}
