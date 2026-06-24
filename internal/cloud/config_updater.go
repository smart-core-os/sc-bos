package cloud

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash"
	"io"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

// installState describes a channel's install situation, reported by reportConfig/reportBinary so the
// caller can enforce the single-install interlock.
type installState struct {
	inFlight  bool // an install for this channel is already underway (blocks starting another)
	startable bool // the server offers a new target this channel could begin installing
}

// ConfigUpdaterOption configures a ConfigUpdater.
type ConfigUpdaterOption func(*ConfigUpdater)

// WithConfigLogger sets the logger for a ConfigUpdater.
func WithConfigLogger(logger *zap.Logger) ConfigUpdaterOption {
	return func(u *ConfigUpdater) { u.logger = logger }
}

// ConfigUpdater installs config deployments on this BOS node when the cloud supplies a new version to install.
// It checks in with the cloud (reporting what is installed), and when a newer deployment is specified, it downloads the
// payload and stages it in the local store ready to install on next boot.
// Cloud calls go through a Client, store mutations through a DeploymentStore.
//
// A staged deployment is not activated automatically: it is left marked installing for the caller to
// verify and then either CommitInstall or FailInstall. You can verify the staged deployment from a different
// ConfigUpdater sharing the same store, e.g. after a reboot into the new deployment.
type ConfigUpdater struct {
	store  *DeploymentStore
	client Client
	logger *zap.Logger
}

// NewConfigUpdater creates a new ConfigUpdater.
func NewConfigUpdater(store *DeploymentStore, client Client, opts ...ConfigUpdaterOption) *ConfigUpdater {
	u := &ConfigUpdater{
		store:  store,
		client: client,
	}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// CheckInRequest builds a check-in request representing the config update stream only.
// It performs corrupted-store recovery (clearing the installing mark when it equals the active mark) as a side effect,
// so it is idempotent but not pure.
//
// running.config reports the config version of the active deployment.
func (u *ConfigUpdater) CheckInRequest(ctx context.Context) (CheckInRequest, error) {
	active, err := u.store.ActiveID()
	if err != nil {
		return CheckInRequest{}, fmt.Errorf("get active deployment id: %w", err)
	}
	installing, err := u.store.InstallingID()
	if err != nil {
		return CheckInRequest{}, fmt.Errorf("get installing deployment id: %w", err)
	}

	if active != "" && active == installing {
		u.logger.Warn("corrupted store - active and installing deployment are the same", zap.String("deploymentId", active))
		if err := u.store.ClearInstalling(ctx); err != nil {
			return CheckInRequest{}, fmt.Errorf("clear installing mark: %w", err)
		}
		installing = ""
	}

	version, err := u.store.ActiveVersion()
	if err != nil {
		return CheckInRequest{}, fmt.Errorf("get active config version: %w", err)
	}

	req := CheckInRequest{}
	if version.ID != "" || version.Version != "" {
		req.Running.Config = &RunningArtefact{Version: version.Version, VersionID: version.ID}
	}
	if installing != "" {
		req.Progress = append(req.Progress, ProgressReport{DeploymentID: installing, State: ProgressInstalling})
	}
	return req, nil
}

// reportConfig reports the config channel's status and returns whether an install is in flight or
// available to start. Does not start or progress an install.
func (u *ConfigUpdater) reportConfig(ctx context.Context, resp CheckInResponse) (installState, error) {
	installing, err := u.store.InstallingID()
	if err != nil {
		return installState{}, fmt.Errorf("get installing deployment id: %w", err)
	}
	if installing != "" {
		return installState{inFlight: true}, nil
	}
	active, err := u.store.ActiveID()
	if err != nil {
		return installState{}, fmt.Errorf("get active deployment id: %w", err)
	}

	latest := resp.LatestConfig
	if latest == nil {
		return installState{}, nil
	}
	if latest.Deployment.ID == active {
		// The server still offers a deployment this node already has active, so its applied report was
		// lost (a completed deployment is not re-offered). Re-report it so the deployment can complete -
		// otherwise it stays in_progress and blocks the config stream.
		if _, err := u.client.CheckIn(ctx, CheckInRequest{
			Progress: []ProgressReport{{DeploymentID: active, State: ProgressApplied}},
		}); err != nil {
			return installState{}, fmt.Errorf("check-in to re-report applied config: %w", err)
		}
		return installState{}, nil
	}

	return installState{startable: true}, nil
}

// installConfig installs the offered config deployment: it reports the deployment installing, downloads
// and verifies the payload, and stages it for the next boot. needReboot is true once the deployment is
// staged. The caller must have established (via the single-install interlock) that starting is allowed.
func (u *ConfigUpdater) installConfig(ctx context.Context, latest *LatestStream) (needReboot bool, err error) {
	deploymentID := latest.Deployment.ID

	// reports the deployment as still installing with a transient error, so the server keeps offering it and a later
	// poll retries.
	failTemp := func(err error) (bool, error) {
		_, reportErr := u.client.CheckIn(ctx, CheckInRequest{
			Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressInstalling, Error: err.Error()}},
		})
		return false, errors.Join(err, reportErr)
	}
	// reports it failed, so the server stops offering it - for errors retrying cannot fix
	failPermanent := func(err error) (bool, error) {
		_, reportErr := u.client.CheckIn(ctx, CheckInRequest{
			Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressFailed, Reason: err.Error()}},
		})
		return false, errors.Join(err, reportErr)
	}

	u.logger.Info("new deployment available, starting installation",
		zap.String("deploymentId", deploymentID),
		zap.String("configVersionId", latest.Version.ID),
	)
	installing := CheckInRequest{Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressInstalling}}}
	if _, err = u.client.CheckIn(ctx, installing); err != nil {
		return false, fmt.Errorf("check-in to report installing: %w", err)
	}

	body, err := u.client.DownloadPayload(ctx, latest.Version.PayloadURL)
	if err != nil {
		return failTemp(fmt.Errorf("download config version: %w", err))
	}
	// payload is the reader handed to the store; closing it closes the underlying body.
	payload := body
	defer func() { _ = payload.Close() }()

	// Verify the download against the version's checksum. A missing or malformed checksum is a server
	// contract violation the node cannot resolve by retrying, so fail the deployment terminally - the
	// same treatment the binary channel gives a supervisor InvalidArgument.
	verified, err := newVerifiedReader(body, latest.Version.Checksum)
	if err != nil {
		return failPermanent(fmt.Errorf("config version checksum: %w", err))
	}
	// whatever is in payload gets closed - and verifiedReader closes the underlying body
	payload = verified

	version := ConfigVersion{ID: latest.Version.ID, Version: latest.Version.Version}
	if err := u.store.WriteInstalling(ctx, deploymentID, version, payload); err != nil {
		return failTemp(fmt.Errorf("write installing deployment: %w", err))
	}

	u.logger.Info("new deployment ready to install on next boot",
		zap.String("deploymentId", deploymentID),
		zap.String("configVersionId", latest.Version.ID),
	)
	return true, nil
}

// CheckIn performs a single check-in with the server using the current store
// state but does not act on the response. Use this to verify connectivity and
// credentials without triggering a deployment.
func (u *ConfigUpdater) CheckIn(ctx context.Context) error {
	installing, err := u.store.InstallingID()
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	req := CheckInRequest{}
	if installing != "" {
		req.Progress = append(req.Progress, ProgressReport{DeploymentID: installing, State: ProgressInstalling})
	}
	_, err = u.client.CheckIn(ctx, req)
	return err
}

// verifiedReader streams an underlying ReadCloser through a hash and, on EOF, fails the read if the
// accumulated digest does not match the expected checksum. It lets a download be verified inline as
// it is consumed, without buffering the whole payload.
type verifiedReader struct {
	r    io.ReadCloser
	hash hash.Hash
	want checksum.Checksum
}

// newVerifiedReader wraps r to verify it against a type-prefixed "<algo>:<base64>" checksum.
func newVerifiedReader(r io.ReadCloser, want string) (*verifiedReader, error) {
	c, err := checksum.Parse(want)
	if err != nil {
		return nil, err
	}
	h, err := c.Algo.NewHash()
	if err != nil {
		return nil, err
	}
	return &verifiedReader{r: r, hash: h, want: c}, nil
}

func (v *verifiedReader) Read(p []byte) (int, error) {
	n, err := v.r.Read(p)
	if n > 0 {
		v.hash.Write(p[:n])
	}
	if errors.Is(err, io.EOF) {
		if got := v.hash.Sum(nil); !bytes.Equal(got, v.want.Digest) {
			return n, fmt.Errorf("checksum mismatch: want %s, got %s", v.want, checksum.Format(v.want.Algo, got))
		}
	}
	return n, err
}

func (v *verifiedReader) Close() error { return v.r.Close() }

// CommitInstall marks the installing deployment as active (filesystem) and reports the successful
// install to the server as progress:applied. A server-reporting failure is logged, not returned -
// the server will offer the same deployment to use again on the next check-in, at which point we
// will mark it as complete again, correcting the error.
func (u *ConfigUpdater) CommitInstall(ctx context.Context) error {
	installingID, err := u.store.InstallingID()
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	if err := u.store.CommitInstall(ctx); err != nil {
		return err
	}
	if _, err := u.client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: installingID, State: ProgressApplied}},
	}); err != nil {
		u.logger.Warn("failed to report successful install to server",
			zap.String("deploymentId", installingID), zap.Error(err))
	}
	return nil
}

// FailInstall clears the installing mark (filesystem) and reports the failure to the server.
// Logs a warning.
// May return a store error.
func (u *ConfigUpdater) FailInstall(ctx context.Context, message string) error {
	installingID, _ := u.store.InstallingID()

	if err := u.store.FailInstall(ctx); err != nil {
		return err
	}

	if _, err := u.client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: installingID, State: ProgressFailed, Reason: message}},
	}); err != nil {
		u.logger.Warn("failed to report install failure to server",
			zap.String("deploymentId", installingID), zap.Error(err))
	}
	return nil
}
