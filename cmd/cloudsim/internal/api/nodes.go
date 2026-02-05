package api

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// Node is the JSON representation of a node.
type Node struct {
	ID         int64     `json:"id"`
	Hostname   string    `json:"hostname"`
	SiteID     int64     `json:"siteId"`
	CreateTime time.Time `json:"createTime"`
}

func toNode(n queries.Node) Node {
	return Node{
		ID:         n.ID,
		Hostname:   n.Hostname,
		SiteID:     n.SiteID,
		CreateTime: n.CreateTime,
	}
}

func toNodes(nodes []queries.Node) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = toNode(n)
	}
	return out
}

type createNodeRequest struct {
	Hostname string `json:"hostname"`
	SiteID   int64  `json:"siteId"`
}

type updateNodeRequest struct {
	Hostname string `json:"hostname"`
	SiteID   int64  `json:"siteId"`
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid pagination", zap.Error(err))
		return
	}

	// Optional siteId filter
	siteID, err := parseID(r.URL.Query().Get("siteId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid siteId filter", zap.Error(err))
		return
	}

	var items []queries.Node
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		if siteID != 0 {
			items, err = tx.ListNodesBySite(r.Context(), queries.ListNodesBySiteParams{
				SiteID:  siteID,
				AfterID: afterID,
				Limit:   limit + 1,
			})
		} else {
			items, err = tx.ListNodes(r.Context(), queries.ListNodesParams{
				AfterID: afterID,
				Limit:   limit + 1,
			})
		}
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list nodes", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[Node]{
		Items:         toNodes(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid json", zap.Error(err))
		return
	}

	var item queries.Node
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.CreateNode(r.Context(), queries.CreateNodeParams{
			Hostname: req.Hostname,
			SiteID:   req.SiteID,
		})
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to create node", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusCreated, toNode(item))
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var item queries.Node
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetNode(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to get node", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, toNode(item))
}

func (s *Server) updateNode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var req updateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid json", zap.Error(err))
		return
	}

	var item queries.Node
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.UpdateNode(r.Context(), queries.UpdateNodeParams{
			ID:       id,
			Hostname: req.Hostname,
			SiteID:   req.SiteID,
		})
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to update node", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, toNode(item))
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteNode(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to delete node", zap.Error(err))
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Error("node not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
