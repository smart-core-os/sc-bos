package sim

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

// CheckInResponse is returned from the check-in endpoint. LatestConfig and LatestBinary are present
// only when there is an update of the respective type that hasn't fully installed yet.
type CheckInResponse struct {
	CheckIn      CheckInAck    `json:"checkIn"`
	LatestConfig *LatestStream `json:"latestConfig,omitempty"`
	LatestBinary *LatestStream `json:"latestBinary,omitempty"`
}

// CheckInAck is the acknowledgement portion of the check-in response.
type CheckInAck struct {
	NodeID      int64     `json:"nodeId,string"`
	CheckInTime time.Time `json:"checkInTime"`
}

// Artefact stream discriminators used in StreamDeployment.Artefact.
const (
	artefactConfig = "config"
	artefactBinary = "binary"
)

// LatestStream is the deployment a stream (config or binary) wants the node to reach plus the version
// it targets. The shape is identical for both streams.
type LatestStream struct {
	Deployment StreamDeployment  `json:"deployment"`
	Version    VersionProjection `json:"version"`
}

// StreamDeployment identifies a deployment (config or binary) that the server is offering for us to install.
type StreamDeployment struct {
	ID        string `json:"id"`
	Artefact  string `json:"artefact"`
	VersionID string `json:"versionId"`
	Status    string `json:"status"`
}

// VersionProjection contains metadata about the update version corresponding to the deployment: where to download it and
// how to verify it.
type VersionProjection struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	PayloadURL  string    `json:"payloadUrl"`
	Checksum    string    `json:"checksum"` // type-prefixed "<algo>:<base64>" digest of the payload; always present
	CreateTime  time.Time `json:"createTime"`
}

// configLatestStream projects an active config deployment and its version into the check-in response.
// It fails if the version has no checksum, since the node requires one to verify the download.
func configLatestStream(r *http.Request, d queries.ConfigDeployment, cv queries.ConfigVersion) (*LatestStream, error) {
	if len(cv.Sha256) == 0 {
		return nil, fmt.Errorf("config version %d has no checksum; run cloudsim -cleanup to backfill", cv.ID)
	}
	ls := &LatestStream{
		Deployment: StreamDeployment{
			ID:        formatDeploymentID(artefactConfig, d.ID),
			Artefact:  artefactConfig,
			VersionID: strconv.FormatInt(d.ConfigVersionID, 10),
			Status:    d.Status,
		},
		Version: VersionProjection{
			ID:         strconv.FormatInt(cv.ID, 10),
			Version:    cv.Version.String,
			PayloadURL: payloadUrl(r, cv.ID),
			Checksum:   checksum.Format(checksum.SHA256, cv.Sha256),
			CreateTime: cv.CreateTime,
		},
	}
	if cv.Description.Valid {
		ls.Version.Description = cv.Description.String
	}
	return ls, nil
}

// binaryLatestStream projects an active binary deployment and its artefact into the check-in response.
// It fails if the artefact has no checksum, since the node requires one to verify the download.
func binaryLatestStream(r *http.Request, d queries.BinaryDeployment, a store.BinaryArtefact) (*LatestStream, error) {
	if len(a.Sha256) == 0 {
		return nil, fmt.Errorf("binary artefact %d has no checksum", a.ID)
	}
	ls := &LatestStream{
		Deployment: StreamDeployment{
			ID:        formatDeploymentID(artefactBinary, d.ID),
			Artefact:  artefactBinary,
			VersionID: strconv.FormatInt(d.BinaryArtefactID, 10),
			Status:    d.Status,
		},
		Version: VersionProjection{
			ID:         strconv.FormatInt(a.ID, 10),
			Version:    a.Version,
			PayloadURL: binaryArtefactPayloadUrl(r, a.ID),
			Checksum:   checksum.Format(checksum.SHA256, a.Sha256),
			CreateTime: a.CreateTime,
		},
	}
	if a.Description.Valid {
		ls.Version.Description = a.Description.String
	}
	return ls, nil
}

// CheckInRequest is the optional request body for the check-in endpoint: what the node is running
// (running) plus progress on its in-flight deployments (progress). A bare "{}" is "just checking in".
type CheckInRequest struct {
	Running  RunningState     `json:"running"`
	Progress []ProgressReport `json:"progress,omitempty"`
}

// RunningState reports what the node is running and its platform. Only Platform is acted on here (it
// reconciles the node's os/arch); Config/Binary versions are accepted for wire compatibility but the
// deployment lifecycle is driven by Progress.
type RunningState struct {
	Config   *RunningArtefact `json:"config,omitempty"`
	Binary   *RunningArtefact `json:"binary,omitempty"`
	Platform *Platform        `json:"platform,omitempty"`
}

// RunningArtefact identifies the version a stream is running.
type RunningArtefact struct {
	Version   string `json:"version,omitempty"`
	VersionID string `json:"versionId,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// Platform is the node's GOOS/GOARCH, reconciled onto the node on check-in.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Progress states a node may report for an in-flight deployment.
const (
	stateInstalling = "installing"
	stateApplied    = "applied"
	stateFailed     = "failed"
)

// ProgressReport updates one in-flight deployment, keyed by its type-prefixed DeploymentID. Error is
// transient (with installing); Reason is terminal (with failed).
type ProgressReport struct {
	DeploymentID string `json:"deploymentId"`
	State        string `json:"state"`
	Attempts     int    `json:"attempts,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// Deployment ids are type-prefixed on the wire so config and binary deployments - which live in
// separate tables with independent autoincrement ids - stay distinct in the flat progress[] namespace.
const (
	deploymentPrefixConfig = "c-"
	deploymentPrefixBinary = "b-"
)

// formatDeploymentID renders a deployment's wire id: "c-<id>" for config, "b-<id>" for binary.
func formatDeploymentID(artefact string, id int64) string {
	prefix := deploymentPrefixConfig
	if artefact == artefactBinary {
		prefix = deploymentPrefixBinary
	}
	return prefix + strconv.FormatInt(id, 10)
}

// parseDeploymentID splits a wire deployment id into its stream and numeric id. An unknown prefix or a
// non-numeric id is an invalid request.
func parseDeploymentID(s string) (artefact string, id int64, err error) {
	var rest string
	if r, ok := strings.CutPrefix(s, deploymentPrefixConfig); ok {
		artefact, rest = artefactConfig, r
	} else if r, ok := strings.CutPrefix(s, deploymentPrefixBinary); ok {
		artefact, rest = artefactBinary, r
	} else {
		return "", 0, fmt.Errorf("deployment id %q: unknown prefix: %w", s, errInvalidRequest)
	}
	id, err = strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("deployment id %q: %w", s, errInvalidRequest)
	}
	return artefact, id, nil
}

// nextDeploymentStatus maps a reported progress state to a deployment's next status given its current
// one, or ok=false when there is no transition. installing advances a pending deployment to
// in_progress; applied completes a non-terminal deployment (even straight from pending); failed fails a
// non-terminal deployment. A terminal deployment (completed/failed/cancelled) is never revived.
func nextDeploymentStatus(current, state string) (string, bool) {
	terminal := current == statusCompleted || current == statusFailed || current == statusCancelled
	switch state {
	case stateInstalling:
		if current == statusPending {
			return statusInProgress, true
		}
	case stateApplied:
		if !terminal {
			return statusCompleted, true
		}
	case stateFailed:
		if !terminal {
			return statusFailed, true
		}
	}
	return "", false
}

// applyConfigProgress applies one progress report to a config deployment, after confirming it belongs
// to the node. A missing or foreign deployment is an invalid request.
func applyConfigProgress(ctx context.Context, tx *store.Tx, nodeID, id int64, p ProgressReport) error {
	row, err := tx.GetConfigDeploymentWithConfigVersion(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("config deployment %d: not found: %w", id, errInvalidRequest)
	}
	if err != nil {
		return err
	}
	if row.NodeID != nodeID {
		return fmt.Errorf("config deployment %d: belongs to a different node: %w", id, errInvalidRequest)
	}
	next, ok := nextDeploymentStatus(row.Status, p.State)
	if !ok {
		return nil
	}
	_, err = tx.UpdateConfigDeploymentStatus(ctx, queries.UpdateConfigDeploymentStatusParams{
		ID:     id,
		Status: next,
		Reason: nullString(p.Reason),
	})
	return err
}

// applyBinaryProgress applies one progress report to an update (binary) deployment, after confirming it
// belongs to the node. Binary deployments carry node_id directly, so no join is needed.
func applyBinaryProgress(ctx context.Context, tx *store.Tx, nodeID, id int64, p ProgressReport) error {
	dep, err := tx.GetBinaryDeployment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("binary deployment %d: not found: %w", id, errInvalidRequest)
	}
	if err != nil {
		return err
	}
	if dep.NodeID != nodeID {
		return fmt.Errorf("binary deployment %d: belongs to a different node: %w", id, errInvalidRequest)
	}
	next, ok := nextDeploymentStatus(dep.Status, p.State)
	if !ok {
		return nil
	}
	_, err = tx.SetBinaryDeploymentStatus(ctx, queries.SetBinaryDeploymentStatusParams{
		ID:     id,
		Status: next,
		Reason: nullString(p.Reason),
	})
	return err
}

// checkInRecord accumulates the per-stream current/installing deployment ids reported via progress[],
// to record on the node_check_ins row.
type checkInRecord struct {
	currentConfig, installingConfig *int64
	configError                     string
	configAttempts                  int
	currentBinary, installingBinary *int64
	binaryError                     string
	binaryAttempts                  int
}

func (a *checkInRecord) record(artefact string, id int64, p ProgressReport) {
	idCopy := id
	switch {
	case artefact == artefactConfig && p.State == stateInstalling:
		a.installingConfig, a.configError, a.configAttempts = &idCopy, p.Error, p.Attempts
	case artefact == artefactConfig && p.State == stateApplied:
		a.currentConfig = &idCopy
	case artefact == artefactBinary && p.State == stateInstalling:
		a.installingBinary, a.binaryError, a.binaryAttempts = &idCopy, p.Error, p.Attempts
	case artefact == artefactBinary && p.State == stateApplied:
		a.currentBinary = &idCopy
	}
}

func (a *checkInRecord) params(nodeID int64) queries.CreateNodeCheckInParams {
	return queries.CreateNodeCheckInParams{
		NodeID:                       nodeID,
		CurrentDeploymentID:          nullInt64(a.currentConfig),
		InstallingDeploymentID:       nullInt64(a.installingConfig),
		InstallingDeploymentError:    nullString(a.configError),
		InstallingDeploymentAttempts: nullInt64FromInt(a.configAttempts),
		CurrentBinaryDeploymentID:    nullInt64(a.currentBinary),
		InstallingBinaryDeploymentID: nullInt64(a.installingBinary),
		InstallingBinaryError:        nullString(a.binaryError),
		InstallingBinaryAttempts:     nullInt64FromInt(a.binaryAttempts),
	}
}

func (s *Server) checkIn(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := s.authenticateNode(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("check-in auth failed", zap.Error(err))
		return
	}

	var req CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	var (
		checkIn       queries.NodeCheckIn
		deployment    queries.ConfigDeployment
		configVersion queries.ConfigVersion
		hasDeploy     bool

		updateDep queries.BinaryDeployment
		hasBinary bool
	)
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		node, err := tx.GetNode(r.Context(), nodeID)
		if err != nil {
			return err
		}

		// Save the node's platform details if we don't know them already. We don't validate that these platform
		// values make sense, but a production server may want to do that.
		// If we do know them already, check that they match - a node should never change its platform, instead it
		// would need to be registered as a new node.
		if p := req.Running.Platform; p != nil {
			if node.Os == "" && node.Arch == "" {
				// no details saved on the node yet, so save them
				if _, err := tx.UpdateNodePlatform(r.Context(), queries.UpdateNodePlatformParams{
					ID:   node.ID,
					Os:   p.OS,
					Arch: p.Arch,
				}); err != nil {
					return err
				}
			} else if p.OS != node.Os || p.Arch != node.Arch {
				return errChangePlatform(Platform{OS: node.Os, Arch: node.Arch}, *p)
			}
		}

		// Apply each progress report to its deployment, routing on the id's stream prefix, and collect
		// the per-stream current/installing ids for the check-in record.
		var rec checkInRecord
		for _, p := range req.Progress {
			artefact, id, err := parseDeploymentID(p.DeploymentID)
			if err != nil {
				return err
			}
			switch artefact {
			case artefactConfig:
				if err := applyConfigProgress(r.Context(), tx, node.ID, id, p); err != nil {
					return err
				}
			case artefactBinary:
				if err := applyBinaryProgress(r.Context(), tx, node.ID, id, p); err != nil {
					return err
				}
			}
			rec.record(artefact, id, p)
		}

		checkIn, err = tx.CreateNodeCheckIn(r.Context(), rec.params(node.ID))
		if err != nil {
			return err
		}

		deployment, err = tx.GetActiveConfigDeploymentByNode(r.Context(), node.ID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			hasDeploy = false
		case err != nil:
			return err
		default:
			hasDeploy = true
			configVersion, err = tx.GetConfigVersion(r.Context(), deployment.ConfigVersionID)
			if err != nil {
				return err
			}
		}

		updateDep, err = tx.GetActiveBinaryDeploymentByNode(r.Context(), node.ID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			hasBinary = false
		case err != nil:
			return err
		default:
			hasBinary = true
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, errUnauthorized)
			logger.Debug("check-in for unknown node")
			return
		} else if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid check-in request", zap.Error(err))
			return
		} else if err, ok := errors.AsType[apiError](err); ok {
			writeError(w, err)
			logger.Error("check-in failed due to bad request", zap.Error(err))
			return
		}
		writeError(w, errInternal)
		logger.Error("check-in failed", zap.Error(err))
		return
	}

	resp := CheckInResponse{
		CheckIn: CheckInAck{
			NodeID:      checkIn.NodeID,
			CheckInTime: checkIn.CheckInTime,
		},
	}
	if hasDeploy {
		resp.LatestConfig, err = configLatestStream(r, deployment, configVersion)
		if err != nil {
			writeError(w, errInternal)
			logger.Error("failed to project config offer", zap.Error(err))
			return
		}
	}
	if hasBinary {
		artefact, err := s.store.GetBinaryArtefact(r.Context(), updateDep.BinaryArtefactID)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// The artefact row exists but its payload file is gone (removed out-of-band, restore issue,
			// orphan sweep). Degrade to "no binary offer this poll" rather than failing the whole
			// check-in, which would also drop the config offer already prepared for this poll.
			logger.Warn("binary artefact payload missing, omitting binary offer from check-in",
				zap.Int64("artefactId", updateDep.BinaryArtefactID))
		case err != nil:
			writeError(w, errInternal)
			logger.Error("failed to read binary artefact", zap.Error(err))
			return
		default:
			resp.LatestBinary, err = binaryLatestStream(r, updateDep, artefact)
			if err != nil {
				writeError(w, errInternal)
				logger.Error("failed to project binary offer", zap.Error(err))
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
