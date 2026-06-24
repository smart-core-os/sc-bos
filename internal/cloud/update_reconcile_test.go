package cloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// seedInProgressUpdate creates an update artefact + deployment and reports it installing to the sim, so
// the deployment is in_progress (the state BOS leaves it in before the Supervisor restarts it). It
// returns the deployment id. version is the artefact's target version.
func seedInProgressUpdate(t *testing.T, env *clientEnv, version string) int64 {
	t.Helper()
	ctx := context.Background()

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, version, []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := env.client.CheckIn(ctx, CheckInRequest{
		InstallingUpdate: &CheckInInstallingDeployment{ID: fmt.Sprintf("%d", depID)},
	}); err != nil {
		t.Fatalf("report installing: %v", err)
	}
	return depID
}

// TestUpdateOutcome_Success: when the Supervisor reports this deployment COMPLETED, the update is
// reported to the cloud as current (completed), even if BOS's running version is unknown.
func TestUpdateOutcome_Success(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithUpdateInstaller(sup)) // running version unknown

	depID := seedInProgressUpdate(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_COMPLETED,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already completed)", fake.calls())
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("deployment status = %q, want completed", dep.Status)
	}
}

// TestUpdateOutcome_Rollback: when BOS is not running the target and the Supervisor reports this
// deployment FAILED, the update is reported to the cloud as failed with the Supervisor's reason.
func TestUpdateOutcome_Rollback(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	const wantReason = "health check failed after install"
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("8.8.8"), // running version != target (rolled back)
	)

	depID := seedInProgressUpdate(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_FAILED,
		Version:      "9.9.9",
		Error:        wantReason,
		DeploymentId: fmt.Sprintf("%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already failed)", fake.calls())
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("deployment status = %q, want failed", dep.Status)
	}
	if dep.Reason != wantReason {
		t.Errorf("deployment reason = %q, want %q", dep.Reason, wantReason)
	}
}

// TestUpdateOutcome_InProgress: when the Supervisor reports this deployment still installing, the update
// is reported installing and the Supervisor is not commanded to install again.
func TestUpdateOutcome_InProgress(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithUpdateInstaller(sup))

	depID := seedInProgressUpdate(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_INSTALLING,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (install already in flight)", fake.calls())
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress", dep.Status)
	}
}

// TestUpdateOutcome_SameVersionRetry: a fresh deployment for a version the Supervisor previously FAILED
// (under a different deployment id) is treated as a new install, not reported failed. This is the case
// the deployment-id correlation exists to disambiguate.
func TestUpdateOutcome_SameVersionRetry(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("8.8.8"), // not running the target
	)

	// The Supervisor's last update was a failed attempt at 9.9.9 under a stale deployment id.
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_FAILED,
		Version:      "9.9.9",
		Error:        "previous attempt failed",
		DeploymentId: "999999",
	})

	// A new deployment for the same version arrives.
	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1 (fresh install of the retried version)", fake.calls())
	}
	if got := fake.lastInstall().GetDeploymentId(); got != fmt.Sprintf("%d", depID) {
		t.Errorf("InstallUpdate deployment id = %q, want %q", got, fmt.Sprintf("%d", depID))
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress", dep.Status)
	}
}
