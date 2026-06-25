package sim

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// Artefact kinds distinguish what a payload is, so the supervisor's own RPM can be distributed
// alongside BOS podman-image tarballs to the same (podman) node.
const (
	// ArtefactKindBOSImage is a BOS container image (podman-save tarball). The default kind.
	ArtefactKindBOSImage = "bos-image"
	// ArtefactKindSupervisorRPM is a Supervisor self-update RPM.
	ArtefactKindSupervisorRPM = "supervisor-rpm"
)

// UpdateArtefact is the JSON representation of a BOS update artefact's metadata. The payload itself is
// never included here; it is downloaded separately from PayloadURL.
type UpdateArtefact struct {
	ID          int64     `json:"id,string"`
	SiteID      *int64    `json:"siteId,string,omitempty"` // nil = generic, available to all sites
	Platform    string    `json:"platform"`
	Kind        string    `json:"kind"`
	Version     string    `json:"version"`
	SHA256      string    `json:"sha256,omitempty"` // hex; empty until computed
	Description string    `json:"description"`
	Size        int64     `json:"size,string"`
	PayloadURL  string    `json:"payloadUrl"`
	CreateTime  time.Time `json:"createTime"`
}

// updateArtefactPayloadUrl constructs a fully qualified URL to download an update artefact's payload.
// Hostname and scheme are inferred based on the request r.
func updateArtefactPayloadUrl(r *http.Request, id int64) string {
	return sameHostURL(r, fmt.Sprintf("/api/v1/management/update-artefacts/%d/payload", id))
}

// toUpdateArtefact maps a stored artefact row to its JSON representation. The payload is never part
// of the row; it is downloaded separately from PayloadURL.
func toUpdateArtefact(r *http.Request, a queries.UpdateArtefact) UpdateArtefact {
	out := UpdateArtefact{
		ID:         a.ID,
		Platform:   a.Platform,
		Kind:       a.Kind,
		Version:    a.Version,
		Size:       a.Size,
		PayloadURL: updateArtefactPayloadUrl(r, a.ID),
		CreateTime: a.CreateTime,
	}
	if a.SiteID.Valid {
		siteID := a.SiteID.Int64
		out.SiteID = &siteID
	}
	if a.Sha256.Valid {
		out.SHA256 = a.Sha256.String
	}
	if a.Description.Valid {
		out.Description = a.Description.String
	}
	return out
}

func toUpdateArtefacts(r *http.Request, as []queries.UpdateArtefact) []UpdateArtefact {
	out := make([]UpdateArtefact, len(as))
	for i, a := range as {
		out[i] = toUpdateArtefact(r, a)
	}
	return out
}

func (s *Server) listUpdateArtefacts(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	// Optional platform filter ("" = no filter).
	platform := r.URL.Query().Get("platform")

	// Optional siteId filter (0 = no filter).
	siteID, err := parseID(r.URL.Query().Get("siteId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid siteId filter", zap.Error(err))
		return
	}

	var items []queries.UpdateArtefact
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		items, err = tx.ListUpdateArtefacts(r.Context(), queries.ListUpdateArtefactsParams{
			AfterID:  afterID,
			Platform: platform,
			SiteID:   siteID,
			Limit:    limit + 1,
		})
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list update artefacts", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[UpdateArtefact]{
		Items:         toUpdateArtefacts(r, items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createUpdateArtefact(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid multipart form", zap.Error(err))
		return
	}

	version := r.FormValue("version")
	if version == "" {
		writeError(w, errInvalidRequest)
		logger.Info("missing required field", zap.String("field", "version"))
		return
	}

	platform := r.FormValue("platform")
	if !validPlatform(platform) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid platform", zap.String("platform", platform))
		return
	}

	// kind is optional; empty defaults to a BOS image. Validate in Go (not the DB) against the known
	// kinds so a typo is caught.
	kind := r.FormValue("kind")
	switch kind {
	case "":
		kind = ArtefactKindBOSImage
	case ArtefactKindBOSImage, ArtefactKindSupervisorRPM:
	default:
		writeError(w, errInvalidRequest)
		logger.Info("invalid kind", zap.String("kind", kind))
		return
	}

	// siteId is optional; empty means a generic artefact (available to all sites).
	var siteID *int64
	if v := r.FormValue("siteId"); v != "" {
		id, err := parseID(v)
		if err != nil {
			writeError(w, errInvalidRequest)
			logger.Info("invalid siteId", zap.Error(err))
			return
		}
		siteID = &id
	}

	// description is optional.
	var description *string
	if v := r.FormValue("description"); v != "" {
		description = &v
	}

	file, header, err := r.FormFile("payload")
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("missing payload file", zap.Error(err))
		return
	}
	defer func() { _ = file.Close() }()
	if header.Size == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("empty payload")
		return
	}

	artefact, err := s.store.CreateUpdateArtefact(r.Context(), store.CreateUpdateArtefactParams{
		SiteID:      siteID,
		Platform:    platform,
		Kind:        kind,
		Version:     version,
		Description: description,
	}, file)
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create update artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to create update artefact", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toUpdateArtefact(r, artefact))
}

func (s *Server) getUpdateArtefact(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	var item queries.UpdateArtefact
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		item, err = tx.GetUpdateArtefact(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get update artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to get update artefact", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toUpdateArtefact(r, item))
}

func (s *Server) getUpdateArtefactPayload(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	// Confirm the artefact exists before streaming so a missing one yields a clean 404.
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		_, err := tx.GetUpdateArtefact(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get update artefact payload", zap.Error(err))
		} else {
			logger.Debug("bad request to get update artefact payload", zap.Error(err))
		}
		return
	}

	// started is set once we begin writing the response, so a failure to open the payload file
	// (e.g. the artefact was deleted in the window since the existence check) still yields a clean
	// error status rather than an implicit empty 200.
	started := false
	err = s.store.ReadUpdateArtefactPayload(r.Context(), id, func(file *os.File, size int64) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"update-%d.tar\"", id))
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
		started = true
		_, err := io.Copy(w, file)
		return err
	})
	if err != nil {
		if !started {
			// Nothing has been written yet, so we can still respond with a proper status.
			resErr := translateDBError(err)
			writeError(w, resErr)
			logger.Error("failed to open update artefact payload", zap.Error(err))
			return
		}
		// The status and some body are already written, so we cannot change the status now.
		logger.Error("failed to stream update artefact payload", zap.Error(err))
		return
	}
}

func (s *Server) deleteUpdateArtefact(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteUpdateArtefact(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete update artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to delete update artefact", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("update artefact not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
