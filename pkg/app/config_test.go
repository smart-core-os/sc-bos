package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
)

// noopClient implements cloud.Client for testing.
// CheckIn always succeeds; DownloadPayload is not expected to be called.
type noopClient struct{}

func (noopClient) CheckIn(_ context.Context, _ cloud.CheckInRequest) (cloud.CheckInResponse, error) {
	return cloud.CheckInResponse{}, nil
}

func (noopClient) DownloadPayload(_ context.Context, _ string) (io.ReadCloser, error) {
	panic("DownloadPayload not expected in config load tests")
}

// setupInstallingState creates a "installing" symlink in storeDir pointing to depID,
// and writes configJSON at deployments/<depID>/config-version/config/app.conf.json.
func setupInstallingState(t *testing.T, storeDir string, depID int64, configJSON string) {
	t.Helper()
	depDir := filepath.Join(storeDir, "deployments", strconv.FormatInt(depID, 10), "config-version", "config")
	if err := os.MkdirAll(depDir, 0755); err != nil {
		t.Fatalf("create deployment dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depDir, "app.conf.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write app.conf.json: %v", err)
	}
	linkPath := filepath.Join(storeDir, "deployments", "installing")
	if err := os.Symlink(strconv.FormatInt(depID, 10), linkPath); err != nil {
		t.Fatalf("create installing symlink: %v", err)
	}
}

// setupActiveState creates an "active" symlink in storeDir pointing to depID,
// and writes configJSON at deployments/<depID>/config-version/config/app.conf.json.
func setupActiveState(t *testing.T, storeDir string, depID int64, configJSON string) {
	t.Helper()
	depDir := filepath.Join(storeDir, "deployments", strconv.FormatInt(depID, 10), "config-version", "config")
	if err := os.MkdirAll(depDir, 0755); err != nil {
		t.Fatalf("create deployment dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depDir, "app.conf.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write app.conf.json: %v", err)
	}
	linkPath := filepath.Join(storeDir, "deployments", "active")
	if err := os.Symlink(strconv.FormatInt(depID, 10), linkPath); err != nil {
		t.Fatalf("create active symlink: %v", err)
	}
}

// newTestEnv creates a Conn and DeploymentStore backed by a temp dir.
// The Conn has a fake registration so that CommitInstall/FailInstall have an updater.
func newTestEnv(t *testing.T) (*cloud.Conn, *cloud.DeploymentStore, string) {
	t.Helper()
	storePath := t.TempDir()
	storeDir, err := os.OpenRoot(storePath)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = storeDir.Close() })
	store := cloud.NewDeploymentStore(storeDir)

	regStore := cloud.NewFileRegistrationStore(filepath.Join(t.TempDir(), "registration.json"))
	conn, err := cloud.OpenConn(t.Context(), regStore, store,
		cloud.WithClientFactory(func(cloud.Registration) cloud.Client { return noopClient{} }),
	)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	// Pre-register so the updater is initialised (needed for CommitInstall/FailInstall).
	if _, err = conn.Register(t.Context(), cloud.Registration{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		BosapiRoot:   "http://localhost",
	}); err != nil {
		t.Fatalf("conn.Register: %v", err)
	}
	return conn, store, storePath
}

func TestLoadCloudInstallingConfig(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("no installing config", func(t *testing.T) {
		conn, store, _ := newTestEnv(t)

		_, loaded := loadCloudInstallingConfig(ctx, store, conn, logger)
		if loaded {
			t.Error("want loaded=false when no installing config")
		}
	})

	t.Run("valid config is committed", func(t *testing.T) {
		conn, store, storePath := newTestEnv(t)
		setupInstallingState(t, storePath, 1, "{}")

		_, loaded := loadCloudInstallingConfig(ctx, store, conn, logger)
		if !loaded {
			t.Fatal("want loaded=true for valid installing config")
		}

		// CommitInstall should have cleared the installing mark
		fsys, err := store.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig after commit: %v", err)
		}
		if fsys != nil {
			_ = fsys.Close()
			t.Error("expected nil installing FS after CommitInstall")
		}
	})

	t.Run("unreadable installing FS triggers FailInstall", func(t *testing.T) {
		conn, store, storePath := newTestEnv(t)

		// Create installing symlink and deployment dir but no config-version subdir
		deploymentsDir := filepath.Join(storePath, "deployments")
		if err := os.MkdirAll(filepath.Join(deploymentsDir, "1"), 0755); err != nil {
			t.Fatalf("create deployment dir: %v", err)
		}
		if err := os.Symlink("1", filepath.Join(deploymentsDir, "installing")); err != nil {
			t.Fatalf("create installing symlink: %v", err)
		}

		_, loaded := loadCloudInstallingConfig(ctx, store, conn, logger)
		if loaded {
			t.Error("want loaded=false when config-version dir is missing")
		}

		// FailInstall should have cleared the installing mark
		fsys, err := store.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig after FailInstall: %v", err)
		}
		if fsys != nil {
			_ = fsys.Close()
			t.Error("expected nil installing FS after FailInstall")
		}
	})

	t.Run("invalid config JSON triggers FailInstall", func(t *testing.T) {
		conn, store, storePath := newTestEnv(t)
		setupInstallingState(t, storePath, 1, "{invalid json}")

		_, loaded := loadCloudInstallingConfig(ctx, store, conn, logger)
		if loaded {
			t.Error("want loaded=false for invalid config JSON")
		}

		// FailInstall should have cleared the installing mark
		fsys, err := store.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig after FailInstall: %v", err)
		}
		if fsys != nil {
			_ = fsys.Close()
			t.Error("expected nil installing FS after FailInstall")
		}
	})
}

func TestLoadCloudAppConfig(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("installing config takes precedence over active", func(t *testing.T) {
		conn, store, storePath := newTestEnv(t)
		setupInstallingState(t, storePath, 1, `{"name":"installing"}`)
		setupActiveState(t, storePath, 2, `{"name":"active"}`)

		cs, err := loadCloudAppConfig(ctx, sysconf.Config{DataDir: t.TempDir()}, store, conn, logger)
		if err != nil {
			t.Fatalf("loadCloudAppConfig: %v", err)
		}
		if cs == nil {
			t.Fatal("want non-nil ConfigStore")
		}
		if _, ok := cs.(*immutableConfigStore); !ok {
			t.Errorf("want *immutableConfigStore, got %T", cs)
		}
		if got := cs.Active().Name; got != "installing" {
			t.Errorf("active config name = %q, want %q", got, "installing")
		}
	})

	t.Run("falls back to active config when no installing", func(t *testing.T) {
		conn, store, storePath := newTestEnv(t)
		setupActiveState(t, storePath, 1, `{"name":"active"}`)

		cs, err := loadCloudAppConfig(ctx, sysconf.Config{DataDir: t.TempDir()}, store, conn, logger)
		if err != nil {
			t.Fatalf("loadCloudAppConfig: %v", err)
		}
		if cs == nil {
			t.Fatal("want non-nil ConfigStore")
		}
		if _, ok := cs.(*immutableConfigStore); !ok {
			t.Errorf("want *immutableConfigStore, got %T", cs)
		}
		if got := cs.Active().Name; got != "active" {
			t.Errorf("active config name = %q, want %q", got, "active")
		}
	})

	t.Run("falls back to local config when no active", func(t *testing.T) {
		conn, store, _ := newTestEnv(t)

		const localConfigName = "localconf.json"
		const localConfig = `{"name":"local"}`
		localConfigPath := filepath.Join(t.TempDir(), localConfigName)
		if err := os.WriteFile(localConfigPath, []byte(localConfig), 0644); err != nil {
			t.Fatalf("write local config: %v", err)
		}

		cs, err := loadCloudAppConfig(ctx, sysconf.Config{DataDir: t.TempDir(), AppConfig: []string{localConfigPath}}, store, conn, logger)
		if err != nil {
			t.Fatalf("loadCloudAppConfig: %v", err)
		}
		if cs == nil {
			t.Fatal("want non-nil ConfigStore (from local fallback)")
		}
		if _, ok := cs.(*immutableConfigStore); ok {
			t.Error("want mutable (local) ConfigStore, not *immutableConfigStore")
		}
		if got := cs.Active().Name; got != "local" {
			t.Errorf("active config name = %q, want %q", got, "local")
		}
	})
}
