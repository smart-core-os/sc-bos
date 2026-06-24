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

// UpdateInstaller is the slice of the Supervisor client the update channel drives. It is satisfied
// by *supervisor.Client.
//
// InstallUpdate is mandatory: a nil UpdateInstaller disables the integration, so the update channel
// attempts no install and persists no intent (BOS cannot apply an update without a Supervisor).
// GetUpdateStatus is best-effort, read only to recover a richer rollback failure reason; when the
// installer is nil the reconciliation falls back to a generic reason.
type UpdateInstaller interface {
	// InstallUpdate asks the Supervisor to download and install the given update. It returns
	// codes.FailedPrecondition when an update is already in progress.
	InstallUpdate(ctx context.Context, version, downloadURL, sha256 string) error
	// GetUpdateStatus returns the state of the most recent or in-progress update.
	GetUpdateStatus(ctx context.Context) (*supervisorpb.UpdateStatus, error)
}

// SoftwareUpdater handles the update channel of a check-in: it inspects the latestUpdate block and,
// when a new update deployment is offered, persists the intent to install it, reports installing to
// SCC, and commands the Supervisor to perform the install. The Supervisor restarts BOS on success;
// the post-restart instance reports the outcome (a later phase).
type SoftwareUpdater struct {
	store     UpdateStore
	installer UpdateInstaller // nil when the Supervisor integration is disabled
	version   string          // the version BOS currently runs; "" when unknown
	logger    *zap.Logger
}

// SoftwareUpdaterOption configures a SoftwareUpdater.
type SoftwareUpdaterOption func(*SoftwareUpdater)

// WithSoftwareUpdaterLogger sets the logger for a SoftwareUpdater.
func WithSoftwareUpdaterLogger(logger *zap.Logger) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.logger = logger }
}

// WithUpdateInstaller sets the Supervisor client the update channel drives. When unset (or nil) the
// Supervisor integration is treated as disabled: no install is attempted and no installing intent is
// reported, since BOS cannot apply an update without a Supervisor.
func WithUpdateInstaller(installer UpdateInstaller) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.installer = installer }
}

// WithSoftwareUpdaterVersion sets the version BOS currently runs, used to skip installing an update
// for a version already running. The version is injected (rather than read from pkg/app) to avoid an
// import cycle: pkg/app imports internal/cloud.
func WithSoftwareUpdaterVersion(version string) SoftwareUpdaterOption {
	return func(u *SoftwareUpdater) { u.version = version }
}

// NewSoftwareUpdater creates a SoftwareUpdater backed by store.
func NewSoftwareUpdater(store UpdateStore, opts ...SoftwareUpdaterOption) *SoftwareUpdater {
	u := &SoftwareUpdater{store: store}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// CheckInState reports the update channel's current state for inclusion in a shared check-in
// request. It surfaces any persisted in-flight intent as the installing update, so SCC keeps seeing
// the in-flight state on every poll until ReconcileStartup reports the outcome and clears the state.
func (u *SoftwareUpdater) CheckInState(ctx context.Context, req *CheckInRequest) error {
	state, ok, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load update state: %w", err)
	}
	if !ok {
		return nil
	}
	req.InstallingUpdate = &CheckInInstallingDeployment{ID: state.DeploymentID, Attempts: state.Attempts}
	return nil
}

// HandleUpdate applies the update channel of a check-in response. When resp names a new update
// deployment (id differs from the persisted in-flight intent), it persists the intent, reports
// installing to SCC via a follow-up check-in, and commands the Supervisor to install the artefact.
//
// client is used for the follow-up check-ins (mirroring the config flow, which check-ins to report
// installing and any error). On a successful InstallUpdate the Supervisor restarts BOS; BOS does not
// self-restart, so this just returns. A FailedPrecondition from the Supervisor (an update already in
// flight) is benign: the intent is kept and no error is returned.
func (u *SoftwareUpdater) HandleUpdate(ctx context.Context, client Client, resp CheckInResponse) error {
	latest := resp.LatestUpdate
	if latest == nil {
		return nil
	}
	deploymentID := latest.UpdateDeployment.ID
	if deploymentID == "" {
		return nil
	}

	state, ok, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load update state: %w", err)
	}
	if ok && state.DeploymentID == deploymentID {
		return nil // already recorded this in-flight intent
	}

	artefact := latest.UpdateArtefact

	// Already-current guard: if we already run the target version, do not install. This mirrors how
	// the config flow skips a latestConfig whose deployment equals the active one, and prevents an
	// install loop when SCC keeps offering a deployment for the running version.
	if u.version != "" && artefact.Version == u.version {
		u.logger.Debug("update deployment names the running version, not installing",
			zap.String("deploymentId", deploymentID),
			zap.String("version", artefact.Version),
		)
		return nil
	}

	// Without a Supervisor BOS cannot apply an update, so skip quietly: do not report installing and
	// do not persist intent (typical in dev, where there is no Supervisor).
	if u.installer == nil {
		u.logger.Debug("update deployment available but Supervisor integration disabled, not installing",
			zap.String("deploymentId", deploymentID),
			zap.String("version", artefact.Version),
		)
		return nil
	}

	// Persist the in-flight intent FIRST, before any check-in or Supervisor call, so a crash
	// mid-flight still lets the post-restart instance report. A different deployment supersedes any
	// persisted intent and same-deployment re-offers are short-circuited above, so this is always
	// this deployment's first tracked attempt.
	attempts := 1
	if err := u.store.Save(ctx, UpdateState{
		DeploymentID: deploymentID,
		Version:      artefact.Version,
		Attempts:     attempts,
		StartTime:    time.Now(),
	}); err != nil {
		return fmt.Errorf("save update state: %w", err)
	}

	u.logger.Info("new update available, installing via supervisor",
		zap.String("deploymentId", deploymentID),
		zap.String("version", artefact.Version),
		zap.Int("attempts", attempts),
	)

	// Report installing to SCC, mirroring the config flow's installing report.
	installing := &CheckInInstallingDeployment{ID: deploymentID, Attempts: attempts}
	if _, err := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing}); err != nil {
		return fmt.Errorf("check-in to report installing update: %w", err)
	}

	err = u.installer.InstallUpdate(ctx, artefact.Version, artefact.PayloadURL, artefact.SHA256)
	switch {
	case err == nil:
		// The Supervisor accepted the install and will restart BOS shortly. Nothing more to do here.
		return nil
	case status.Code(err) == codes.FailedPrecondition:
		// The Supervisor is already working on an update. Benign: keep the intent, no error.
		u.logger.Info("supervisor reports an update already in progress, keeping intent",
			zap.String("deploymentId", deploymentID))
		return nil
	default:
		// Report the failure to SCC (mirroring config's error reporting) and keep the intent so the
		// next poll or boot can resolve it. Return the error for the caller to log.
		installing.Error = err.Error()
		_, reportErr := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing})
		return errors.Join(fmt.Errorf("install update via supervisor: %w", err), reportErr)
	}
}

// ReconcileStartup resolves an in-flight update after the Supervisor has restarted BOS. It compares
// the running version (injected via WithSoftwareUpdaterVersion) to the in-flight target and reports
// the outcome to SCC, then clears the persisted intent:
//   - running == target -> the update succeeded -> report currentUpdate (mirrors config CommitInstall).
//   - running != target -> the Supervisor rolled back -> report failedUpdate with a reason taken from
//     the Supervisor's GetUpdateStatus().Error (or a generic fallback) (mirrors config FailInstall).
//
// A server-report failure is logged but not returned: the intent is still cleared (matching config's
// CommitInstall/FailInstall, which never fail BOS on a reporting error). When no intent is persisted
// this is a no-op. client is used for the outcome check-in, mirroring the config flow.
func (u *SoftwareUpdater) ReconcileStartup(ctx context.Context, client Client) error {
	state, ok, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load update state: %w", err)
	}
	if !ok {
		return nil // no in-flight update to reconcile
	}

	if u.version != "" && u.version == state.Version {
		// The new instance runs the target version: the update succeeded.
		u.logger.Info("update succeeded, reporting current to server",
			zap.String("deploymentId", state.DeploymentID),
			zap.String("version", state.Version),
		)
		if _, err := client.CheckIn(ctx, CheckInRequest{
			CurrentUpdate: &CheckInDeploymentRef{ID: state.DeploymentID},
		}); err != nil {
			u.logger.Warn("failed to report successful update to server",
				zap.String("deploymentId", state.DeploymentID), zap.Error(err))
		}
	} else {
		// The running version differs from the target: the Supervisor rolled back.
		reason := u.rollbackReason(ctx)
		u.logger.Warn("update rolled back, reporting failure to server",
			zap.String("deploymentId", state.DeploymentID),
			zap.String("targetVersion", state.Version),
			zap.String("runningVersion", u.version),
			zap.String("reason", reason),
		)
		if _, err := client.CheckIn(ctx, CheckInRequest{
			FailedUpdate: &CheckInFailedDeployment{ID: state.DeploymentID, Reason: reason},
		}); err != nil {
			u.logger.Warn("failed to report update failure to server",
				zap.String("deploymentId", state.DeploymentID), zap.Error(err))
		}
	}

	if err := u.store.Clear(ctx); err != nil {
		return fmt.Errorf("clear update state: %w", err)
	}
	return nil
}

// rollbackReason returns a human-readable reason for a rolled-back update, preferring the
// Supervisor's reported error and falling back to a generic message when unavailable.
func (u *SoftwareUpdater) rollbackReason(ctx context.Context) string {
	const generic = "update rolled back: running version does not match the target"
	if u.installer == nil {
		return generic
	}
	st, err := u.installer.GetUpdateStatus(ctx)
	if err != nil {
		u.logger.Debug("failed to read supervisor update status", zap.Error(err))
		return generic
	}
	if msg := st.GetError(); msg != "" {
		return msg
	}
	return generic
}
