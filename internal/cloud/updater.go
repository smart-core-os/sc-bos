package cloud

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// UpdaterOption configures a DeploymentUpdater.
type UpdaterOption func(*DeploymentUpdater)

// WithLogger sets the logger for a DeploymentUpdater.
func WithLogger(logger *zap.Logger) UpdaterOption {
	return func(u *DeploymentUpdater) {
		u.logger = logger
	}
}

// DeploymentUpdater manages deployment state on disk and coordinates with the cloud check-in API.
//
// Methods are safe to call concurrently.
type DeploymentUpdater struct {
	dir    *os.Root
	client Client
	logger *zap.Logger
	lockCh chan struct{}
}

// NewDeploymentUpdater creates a new DeploymentUpdater.
// The storeDir is used to store config version packages, and retain state between BOS boots.
func NewDeploymentUpdater(storeDir *os.Root, client Client, opts ...UpdaterOption) *DeploymentUpdater {
	u := &DeploymentUpdater{
		dir:    storeDir,
		client: client,
		lockCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(u)
	}
	if u.logger == nil {
		u.logger = zap.NewNop()
	}
	return u
}

// InstallingConfig returns an fs.FS containing the config files for the currently installing deployment, which has
// not yet been committed. Once the deployment has been applied, call either CommitInstall or FailInstall.
// Returns a nil fs.FS with no error if there is no deployment currently being installed.
func (c *DeploymentUpdater) InstallingConfig() (fs.FS, error) {
	return c.extractedConfigByMark(markInstalling)
}

func (c *DeploymentUpdater) CommitInstall(ctx context.Context) error {
	if !c.lock(ctx) {
		return ctx.Err()
	}
	defer c.unlock()

	installingID, err := c.deploymentIDByMark(markInstalling)
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	oldActiveID, err := c.deploymentIDByMark(markActive)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("get active deployment id: %w", err)
	}
	// if there is no active deployment, then oldActiveID = 0

	err = c.mark(markActive, installingID)
	if err != nil {
		return fmt.Errorf("mark active deployment: %w", err)
	}
	err = c.clearMark(markInstalling)
	if err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}
	c.logger.Info("marked deployment as active", zap.Int64("deploymentId", installingID))

	// tell server that we have installed the deployment
	_, err = c.client.CheckIn(ctx, CheckInRequest{
		CurrentDeployment: &CheckInDeploymentRef{ID: installingID},
	})
	if err != nil {
		return fmt.Errorf("check-in after commit: %w", err)
	}

	if oldActiveID != 0 {
		// clean up old deployment storage after successful commit of new deployment
		oldDir := c.deploymentDirPath(oldActiveID)
		c.logger.Debug("cleaning up old deployment storage", zap.Int64("deploymentId", oldActiveID), zap.String("path", oldDir))
		err = c.dir.RemoveAll(oldDir)
		if err != nil {
			return fmt.Errorf("remove old deployment storage: %w", err)
		}
	}

	return nil
}

func (c *DeploymentUpdater) FailInstall(ctx context.Context, message string) error {
	if !c.lock(ctx) {
		return ctx.Err()
	}
	defer c.unlock()

	installingID, err := c.deploymentIDByMark(markInstalling)
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}

	activeID, err := c.deploymentIDByMark(markActive)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("get active deployment id: %w", err)
	}
	// if there is no active deployment, then activeID = 0

	c.logger.Error("marking deployment as failed to install", zap.Int64("deploymentId", installingID), zap.String("reason", message))
	err = c.clearMark(markInstalling)
	if err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}

	// tell server that we failed to install the deployment
	req := CheckInRequest{
		FailedDeployment: &CheckInFailedDeployment{ID: installingID, Reason: message},
	}
	if activeID != 0 {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: activeID}
	}
	_, err = c.client.CheckIn(ctx, req)
	if err != nil {
		return fmt.Errorf("check-in after rollback: %w", err)
	}

	return nil
}

// ActiveConfig returns an fs.FS containing the config files for the currently active deployment.
// Unlike InstallingConfig, this has already been committed, so should be considered safe.
// Returns a nil fs.FS with no error if there is no active deployment.
func (c *DeploymentUpdater) ActiveConfig() (fs.FS, error) {
	return c.extractedConfigByMark(markActive)
}

func (c *DeploymentUpdater) extractedConfigByMark(name string) (fs.FS, error) {
	root, err := c.deploymentRootByMark(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	extractedRoot, err := root.OpenRoot(dirConfigVersion)
	if err != nil {
		return nil, fmt.Errorf("open extracted config version directory: %w", err)
	}
	return extractedRoot.FS(), nil
}

// PollOnce performs a check-in with the server and updates the local state of the updater accordingly.
// If there is a pending deployment, we report to the server that it's in progress, the config version package will
// be downloaded and extracted, and marked as the installing config returned by InstallingConfig.
// In this case, needReboot will be true, indicating that the node should restart to apply the new config.
func (c *DeploymentUpdater) PollOnce(ctx context.Context) (needReboot bool, err error) {
	if !c.lock(ctx) {
		return false, ctx.Err()
	}
	defer c.unlock()

	// ID = 0 is a placeholder for "no such mark"
	activeDeploymentID, err := c.deploymentIDByMark(markActive)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("get active deployment id: %w", err)
	}
	installingDeploymentID, err := c.deploymentIDByMark(markInstalling)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("get installing deployment id: %w", err)
	}

	req := CheckInRequest{}
	if activeDeploymentID != 0 {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: activeDeploymentID}
	}
	if installingDeploymentID != 0 {
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: installingDeploymentID}
	}

	c.logger.Debug("checking in with deployment server", zap.Int64("activeDeploymentId", activeDeploymentID), zap.Int64("installingDeploymentId", installingDeploymentID))
	resp, err := c.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in: %w", err)
	}

	// if we are already installing a deployment, then we won't start installing another one
	if installingDeploymentID != 0 {
		return true, nil
	}

	// find out if there is a new deployment to install
	latest := resp.LatestConfig
	if latest == nil {
		return false, nil
	}
	if latest.Deployment.ID == activeDeploymentID {
		// already on the latest deployment, nothing to do
		return false, nil
	}

	// report that we will install the new deployment
	c.logger.Info("new deployment available, starting installation", zap.Int64("deploymentId", latest.Deployment.ID), zap.Int64("configVersionId", latest.ConfigVersion.ID))
	req.InstallingDeployment = &CheckInInstallingDeployment{ID: latest.Deployment.ID}
	_, err = c.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in to report installing: %w", err)
	}

	// download config version
	err = c.downloadConfigVersion(ctx, latest.Deployment, latest.ConfigVersion)
	if err != nil {
		c.logger.Error("failed to download config version package", zap.Int64("deploymentId", latest.Deployment.ID), zap.Int64("configVersionId", latest.ConfigVersion.ID), zap.Error(err))
		// report transient failure to install the deployment
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: latest.Deployment.ID, Error: err.Error()}
		_, reportErr := c.client.CheckIn(ctx, req)
		return false, errors.Join(fmt.Errorf("download config version: %w", err), reportErr)
	}

	// mark the new deployment as installing for when the node reboots
	err = c.mark(markInstalling, latest.Deployment.ID)
	if err != nil {
		return false, fmt.Errorf("mark installing deployment: %w", err)
	}

	c.logger.Info("new deployment ready to install on next boot", zap.Int64("deploymentId", latest.Deployment.ID), zap.Int64("configVersionId", latest.ConfigVersion.ID))
	return true, nil
}

// AutoPoll will continuously poll the server for new deployments and apply them as they come in, until the context is
// cancelled, or we determine that the controller needs to reboot.
// Will poll every interval. Panics if interval <= 0.
func (c *DeploymentUpdater) AutoPoll(ctx context.Context, interval time.Duration) (needsReboot bool) {
	if interval <= 0 {
		panic("invalid interval")
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
			needReboot, err := c.PollOnce(ctx)
			if err != nil {
				c.logger.Error("failed to poll deployment server", zap.Error(err))
			} else if needReboot {
				return true
			}
		}
	}
}

func (c *DeploymentUpdater) mark(name string, deploymentID int64) error {
	err := c.clearMark(name)
	if err != nil {
		return err
	}
	linkPath := filepath.Join(dirDeployments, name)
	err = c.dir.Symlink(strconv.FormatInt(deploymentID, 10), linkPath)
	if err != nil {
		return fmt.Errorf("create %s symlink: %w", name, err)
	}
	return nil
}

func (c *DeploymentUpdater) clearMark(name string) error {
	linkPath := filepath.Join(dirDeployments, name)
	err := c.dir.Remove(linkPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old %s link: %w", name, err)
	}
	return nil
}

func (c *DeploymentUpdater) downloadConfigVersion(ctx context.Context, deployment Deployment, configVersion ConfigVersion) error {
	dstDir, err := c.deploymentRoot(deployment.ID)
	if err != nil {
		return fmt.Errorf("open deployment storage: %w", err)
	}

	body, err := c.client.DownloadPayload(ctx, configVersion.PayloadURL)
	if err != nil {
		return err
	}
	defer body.Close()

	// extract tar.gz directly from response body stream to avoid storing the entire package on disk twice
	extractedName := dirConfigVersion
	err = dstDir.MkdirAll(extractedName, 0755)
	if err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	extractRoot, err := dstDir.OpenRoot(extractedName)
	if err != nil {
		return fmt.Errorf("open extraction directory: %w", err)
	}
	err = extractTarGZ(body, extractRoot)
	if err != nil {
		// If the extract failed, we don't want to leave a partially extracted package around, as it may interfere
		// with a future successful attempt.
		// It's still useful to leave the remains around to assist with debugging, but we should move it out of the way.
		moveTo := fmt.Sprintf("failed-download-%d", time.Now().UnixMilli())
		c.logger.Error("failed to download and extract config version package",
			zap.Int64("deploymentId", deployment.ID),
			zap.Int64("configVersionId", configVersion.ID),
			zap.Error(err),
			zap.String("moveFailedExtractTo", moveTo),
		)
		moveErr := dstDir.Rename(extractedName, moveTo)
		return errors.Join(fmt.Errorf("extract config version package: %w", err), moveErr)
	}

	return nil
}

func (c *DeploymentUpdater) deploymentDirPath(deploymentID int64) string {
	return filepath.Join(dirDeployments, strconv.FormatInt(deploymentID, 10))
}

func (c *DeploymentUpdater) deploymentRoot(deploymentID int64) (*os.Root, error) {
	dir := c.deploymentDirPath(deploymentID)
	err := c.dir.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("create deployment storage: %w", err)
	}

	return c.dir.OpenRoot(dir)
}

func (c *DeploymentUpdater) deploymentRootByMark(name string) (*os.Root, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := c.dir.Readlink(linkPath)
	if err != nil {
		return nil, fmt.Errorf("read %s link: %w", name, err)
	}
	return c.dir.OpenRoot(filepath.Join(dirDeployments, target))
}

func (c *DeploymentUpdater) deploymentIDByMark(name string) (int64, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := c.dir.Readlink(linkPath)
	if err != nil {
		return 0, fmt.Errorf("read %s link: %w", name, err)
	}
	id, err := strconv.ParseInt(filepath.Base(target), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse deployment id from link target: %w", err)
	}
	return id, nil
}

func (c *DeploymentUpdater) lock(ctx context.Context) (ok bool) {
	select {
	case c.lockCh <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// only call when holding the lock
func (c *DeploymentUpdater) unlock() {
	select {
	case <-c.lockCh:
	default:
		panic("unlock called without lock held")
	}
}

func extractTarGZ(src io.Reader, dst *os.Root) error {
	srcUncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer func() {
		_ = srcUncompressed.Close()
	}()

	tarReader := tar.NewReader(srcUncompressed)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = dst.MkdirAll(header.Name, 0755)
			if err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		case tar.TypeReg:
			err = copyFile(dst, header.Name, tarReader, false)
			if err != nil {
				return fmt.Errorf("copy file: %w", err)
			}

		default:
			return fmt.Errorf("unsupported tar entry type: %c in file %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func copyFile(root *os.Root, path string, src io.Reader, mkdir bool) error {
	if mkdir {
		dirName := filepath.Dir(path)
		err := root.MkdirAll(dirName, 0755)
		if err != nil {
			return fmt.Errorf("create parent directories: %w", err)
		}
	}

	dstFile, err := root.Create(path)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, src)
	if err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}
	return nil
}

const (
	markInstalling = "installing"
	markActive     = "active"
)

const (
	dirDeployments   = "deployments"
	dirConfigVersion = "config-version"
)
