package cloud

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// DeploymentClient is an HTTP client for the cloud check-in API.
type DeploymentClient struct {
	dir        *os.Root
	baseURL    string
	secret     string
	httpClient *http.Client
	logger     *zap.Logger
}

// Option configures a DeploymentClient.
type Option func(*DeploymentClient)

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(c *http.Client) Option {
	return func(client *DeploymentClient) {
		client.httpClient = c
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(client *DeploymentClient) {
		client.logger = logger
	}
}

// OpenDeploymentClient creates a new DeploymentClient.
// The storeDir is used to store config version packages, and retain state between BOS boots.
// The secret is used verbatim as the Bearer token value in authenticated requests to the server.
func OpenDeploymentClient(storeDir *os.Root, baseURL string, secret string, opts ...Option) (*DeploymentClient, error) {
	c := &DeploymentClient{
		dir:        storeDir,
		baseURL:    baseURL,
		secret:     secret,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.logger == nil {
		c.logger = zap.NewNop()
	}

	// TODO: load currentDeploymentID from disk to retain state across reboots
	return c, nil
}

// InstallingConfig returns an fs.FS containing the config files for the currently installing deployment, which has
// not yet been commited. Once the deployment has been applied, call either CommitInstall or FailInstall.
// Returns a nil fs.FS with no error if there is no deployment currently being installed.
func (c *DeploymentClient) InstallingConfig() (fs.FS, error) {
	return c.extractedConfigByMark(markInstalling)
}

func (c *DeploymentClient) CommitInstall(ctx context.Context) error {
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
	_, err = c.doCheckIn(ctx, CheckInRequest{
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

func (c *DeploymentClient) FailInstall(ctx context.Context, message string) error {
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
	_, err = c.doCheckIn(ctx, req)
	if err != nil {
		return fmt.Errorf("check-in after rollback: %w", err)
	}

	return nil
}

// ActiveConfig returns an fs.FS containing the config files for the currently active deployment.
// Unlike InstallingConfig, this has already been committed, so should be considered safe.
// Returns a nil fs.FS with no error if there is no active deployment.
func (c *DeploymentClient) ActiveConfig() (fs.FS, error) {
	return c.extractedConfigByMark(markActive)
}

func (c *DeploymentClient) extractedConfigByMark(name string) (fs.FS, error) {
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

// PollOnce performs a check-in with the server and updates the local state of the client accordingly.
// If there is a pending deployment, we report to the server that it's in progress, the config version package will
// be downloaded and extracted, and marked as the installing config returned by InstallingConfig.
// In this case, needReboot will be true, indicating that the node should restart to apply the new config.
func (c *DeploymentClient) PollOnce(ctx context.Context) (needReboot bool, err error) {
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
	resp, err := c.doCheckIn(ctx, req)
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
	_, err = c.doCheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in to report installing: %w", err)
	}

	// download config version
	err = c.downloadConfigVersion(ctx, latest.Deployment, latest.ConfigVersion)
	if err != nil {
		c.logger.Error("failed to download config version package", zap.Int64("deploymentId", latest.Deployment.ID), zap.Int64("configVersionId", latest.ConfigVersion.ID), zap.Error(err))
		// report transient failure to install the deployment
		req.InstallingDeployment = &CheckInInstallingDeployment{ID: latest.Deployment.ID, Error: err.Error()}
		_, reportErr := c.doCheckIn(ctx, req)
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
func (c *DeploymentClient) AutoPoll(ctx context.Context, interval time.Duration) (needsReboot bool) {
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

func (c *DeploymentClient) mark(name string, deploymentID int64) error {
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

func (c *DeploymentClient) clearMark(name string) error {
	linkPath := filepath.Join(dirDeployments, name)
	err := c.dir.Remove(linkPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old %s link: %w", name, err)
	}
	return nil
}

func (c *DeploymentClient) downloadConfigVersion(ctx context.Context, deployment Deployment, configVersion ConfigVersion) error {
	dstDir, err := c.deploymentRoot(deployment.ID)
	if err != nil {
		return fmt.Errorf("open deployment storage: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configVersion.PayloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// start download
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("download config version: server returned status %d", httpResp.StatusCode)
	}

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
	err = extractTarGZ(httpResp.Body, extractRoot)
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

func (c *DeploymentClient) deploymentDirPath(deploymentID int64) string {
	return filepath.Join(dirDeployments, strconv.FormatInt(deploymentID, 10))
}

func (c *DeploymentClient) deploymentRoot(deploymentID int64) (*os.Root, error) {
	dir := c.deploymentDirPath(deploymentID)
	err := c.dir.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("create deployment storage: %w", err)
	}

	return c.dir.OpenRoot(dir)
}

func (c *DeploymentClient) deploymentRootByMark(name string) (*os.Root, error) {
	linkPath := filepath.Join(dirDeployments, name)
	target, err := c.dir.Readlink(linkPath)
	if err != nil {
		return nil, fmt.Errorf("read %s link: %w", name, err)
	}
	return c.dir.OpenRoot(filepath.Join(dirDeployments, target))
}

func (c *DeploymentClient) deploymentIDByMark(name string) (int64, error) {
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

// doCheckIn sends a POST request to the check-in endpoint and returns the server response.
// A zero-valued req is valid; the server accepts an empty body.
//
// The error return may be an *APIError which contains additional details about the error response from the server.
func (c *DeploymentClient) doCheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/check-in", bytes.NewReader(body))
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.secret)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	// cap at 1 MiB — the check-in response is a small JSON payload
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxCheckInBodySize))
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var apiErr APIError
		decodeErr := json.Unmarshal(respBody, &apiErr)
		apiErr.StatusCode = httpResp.StatusCode
		return CheckInResponse{}, errors.Join(&apiErr, decodeErr)
	}

	var resp CheckInResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return CheckInResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return resp, nil
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

const maxCheckInBodySize = 1024 * 1024 // 1 MiB
