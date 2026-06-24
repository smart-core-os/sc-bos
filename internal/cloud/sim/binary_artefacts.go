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
	"github.com/smart-core-os/sc-bos/internal/util/checksum"
	"github.com/smart-core-os/sc-bos/internal/util/ocitag"
)

// BinaryArtefact contains metadata about a BOS binary artefact, for JSON serialisation.
type BinaryArtefact struct {
	ID          int64     `json:"id,string"`
	SiteID      *int64    `json:"siteId,string,omitempty"` // nil = generic, available to all sites
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Version     string    `json:"version"`
	Checksum    string    `json:"checksum,omitempty"` // type-prefixed "<algo>:<base64>", empty until computed
	Description string    `json:"description"`
	Size        int64     `json:"size,string"`
	PayloadURL  string    `json:"payloadUrl"`
	CreateTime  time.Time `json:"createTime"`
}

// binaryArtefactPayloadUrl constructs a fully qualified URL to download a binary artefact's payload.
// Hostname and scheme are inferred based on the request r.
func binaryArtefactPayloadUrl(r *http.Request, id int64) string {
	return sameHostURL(r, fmt.Sprintf("/api/v1/management/binary-artefacts/%d/payload", id))
}

// toBinaryArtefact maps stored artefact metadata to its JSON representation. The payload is never part
// of the row; it is downloaded separately from PayloadURL. The sha256 digest is emitted type-prefixed
// ("sha256:<base64>").
func toBinaryArtefact(r *http.Request, a store.BinaryArtefact) BinaryArtefact {
	out := BinaryArtefact{
		ID:         a.ID,
		OS:         a.Os,
		Arch:       a.Arch,
		Version:    a.Version,
		Size:       a.Size,
		PayloadURL: binaryArtefactPayloadUrl(r, a.ID),
		CreateTime: a.CreateTime,
	}
	if a.SiteID.Valid {
		siteID := a.SiteID.Int64
		out.SiteID = &siteID
	}
	if len(a.Sha256) > 0 {
		out.Checksum = checksum.Format(checksum.SHA256, a.Sha256)
	}
	if a.Description.Valid {
		out.Description = a.Description.String
	}
	return out
}

func toBinaryArtefacts(r *http.Request, as []store.BinaryArtefact) []BinaryArtefact {
	out := make([]BinaryArtefact, len(as))
	for i, a := range as {
		out[i] = toBinaryArtefact(r, a)
	}
	return out
}

func (s *Server) listBinaryArtefacts(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	afterID, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid pagination", zap.Error(err))
		return
	}

	// Optional os/arch filters ("" = no filter).
	os := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("arch")

	// Optional siteId filter (0 = no filter).
	siteID, err := parseID(r.URL.Query().Get("siteId"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid siteId filter", zap.Error(err))
		return
	}

	items, err := s.store.ListBinaryArtefacts(r.Context(), queries.ListBinaryArtefactsParams{
		AfterID: afterID,
		Os:      os,
		Arch:    arch,
		SiteID:  siteID,
		Limit:   limit + 1,
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to list binary artefacts", zap.Error(err))
		return
	}

	var nextToken string
	if int64(len(items)) > limit {
		nextToken = encodePageToken(items[limit-1].ID)
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, ListResponse[BinaryArtefact]{
		Items:         toBinaryArtefacts(r, items),
		NextPageToken: nextToken,
	})
}

func (s *Server) createBinaryArtefact(w http.ResponseWriter, r *http.Request) {
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
	// The version doubles as the image tag the supervisor loads, so reject one that isn't a valid OCI tag
	// here rather than letting it fail only at install time.
	if !ocitag.Valid(version) {
		writeError(w, errInvalidRequest)
		logger.Info("version is not a valid image tag", zap.String("version", version))
		return
	}

	// An artefact always targets a concrete platform, so os and arch are both required and must be
	// supported values.
	os := r.FormValue("os")
	if !validOS(os) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid os", zap.String("os", os))
		return
	}
	arch := r.FormValue("arch")
	if !validArch(arch) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid arch", zap.String("arch", arch))
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

	artefact, err := s.store.CreateBinaryArtefact(r.Context(), store.CreateBinaryArtefactParams{
		SiteID:      siteID,
		OS:          os,
		Arch:        arch,
		Version:     version,
		Description: description,
	}, file)
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create binary artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to create binary artefact", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toBinaryArtefact(r, artefact))
}

func (s *Server) getBinaryArtefact(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	item, err := s.store.GetBinaryArtefact(r.Context(), id)
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get binary artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to get binary artefact", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, toBinaryArtefact(r, item))
}

func (s *Server) getBinaryArtefactPayload(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid id", zap.Error(err))
		return
	}

	// Confirm the artefact exists before streaming so a missing one yields a clean 404.
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		_, err := tx.GetBinaryArtefact(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to get binary artefact payload", zap.Error(err))
		} else {
			logger.Debug("bad request to get binary artefact payload", zap.Error(err))
		}
		return
	}

	// started is set once we begin writing the response, so a failure to open the payload file
	// (e.g. the artefact was deleted in the window since the existence check) still yields a clean
	// error status rather than an implicit empty 200.
	started := false
	err = s.store.ReadBinaryArtefactPayload(r.Context(), id, func(file *os.File, size int64) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"binary-%d.tar\"", id))
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
			logger.Error("failed to open binary artefact payload", zap.Error(err))
			return
		}
		// The status and some body are already written, so we cannot change the status now.
		logger.Error("failed to stream binary artefact payload", zap.Error(err))
		return
	}
}

func (s *Server) deleteBinaryArtefact(w http.ResponseWriter, r *http.Request) {
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
		affected, err = tx.DeleteBinaryArtefact(r.Context(), id)
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to delete binary artefact", zap.Error(err))
		} else {
			logger.Debug("bad request to delete binary artefact", zap.Error(err))
		}
		return
	}

	if affected == 0 {
		writeError(w, errNotFound)
		logger.Debug("binary artefact not found", zap.Int64("id", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
