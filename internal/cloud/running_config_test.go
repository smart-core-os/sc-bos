package cloud

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestDeploymentStore_ActiveVersion covers persisting the config version a deployment carries and
// reading it back for the active deployment, including graceful handling of deployments staged
// before version recording (no metadata file).
func TestDeploymentStore_ActiveVersion(t *testing.T) {
	ctx := context.Background()
	storePath := t.TempDir()
	storeDir, err := os.OpenRoot(storePath)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = storeDir.Close() })
	store := NewDeploymentStore(storeDir)

	// Fresh store: no active deployment, nothing to report.
	if got, err := store.ActiveVersion(); err != nil || got != (ConfigVersion{}) {
		t.Fatalf("ActiveVersion on fresh store = %+v, %v; want zero, nil", got, err)
	}

	want := ConfigVersion{ID: "42", Version: "1.2.3"}
	payload := txtarToTarGZ(t, "single.txtar")
	if err := store.WriteInstalling(ctx, "c-1", want, bytes.NewReader(payload)); err != nil {
		t.Fatalf("WriteInstalling: %v", err)
	}

	// The deployment is only installing, not active, so nothing is reported yet.
	if got, err := store.ActiveVersion(); err != nil || got != (ConfigVersion{}) {
		t.Fatalf("ActiveVersion before commit = %+v, %v; want zero, nil", got, err)
	}

	if err := store.CommitInstall(ctx); err != nil {
		t.Fatalf("CommitInstall: %v", err)
	}
	if got, err := store.ActiveVersion(); err != nil || got != want {
		t.Fatalf("ActiveVersion after commit = %+v, %v; want %+v, nil", got, err, want)
	}

	// A deployment staged before version recording (no metadata file) reports nothing rather than erroring.
	if err := os.Remove(filepath.Join(storePath, "deployments", "c-1", fileVersionMeta)); err != nil {
		t.Fatalf("remove version metadata: %v", err)
	}
	if got, err := store.ActiveVersion(); err != nil || got != (ConfigVersion{}) {
		t.Fatalf("ActiveVersion for legacy deployment = %+v, %v; want zero, nil", got, err)
	}
}

// TestConfigFlow_ReportsRunningConfig locks the end-to-end behaviour over the production Conn.Update
// path: the client reports running.config for the config version it is actually running, and reports
// the active version (not a freshly staged one) while a newer deployment is only installing.
func TestConfigFlow_ReportsRunningConfig(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	// Fresh node: nothing running, running.config omitted.
	req, err := env.updater.CheckInRequest(ctx)
	if err != nil {
		t.Fatalf("CheckInRequest (fresh): %v", err)
	}
	if req.Running.Config != nil {
		t.Errorf("fresh node running.config = %+v, want nil", req.Running.Config)
	}

	// Install and commit a first deployment.
	payload := txtarToTarGZ(t, "single.txtar")
	cv1 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
	createPendingDeployment(t, env.httpClient, env.testServer.URL, cv1)
	if _, err := env.conn.Update(ctx); err != nil {
		t.Fatalf("Update (install dep1): %v", err)
	}
	if err := env.updater.CommitInstall(ctx); err != nil {
		t.Fatalf("CommitInstall: %v", err)
	}

	// The node now reports the config version it is running, including the version string cloudsim
	// assigned (see createConfigVersion helper).
	wantID := strconv.FormatInt(cv1, 10)
	req, err = env.updater.CheckInRequest(ctx)
	if err != nil {
		t.Fatalf("CheckInRequest (after commit): %v", err)
	}
	if req.Running.Config == nil || req.Running.Config.VersionID != wantID || req.Running.Config.Version != "1.0.0" {
		t.Fatalf("running.config = %+v, want VersionID %q and Version %q", req.Running.Config, wantID, "1.0.0")
	}

	// A newer deployment is offered and staged (installing) but not yet committed: running.config
	// still reports the version actually active (cv1), not the staged one.
	cv2 := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
	createPendingDeployment(t, env.httpClient, env.testServer.URL, cv2)
	if _, err := env.conn.Update(ctx); err != nil {
		t.Fatalf("Update (stage dep2): %v", err)
	}
	req, err = env.updater.CheckInRequest(ctx)
	if err != nil {
		t.Fatalf("CheckInRequest (dep2 staged): %v", err)
	}
	if req.Running.Config == nil || req.Running.Config.VersionID != wantID {
		t.Fatalf("running.config while dep2 staged = %+v, want VersionID %q (still cv1)", req.Running.Config, wantID)
	}
}
