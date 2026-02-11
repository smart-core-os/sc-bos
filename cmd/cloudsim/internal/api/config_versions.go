package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// ConfigVersion is the JSON representation of a config version.
type ConfigVersion struct {
	ID          int64     `json:"id"`
	NodeID      int64     `json:"nodeId"`
	Description string    `json:"description"`
	Payload     []byte    `json:"payload"`
	CreateTime  time.Time `json:"createTime"`
}

func toConfigVersion(cv queries.ConfigVersion) ConfigVersion {
	description := ""
	if cv.Description.Valid {
		description = cv.Description.String
	}
	return ConfigVersion{
		ID:          cv.ID,
		NodeID:      cv.NodeID,
		Description: description,
		Payload:     cv.Payload,
		CreateTime:  cv.CreateTime,
	}
}

func toConfigVersions(cvs []queries.ConfigVersion) []ConfigVersion {
	out := make([]ConfigVersion, len(cvs))
	for i, cv := range cvs {
		out[i] = toConfigVersion(cv)
	}
	return out
}

type createConfigVersionRequest struct {
	NodeID      int64  `json:"nodeId"`
	Description string `json:"description"`
	Payload     []byte `json:"payload"`
}

func (s *Server) listConfigVersions(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid pagination", zap.Error(err))
		return
	}

	// Optional nodeId filter
	nodeID, err := parseID(r.URL.Query().Get("nodeId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid nodeId filter", zap.Error(err))
		return
	}

	var items []queries.ConfigVersion
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		if nodeID != 0 {
			items, err = tx.ListConfigVersionsByNode(r.Context(), queries.ListConfigVersionsByNodeParams{
				NodeID:  nodeID,
				AfterID: afterID,
				Limit:   limit + 1,
			})
		} else {
			items, err = tx.ListConfigVersions(r.Context(), queries.ListConfigVersionsParams{
				AfterID: afterID,
				Limit:   limit + 1,
			})
		}
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list config versions", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[ConfigVersion]{
		Items:         toConfigVersions(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createConfigVersion(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createConfigVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid json", zap.Error(err))
		return
	}

	var item queries.ConfigVersion
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		description := sql.NullString{Valid: false}
		if req.Description != "" {
			description = sql.NullString{String: req.Description, Valid: true}
		}
		item, err = tx.CreateConfigVersion(r.Context(), queries.CreateConfigVersionParams{
			NodeID:      req.NodeID,
			Description: description,
			Payload:     req.Payload,
		})
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to create config version", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusCreated, toConfigVersion(item))
}

func (s *Server) getConfigVersion(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var item queries.ConfigVersion
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetConfigVersion(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to get config version", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, toConfigVersion(item))
}

func (s *Server) deleteConfigVersion(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var affected int64
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		affected, err = tx.DeleteConfigVersion(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to delete config version", zap.Error(err))
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Error("config version not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
