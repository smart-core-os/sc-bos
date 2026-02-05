package api

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// Site is the JSON representation of a site.
type Site struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	CreateTime time.Time `json:"createTime"`
}

func toSite(s queries.Site) Site {
	return Site{
		ID:         s.ID,
		Name:       s.Name,
		CreateTime: s.CreateTime,
	}
}

func toSites(sites []queries.Site) []Site {
	out := make([]Site, len(sites))
	for i, s := range sites {
		out[i] = toSite(s)
	}
	return out
}

type createSiteRequest struct {
	Name string `json:"name"`
}

type updateSiteRequest struct {
	Name string `json:"name"`
}

func (s *Server) listSites(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid pagination", zap.Error(err))
		return
	}

	var items []queries.Site
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		items, err = tx.ListSites(r.Context(), queries.ListSitesParams{
			AfterID: afterID,
			Limit:   limit + 1, // fetch one extra to check for next page
		})
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list sites", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[Site]{
		Items:         toSites(items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createSite(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	var req createSiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid json", zap.Error(err))
		return
	}

	if req.Name == "" {
		writeError(w, errInvalidRequest)
		logger.Error("missing required field", zap.String("field", "name"))
		return
	}

	var item queries.Site
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.CreateSite(r.Context(), req.Name)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to create site", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusCreated, toSite(item))
}

func (s *Server) getSite(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var item queries.Site
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetSite(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to get site", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, toSite(item))
}

func (s *Server) updateSite(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid id", zap.Error(err))
		return
	}

	var req updateSiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Error("invalid json", zap.Error(err))
		return
	}

	var item queries.Site
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.UpdateSite(r.Context(), queries.UpdateSiteParams{
			ID:   id,
			Name: req.Name,
		})
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to update site", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, toSite(item))
}

func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteSite(r.Context(), id)
		return err
	})
	if err != nil {
		writeError(w, translateDBError(err))
		logger.Error("failed to delete site", zap.Error(err))
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Error("site not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
