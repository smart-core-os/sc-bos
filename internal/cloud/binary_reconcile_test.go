package cloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// seedInProgressBinary creates a binary artefact + deployment and reports it installing to the sim, so
// the deployment is in_progress (the state BOS leaves it in before the Supervisor restarts it). It
// returns the deployment id. version is the artefact's target version.
func seedInProgressBinary(t *testing.T, env *clientEnv, version string) int64 {
	t.Helper()
	ctx := context.Background()

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, version, []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := env.client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: fmt.Sprintf("b-%d", depID), State: ProgressInstalling}},
	}); err != nil {
		t.Fatalf("report installing: %v", err)
	}
	return depID
}

// TestBinaryOutcome_Success: when the Supervisor reports this deployment COMPLETED, the update is
// reported to the cloud as current (completed), even if BOS's running version is unknown.
func TestBinaryOutcome_Success(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup)) // running version unknown

	depID := seedInProgressBinary(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_COMPLETED,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("b-%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already completed)", fake.calls())
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("deployment status = %q, want completed", dep.Status)
	}
}

// TestBinaryOutcome_CompletesPendingDeployment: reporting a deployment current completes it even when
// the server never saw an intermediate installing report (it is still pending). Reporting current is
// sufficient on its own - the node does not also have to claim it is installing.
func TestBinaryOutcome_CompletesPendingDeployment(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := env.client.CheckIn(ctx, CheckInRequest{
		Progress: []ProgressReport{{DeploymentID: fmt.Sprintf("b-%d", depID), State: ProgressApplied}},
	}); err != nil {
		t.Fatalf("report current: %v", err)
	}

	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("deployment status = %q, want completed", dep.Status)
	}
}

// TestBinaryOutcome_Rollback: when BOS is not running the target and the Supervisor reports this
// deployment FAILED, the update is reported to the cloud as failed with the Supervisor's reason.
func TestBinaryOutcome_Rollback(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	const wantReason = "health check failed after install"
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithBinaryInstaller(sup),
		WithBinaryVersion("8.8.8"), // running version != target (rolled back)
	)

	depID := seedInProgressBinary(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_FAILED,
		Version:      "9.9.9",
		Error:        wantReason,
		DeploymentId: fmt.Sprintf("b-%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already failed)", fake.calls())
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("deployment status = %q, want failed", dep.Status)
	}
	if dep.Reason != wantReason {
		t.Errorf("deployment reason = %q, want %q", dep.Reason, wantReason)
	}
}

// TestBinaryOutcome_InProgress: when the Supervisor reports this deployment still installing, the update
// is reported installing and the Supervisor is not commanded to install again.
func TestBinaryOutcome_InProgress(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup))

	depID := seedInProgressBinary(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_INSTALLING,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("b-%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (install already in flight)", fake.calls())
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress", dep.Status)
	}
}

// TestBinaryOutcome_RunningTargetButSupervisorInstalling is a regression test for C2: when BOS already
// runs the target version but the Supervisor is still installing this deployment (awaiting the commit,
// able to roll back), the update is reported installing, not applied. Completing it prematurely would let
// the cloud mark it done and never observe a subsequent rollback.
func TestBinaryOutcome_RunningTargetButSupervisorInstalling(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithBinaryInstaller(sup),
		WithBinaryVersion("9.9.9"), // already running the target
	)

	depID := seedInProgressBinary(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_INSTALLING,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("b-%d", depID),
	})

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0", fake.calls())
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress (Supervisor may still roll back)", dep.Status)
	}
}

// TestBinaryOutcome_SameVersionRetry: a fresh deployment for a version the Supervisor previously FAILED
// (under a different deployment id) is treated as a new install, not reported failed. This is the case
// the deployment-id correlation exists to disambiguate.
func TestBinaryOutcome_SameVersionRetry(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithBinaryInstaller(sup),
		WithBinaryVersion("8.8.8"), // not running the target
	)

	// The Supervisor's last update was a failed attempt at 9.9.9 under a stale deployment id.
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_FAILED,
		Version:      "9.9.9",
		Error:        "previous attempt failed",
		DeploymentId: "999999",
	})

	// A new deployment for the same version arrives.
	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1 (fresh install of the retried version)", fake.calls())
	}
	if got := fake.lastInstall().GetDeploymentId(); got != fmt.Sprintf("b-%d", depID) {
		t.Errorf("InstallUpdate deployment id = %q, want %q", got, fmt.Sprintf("b-%d", depID))
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress", dep.Status)
	}
}
