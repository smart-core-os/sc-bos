package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// supervisorCallTimeout bounds each Supervisor RPC so a hung Supervisor cannot block the poll loop
// indefinitely. All supervisor RPCs should complete quickly so a short fixed deadline suffices.
const supervisorCallTimeout = 10 * time.Second

// rollbackReasonFallback is reported when the Supervisor rolled an update back but gave no reason.
const rollbackReasonFallback = "update rolled back: running version does not match the target"

// noSupervisorReason is reported when BOS is offered an update it can't install because it has no Supervisor.
const noSupervisorReason = "software updates are not supported on this node: no Supervisor configured"

// BinaryUpdater handles the binary channel of a check-in. The Supervisor is the source of truth for an
// update's outcome, and the cloud re-offers the active deployment on every poll until BOS reports a
// terminal result. The only state BOS keeps is the cloud's version id for each version it was asked to
// run, so it can report which cloud version it is running.
type BinaryUpdater struct {
	// installer is the Supervisor's gRPC client, or nil when the integration is disabled. When nil the
	// binary channel attempts no install (BOS cannot apply an update without a Supervisor) and reports
	// any update it isn't already running as failed.
	installer supervisorpb.SupervisorApiClient
	version   string          // the version BOS currently runs; "" when unknown
	versions  *binaryVersions // optional - if nil, BinaryUpdater reports versions without version IDs
	logger    *zap.Logger
}

// BinaryUpdaterOption configures a BinaryUpdater.
type BinaryUpdaterOption func(*BinaryUpdater)

// WithBinaryLogger sets the logger for a BinaryUpdater.
func WithBinaryLogger(logger *zap.Logger) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.logger = logger }
}

// WithBinaryInstaller sets the Supervisor client the binary channel drives. When unset (or nil) the
// Supervisor integration is treated as disabled: no install is attempted, and an update BOS isn't
// already running is reported failed, since BOS cannot apply an update without a Supervisor.
func WithBinaryInstaller(installer supervisorpb.SupervisorApiClient) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.installer = installer }
}

// WithBinaryVersion sets the version BOS currently runs. It is the primary signal that an update
// succeeded (BOS came back running the target). The version is injected (rather than read from pkg/app)
// to avoid an import cycle: pkg/app imports internal/cloud.
func WithBinaryVersion(version string) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.version = version }
}

// WithBinaryVersionsFile enables the storage of recently deployed version ID mappings in a file with the given path.
// This allows the BinaryUpdater to remember which cloud versionID is associated with each version.
// A fixed number of mappings are stored, with mappings for newly deployed versions replacing the oldest version.
func WithBinaryVersionsFile(path string) BinaryUpdaterOption {
	return func(u *BinaryUpdater) { u.versions = &binaryVersions{path: path} }
}

// runningBinary reports the binary version BOS is running, or nil when unknown. Attempts to resolve version ID from
// u.versions but returns null version ID if not found, or empty version ID if attempt to resolve it failed.
func (u *BinaryUpdater) runningBinary() *RunningArtefact {
	if u.version == "" {
		return nil
	}
	running := &RunningArtefact{Version: u.version}
	if u.versions == nil {
		return running
	}
	id, ok, err := u.versions.lookup(u.version)
	switch {
	case err != nil:
		u.logger.Error("can't read recorded software version ids, not reporting one", zap.Error(err))
	case ok:
		running.VersionID = ReportID(id)
	default:
		running.VersionID = ReportNoID()
	}
	return running
}

// storeVersionMapping records the cloud's version id associated with an offered version.
// operation may fail - this is logged. It isn't returned because it is noncritical and there is no real way to handle it.
func (u *BinaryUpdater) storeVersionMapping(v VersionProjection) {
	if u.versions == nil {
		return
	}
	if err := u.versions.add(v.Version, v.ID); err != nil {
		u.logger.Error("can't record software version id",
			zap.String("version", v.Version), zap.String("versionId", v.ID), zap.Error(err))
	}
}

// canInstall reports whether BOS can install software updates, which needs a Supervisor.
func (u *BinaryUpdater) canInstall() bool {
	return u.installer != nil
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

// updateStatus reports the Supervisor's current update status, waiting up to supervisorCallTimeout for the
// Supervisor to become available. It returns nil (no error) when the Supervisor integration is disabled, so
// callers treat a disabled updater as having no update in flight.
func (u *BinaryUpdater) updateStatus(ctx context.Context) (*supervisorpb.UpdateStatus, error) {
	if u.installer == nil {
		return nil, nil
	}
	statusCtx, cancel := context.WithTimeout(ctx, supervisorCallTimeout)
	defer cancel()
	resp, err := u.installer.GetUpdateStatus(statusCtx, &supervisorpb.GetUpdateStatusRequest{}, grpc.WaitForReady(true))
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

// mayRollBack reports whether the Supervisor might still roll back.
func mayRollBack(st *supervisorpb.UpdateStatus, stErr error) bool {
	return stErr != nil || installInFlight(st)
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
// (per st) and whether a new deployment is available to install. With no Supervisor configured BOS
// reports current when it already runs the target, and failed otherwise.
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

	// Without a Supervisor BOS can neither apply nor confirm an update. If BOS already runs the target,
	// report it current; otherwise the update can never be installed here, so report it failed.
	if !u.canInstall() {
		if runningTarget {
			u.logger.Info("running the deployment's target version, reporting current to server",
				zap.String("deploymentId", deploymentID), zap.String("version", version.Version))
			return installState{}, u.reportApplied(ctx, client, latest)
		}
		u.logger.Warn("binary deployment received but no Supervisor configured, reporting failure to server",
			zap.String("deploymentId", deploymentID), zap.String("version", version.Version))
		return installState{}, u.reportFailed(ctx, client, deploymentID, noSupervisorReason)
	}

	// The Supervisor correlates its version-keyed outcome with this deployment via the opaque id BOS
	// passed on InstallUpdate. Only act on the Supervisor's terminal/in-flight state when it is about
	// this same deployment; otherwise fall through to the running-version check below.
	if st.GetDeploymentId() == deploymentID {
		switch st.GetState() {
		case supervisorpb.UpdateStatus_COMPLETED:
			// The Supervisor confirmed the update. Report current; the server completes the deployment
			// whether it was still pending or already in_progress.
			return installState{}, u.reportApplied(ctx, client, latest)
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
			return installState{}, u.reportFailed(ctx, client, deploymentID, reason)
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
		return installState{}, u.reportApplied(ctx, client, latest)
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

	// Recorded before installing, since BOS may restart into the new version at any point after this.
	u.storeVersionMapping(version)

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
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err),
			u.reportFailed(ctx, client, deploymentID, reason))
	default:
		// A transient or unknown failure: report installing-with-error and let a later poll retry.
		installing.Error = err.Error()
		_, reportErr := client.CheckIn(ctx, CheckInRequest{Progress: []ProgressReport{installing}})
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err), reportErr)
	}
}

// reportApplied records the version id of latest, then reports its deployment as running its target
// version (progress:applied) via a follow-up check-in that also carries the running binary, so the cloud
// learns its version id without waiting for the next poll. The server completes the deployment whether
// it was still pending or already in_progress.
func (u *BinaryUpdater) reportApplied(ctx context.Context, client Client, latest *LatestStream) error {
	u.storeVersionMapping(latest.Version)
	if _, err := client.CheckIn(ctx, CheckInRequest{
		Running:  RunningState{Binary: u.runningBinary()},
		Progress: []ProgressReport{{DeploymentID: latest.Deployment.ID, State: ProgressApplied}},
	}); err != nil {
		return fmt.Errorf("check-in to report current update: %w", err)
	}
	return nil
}

// reportFailed reports the deployment as permanently failed with reason (progress:failed) via a follow-up
// check-in. The server stops offering a failed deployment.
func (u *BinaryUpdater) reportFailed(ctx context.Context, client Client, deploymentID, reason string) error {
	if _, err := client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: deploymentID, State: ProgressFailed, Reason: reason}},
	}); err != nil {
		return fmt.Errorf("check-in to report failed update: %w", err)
	}
	return nil
}
