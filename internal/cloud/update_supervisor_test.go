package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// createUpdateArtefactKind creates an update artefact of the given kind and returns its id.
func createUpdateArtefactKind(t *testing.T, client *http.Client, baseURL, kind, version string, payload []byte) int64 {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("version", version)
	_ = mw.WriteField("platform", "podman")
	_ = mw.WriteField("kind", kind)
	fw, err := mw.CreateFormFile("payload", "update.rpm")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req, err := http.NewRequest("POST", baseURL+"/api/v1/management/update-artefacts", &body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create artefact: expected 201, got %d", resp.StatusCode)
	}
	var a sim.UpdateArtefact
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return a.ID
}

// supervisorUpdateOffer builds a CheckInResponse offering the given supervisor-rpm deployment.
func supervisorUpdateOffer(depID int64, version string) CheckInResponse {
	return CheckInResponse{
		LatestSupervisorUpdate: &LatestUpdate{
			UpdateDeployment: UpdateDeployment{ID: fmt.Sprintf("%d", depID)},
			UpdateArtefact:   UpdateArtefact{Version: version, PayloadURL: "http://artefact/sup.rpm", SHA256: "abc"},
		},
	}
}

// reportInstalling moves a deployment to in_progress (the precondition for a later completed/failed).
func reportInstalling(t *testing.T, env *clientEnv, depID int64) {
	t.Helper()
	if _, err := env.client.CheckIn(context.Background(), CheckInRequest{
		InstallingUpdate: &CheckInInstallingDeployment{ID: fmt.Sprintf("%d", depID), Attempts: 1},
	}); err != nil {
		t.Fatalf("report installing: %v", err)
	}
}

func newSupervisorUpdater(t *testing.T, sup SupervisorInstaller) (*SupervisorUpdater, UpdateStore) {
	t.Helper()
	store := NewFileUpdateStore(filepath.Join(t.TempDir(), "supervisor-update.json"))
	opts := []SupervisorUpdaterOption{}
	if sup != nil {
		opts = append(opts, WithSupervisorInstaller(sup))
	}
	return NewSupervisorUpdater(store, opts...), store
}

// TestSupervisorUpdate_InstallsWhenOffered: a supervisor-rpm offer for a version the Supervisor is not
// running is installed, the intent is persisted, and the deployment moves to in_progress.
func TestSupervisorUpdate_InstallsWhenOffered(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{supervisorVersion: "1.0.0"}
	sup := dialFakeSupervisor(t, fake)
	updater, store := newSupervisorUpdater(t, sup)

	artID := createUpdateArtefactKind(t, env.httpClient, env.testServer.URL, "supervisor-rpm", "2.0.0", []byte("rpm"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if err := updater.HandleSupervisorUpdate(ctx, env.client, supervisorUpdateOffer(depID, "2.0.0")); err != nil {
		t.Fatalf("HandleSupervisorUpdate: %v", err)
	}

	if calls, req := fake.supInstall(); calls != 1 || req.GetVersion() != "2.0.0" {
		t.Errorf("InstallSupervisorUpdate calls=%d req=%v, want one call for 2.0.0", calls, req)
	}
	if dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "in_progress" {
		t.Errorf("deployment status = %q, want in_progress", dep.Status)
	}
	if st, ok, _ := store.Load(ctx); !ok || st.Version != "2.0.0" {
		t.Errorf("persisted intent = %+v ok=%v, want version 2.0.0", st, ok)
	}
}

// TestSupervisorUpdate_ReportsCompletedWhenVersionMatches: once the Supervisor reports the target
// version, the deployment is reported completed and the intent is cleared.
func TestSupervisorUpdate_ReportsCompletedWhenVersionMatches(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{supervisorVersion: "2.0.0"} // already running the target
	sup := dialFakeSupervisor(t, fake)
	updater, store := newSupervisorUpdater(t, sup)

	artID := createUpdateArtefactKind(t, env.httpClient, env.testServer.URL, "supervisor-rpm", "2.0.0", []byte("rpm"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)
	reportInstalling(t, env, depID)
	if err := store.Save(ctx, UpdateState{DeploymentID: fmt.Sprintf("%d", depID), Version: "2.0.0", Attempts: 1}); err != nil {
		t.Fatalf("seed intent: %v", err)
	}

	if err := updater.HandleSupervisorUpdate(ctx, env.client, CheckInResponse{}); err != nil {
		t.Fatalf("HandleSupervisorUpdate: %v", err)
	}

	if dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "completed" {
		t.Errorf("deployment status = %q, want completed", dep.Status)
	}
	if _, ok, _ := store.Load(ctx); ok {
		t.Error("expected intent cleared after success")
	}
}

// TestSupervisorUpdate_ReportsFailedOnRollback: when the Supervisor rolled back (running the old
// version, self-update FAILED), the deployment is reported failed with the Supervisor's reason.
func TestSupervisorUpdate_ReportsFailedOnRollback(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	const wantReason = "new version did not come up healthy"
	fake := &fakeSupervisor{
		supervisorVersion:    "1.0.0", // rolled back to the old version
		supervisorSelfUpdate: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_FAILED, Error: wantReason},
	}
	sup := dialFakeSupervisor(t, fake)
	updater, store := newSupervisorUpdater(t, sup)

	artID := createUpdateArtefactKind(t, env.httpClient, env.testServer.URL, "supervisor-rpm", "2.0.0", []byte("rpm"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)
	reportInstalling(t, env, depID)
	if err := store.Save(ctx, UpdateState{DeploymentID: fmt.Sprintf("%d", depID), Version: "2.0.0", Attempts: 1}); err != nil {
		t.Fatalf("seed intent: %v", err)
	}

	if err := updater.HandleSupervisorUpdate(ctx, env.client, CheckInResponse{}); err != nil {
		t.Fatalf("HandleSupervisorUpdate: %v", err)
	}

	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("deployment status = %q, want failed", dep.Status)
	}
	if dep.Reason != wantReason {
		t.Errorf("deployment reason = %q, want %q", dep.Reason, wantReason)
	}
	if _, ok, _ := store.Load(ctx); ok {
		t.Error("expected intent cleared after rollback")
	}
}

// TestSupervisorUpdate_SkipsRunningVersion: an offer naming the version already running is not installed.
func TestSupervisorUpdate_SkipsRunningVersion(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{supervisorVersion: "2.0.0"}
	sup := dialFakeSupervisor(t, fake)
	updater, store := newSupervisorUpdater(t, sup)

	artID := createUpdateArtefactKind(t, env.httpClient, env.testServer.URL, "supervisor-rpm", "2.0.0", []byte("rpm"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if err := updater.HandleSupervisorUpdate(ctx, env.client, supervisorUpdateOffer(depID, "2.0.0")); err != nil {
		t.Fatalf("HandleSupervisorUpdate: %v", err)
	}
	if calls, _ := fake.supInstall(); calls != 0 {
		t.Errorf("InstallSupervisorUpdate calls = %d, want 0 (already running target)", calls)
	}
	if _, ok, _ := store.Load(ctx); ok {
		t.Error("expected no intent persisted")
	}
}

// TestSupervisorUpdate_DisabledNoInstaller: with no installer the channel is a no-op.
func TestSupervisorUpdate_DisabledNoInstaller(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	updater, _ := newSupervisorUpdater(t, nil)

	if err := updater.HandleSupervisorUpdate(ctx, env.client, supervisorUpdateOffer(123, "2.0.0")); err != nil {
		t.Fatalf("HandleSupervisorUpdate (disabled): %v", err)
	}
}
