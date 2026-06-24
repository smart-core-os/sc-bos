package sim

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// Node is the JSON representation of a node.
type Node struct {
	ID         int64     `json:"id,string"`
	Hostname   string    `json:"hostname"`
	SiteID     int64     `json:"siteId,string"`
	OS         string    `json:"os"`   // GOOS, empty until the node reports its platform on check-in
	Arch       string    `json:"arch"` // GOARCH, empty until the node reports its platform on check-in
	CreateTime time.Time `json:"createTime"`
}

func toNode(n queries.Node) Node {
	return Node{
		ID:         n.ID,
		Hostname:   n.Hostname,
		SiteID:     n.SiteID,
		OS:         n.Os,
		Arch:       n.Arch,
		CreateTime: n.CreateTime,
	}
}

// validOS reports whether os is a GOOS value the system supports.
func validOS(os string) bool {
	return os == "linux" || os == "freebsd"
}

// validArch reports whether arch is a GOARCH value the system supports.
func validArch(arch string) bool {
	return arch == "amd64" || arch == "arm64"
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
	SiteID   int64  `json:"siteId,string"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

type updateNodeRequest struct {
	Hostname string `json:"hostname"`
	SiteID   int64  `json:"siteId,string"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

// validNodePlatform reports whether a node's os/arch are acceptable: either both empty (the platform
// is unknown until the node first reports it on check-in) or both populated with supported values. A
// half-specified platform (one set, one empty) is rejected.
func validNodePlatform(os, arch string) bool {
	if os == "" && arch == "" {
		return true
	}
	return validOS(os) && validArch(arch)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	// Optional siteId filter
	siteID, err := parseID(r.URL.Query().Get("siteId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid siteId filter", zap.Error(err))
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
		logger.Info("invalid json", zap.Error(err))
		return
	}

	if req.Hostname == "" {
		writeError(w, errInvalidRequest)
		logger.Info("missing required field", zap.String("field", "hostname"))
		return
	}

	if !validNodePlatform(req.OS, req.Arch) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid platform", zap.String("os", req.OS), zap.String("arch", req.Arch))
		return
	}

	var item queries.Node
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.CreateNode(r.Context(), queries.CreateNodeParams{
			Hostname: req.Hostname,
			SiteID:   req.SiteID,
			Os:       req.OS,
			Arch:     req.Arch,
		})
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create node", zap.Error(err))
		} else {
			logger.Debug("bad request to create node", zap.Error(err))
		}
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
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get node", zap.Error(err))
		} else {
			logger.Debug("bad request to get node", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toNode(item))
}

func (s *Server) updateNode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var req updateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	if req.Hostname == "" {
		writeError(w, errInvalidRequest)
		logger.Info("missing required field", zap.String("field", "hostname"))
		return
	}

	if !validNodePlatform(req.OS, req.Arch) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid platform", zap.String("os", req.OS), zap.String("arch", req.Arch))
		return
	}

	var item queries.Node
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.UpdateNode(r.Context(), queries.UpdateNodeParams{
			ID:       id,
			Hostname: req.Hostname,
			SiteID:   req.SiteID,
			Os:       req.OS,
			Arch:     req.Arch,
		})
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to update node", zap.Error(err))
		} else {
			logger.Debug("bad request to update node", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toNode(item))
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteNode(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete node", zap.Error(err))
		} else {
			logger.Debug("bad request to delete node", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("node not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
