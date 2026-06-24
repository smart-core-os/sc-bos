package cloud

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

// These characterization tests lock the external behaviour of the config flow (Conn.Update /
// CommitInstall / FailInstall and the poll) on the production shared-check-in path.

// TestConfigFlow_NewDeployment locks: a brand-new deployment is reported installing, downloaded,
// staged on disk, and reported needReboot=true.
func TestConfigFlow_NewDeployment(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	payload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
	depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	needReboot, err := env.conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !needReboot {
		t.Fatal("expected needReboot=true after new deployment")
	}

	// staged: installing symlink points at the deployment and the payload is extracted.
	if !symlinkExists(env.storePath, "deployments/installing") {
		t.Fatal("expected installing symlink to exist")
	}
	target := readSymlinkTarget(t, env.storePath, "deployments/installing")
	if target != fmt.Sprintf("c-%d", depID) {
		t.Errorf("installing symlink target = %q, want %q", target, fmt.Sprintf("c-%d", depID))
	}
	assertFileContent(t, os.DirFS(env.storePath), path.Join("deployments", target, "config-version", "config.json"), `{"key":"value"}`)

	// reported installing: the server moved the deployment to in_progress.
	dep := getDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("server deployment status = %q, want in_progress", dep.Status)
	}
}

// TestConfigFlow_AlreadyInstalling locks: when a deployment is already staged as installing, a
// second Update short-circuits to needReboot=true without acting on latestConfig.
func TestConfigFlow_AlreadyInstalling(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	payload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
	createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	if needReboot, err := env.conn.Update(ctx); err != nil || !needReboot {
		t.Fatalf("first Update: needReboot=%v err=%v", needReboot, err)
	}

	needReboot, err := env.conn.Update(ctx)
	if err != nil {
		t.Fatalf("second Update: %v", err)
	}
	if !needReboot {
		t.Error("expected needReboot=true when already installing")
	}
}

// TestConfigFlow_NoLatestConfig locks: when the server returns no latestConfig, Update is a no-op
// (needReboot=false, nothing staged).
func TestConfigFlow_NoLatestConfig(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	needReboot, err := env.conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if needReboot {
		t.Error("expected needReboot=false when no latestConfig")
	}
	if symlinkExists(env.storePath, "deployments/installing") {
		t.Error("expected no installing symlink when no latestConfig")
	}
}

// TestConfigFlow_DeploymentEqualsActive locks: when latestConfig names the already-active
// deployment, Update is a no-op (needReboot=false).
func TestConfigFlow_DeploymentEqualsActive(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	payload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, payload)
	createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	if _, err := env.conn.Update(ctx); err != nil {
		t.Fatalf("first Update: %v", err)
	}
	if err := env.updater.CommitInstall(ctx); err != nil {
		t.Fatalf("CommitInstall: %v", err)
	}

	needReboot, err := env.conn.Update(ctx)
	if err != nil {
		t.Fatalf("second Update: %v", err)
	}
	if needReboot {
		t.Error("expected needReboot=false when latestConfig matches active deployment")
	}
}

// a config payload that cannot be installed (e.g. invalid tar.gz) is retried, but after
// maxInstallAttempts the deployment is reported permanently failed so that the cloud stops
// re-offering it - prevents downloading in a loop forever.
func TestConfigFlow_TransientFailuresCapAtThree(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	// The download and checksum succeed, but staging fails on extraction every poll - a repeatable
	// transient install failure.
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, []byte("not a tar.gz"))
	depID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	for i := 1; i < maxInstallAttempts; i++ {
		if _, err := env.conn.Update(ctx); err == nil {
			t.Fatalf("Update %d: want transient install error, got nil", i)
		}
		if dep := getDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "in_progress" {
			t.Fatalf("after Update %d: config deployment status = %q, want in_progress", i, dep.Status)
		}
	}

	if _, err := env.conn.Update(ctx); err != nil {
		t.Fatalf("Update %d: want nil once the cap is reached, got %v", maxInstallAttempts, err)
	}
	dep := getDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Fatalf("config deployment status = %q, want failed after %d attempts", dep.Status, maxInstallAttempts)
	}
	if !strings.Contains(dep.Reason, "attempts") {
		t.Errorf("config deployment reason = %q, want it to mention the attempt cap", dep.Reason)
	}

	// A further poll has nothing to install: the failed deployment is no longer offered.
	if _, err := env.conn.Update(ctx); err != nil {
		t.Fatalf("Update after cap: %v", err)
	}
	if dep := getDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "failed" {
		t.Errorf("config deployment status = %q, want it to stay failed", dep.Status)
	}
}
