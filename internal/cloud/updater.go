package cloud

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

// UpdaterOption configures a DeploymentUpdater.
type UpdaterOption func(*DeploymentUpdater)

// WithLogger sets the logger for a DeploymentUpdater.
func WithLogger(logger *zap.Logger) UpdaterOption {
	return func(u *DeploymentUpdater) { u.logger = logger }
}

// DeploymentUpdater checks for updates deployments from a Client and installs
// them into a store when Update is called.
// It delegates all filesystem operations to a DeploymentStore and all cloud
// communication to a Client.
//
// Methods are safe to call concurrently, though Update does not hold the
// store lock across HTTP calls — it is intended to be driven by a single loop
// (see AutoPoll).
//
// Newly installed deployments are marked as installing - it is the callers
// responsibility to retrieve the installing deployment from the store,
// verify it, and call CommitInstall or FailInstall accordingly. This can
// be done on a separate DeploymentUpdater sharing the same store e.g. after
// a process restart.
type DeploymentUpdater struct {
	store  *DeploymentStore
	client Client
	logger *zap.Logger
}

// NewDeploymentUpdater creates a new DeploymentUpdater.
func NewDeploymentUpdater(store *DeploymentStore, client Client, opts ...UpdaterOption) *DeploymentUpdater {
	u := &DeploymentUpdater{
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

// Update performs a check-in with the server and updates local deployment state accordingly.
// If a new deployment is available it is downloaded, extracted, and marked as installing;
// needReboot is true whenever an installing deployment is pending (either pre-existing or just staged).
func (u *DeploymentUpdater) Update(ctx context.Context) (needReboot bool, err error) {
	active, err := u.store.ActiveID()
	if err != nil {
		return false, fmt.Errorf("get active deployment id: %w", err)
	}
	installing, err := u.store.InstallingID()
	if err != nil {
		return false, fmt.Errorf("get installing deployment id: %w", err)
	}

	if active != "" && active == installing {
		u.logger.Warn("corrupted store - active and installing deployment are the same", zap.String("deploymentId", active))
		if err := u.store.ClearInstalling(ctx); err != nil {
			return false, fmt.Errorf("clear installing mark: %w", err)
		}
		installing = ""
	}

	req := CheckInRequest{}
	if active != "" {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: active}
	}
	if installing != "" {
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: installing}
	}

	u.logger.Debug("checking in with deployment server",
		zap.String("activeDeploymentId", active),
		zap.String("installingDeploymentId", installing),
	)
	resp, err := u.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in: %w", err)
	}

	if installing != "" {
		return true, nil
	}

	latest := resp.LatestConfig
	if latest == nil {
		return false, nil
	}
	if latest.Deployment.ID == active {
		return false, nil
	}

	u.logger.Info("new deployment available, starting installation",
		zap.String("deploymentId", latest.Deployment.ID),
		zap.String("configVersionId", latest.ConfigVersion.ID),
	)
	req.InstallingDeployment = &CheckInInstallingDeployment{ID: latest.Deployment.ID}
	if _, err = u.client.CheckIn(ctx, req); err != nil {
		return false, fmt.Errorf("check-in to report installing: %w", err)
	}

	body, err := u.client.DownloadPayload(ctx, latest.ConfigVersion.PayloadURL)
	if err != nil {
		req.InstallingDeployment.Error = err.Error()
		_, reportErr := u.client.CheckIn(ctx, req)
		return false, errors.Join(fmt.Errorf("download config version: %w", err), reportErr)
	}
	defer body.Close()

	if err := u.store.WriteInstalling(ctx, latest.Deployment.ID, body); err != nil {
		req.InstallingDeployment.Error = err.Error()
		_, reportErr := u.client.CheckIn(ctx, req)
		return false, errors.Join(fmt.Errorf("write installing deployment: %w", err), reportErr)
	}

	u.logger.Info("new deployment ready to install on next boot",
		zap.String("deploymentId", latest.Deployment.ID),
		zap.String("configVersionId", latest.ConfigVersion.ID),
	)
	return true, nil
}

// CheckIn performs a single check-in with the server using the current store
// state but does not act on the response. Use this to verify connectivity and
// credentials without triggering a deployment.
func (u *DeploymentUpdater) CheckIn(ctx context.Context) error {
	active, err := u.store.ActiveID()
	if err != nil {
		return fmt.Errorf("get active deployment id: %w", err)
	}
	installing, err := u.store.InstallingID()
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	req := CheckInRequest{}
	if active != "" {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: active}
	}
	if installing != "" {
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: installing}
	}
	_, err = u.client.CheckIn(ctx, req)
	return err
}

// CommitInstall marks the installing deployment as active (filesystem) and reports
// the successful install to the server. A server-reporting failure is logged but not
// returned — it will be corrected on the next AutoPoll check-in.
func (u *DeploymentUpdater) CommitInstall(ctx context.Context) error {
	installingID, err := u.store.InstallingID()
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	if err := u.store.CommitInstall(ctx); err != nil {
		return err
	}
	if _, err := u.client.CheckIn(ctx, CheckInRequest{
		CurrentDeployment: &CheckInDeploymentRef{ID: installingID},
	}); err != nil {
		u.logger.Warn("failed to report successful install to server",
			zap.String("deploymentId", installingID), zap.Error(err))
	}
	return nil
}

// FailInstall clears the installing mark (filesystem) and reports the failure to the server.
// A server-reporting failure is logged but not returned.
func (u *DeploymentUpdater) FailInstall(ctx context.Context, message string) error {
	installingID, _ := u.store.InstallingID()
	activeID, _ := u.store.ActiveID()

	if err := u.store.FailInstall(ctx); err != nil {
		return err
	}

	req := CheckInRequest{
		FailedDeployment: &CheckInFailedDeployment{ID: installingID, Reason: message},
	}
	if activeID != "" {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: activeID}
	}
	if _, err := u.client.CheckIn(ctx, req); err != nil {
		u.logger.Warn("failed to report install failure to server",
			zap.String("deploymentId", installingID), zap.Error(err))
	}
	return nil
}
