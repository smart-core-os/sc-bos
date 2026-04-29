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

const (
	markInstalling = "installing"
	markActive     = "active"

	dirDeployments   = "deployments"
	dirConfigVersion = "config-version"
)

// FS is a closeable fs.FS returned by DeploymentStore config accessors.
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

// StoreOption configures a DeploymentStore.
type StoreOption func(*DeploymentStore)

// WithStoreLogger sets the logger for a DeploymentStore.
func WithStoreLogger(logger *zap.Logger) StoreOption {
	return func(s *DeploymentStore) { s.logger = logger }
}

// WithPreserveDownloads controls whether failed or superseded deployment packages
// are kept on disk rather than deleted.
func WithPreserveDownloads(preserveDownloads bool) StoreOption {
	return func(s *DeploymentStore) { s.preserveDownloads = preserveDownloads }
}

// WithMaxDeploymentSize sets the maximum allowed decompressed size (bytes) of a deployment
// package. Packages exceeding this limit are rejected. Zero or negative means no limit.
func WithMaxDeploymentSize(maxSize int64) StoreOption {
	return func(s *DeploymentStore) { s.maxDeploymentSize = maxSize }
}

// DeploymentStore manages deployment state on disk.
// It handles marks (symlinks), extraction of config payloads, and per-deployment directories.
// It has no knowledge of the cloud API.
//
// Methods are safe to call concurrently.
type DeploymentStore struct {
	dir               *os.Root
	logger            *zap.Logger
	preserveDownloads bool
	maxDeploymentSize int64
	lockCh            chan struct{}
}

// NewDeploymentStore creates a new DeploymentStore rooted at dir.
func NewDeploymentStore(dir *os.Root, opts ...StoreOption) *DeploymentStore {
	s := &DeploymentStore{
		dir:    dir,
		lockCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.logger == nil {
		s.logger = zap.NewNop()
	}
	return s
}

// Close closes the underlying filesystem root.
func (s *DeploymentStore) Close() error {
	return s.dir.Close()
}

// ActiveID returns the deployment ID of the currently active deployment,
// or "" if there is no active deployment.
func (s *DeploymentStore) ActiveID() (string, error) {
	id, err := s.deploymentIDByMark(markActive)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return id, err
}

// InstallingID returns the deployment ID of the currently installing deployment,
// or "" if there is no installing deployment.
func (s *DeploymentStore) InstallingID() (string, error) {
	id, err := s.deploymentIDByMark(markInstalling)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return id, err
}

// InstallingConfig returns an FS containing the config files for the currently installing
// deployment. Returns nil, nil if no deployment is being installed.
// The caller must close the returned FS.
func (s *DeploymentStore) InstallingConfig() (FS, error) {
	return s.extractedConfigByMark(markInstalling)
}

// ActiveConfig returns an FS containing the config files for the active deployment.
// Returns nil, nil if there is no active deployment.
// The caller must close the returned FS.
func (s *DeploymentStore) ActiveConfig() (FS, error) {
	return s.extractedConfigByMark(markActive)
}

// WriteInstalling extracts the tar.gz payload into a new per-deployment directory
// and marks it as the installing deployment. On failure, the partial directory is
// removed (or renamed if WithPreserveDownloads is set).
func (s *DeploymentStore) WriteInstalling(ctx context.Context, deploymentID string, payload io.Reader) error {
	if !s.lock(ctx) {
		return ctx.Err()
	}
	defer s.unlock()

	dstDir, err := s.deploymentRoot(deploymentID)
	if err != nil {
		return fmt.Errorf("open deployment storage: %w", err)
	}
	var deleteDstDir bool
	defer func() {
		_ = dstDir.Close()
		if deleteDstDir {
			dir := s.deploymentDirPath(deploymentID)
			if err := s.dir.Remove(dir); err != nil {
				s.logger.Warn("failed to delete empty directory")
			}
		}
	}()

	extractedName := dirConfigVersion
	err = dstDir.MkdirAll(extractedName, 0755)
	if err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	extractRoot, err := dstDir.OpenRoot(extractedName)
	if err != nil {
		return fmt.Errorf("open extraction directory: %w", err)
	}
	err = extractTarGZ(payload, extractRoot, s.maxDeploymentSize)
	_ = extractRoot.Close()
	if err != nil {
		var cleanupErr error
		if s.preserveDownloads {
			moveTo := fmt.Sprintf("failed-download-%d", time.Now().UnixMilli())
			s.logger.Error("failed to extract deployment package, moving aside",
				zap.String("deploymentId", deploymentID),
				zap.Error(err),
				zap.String("moveFailedExtractTo", moveTo),
			)
			cleanupErr = dstDir.Rename(extractedName, moveTo)
		} else {
			cleanupErr = dstDir.RemoveAll(extractedName)
			deleteDstDir = true
		}
		return errors.Join(fmt.Errorf("extract config version package: %w", err), cleanupErr)
	}

	if err := s.mark(markInstalling, deploymentID); err != nil {
		return fmt.Errorf("mark installing deployment: %w", err)
	}
	s.logger.Info("new deployment ready to install on next boot", zap.String("deploymentId", deploymentID))
	return nil
}

// CommitInstall promotes the installing deployment to active, removes the old active
// deployment directory (unless WithPreserveDownloads is set), and clears the installing mark.
func (s *DeploymentStore) CommitInstall(ctx context.Context) error {
	if !s.lock(ctx) {
		return ctx.Err()
	}
	defer s.unlock()

	installingID, err := s.deploymentIDByMark(markInstalling)
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}
	oldActiveID, err := s.deploymentIDByMark(markActive)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("get active deployment id: %w", err)
	}

	// clear installing mark before adding active mark to prevent edge case where both point
	// to the same deployment if we stop in between
	if err = s.clearMark(markInstalling); err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}
	if err = s.mark(markActive, installingID); err != nil {
		return fmt.Errorf("mark active deployment: %w", err)
	}
	s.logger.Info("marked deployment as active", zap.String("deploymentId", installingID))

	if oldActiveID != "" && !s.preserveDownloads {
		oldDir := s.deploymentDirPath(oldActiveID)
		s.logger.Debug("cleaning up old deployment storage", zap.String("deploymentId", oldActiveID), zap.String("path", oldDir))
		if err = s.dir.RemoveAll(oldDir); err != nil {
			return fmt.Errorf("remove old deployment storage: %w", err)
		}
	}
	return nil
}

// FailInstall clears the installing mark and removes the installing deployment directory
// (unless WithPreserveDownloads is set).
func (s *DeploymentStore) FailInstall(ctx context.Context) error {
	if !s.lock(ctx) {
		return ctx.Err()
	}
	defer s.unlock()

	installingID, err := s.deploymentIDByMark(markInstalling)
	if err != nil {
		return fmt.Errorf("get installing deployment id: %w", err)
	}

	s.logger.Error("marking deployment as failed to install", zap.String("deploymentId", installingID))
	if err = s.clearMark(markInstalling); err != nil {
		return fmt.Errorf("clear installing mark: %w", err)
	}

	if installingID != "" && !s.preserveDownloads {
		dir := s.deploymentDirPath(installingID)
		s.logger.Debug("cleaning up failed deployment storage", zap.String("deploymentId", installingID), zap.String("path", dir))
		if err = s.dir.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove failed deployment storage: %w", err)
		}
	}
	return nil
}

// ClearInstalling removes the installing mark without deleting the deployment directory.
// Used to recover from store corruption where the installing deployment is the same as active.
func (s *DeploymentStore) ClearInstalling(ctx context.Context) error {
	if !s.lock(ctx) {
		return ctx.Err()
	}
	defer s.unlock()
	return s.clearMark(markInstalling)
}

func (s *DeploymentStore) extractedConfigByMark(name string) (FS, error) {
	root, err := s.deploymentRootByMark(name)
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

func (s *DeploymentStore) mark(name string, deploymentID string) error {
	if err := s.clearMark(name); err != nil {
		return err
	}
	linkPath := filepath.Join(dirDeployments, name)
	if err := s.dir.Symlink(deploymentID, linkPath); err != nil {
		return fmt.Errorf("create %s symlink: %w", name, err)
	}
	return nil
}

func (s *DeploymentStore) clearMark(name string) error {
	linkPath := filepath.Join(dirDeployments, name)
	err := s.dir.Remove(linkPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old %s link: %w", name, err)
	}
	return nil
}

func (s *DeploymentStore) deploymentDirPath(deploymentID string) string {
	return filepath.Join(dirDeployments, deploymentID)
}

func (s *DeploymentStore) deploymentRoot(deploymentID string) (*os.Root, error) {
	dir := s.deploymentDirPath(deploymentID)
	if err := s.dir.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create deployment storage: %w", err)
	}
	return s.dir.OpenRoot(dir)
}

func (s *DeploymentStore) deploymentRootByMark(name string) (*os.Root, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := s.dir.Readlink(linkPath)
	if err != nil {
		return nil, fmt.Errorf("read %s link: %w", name, err)
	}
	return s.dir.OpenRoot(filepath.Join(dirDeployments, target))
}

func (s *DeploymentStore) deploymentIDByMark(name string) (string, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := s.dir.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("read %s link: %w", name, err)
	}
	return filepath.Base(target), nil
}

func (s *DeploymentStore) lock(ctx context.Context) bool {
	select {
	case s.lockCh <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *DeploymentStore) unlock() {
	select {
	case <-s.lockCh:
	default:
		panic("unlock called without lock held")
	}
}

// maxSize, if positive, limits the size of the decompressed tarball.
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
			if err = dst.MkdirAll(header.Name, 0755); err != nil {
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
		if err := root.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create parent directories: %w", err)
		}
	}

	dstFile, err := root.Create(path)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, src); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}
	return nil
}

type limitedReader struct {
	R io.Reader
	N int64
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, errLimitReached
	}
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
