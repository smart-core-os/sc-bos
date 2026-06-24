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

	mu             sync.Mutex
	status         *supervisorpb.UpdateStatus // returned by GetUpdateStatus; nil yields an empty status
	installCalls   int
	lastInstallReq *supervisorpb.InstallUpdateRequest
}

func (f *fakeSupervisor) GetUpdateStatus(_ context.Context, _ *supervisorpb.GetUpdateStatusRequest) (*supervisorpb.GetUpdateStatusResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &supervisorpb.GetUpdateStatusResponse{Status: f.status}, nil
}

// setStatus sets the status GetUpdateStatus reports, simulating the Supervisor's view of an update.
func (f *fakeSupervisor) setStatus(st *supervisorpb.UpdateStatus) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = st
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

// dialFakeSupervisor serves fake over a real Unix socket and returns a connected supervisorpb client.
// The socket lives under a short os.MkdirTemp directory to stay within macOS's 104-char limit.
func dialFakeSupervisor(t *testing.T, fake *fakeSupervisor) supervisorpb.SupervisorApiClient {
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

	conn, err := supervisor.Dial(sockPath)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return supervisorpb.NewSupervisorApiClient(conn)
}

// TestUpdateInstall_TriggersSupervisor: a new latestUpdate (no matching Supervisor status) is reported
// installing to SCC and the Supervisor is commanded with the artefact details and the deployment id.
func TestUpdateInstall_TriggersSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithUpdateInstaller(sup))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Supervisor commanded with the artefact's version, payload URL, and the deployment id (the opaque
	// correlation token); the sha is server-derived.
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1", fake.calls())
	}
	req := fake.lastInstall()
	if got := req.GetVersion(); got != "9.9.9" {
		t.Errorf("InstallUpdate version = %q, want %q", got, "9.9.9")
	}
	if req.GetDownloadUrl() == "" {
		t.Error("InstallUpdate download_url empty, want artefact payload URL")
	}
	if got := req.GetDeploymentId(); got != fmt.Sprintf("%d", depID) {
		t.Errorf("InstallUpdate deployment id = %q, want %q", got, fmt.Sprintf("%d", depID))
	}

	// installingUpdate was reported to SCC: the update deployment is now in_progress.
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("update deployment status = %q, want in_progress", dep.Status)
	}
}

// TestUpdateInstall_AlreadyInProgress: a FailedPrecondition from the Supervisor is benign — no error.
func TestUpdateInstall_AlreadyInProgress(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{locked: true}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithUpdateInstaller(sup))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: expected no error from FailedPrecondition, got %v", err)
	}

	// The deployment was still reported installing before the (rejected) install attempt.
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("update deployment status = %q, want in_progress", dep.Status)
	}
}

// TestUpdateInstall_DisabledSupervisor: with no Supervisor wired, no install is attempted and the
// deployment is left untouched (still pending).
func TestUpdateInstall_DisabledSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn := newConnEnv(t, env) // no WithUpdateInstaller

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "pending" {
		t.Errorf("update deployment status = %q, want pending (Supervisor disabled)", dep.Status)
	}
}

// TestUpdateInstall_AlreadyCurrentVersion: when the artefact names the running version, the update is
// reported successful (current) without commanding the Supervisor to install.
func TestUpdateInstall_AlreadyCurrentVersion(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithUpdateInstaller(sup),
		WithSoftwareUpdaterVersion("9.9.9"),
	)

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already current)", fake.calls())
	}
	// Running the target version is reported as success: the deployment completes.
	dep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("update deployment status = %q, want completed", dep.Status)
	}
}
