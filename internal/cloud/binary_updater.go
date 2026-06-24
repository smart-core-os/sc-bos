package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// supervisorCallTimeout bounds each Supervisor RPC so a hung Supervisor cannot block the poll loop
// indefinitely. All supervisor RPCs should complete quickly so a short fixed deadline suffices.
const supervisorCallTimeout = 10 * time.Second

// rollbackReasonFallback is reported when the Supervisor rolled an update back but gave no reason.
const rollbackReasonFallback = "update rolled back: running version does not match the target"

// BinaryUpdater handles the binary channel of a check-in. It is stateless: the
// Supervisor is the source of truth for an update's outcome, and the cloud re-offers the active
// deployment on every poll until BOS reports a terminal result, so BOS persists nothing.
type BinaryUpdater struct {
	// installer is the Supervisor's gRPC client, or nil when the integration is disabled. When nil the
	// binary channel attempts no install (BOS cannot apply an update without a Supervisor) and reports
	// nothing.
	installer supervisorpb.SupervisorApiClient
	version   string // the version BOS currently runs; "" when unknown
	logger    *zap.Logger
}

// BinaryUpdaterOption configures a BinaryUpdater.
type BinaryUpdaterOption func(*BinaryUpdater)

// WithBinaryLogger sets the logger for a BinaryUpdater.
func WithBinaryLogger(logger *zap.Logger) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.logger = logger }
}

// WithBinaryInstaller sets the Supervisor client the binary channel drives. When unset (or nil) the
// Supervisor integration is treated as disabled: no install is attempted and no result is reported,
// since BOS cannot apply an update without a Supervisor.
func WithBinaryInstaller(installer supervisorpb.SupervisorApiClient) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.installer = installer }
}

// WithBinaryVersion sets the version BOS currently runs. It is the primary signal that an update
// succeeded (BOS came back running the target). The version is injected (rather than read from pkg/app)
// to avoid an import cycle: pkg/app imports internal/cloud.
func WithBinaryVersion(version string) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.version = version }
}

// runningBinary reports the binary version BOS is running, or nil when unknown.
func (u *BinaryUpdater) runningBinary() *RunningArtefact {
	if u.version == "" {
		return nil
	}
	return &RunningArtefact{Version: u.version}
}

// NewBinaryUpdater creates a BinaryUpdater.
func NewBinaryUpdater(opts ...BinaryUpdaterOption) *BinaryUpdater {
	u := &BinaryUpdater{}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// updateStatus reports the Supervisor's current update status. It returns nil (no error) when the
// Supervisor integration is disabled, so callers treat a disabled updater as having no update in flight.
func (u *BinaryUpdater) updateStatus(ctx context.Context) (*supervisorpb.UpdateStatus, error) {
	if u.installer == nil {
		return nil, nil
	}
	statusCtx, cancel := context.WithTimeout(ctx, supervisorCallTimeout)
	defer cancel()
	resp, err := u.installer.GetUpdateStatus(statusCtx, &supervisorpb.GetUpdateStatusRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetStatus(), nil
}

// installInFlight reports whether the Supervisor is mid-install: downloading, or installing (which also
// covers the window awaiting BOS's commit). A nil status is not in flight.
func installInFlight(st *supervisorpb.UpdateStatus) bool {
	switch st.GetState() {
	case supervisorpb.UpdateStatus_DOWNLOADING, supervisorpb.UpdateStatus_INSTALLING:
		return true
	default:
		return false
	}
}

// reportBinary reports the binary channel's status from a check-in response, using the Supervisor status
// st (fetched once per poll by the caller). It reconciles the offered latestBinary against st and reports
// the appropriate outcome via a follow-up check-in:
//
//   - the Supervisor's last update is this deployment and it COMPLETED: report it current.
//   - the Supervisor's last update is this deployment and it FAILED: it was rolled back; report it failed
//     with the Supervisor's reason.
//   - the Supervisor is downloading or installing this deployment: report it installing - even if BOS
//     already runs the target version, since the Supervisor may still roll back and completing now would
//     leave the cloud unable to observe that.
//   - the Supervisor holds no state for this deployment but BOS already runs the target: report it current
//     without reinstalling.
//   - otherwise the deployment is new: return with startable=true
//
// Does not start an installation. The returned installState tells the caller whether an update is already in flight
// (per st) and whether a new deployment is available to install. With no Supervisor configured BOS can
// only report current when it already runs the target.
func (u *BinaryUpdater) reportBinary(ctx context.Context, client Client, resp CheckInResponse, st *supervisorpb.UpdateStatus) (installState, error) {
	// The Supervisor's status is the authoritative in-flight signal, independent of what the server offers:
	// a running install must block a config reboot even on a poll where the binary is not offered.
	inFlight := installInFlight(st)

	latest := resp.LatestBinary
	if latest == nil {
		return installState{inFlight: inFlight}, nil
	}
	deploymentID := latest.Deployment.ID
	if deploymentID == "" {
		return installState{inFlight: inFlight}, nil
	}
	version := latest.Version
	runningTarget := u.version != "" && version.Version == u.version

	// Without a Supervisor BOS can neither apply nor confirm an update. The only signal available is
	// whether BOS already runs the target version; report that current, otherwise stay quiet (typical in
	// dev, where there is no Supervisor).
	if u.installer == nil {
		if runningTarget {
			u.logger.Info("running the deployment's target version, reporting current to server",
				zap.String("deploymentId", deploymentID), zap.String("version", version.Version))
			return installState{}, u.reportApplied(ctx, client, deploymentID)
		}
		u.logger.Debug("binary deployment available but Supervisor integration disabled, not installing",
			zap.String("deploymentId", deploymentID), zap.String("version", version.Version))
		return installState{}, nil
	}

	// The Supervisor correlates its version-keyed outcome with this deployment via the opaque id BOS
	// passed on InstallUpdate. Only act on the Supervisor's terminal/in-flight state when it is about
	// this same deployment; otherwise fall through to the running-version check below.
	if st.GetDeploymentId() == deploymentID {
		switch st.GetState() {
		case supervisorpb.UpdateStatus_COMPLETED:
			// The Supervisor confirmed the update. Report current; the server completes the deployment
			// whether it was still pending or already in_progress.
			return installState{}, u.reportApplied(ctx, client, deploymentID)
		case supervisorpb.UpdateStatus_FAILED:
			reason := st.GetError()
			if reason == "" {
				reason = rollbackReasonFallback
			}
			u.logger.Warn("update rolled back, reporting failure to server",
				zap.String("deploymentId", deploymentID),
				zap.String("targetVersion", version.Version),
				zap.String("runningVersion", u.version),
				zap.String("reason", reason),
			)
			if _, err := client.CheckIn(ctx, CheckInRequest{
				Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressFailed, Reason: reason}},
			}); err != nil {
				return installState{}, fmt.Errorf("check-in to report failed update: %w", err)
			}
			return installState{}, nil
		case supervisorpb.UpdateStatus_DOWNLOADING, supervisorpb.UpdateStatus_INSTALLING:
			// In flight: report installing and wait for a later poll to see the outcome. This holds even
			// when BOS already runs the target version - the Supervisor has not yet confirmed the commit
			// and may still roll back, so completing the deployment now would be premature and leave the
			// cloud unable to observe that rollback.
			if _, err := client.CheckIn(ctx, CheckInRequest{
				Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressInstalling}},
			}); err != nil {
				return installState{}, fmt.Errorf("check-in to report installing update: %w", err)
			}
			return installState{inFlight: inFlight}, nil
		}
	}

	// The Supervisor holds no in-flight state for this deployment. If BOS already runs the target it is
	// applied - and reinstalling would only force a needless restart - so report it current.
	if runningTarget {
		u.logger.Info("running the deployment's target version, reporting current to server",
			zap.String("deploymentId", deploymentID), zap.String("version", version.Version))
		return installState{}, u.reportApplied(ctx, client, deploymentID)
	}

	// A genuinely new deployment. Leave it for the caller to install (subject to the interlock); report
	// whether the Supervisor is otherwise busy so the caller does not start a second install.
	return installState{inFlight: inFlight, startable: true}, nil
}

// installBinary installs the offered binary deployment: it reports the deployment installing, then
// commands the Supervisor to install the artefact and reports a terminal failure if the Supervisor
// rejects it as invalid. The caller must have established (via the single-install interlock) that
// starting is allowed.
func (u *BinaryUpdater) installBinary(ctx context.Context, client Client, latest *LatestStream) error {
	deploymentID := latest.Deployment.ID
	version := latest.Version

	u.logger.Info("new update available, installing via supervisor",
		zap.String("deploymentId", deploymentID),
		zap.String("version", version.Version),
	)
	installing := ProgressReport{DeploymentID: deploymentID, State: ProgressInstalling}
	if _, err := client.CheckIn(ctx, CheckInRequest{Progress: []ProgressReport{installing}}); err != nil {
		return fmt.Errorf("check-in to report installing update: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, supervisorCallTimeout)
	defer cancel()
	_, err := u.installer.InstallUpdate(installCtx, &supervisorpb.InstallUpdateRequest{
		Version:      version.Version,
		DownloadUrl:  version.PayloadURL,
		Checksum:     version.Checksum,
		DeploymentId: deploymentID,
	})
	switch {
	case err == nil:
		// The Supervisor accepted the install and will restart BOS shortly. Nothing more to do here.
		return nil
	case status.Code(err) == codes.FailedPrecondition:
		// The Supervisor is already working on an update. Benign: a later poll resolves the outcome.
		u.logger.Info("supervisor reports an update already in progress",
			zap.String("deploymentId", deploymentID))
		return nil
	case status.Code(err) == codes.InvalidArgument:
		// The Supervisor rejected the artefact as malformed (e.g. the version is not a valid image tag).
		// Retrying the same artefact cannot succeed, so drive the deployment to a terminal failed state
		// rather than re-reporting installing and being re-offered every poll.
		reason := status.Convert(err).Message()
		u.logger.Warn("supervisor rejected the update as invalid, reporting failure to server",
			zap.String("deploymentId", deploymentID),
			zap.String("version", version.Version),
			zap.String("reason", reason),
		)
		_, reportErr := client.CheckIn(ctx, CheckInRequest{
			Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressFailed, Reason: reason}},
		})
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err), reportErr)
	default:
		// A transient or unknown failure: report installing-with-error and let a later poll retry.
		installing.Error = err.Error()
		_, reportErr := client.CheckIn(ctx, CheckInRequest{Progress: []ProgressReport{installing}})
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err), reportErr)
	}
}

// reportApplied reports the deployment as running its target version (progress:applied) via a follow-up
// check-in. The server completes the deployment whether it was still pending or already in_progress.
func (u *BinaryUpdater) reportApplied(ctx context.Context, client Client, deploymentID string) error {
	if _, err := client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressApplied}},
	}); err != nil {
		return fmt.Errorf("check-in to report current update: %w", err)
	}
	return nil
}
