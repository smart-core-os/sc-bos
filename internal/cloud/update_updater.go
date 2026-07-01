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

// SoftwareUpdater handles the update channel of a check-in (see HandleUpdate). It is stateless: the
// Supervisor is the source of truth for an update's outcome, and the cloud re-offers the active
// deployment on every poll until BOS reports a terminal result, so BOS persists nothing.
type SoftwareUpdater struct {
	// installer is the Supervisor's gRPC client, or nil when the integration is disabled. When nil the
	// update channel attempts no install (BOS cannot apply an update without a Supervisor) and reports
	// nothing.
	installer supervisorpb.SupervisorApiClient
	version   string // the version BOS currently runs; "" when unknown
	logger    *zap.Logger
}

// SoftwareUpdaterOption configures a SoftwareUpdater.
type SoftwareUpdaterOption func(*SoftwareUpdater)

// WithSoftwareUpdaterLogger sets the logger for a SoftwareUpdater.
func WithSoftwareUpdaterLogger(logger *zap.Logger) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.logger = logger }
}

// WithUpdateInstaller sets the Supervisor client the update channel drives. When unset (or nil) the
// Supervisor integration is treated as disabled: no install is attempted and no result is reported,
// since BOS cannot apply an update without a Supervisor.
func WithUpdateInstaller(installer supervisorpb.SupervisorApiClient) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.installer = installer }
}

// WithSoftwareUpdaterVersion sets the version BOS currently runs. It is the primary signal that an update
// succeeded (BOS came back running the target). The version is injected (rather than read from pkg/app)
// to avoid an import cycle: pkg/app imports internal/cloud.
func WithSoftwareUpdaterVersion(version string) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.version = version }
}

// NewSoftwareUpdater creates a SoftwareUpdater.
func NewSoftwareUpdater(opts ...SoftwareUpdaterOption) *SoftwareUpdater {
	u := &SoftwareUpdater{}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// HandleUpdate applies the update channel of a check-in response. It reconciles the deployment the cloud
// offers in latestUpdate against the Supervisor's status and reports the appropriate outcome via a
// follow-up check-in (client):
//
//   - BOS already runs the target version: the update succeeded; report it current.
//   - the Supervisor's last update is this deployment and it FAILED: it was rolled back; report it failed
//     with the Supervisor's reason.
//   - the Supervisor is downloading or installing this deployment: report it installing.
//   - otherwise the deployment is new (or the Supervisor's last update was a different one): report it
//     installing and command the Supervisor to install it.
//
// It persists nothing, so it is safe to call on every poll.
func (u *SoftwareUpdater) HandleUpdate(ctx context.Context, client Client, resp CheckInResponse) error {
	latest := resp.LatestUpdate
	if latest == nil {
		return nil
	}
	deploymentID := latest.UpdateDeployment.ID
	if deploymentID == "" {
		return nil
	}
	artefact := latest.UpdateArtefact

	// BOS is running the target version, so the update succeeded. Send installing and current together so
	// a deployment still pending server-side moves pending -> in_progress -> completed in one check-in.
	if u.version != "" && artefact.Version == u.version {
		u.logger.Info("running the deployment's target version, reporting current to server",
			zap.String("deploymentId", deploymentID),
			zap.String("version", artefact.Version),
		)
		if _, err := client.CheckIn(ctx, CheckInRequest{
			InstallingUpdate: &CheckInInstallingDeployment{ID: deploymentID},
			CurrentUpdate:    &CheckInDeploymentRef{ID: deploymentID},
		}); err != nil {
			return fmt.Errorf("check-in to report current update: %w", err)
		}
		return nil
	}

	// Without a Supervisor BOS cannot apply an update, so skip quietly (typical in dev, where there is no
	// Supervisor): do not report and do not install.
	if u.installer == nil {
		u.logger.Debug("update deployment available but Supervisor integration disabled, not installing",
			zap.String("deploymentId", deploymentID),
			zap.String("version", artefact.Version),
		)
		return nil
	}

	statusCtx, cancel := context.WithTimeout(ctx, supervisorCallTimeout)
	st, err := u.installer.GetUpdateStatus(statusCtx, &supervisorpb.GetUpdateStatusRequest{})
	cancel()
	if err != nil {
		return fmt.Errorf("get supervisor update status: %w", err)
	}
	supStatus := st.GetStatus()

	// The Supervisor correlates its version-keyed outcome with this deployment via the opaque id BOS
	// passed on InstallUpdate. Only act on the Supervisor's terminal/in-flight state when it is about
	// this same deployment; otherwise treat the deployment as new and install it.
	if supStatus.GetDeploymentId() == deploymentID {
		switch supStatus.GetState() {
		case supervisorpb.UpdateStatus_COMPLETED:
			// The Supervisor confirmed the update, but BOS is not (yet) running the target version. Report
			// current to match the Supervisor's view.
			if _, err := client.CheckIn(ctx, CheckInRequest{
				CurrentUpdate: &CheckInDeploymentRef{ID: deploymentID},
			}); err != nil {
				return fmt.Errorf("check-in to report current update: %w", err)
			}
			return nil
		case supervisorpb.UpdateStatus_FAILED:
			reason := supStatus.GetError()
			if reason == "" {
				reason = rollbackReasonFallback
			}
			u.logger.Warn("update rolled back, reporting failure to server",
				zap.String("deploymentId", deploymentID),
				zap.String("targetVersion", artefact.Version),
				zap.String("runningVersion", u.version),
				zap.String("reason", reason),
			)
			if _, err := client.CheckIn(ctx, CheckInRequest{
				FailedUpdate: &CheckInFailedDeployment{ID: deploymentID, Reason: reason},
			}); err != nil {
				return fmt.Errorf("check-in to report failed update: %w", err)
			}
			return nil
		case supervisorpb.UpdateStatus_DOWNLOADING, supervisorpb.UpdateStatus_INSTALLING:
			// In flight: report installing and wait for a later poll to see the outcome.
			if _, err := client.CheckIn(ctx, CheckInRequest{
				InstallingUpdate: &CheckInInstallingDeployment{ID: deploymentID},
			}); err != nil {
				return fmt.Errorf("check-in to report installing update: %w", err)
			}
			return nil
		}
	}

	// A new deployment: report installing, then command the Supervisor to install the artefact.
	u.logger.Info("new update available, installing via supervisor",
		zap.String("deploymentId", deploymentID),
		zap.String("version", artefact.Version),
	)
	installing := &CheckInInstallingDeployment{ID: deploymentID}
	if _, err := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing}); err != nil {
		return fmt.Errorf("check-in to report installing update: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, supervisorCallTimeout)
	defer cancel()
	_, err = u.installer.InstallUpdate(installCtx, &supervisorpb.InstallUpdateRequest{
		Version:      artefact.Version,
		DownloadUrl:  artefact.PayloadURL,
		Sha256:       artefact.SHA256,
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
	default:
		// Report the failure to SCC and return the error for the caller to log.
		installing.Error = err.Error()
		_, reportErr := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing})
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err), reportErr)
	}
}
