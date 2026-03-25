package cloud

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"
	"golang.org/x/tools/txtar"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
)

// clientEnv holds test fixtures for DeploymentUpdater tests.
type clientEnv struct {
	testServer *httptest.Server
	httpClient *http.Client
	storeDir   *os.Root
	storePath  string // temp dir absolute path for filesystem assertions
	updater    *DeploymentUpdater
	nodeID     int64
}

// setupClientEnv creates a test environment with an httptest server and DeploymentClient.
func setupClientEnv(t *testing.T) *clientEnv {
	t.Helper()

	logger := zap.NewNop()
	s := store.NewMemoryStore(logger)
	t.Cleanup(func() { _ = s.Close() })

	apiServer, err := sim.NewServer(s, logger)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	client := ts.Client()

	// Create site
	var site sim.Site
	resp := doSimRequest(t, client, "POST", ts.URL+"/api/v1/management/sites",
		map[string]string{"name": "Test Site"}, &site)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create site: expected 201, got %d", resp.StatusCode)
	}

	// Create node (capture raw secret bytes)
	var created sim.CreateNodeResponse
	resp = doSimRequest(t, client, "POST", ts.URL+"/api/v1/management/nodes", map[string]any{
		"hostname": "test-node",
		"siteId":   strconv.FormatInt(site.ID, 10),
	}, &created)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create node: expected 201, got %d", resp.StatusCode)
	}

	// Base64-encode raw secret for use with the token endpoint
	encodedSecret := base64.StdEncoding.EncodeToString(created.Secret)

	// Create temp store dir
	storePath := t.TempDir()
	storeDir, err := os.OpenRoot(storePath)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = storeDir.Close() })

	httpClient := NewHTTPClient(Credentials{
		CheckInEndpoint: ts.URL + "/v1/device/check-in",
		TokenEndpoint:   ts.URL + "/v1/device/token",
		ClientID:        strconv.FormatInt(created.ID, 10),
		ClientSecret:    encodedSecret,
	}, WithHTTPClient(ts.Client()))
	updater := NewDeploymentUpdater(storeDir, httpClient)

	return &clientEnv{
		testServer: ts,
		httpClient: client,
		storeDir:   storeDir,
		storePath:  storePath,
		updater:    updater,
		nodeID:     created.ID,
	}
}

//go:embed testdata
var testdata embed.FS

func readTestData(t *testing.T, name string) []byte {
	t.Helper()
	data, err := testdata.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read test data %q: %v", name, err)
	}
	return data
}

func txtarToTarGZ(t *testing.T, testdataName string) []byte {
	txtarData := readTestData(t, testdataName)

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	t.Helper()
	ar := txtar.Parse(txtarData)
	for _, file := range ar.Files {
		if strings.HasSuffix(file.Name, "/") {
			_ = tarWriter.WriteHeader(&tar.Header{
				Name:     file.Name[:len(file.Name)-1], // strip trailing slash for directory entry
				Mode:     0755,
				Typeflag: tar.TypeDir,
			})
		} else if link, target, found := strings.Cut(file.Name, " -> "); found {
			_ = tarWriter.WriteHeader(&tar.Header{
				Name:     link,
				Mode:     0644,
				Typeflag: tar.TypeSymlink,
				Linkname: target,
			})
		} else {
			_ = tarWriter.WriteHeader(&tar.Header{
				Name:     file.Name,
				Mode:     0644,
				Typeflag: tar.TypeReg,
				Size:     int64(len(file.Data)),
			})
			_, _ = tarWriter.Write(file.Data)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// doSimRequest is a JSON HTTP helper for management API calls.
func doSimRequest(t *testing.T, client *http.Client, method, url string, req, res any) *http.Response {
	t.Helper()
	var reqBody *bytes.Buffer
	if req != nil {
		b, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	httpReq, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if req != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if res != nil {
		if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp
}

// createConfigVersion creates a config version with the given payload and returns its ID.
func createConfigVersion(t *testing.T, client *http.Client, baseURL string, nodeID int64, payload []byte) int64 {
	t.Helper()
	var cv sim.ConfigVersion
	resp := doSimRequest(t, client, "POST", baseURL+"/api/v1/management/config-versions", map[string]any{
		"nodeId":      strconv.FormatInt(nodeID, 10),
		"description": "test config",
		"payload":     payload,
	}, &cv)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create config version: expected 201, got %d", resp.StatusCode)
	}
	return cv.ID
}

// createPendingDeployment creates a PENDING deployment for the given config version ID and returns the deployment ID.
func createPendingDeployment(t *testing.T, client *http.Client, baseURL string, configVersionID int64) int64 {
	t.Helper()
	var dep sim.Deployment
	resp := doSimRequest(t, client, "POST", baseURL+"/api/v1/management/deployments", map[string]any{
		"configVersionId": strconv.FormatInt(configVersionID, 10),
		"status":          "pending",
	}, &dep)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create deployment: expected 201, got %d", resp.StatusCode)
	}
	return dep.ID
}

// getDeployment fetches a deployment by ID.
func getDeployment(t *testing.T, client *http.Client, baseURL string, deploymentID int64) sim.Deployment {
	t.Helper()
	var dep sim.Deployment
	resp := doSimRequest(t, client, "GET",
		fmt.Sprintf("%s/api/v1/management/deployments/%d", baseURL, deploymentID),
		nil, &dep)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get deployment: expected 200, got %d", resp.StatusCode)
	}
	return dep
}

// assertFileContent reads a file from an fs.FS and compares its content.
func assertFileContent(t *testing.T, fsys fs.FS, path, want string) {
	t.Helper()
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if got := string(data); strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Errorf("file %q: got %q, want %q", path, got, want)
	}
}

// readSymlinkTarget reads the target of a symlink at the given relative path within dir.
func readSymlinkTarget(t *testing.T, dir, relPath string) string {
	t.Helper()
	target, err := os.Readlink(filepath.Join(dir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("readlink %q: %v", relPath, err)
	}
	return target
}

// symlinkExists returns true if there is a symlink (or any file) at the given relative path within dir.
func symlinkExists(dir, relPath string) bool {
	_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(relPath)))
	return err == nil
}

func countDirectories(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

// TestExtractTarGZ is a unit test for the private extractTarGZ function.
func TestExtractTarGZ(t *testing.T) {
	t.Run("files and directories", func(t *testing.T) {
		data := txtarToTarGZ(t, "nested.txtar")

		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer root.Close()

		if err := extractTarGZ(bytes.NewReader(data), root, -1); err != nil {
			t.Fatalf("extractTarGZ: %v", err)
		}

		fsys := root.FS()
		assertFileContent(t, fsys, "config.json", `{"key":"value"}`)
		assertFileContent(t, fsys, "sub/nested.txt", "nested content")
	})

	t.Run("empty archive", func(t *testing.T) {
		data := txtarToTarGZ(t, "empty.txtar")

		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer root.Close()

		if err := extractTarGZ(bytes.NewReader(data), root, -1); err != nil {
			t.Errorf("expected no error for empty archive, got: %v", err)
		}
	})

	t.Run("invalid gzip", func(t *testing.T) {
		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer root.Close()

		err = extractTarGZ(bytes.NewReader([]byte("not gzip")), root, -1)
		if err == nil {
			t.Error("expected error for invalid gzip, got nil")
		}
	})

	t.Run("unsupported entry type (symlink)", func(t *testing.T) {
		data := txtarToTarGZ(t, "symlink.txtar")

		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer root.Close()

		err = extractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Fatal("expected error for symlink entry, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("expected error to contain %q, got: %v", "unsupported", err)
		}
	})

	t.Run("truncated tar", func(t *testing.T) {
		// Valid gzip wrapping truncated tar data
		data := txtarToTarGZ(t, "nested.txtar")
		data = data[:len(data)/2]

		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer root.Close()

		err = extractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Error("expected error for truncated tar, got nil")
		}
	})

	// oversized files of various magnitudes
	type oversizeTest struct {
		name string
		text string
		size int64
	}
	oversizeTests := []oversizeTest{
		{"oversized file bytes", "123 B", 123},
		{"oversized file kibibytes", "100 KiB", 100 * 1024},
		{"oversized file mebibytes", "15 MiB", 15 * 1024 * 1024},
	}
	for _, ot := range oversizeTests {
		t.Run(ot.name, func(t *testing.T) {
			// create a tar.gz with a single file of the specified size
			// with the tar headers included, this will exceed the size limit
			data := genLargeFileTarGZ(t, ot.size)

			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatalf("open root: %v", err)
			}
			defer root.Close()

			err = extractTarGZ(bytes.NewReader(data), root, ot.size)
			t.Log(err)
			if err == nil {
				t.Fatalf("expected error for oversized file of size %d, got nil", ot.size)
			}
			if !strings.Contains(err.Error(), ot.text) {
				t.Errorf("expected error to contain %q, got: %v", ot.text, err)
			}
		})
	}
}

func TestPoll(t *testing.T) {
	ctx := context.Background()

	t.Run("no deployment available", func(t *testing.T) {
		env := setupClientEnv(t)

		needReboot, err := env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("PollOnce: %v", err)
		}
		if needReboot {
			t.Error("expected needReboot=false when no deployment available")
		}
	})

	t.Run("new deployment downloads and installs", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		needReboot, err := env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("PollOnce: %v", err)
		}
		if !needReboot {
			t.Fatal("expected needReboot=true after new deployment")
		}

		// Installing symlink should exist pointing to the deployment ID
		if !symlinkExists(env.storePath, "deployments/installing") {
			t.Fatal("expected installing symlink to exist")
		}
		target := readSymlinkTarget(t, env.storePath, "deployments/installing")
		if target != fmt.Sprintf("%d", depID) {
			t.Errorf("installing symlink target = %q, want %q", target, fmt.Sprintf("%d", depID))
		}

		// Extracted files should be readable at deployments/<id>/config-version/
		assertFileContent(t, os.DirFS(env.storePath), path.Join("deployments", target, "config-version", "config.json"), `{"key":"value"}`)
	})

	t.Run("already on latest deployment", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		// First poll: installs deployment
		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("first PollOnce: %v", err)
		}

		// CommitInstall to mark it active
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall: %v", err)
		}

		// Second poll: no new deployment (completed one is gone from active list)
		needReboot, err := env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("second PollOnce: %v", err)
		}
		if needReboot {
			t.Error("expected needReboot=false when already on latest deployment")
		}
	})

	t.Run("already installing", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		// First poll marks installing
		needReboot, err := env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("first PollOnce: %v", err)
		}
		if !needReboot {
			t.Fatal("expected needReboot=true on first PollOnce")
		}

		// Second poll without commit — already installing, should return needReboot=true
		needReboot, err = env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("second PollOnce: %v", err)
		}
		if !needReboot {
			t.Error("expected needReboot=true when already installing")
		}
	})

	t.Run("invalid tarball payload", func(t *testing.T) {
		env := setupClientEnv(t)

		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, []byte("not a tarball"))
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		needReboot, err := env.updater.PollOnce(ctx)
		if err == nil {
			t.Error("expected error for invalid tarball payload, got nil")
		}
		if needReboot {
			t.Error("expected needReboot=false when tarball extraction fails")
		}

		// No installing symlink should exist
		if symlinkExists(env.storePath, "deployments/installing") {
			t.Error("expected installing symlink to not exist after failed extraction")
		}
		// No deployment directories should be left
		if numDirs := countDirectories(t, filepath.Join(env.storePath, "deployments")); numDirs != 0 {
			t.Errorf("expected 0 deployment directories after failed extraction, found %d", numDirs)
		}
	})

	t.Run("corrupted store - active and installing point to same deployment", func(t *testing.T) {
		env := setupClientEnv(t)

		// Deploy and commit a deployment to establish an active mark.
		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall: %v", err)
		}

		// Simulate corruption: manually create an installing symlink pointing to the same
		// deployment that is already active. This can happen if the process is interrupted
		// between clearing the installing mark and setting the active mark in CommitInstall.
		installingLink := filepath.Join(env.storePath, "deployments", "installing")
		if err := os.Symlink(fmt.Sprintf("%d", depID), installingLink); err != nil {
			t.Fatalf("create corrupted installing symlink: %v", err)
		}

		// PollOnce should detect and recover from the corruption.
		needReboot, err := env.updater.PollOnce(ctx)
		if err != nil {
			t.Fatalf("PollOnce with corrupted store: %v", err)
		}
		if needReboot {
			t.Error("expected needReboot=false when active and installing point to same deployment")
		}

		// The installing symlink must be cleared as part of recovery.
		if symlinkExists(env.storePath, "deployments/installing") {
			t.Error("expected installing symlink to be removed after corruption recovery")
		}
	})

	t.Run("server auth error", func(t *testing.T) {
		env := setupClientEnv(t)

		// Create an updater with a wrong client secret — token endpoint will reject it
		badHTTPClient := NewHTTPClient(Credentials{
			CheckInEndpoint: env.testServer.URL + "/v1/device/check-in",
			TokenEndpoint:   env.testServer.URL + "/v1/device/token",
			ClientID:        "999999",
			ClientSecret:    "wrong-secret",
		}, WithHTTPClient(env.httpClient))
		badUpdater := NewDeploymentUpdater(env.storeDir, badHTTPClient)

		_, err := badUpdater.PollOnce(ctx)
		if err == nil {
			t.Error("expected error with wrong secret, got nil")
		}
	})
}

func TestCommit(t *testing.T) {
	ctx := context.Background()

	t.Run("normal commit", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}

		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall: %v", err)
		}

		// Active symlink should point to the deployment
		if !symlinkExists(env.storePath, "deployments/active") {
			t.Fatal("expected active symlink to exist after commit")
		}
		activeTarget := readSymlinkTarget(t, env.storePath, "deployments/active")
		if activeTarget != fmt.Sprintf("%d", depID) {
			t.Errorf("active symlink target = %q, want %q", activeTarget, fmt.Sprintf("%d", depID))
		}

		// Installing symlink should be gone
		if symlinkExists(env.storePath, "deployments/installing") {
			t.Error("expected installing symlink to be removed after commit")
		}

		// Server deployment should be COMPLETED
		dep := getDeployment(t, env.httpClient, env.testServer.URL, depID)
		if dep.Status != "completed" {
			t.Errorf("deployment status = %q, want COMPLETED", dep.Status)
		}
	})

	t.Run("no prior active deployment", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}

		// CommitInstall with no prior active — should not error
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Errorf("CommitInstall with no prior active: %v", err)
		}
	})

	t.Run("cleans up old deployment dir", func(t *testing.T) {
		env := setupClientEnv(t)

		// Deploy and commit v1
		payload1 := txtarToTarGZ(t, "v1.txtar")
		cvID1 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload1)
		depID1 := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID1)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce v1: %v", err)
		}
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall v1: %v", err)
		}

		dep1Dir := filepath.Join(env.storePath, "deployments", fmt.Sprintf("%d", depID1))
		if _, err := os.Stat(dep1Dir); err != nil {
			t.Fatalf("v1 deployment dir should exist after commit: %v", err)
		}

		// Deploy and commit v2
		payload2 := txtarToTarGZ(t, "v2.txtar")
		cvID2 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload2)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID2)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce v2: %v", err)
		}
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall v2: %v", err)
		}

		// v1 directory should have been cleaned up
		if _, err := os.Stat(dep1Dir); err == nil {
			t.Error("expected v1 deployment dir to be removed after v2 commit")
		}
	})
}

func TestRollback(t *testing.T) {
	ctx := context.Background()

	t.Run("clears installing mark", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		// starts installing
		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}

		if err := env.updater.FailInstall(ctx, "test reason"); err != nil {
			t.Fatalf("FailInstall: %v", err)
		}

		// Installing symlink should be removed
		if symlinkExists(env.storePath, "deployments/installing") {
			t.Error("expected installing symlink to be removed after rollback")
		}

		// Server deployment should be FAILED with the reason
		dep := getDeployment(t, env.httpClient, env.testServer.URL, depID)
		if dep.Status != "failed" {
			t.Errorf("deployment status = %q, want FAILED", dep.Status)
		}
		if dep.Reason != "test reason" {
			t.Errorf("deployment reason = %q, want %q", dep.Reason, "test reason")
		}
	})

	t.Run("with active deployment", func(t *testing.T) {
		env := setupClientEnv(t)

		// CommitInstall v1
		payload1 := txtarToTarGZ(t, "v1.txtar")
		cvID1 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload1)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID1)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce v1: %v", err)
		}
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall v1: %v", err)
		}

		// PollOnce v2
		payload2 := txtarToTarGZ(t, "v2.txtar")
		cvID2 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload2)
		depID2 := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID2)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce v2: %v", err)
		}

		// FailInstall v2
		if err := env.updater.FailInstall(ctx, "rolled back"); err != nil {
			t.Fatalf("FailInstall: %v", err)
		}

		// Active symlink should still exist (v1 still active)
		if !symlinkExists(env.storePath, "deployments/active") {
			t.Error("expected active symlink to still exist after rollback")
		}

		// Installing symlink should be removed
		if symlinkExists(env.storePath, "deployments/installing") {
			t.Error("expected installing symlink to be removed after rollback")
		}

		// v2 deployment should be FAILED
		dep2 := getDeployment(t, env.httpClient, env.testServer.URL, depID2)
		if dep2.Status != "failed" {
			t.Errorf("v2 deployment status = %q, want FAILED", dep2.Status)
		}
	})

	t.Run("without prior active", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "single.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}

		// FailInstall with no prior active deployment — should not error
		if err := env.updater.FailInstall(ctx, "no reason"); err != nil {
			t.Errorf("FailInstall without prior active: %v", err)
		}
	})
}

func TestInstallingConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("no installing config", func(t *testing.T) {
		env := setupClientEnv(t)

		fsys, err := env.updater.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig: %v", err)
		}
		if fsys != nil {
			t.Error("expected nil fs.FS when no installing config")
			defer func() {
				_ = fsys.Close()
			}()
		}
	})

	t.Run("has installing config", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "nested.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}

		fsys, err := env.updater.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig: %v", err)
		}
		if fsys == nil {
			t.Fatal("expected non-nil fs.FS after installing deployment")
		}
		defer func() {
			_ = fsys.Close()
		}()

		assertFileContent(t, fsys, "config.json", `{"key":"value"}`)
		assertFileContent(t, fsys, "sub/nested.txt", "nested content")
	})
}

func TestDownloadPayload_InsecureURLBlocked(t *testing.T) {
	// When the client's endpoint is HTTPS, HTTP download URLs must be rejected
	// before any network request is made.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// checkInEndpoint uses a different HTTPS host so downloads from ts go through plainHTTP.
	client := NewHTTPClient(Credentials{
		CheckInEndpoint: "https://other-host/v1/device/check-in",
		TokenEndpoint:   "https://other-host/v1/device/token",
		ClientID:        "id",
		ClientSecret:    "secret",
	}, WithHTTPClient(ts.Client()))

	tests := []struct {
		name        string
		downloadURL string
		wantErr     bool
	}{
		{
			name:        "http download URL blocked when endpoint is https",
			downloadURL: "http://example.com/payload.tar.gz",
			wantErr:     true,
		},
		{
			name:        "https download URL allowed when endpoint is https",
			downloadURL: ts.URL + "/payload.tar.gz",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := client.DownloadPayload(context.Background(), tt.downloadURL)
			if rc != nil {
				_ = rc.Close()
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error for insecure download URL, got nil")
				}
				if !errors.Is(err, errInsecureDownloadURL) {
					t.Errorf("expected %v, got: %v", errInsecureDownloadURL, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for secure download URL: %v", err)
				}
			}
		})
	}
}

func TestDownloadPayload_Authorization(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// checkInEndpoint is on a different host than ts, so downloads from ts should not include auth.
	client := NewHTTPClient(Credentials{
		CheckInEndpoint: "http://other-host/v1/device/check-in",
		TokenEndpoint:   "http://other-host/v1/device/token",
		ClientID:        "id",
		ClientSecret:    "secret",
	})

	rc, err := client.DownloadPayload(context.Background(), ts.URL+"/payload.tar.gz")
	if err != nil {
		t.Fatalf("DownloadPayload: %v", err)
	}
	_ = rc.Close()
	if gotAuth != "" {
		t.Errorf("expected no Authorization header on external download, got %q", gotAuth)
	}
}

func TestActiveConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("no active config", func(t *testing.T) {
		env := setupClientEnv(t)

		fsys, err := env.updater.ActiveConfig()
		if err != nil {
			t.Fatalf("ActiveConfig: %v", err)
		}
		if fsys != nil {
			t.Error("expected nil fs.FS when no active config")
		}
	})

	t.Run("has active config", func(t *testing.T) {
		env := setupClientEnv(t)

		payload := txtarToTarGZ(t, "v1.txtar")
		cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
		createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

		if _, err := env.updater.PollOnce(ctx); err != nil {
			t.Fatalf("PollOnce: %v", err)
		}
		if err := env.updater.CommitInstall(ctx); err != nil {
			t.Fatalf("CommitInstall: %v", err)
		}

		fsys, err := env.updater.ActiveConfig()
		if err != nil {
			t.Fatalf("ActiveConfig: %v", err)
		}
		if fsys == nil {
			t.Fatal("expected non-nil fs.FS after commit")
		}

		assertFileContent(t, fsys, "config.json", `{"version":1}`)
	})
}

func genLargeFileTarGZ(t *testing.T, size int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name:     "file.bin",
		Mode:     0644,
		Typeflag: tar.TypeReg,
		Size:     size,
	}
	contents := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, int((size+3)/4))[:size] // repeat pattern to fill size
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tarWriter.Write(contents); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
