package cloud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// seedInflightUpdate creates an update artefact + deployment, reports it installing to the sim (so the
// deployment is in_progress), and persists a matching in-flight UpdateState. It returns the deployment
// id so the test can assert the reported outcome. version is the artefact's version and the persisted
// target version.
func seedInflightUpdate(t *testing.T, env *clientEnv, store UpdateStore, version string) int64 {
	t.Helper()
	ctx := context.Background()

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, version, []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	// Report installing so the sim moves the deployment to in_progress (the precondition for it to
	// later transition to completed on a currentUpdate report).
	if _, err := env.client.CheckIn(ctx, CheckInRequest{
		InstallingUpdate: &CheckInInstallingDeployment{ID: fmt.Sprintf("%d", depID), Attempts: 1},
	}); err != nil {
		t.Fatalf("report installing: %v", err)
	}

	if err := store.Save(ctx, UpdateState{
		DeploymentID: fmt.Sprintf("%d", depID),
		Version:      version,
		Attempts:     1,
		StartTime:    time.Now(),
	}); err != nil {
		t.Fatalf("save update state: %v", err)
	}
	return depID
}

// TestReconcileStartup_Success: an in-flight update whose target version equals the running version is
// reported as currentUpdate (deployment -> completed) and the intent is cleared.
func TestReconcileStartup_Success(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn, updateStore := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("9.9.9"), // EffectiveVersion() == target version
	)

	depID := seedInflightUpdate(t, env, updateStore, "9.9.9")

	if err := conn.ReconcileUpdate(ctx); err != nil {
		t.Fatalf("ReconcileUpdate: %v", err)
	}

	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("deployment status = %q, want completed", dep.Status)
	}
	if _, ok, _ := updateStore.Load(ctx); ok {
		t.Error("expected update state cleared after successful reconciliation")
	}
}

// TestReconcileStartup_Rollback: an in-flight update whose target version differs from the running
// version is reported as failedUpdate with the Supervisor's reason (deployment -> failed) and the
// intent is cleared.
func TestReconcileStartup_Rollback(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	const wantReason = "health check failed after install"
	fake := &fakeSupervisor{status: &supervisorpb.UpdateStatus{
		State: supervisorpb.UpdateStatus_FAILED,
		Error: wantReason,
	}}
	sup := dialFakeSupervisor(t, fake)
	conn, updateStore := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("8.8.8"), // running version != target (rolled back)
	)

	depID := seedInflightUpdate(t, env, updateStore, "9.9.9")

	if err := conn.ReconcileUpdate(ctx); err != nil {
		t.Fatalf("ReconcileUpdate: %v", err)
	}

	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("deployment status = %q, want failed", dep.Status)
	}
	if dep.Reason != wantReason {
		t.Errorf("deployment reason = %q, want %q", dep.Reason, wantReason)
	}
	if _, ok, _ := updateStore.Load(ctx); ok {
		t.Error("expected update state cleared after rollback reconciliation")
	}
}

// TestReconcileStartup_DisabledSupervisor: with no SoftwareUpdater status getter / installer wired the
// reconciliation still resolves (falls back to a generic reason) without panicking. It also covers the
// no-in-flight-state no-op.
func TestReconcileStartup_DisabledSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn, updateStore := newConnEnv(t, env) // no Supervisor wired

	// No in-flight state: ReconcileUpdate is a no-op.
	if err := conn.ReconcileUpdate(ctx); err != nil {
		t.Fatalf("ReconcileUpdate (no state): %v", err)
	}

	// In-flight rollback with no status getter falls back to a generic reason and clears state.
	depID := seedInflightUpdate(t, env, updateStore, "9.9.9")
	if err := conn.ReconcileUpdate(ctx); err != nil {
		t.Fatalf("ReconcileUpdate: %v", err)
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("deployment status = %q, want failed", dep.Status)
	}
	if dep.Reason == "" {
		t.Error("expected a generic failure reason, got empty")
	}
	if _, ok, _ := updateStore.Load(ctx); ok {
		t.Error("expected update state cleared")
	}
}
