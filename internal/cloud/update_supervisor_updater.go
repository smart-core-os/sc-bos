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

// SupervisorInstaller is the slice of the Supervisor client the supervisor-update channel drives. It is
// satisfied by *supervisor.Client. A nil installer disables the channel (no install is attempted).
type SupervisorInstaller interface {
	// InstallSupervisorUpdate asks the Supervisor to update itself. It returns codes.FailedPrecondition
	// when a self-update is already in progress.
	InstallSupervisorUpdate(ctx context.Context, version, downloadURL, sha256 string) error
	// SupervisorInfo returns the running Supervisor's own version and its most recent self-update status.
	SupervisorInfo(ctx context.Context) (version string, selfUpdate *supervisorpb.UpdateStatus, err error)
}

// SupervisorUpdater drives the supervisor-update channel of a check-in: it installs a new Supervisor
// RPM when one is offered, and reports the outcome to SCC. Unlike a BOS update, a Supervisor self-update
// does not restart BOS, so there is no startup reconcile - the outcome is observed and reported on the
// regular poll by comparing the Supervisor's reported version (and self-update status) to the in-flight
// intent.
type SupervisorUpdater struct {
	store     UpdateStore
	installer SupervisorInstaller // nil when the Supervisor integration is disabled
	logger    *zap.Logger
}

// SupervisorUpdaterOption configures a SupervisorUpdater.
type SupervisorUpdaterOption func(*SupervisorUpdater)

// WithSupervisorUpdaterLogger sets the logger.
func WithSupervisorUpdaterLogger(logger *zap.Logger) SupervisorUpdaterOption {
	return func(u *SupervisorUpdater) { u.logger = logger }
}

// WithSupervisorInstaller sets the Supervisor client the channel drives. When unset (or nil) the channel
// is disabled: no install is attempted, since BOS cannot update the Supervisor without it.
func WithSupervisorInstaller(installer SupervisorInstaller) SupervisorUpdaterOption {
	return func(u *SupervisorUpdater) { u.installer = installer }
}

// NewSupervisorUpdater creates a SupervisorUpdater backed by store.
func NewSupervisorUpdater(store UpdateStore, opts ...SupervisorUpdaterOption) *SupervisorUpdater {
	u := &SupervisorUpdater{store: store}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// HandleSupervisorUpdate, called once per poll, does two things in order: it reconciles any in-flight
// self-update against the Supervisor's reported version/status (reporting current or failed to SCC and
// clearing the intent once settled), then, if a new supervisor-rpm deployment is offered for a version
// the Supervisor is not already running, it installs it.
//
// client is used for the follow-up outcome/installing check-ins, mirroring the config flow. Reading the
// Supervisor's info can fail transiently (e.g. mid-restart); that is logged and the poll skipped rather
// than failing the whole check-in.
func (u *SupervisorUpdater) HandleSupervisorUpdate(ctx context.Context, client Client, resp CheckInResponse) error {
	if u.installer == nil {
		return nil
	}

	state, hasIntent, err := u.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load supervisor update state: %w", err)
	}
	latest := resp.LatestSupervisorUpdate
	hasOffer := latest != nil && latest.UpdateDeployment.ID != ""
	if !hasIntent && !hasOffer {
		return nil // nothing to do; avoid an unnecessary socket call
	}

	version, selfUpdate, err := u.installer.SupervisorInfo(ctx)
	if err != nil {
		u.logger.Debug("read supervisor info, skipping supervisor update this poll", zap.Error(err))
		return nil
	}

	if hasIntent {
		if resolved := u.reconcile(ctx, client, state, version, selfUpdate); resolved {
			hasIntent = false
		}
	}

	if !hasOffer {
		return nil
	}
	deploymentID := latest.UpdateDeployment.ID
	if hasIntent && state.DeploymentID == deploymentID {
		return nil // already installing this deployment; awaiting its outcome
	}
	artefact := latest.UpdateArtefact
	if version == artefact.Version {
		u.logger.Debug("supervisor update names the running version, not installing",
			zap.String("deploymentId", deploymentID), zap.String("version", artefact.Version))
		return nil
	}
	return u.begin(ctx, client, deploymentID, artefact)
}

// reconcile reports a settled self-update to SCC and clears the intent, returning true when it did. A
// still-in-flight update (Supervisor not yet on the target version and not failed) is left alone.
func (u *SupervisorUpdater) reconcile(ctx context.Context, client Client, state UpdateState, version string, selfUpdate *supervisorpb.UpdateStatus) (resolved bool) {
	switch {
	case version == state.Version:
		u.logger.Info("supervisor update succeeded, reporting current to server",
			zap.String("deploymentId", state.DeploymentID), zap.String("version", state.Version))
		if _, err := client.CheckIn(ctx, CheckInRequest{
			CurrentUpdate: &CheckInDeploymentRef{ID: state.DeploymentID},
		}); err != nil {
			u.logger.Warn("failed to report successful supervisor update",
				zap.String("deploymentId", state.DeploymentID), zap.Error(err))
		}
	case selfUpdate.GetState() == supervisorpb.UpdateStatus_FAILED:
		reason := selfUpdate.GetError()
		if reason == "" {
			reason = "supervisor update rolled back: running version does not match the target"
		}
		u.logger.Warn("supervisor update rolled back, reporting failure to server",
			zap.String("deploymentId", state.DeploymentID),
			zap.String("targetVersion", state.Version), zap.String("runningVersion", version),
			zap.String("reason", reason))
		if _, err := client.CheckIn(ctx, CheckInRequest{
			FailedUpdate: &CheckInFailedDeployment{ID: state.DeploymentID, Reason: reason},
		}); err != nil {
			u.logger.Warn("failed to report supervisor update failure",
				zap.String("deploymentId", state.DeploymentID), zap.Error(err))
		}
	default:
		return false // still in flight
	}
	if err := u.store.Clear(ctx); err != nil {
		u.logger.Warn("clear supervisor update state", zap.Error(err))
	}
	return true
}

// begin persists the install intent, reports installing to SCC, and commands the Supervisor to install
// the RPM. Mirrors SoftwareUpdater.HandleUpdate: a FailedPrecondition (a self-update already in flight)
// is benign.
func (u *SupervisorUpdater) begin(ctx context.Context, client Client, deploymentID string, artefact UpdateArtefact) error {
	if err := u.store.Save(ctx, UpdateState{
		DeploymentID: deploymentID,
		Version:      artefact.Version,
		Attempts:     1,
		StartTime:    time.Now(),
	}); err != nil {
		return fmt.Errorf("save supervisor update state: %w", err)
	}
	u.logger.Info("new supervisor update available, installing",
		zap.String("deploymentId", deploymentID), zap.String("version", artefact.Version))

	installing := &CheckInInstallingDeployment{ID: deploymentID, Attempts: 1}
	if _, err := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing}); err != nil {
		return fmt.Errorf("check-in to report installing supervisor update: %w", err)
	}

	err := u.installer.InstallSupervisorUpdate(ctx, artefact.Version, artefact.PayloadURL, artefact.SHA256)
	switch {
	case err == nil:
		return nil
	case status.Code(err) == codes.FailedPrecondition:
		u.logger.Info("supervisor reports a self-update already in progress, keeping intent",
			zap.String("deploymentId", deploymentID))
		return nil
	default:
		installing.Error = err.Error()
		_, reportErr := client.CheckIn(ctx, CheckInRequest{InstallingUpdate: installing})
		return errors.Join(fmt.Errorf("install supervisor update: %w", err), reportErr)
	}
}
