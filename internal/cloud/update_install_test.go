package cloud

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// getUpdateDeployment fetches a single update deployment from the sim server.
func getUpdateDeployment(t *testing.T, client *http.Client, baseURL string, deploymentID int64) sim.UpdateDeployment {
	t.Helper()
	var dep sim.UpdateDeployment
	resp := doSimRequest(t, client, "GET",
		fmt.Sprintf("%s/api/v1/management/update-deployments/%d", baseURL, deploymentID),
		nil, &dep)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get update deployment: expected 200, got %d", resp.StatusCode)
	}
	return dep
}

// fakeSupervisor is a minimal SupervisorApiServer that records the last InstallUpdate request and
// optionally rejects it with FailedPrecondition (an update already in progress).
type fakeSupervisor struct {
	supervisorpb.UnimplementedSupervisorApiServer

	locked       bool // if true, InstallUpdate returns FailedPrecondition
	installFails bool // if true, InstallUpdate returns a generic (non-FailedPrecondition) error

	status *supervisorpb.UpdateStatus // returned by GetUpdateStatus; nil yields an empty status

	mu             sync.Mutex
	installCalls   int
	lastInstallReq *supervisorpb.InstallUpdateRequest
}

func (f *fakeSupervisor) GetUpdateStatus(_ context.Context, _ *supervisorpb.GetUpdateStatusRequest) (*supervisorpb.GetUpdateStatusResponse, error) {
	return &supervisorpb.GetUpdateStatusResponse{Status: f.status}, nil
}

func (f *fakeSupervisor) InstallUpdate(_ context.Context, req *supervisorpb.InstallUpdateRequest) (*supervisorpb.InstallUpdateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locked {
		return nil, status.Error(codes.FailedPrecondition, "an update is already in progress")
	}
	if f.installFails {
		return nil, status.Error(codes.Internal, "install failed")
	}
	f.installCalls++
	f.lastInstallReq = req
	return &supervisorpb.InstallUpdateResponse{
		Status: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_DOWNLOADING, Version: req.GetVersion()},
	}, nil
}

func (f *fakeSupervisor) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.installCalls
}

func (f *fakeSupervisor) lastInstall() *supervisorpb.InstallUpdateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastInstallReq
}

// dialFakeSupervisor serves fake over a real Unix socket and returns a connected supervisor.Client.
// The socket lives under a short os.MkdirTemp directory to stay within macOS's 104-char limit.
func dialFakeSupervisor(t *testing.T, fake *fakeSupervisor) *supervisor.Client {
	t.Helper()
	dir, err := os.MkdirTemp("", "sup-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "s.sock")

	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s) = %v", sockPath, err)
	}
	srv := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(srv, fake)
	t.Cleanup(func() { srv.Stop() })
	go func() { _ = srv.Serve(lis) }()

	c, err := supervisor.Dial(sockPath, 5*time.Second)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestUpdateInstall_TriggersSupervisor exercises the full Phase 3 path: a new latestUpdate persists
// intent, reports installingUpdate to SCC, and commands the Supervisor with the artefact details.
func TestUpdateInstall_TriggersSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn, updateStore := newConnEnv(t, env, WithUpdateInstaller(sup))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// intent persisted with Attempts = 1.
	state, ok, err := updateStore.Load(ctx)
	if err != nil || !ok {
		t.Fatalf("expected persisted intent; ok=%v err=%v", ok, err)
	}
	wantID := fmt.Sprintf("%d", depID)
	if state.DeploymentID != wantID {
		t.Errorf("DeploymentID = %q, want %q", state.DeploymentID, wantID)
	}
	if state.Version != "9.9.9" {
		t.Errorf("Version = %q, want %q", state.Version, "9.9.9")
	}
	if state.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", state.Attempts)
	}

	// Supervisor commanded with the artefact's version (url/sha are server-derived; assert version).
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1", fake.calls())
	}
	if got := fake.lastInstall().GetVersion(); got != "9.9.9" {
		t.Errorf("InstallUpdate version = %q, want %q", got, "9.9.9")
	}
	if fake.lastInstall().GetDownloadUrl() == "" {
		t.Error("InstallUpdate download_url empty, want artefact payload URL")
	}

	// installingUpdate was reported to SCC: the update deployment is now in_progress.
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("update deployment status = %q, want in_progress", dep.Status)
	}
}

// TestUpdateInstall_AlreadyInProgress: a FailedPrecondition from the Supervisor is benign — no error,
// intent retained.
func TestUpdateInstall_AlreadyInProgress(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{locked: true}
	sup := dialFakeSupervisor(t, fake)
	conn, updateStore := newConnEnv(t, env, WithUpdateInstaller(sup))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: expected no error from FailedPrecondition, got %v", err)
	}

	state, ok, err := updateStore.Load(ctx)
	if err != nil || !ok {
		t.Fatalf("expected intent retained; ok=%v err=%v", ok, err)
	}
	if state.DeploymentID != fmt.Sprintf("%d", depID) {
		t.Errorf("DeploymentID = %q, want %q", state.DeploymentID, fmt.Sprintf("%d", depID))
	}
}

// TestUpdateInstall_DisabledSupervisor: with no Supervisor wired, no install is attempted and (per the
// chosen policy) no intent is persisted.
func TestUpdateInstall_DisabledSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn, updateStore := newConnEnv(t, env) // no WithUpdateInstaller

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok, err := updateStore.Load(ctx); err != nil || ok {
		t.Errorf("expected no persisted intent with disabled Supervisor; ok=%v err=%v", ok, err)
	}
}

// TestUpdateInstall_AlreadyCurrentVersion: when the artefact names the running version, no install is
// attempted and no intent is persisted.
func TestUpdateInstall_AlreadyCurrentVersion(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn, updateStore := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("9.9.9"),
	)

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already current)", fake.calls())
	}
	if _, ok, _ := updateStore.Load(ctx); ok {
		t.Error("expected no persisted intent when already running the target version")
	}
}
