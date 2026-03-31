package cloud

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
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

func WithPreserveDownloads(preserveDownloads bool) UpdaterOption {
	return func(u *DeploymentUpdater) {
		u.preserveDownloads = preserveDownloads
	}
}

// WithMaxDeploymentSize sets the maximum allowed size of a decompressed deployment config, in bytes. If a deployment
// exceeds this size, it will be rejected and not installed. This is to prevent the updater from filling up disk space
// with huge deployments, which is probably a sign of a bad deployment package.
// If maxSize <= 0, then there is no limit, which is the default behaviour.
func WithMaxDeploymentSize(maxSize int64) UpdaterOption {
	return func(u *DeploymentUpdater) {
		u.maxDeploymentSize = maxSize
	}
}

// DeploymentUpdater manages deployment state on disk and coordinates with the cloud check-in API.
//
// Methods are safe to call concurrently.
type DeploymentUpdater struct {
	dir               *os.Root
	client            Client
	logger            *zap.Logger
	preserveDownloads bool
	maxDeploymentSize int64 // max size, in bytes, of a decompressed deployment config to permit
	lockCh            chan struct{}
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

// InstallingConfig returns an FS containing the config files for the currently installing deployment, which has
// not yet been committed. Once the deployment has been applied, call either CommitInstall or FailInstall.
// Returns a nil FS with no error if there is no deployment currently being installed.
//
// Returned FS must be closed after use to prevent a resource leak.
func (c *DeploymentUpdater) InstallingConfig() (FS, error) {
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
	// if there is no active deployment, then oldActiveID = ""

	// clean installing mark before adding active mark to prevent edge case where both point to the same deployment
	// if we stop in between
	err = c.clearMark(markInstalling)
	if err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}
	err = c.mark(markActive, installingID)
	if err != nil {
		return fmt.Errorf("mark active deployment: %w", err)
	}
	c.logger.Info("marked deployment as active", zap.String("deploymentId", installingID))

	// tell server that we have installed the deployment
	_, err = c.client.CheckIn(ctx, CheckInRequest{
		CurrentDeployment: &CheckInDeploymentRef{ID: installingID},
	})
	if err != nil {
		// it's not vital we report this right here, as it's already committed locally so will be reported at the
		// next check-in anyway.
		c.logger.Warn("failed to report successful install to server", zap.String("deploymentId", installingID), zap.Error(err))
	}

	if oldActiveID != "" && !c.preserveDownloads {
		// clean up old deployment storage after successful commit of new deployment
		oldDir := c.deploymentDirPath(oldActiveID)
		c.logger.Debug("cleaning up old deployment storage", zap.String("deploymentId", oldActiveID), zap.String("path", oldDir))
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
	// if there is no active deployment, then activeID = ""

	c.logger.Error("marking deployment as failed to install", zap.String("deploymentId", installingID), zap.String("reason", message))
	err = c.clearMark(markInstalling)
	if err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}

	// tell server that we failed to install the deployment
	req := CheckInRequest{
		FailedDeployment: &CheckInFailedDeployment{ID: installingID, Reason: message},
	}
	if activeID != "" {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: activeID}
	}
	_, err = c.client.CheckIn(ctx, req)
	if err != nil {
		c.logger.Warn("failed to report install failure to server", zap.String("deploymentId", installingID), zap.Error(err))
	}

	if installingID != "" && !c.preserveDownloads {
		// clean up failed deployment storage
		dir := c.deploymentDirPath(installingID)
		c.logger.Debug("cleaning up failed deployment storage", zap.String("deploymentId", installingID), zap.String("path", dir))
		err = c.dir.RemoveAll(dir)
		if err != nil {
			return fmt.Errorf("remove failed deployment storage: %w", err)
		}
	}

	return nil
}

// ActiveConfig returns an FS containing the config files for the currently active deployment.
// Unlike InstallingConfig, this has already been committed, so should be considered safe.
// Returns a nil FS with no error if there is no active deployment.
//
// Returned FS must be closed after use to prevent a resource leak.
func (c *DeploymentUpdater) ActiveConfig() (FS, error) {
	return c.extractedConfigByMark(markActive)
}

func (c *DeploymentUpdater) extractedConfigByMark(name string) (FS, error) {
	root, err := c.deploymentRootByMark(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer func() {
		_ = root.Close()
	}()

	extractedRoot, err := root.OpenRoot(dirConfigVersion)
	if err != nil {
		return nil, fmt.Errorf("open extracted config version directory: %w", err)
	}
	return &rootFS{FS: extractedRoot.FS(), root: extractedRoot}, nil
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

	// empty string is a placeholder for "no such mark"
	activeDeploymentID, err := c.deploymentIDByMark(markActive)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("get active deployment id: %w", err)
	}
	installingDeploymentID, err := c.deploymentIDByMark(markInstalling)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("get installing deployment id: %w", err)
	}

	// catch store corruption where both active and installing marks point to the same deployment
	// in that case we can ignore the installing mark because it's already installed, but we should correct it
	if activeDeploymentID != "" && activeDeploymentID == installingDeploymentID {
		c.logger.Warn("corrupted store - active and installing deployment are the same", zap.String("deploymentId", activeDeploymentID))
		err = c.clearMark(markInstalling)
		if err != nil {
			return false, fmt.Errorf("clear installing mark: %w", err)
		}
		installingDeploymentID = ""
	}

	req := CheckInRequest{}
	if activeDeploymentID != "" {
		req.CurrentDeployment = &CheckInDeploymentRef{ID: activeDeploymentID}
	}
	if installingDeploymentID != "" {
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: installingDeploymentID}
	}

	c.logger.Debug("checking in with deployment server", zap.String("activeDeploymentId", activeDeploymentID), zap.String("installingDeploymentId", installingDeploymentID))
	resp, err := c.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in: %w", err)
	}

	// if we are already installing a deployment, then we won't start installing another one
	if installingDeploymentID != "" {
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
	c.logger.Info("new deployment available, starting installation", zap.String("deploymentId", latest.Deployment.ID), zap.String("configVersionId", latest.ConfigVersion.ID))
	req.InstallingDeployment = &CheckInInstallingDeployment{ID: latest.Deployment.ID}
	_, err = c.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in to report installing: %w", err)
	}

	// download config version
	err = c.downloadConfigVersion(ctx, latest.Deployment, latest.ConfigVersion)
	if err != nil {
		c.logger.Error("failed to download config version package", zap.String("deploymentId", latest.Deployment.ID), zap.String("configVersionId", latest.ConfigVersion.ID), zap.Error(err))
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

	c.logger.Info("new deployment ready to install on next boot", zap.String("deploymentId", latest.Deployment.ID), zap.String("configVersionId", latest.ConfigVersion.ID))
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

func (c *DeploymentUpdater) mark(name string, deploymentID string) error {
	err := c.clearMark(name)
	if err != nil {
		return err
	}
	linkPath := filepath.Join(dirDeployments, name)
	err = c.dir.Symlink(deploymentID, linkPath)
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
	var deleteDstDir bool // remove empty directory
	defer func() {
		_ = dstDir.Close()
		if deleteDstDir {
			dir := c.deploymentDirPath(deployment.ID)
			if err := c.dir.Remove(dir); err != nil {
				c.logger.Warn("failed to delete empty directory")
			}
		}
	}()

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
	err = extractTarGZ(body, extractRoot, c.maxDeploymentSize)
	_ = extractRoot.Close()
	if err != nil {
		var cleanupErr error
		if c.preserveDownloads {
			// If the extract failed, we don't want to leave a partially extracted package in place, as it may interfere
			// with a future successful attempt.
			// It's still useful to leave the remains around to assist with debugging, but we should move it out of the way.
			moveTo := fmt.Sprintf("failed-download-%d", time.Now().UnixMilli())
			c.logger.Error("failed to download and extract config version package",
				zap.String("deploymentId", deployment.ID),
				zap.String("configVersionId", configVersion.ID),
				zap.Error(err),
				zap.String("moveFailedExtractTo", moveTo),
			)
			cleanupErr = dstDir.Rename(extractedName, moveTo)
		} else {
			// delete the extracted directory
			cleanupErr = dstDir.RemoveAll(extractedName)
			deleteDstDir = true // dstDir will be empty at this point, remove it to limit filesystem clutter
		}
		return errors.Join(fmt.Errorf("extract config version package: %w", err), cleanupErr)
	}

	return nil
}

func (c *DeploymentUpdater) deploymentDirPath(deploymentID string) string {
	return filepath.Join(dirDeployments, deploymentID)
}

func (c *DeploymentUpdater) deploymentRoot(deploymentID string) (*os.Root, error) {
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

func (c *DeploymentUpdater) deploymentIDByMark(name string) (string, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := c.dir.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("read %s link: %w", name, err)
	}
	return filepath.Base(target), nil
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

type FS interface {
	fs.FS
	Close() error
}

type rootFS struct {
	fs.FS
	root *os.Root
}

func (r *rootFS) Close() error {
	return r.root.Close()
}

// maxSize, if positive, limits the size of the decompressed tarball. Decompression will stop early with an error
// if more than maxSize bytes are decompressed.
//
// We apply the limit to the decompressed data rather than the compressed date to catch ZIP bombs, where a small
// file decompresses to a huge size.
func extractTarGZ(src io.Reader, dst *os.Root, maxSize int64) error {
	srcUncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer func() {
		_ = srcUncompressed.Close()
	}()

	var srcLimited io.Reader = srcUncompressed
	if maxSize > 0 {
		srcLimited = limitReader(srcUncompressed, maxSize)
	}

	tarReader := tar.NewReader(srcLimited)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, errLimitReached) {
			return sizeLimitError(maxSize)
		} else if errors.Is(err, io.EOF) {
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
			if errors.Is(err, errLimitReached) {
				return sizeLimitError(maxSize)
			} else if err != nil {
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

// like io.LimitReader but returns a more specific error message when the limit is reached, so we can distinguish
// it from an io.EOF from the source reader
type limitedReader struct {
	R io.Reader
	N int64
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, errLimitReached
	}
	// never read more than the N limit, even if the caller passed in a larger buffer
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return n, err
}

func limitReader(src io.Reader, n int64) io.Reader {
	return &limitedReader{R: src, N: n}
}

var errLimitReached = errors.New("read limit reached")

func humaniseBytes(n int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	size := float64(n)
	for len(units) > 1 && math.Abs(size) >= 1024 {
		size /= 1024
		units = units[1:]
	}
	return fmt.Sprintf("%.3g %s", size, units[0])
}

func sizeLimitError(maxSize int64) error {
	return fmt.Errorf("deployment package exceeds size limit of %s", humaniseBytes(maxSize))
}

const (
	markInstalling = "installing"
	markActive     = "active"
)

const (
	dirDeployments   = "deployments"
	dirConfigVersion = "config-version"
)
