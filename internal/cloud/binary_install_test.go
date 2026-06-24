package cloud

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// getBinaryDeployment fetches a single binary deployment from the sim server.
func getBinaryDeployment(t *testing.T, client *http.Client, baseURL string, deploymentID int64) sim.BinaryDeployment {
	t.Helper()
	var dep sim.BinaryDeployment
	resp := doSimRequest(t, client, "GET",
		fmt.Sprintf("%s/api/v1/management/binary-deployments/%d", baseURL, deploymentID),
		nil, &dep)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get binary deployment: expected 200, got %d", resp.StatusCode)
	}
	return dep
}

// fakeSupervisor is a minimal SupervisorApiServer that records the last InstallUpdate request and
// optionally rejects it with FailedPrecondition (an update already in progress).
type fakeSupervisor struct {
	supervisorpb.UnimplementedSupervisorApiServer

	locked        bool // if true, InstallUpdate returns FailedPrecondition
	installFails  bool // if true, InstallUpdate returns a generic (non-FailedPrecondition) error
	installReject bool // if true, InstallUpdate returns InvalidArgument (a permanent rejection)

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

// setStatus sets the status GetUpdateStatus reports, simulating the Supervisor's view of a binary.
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
	if f.installReject {
		return nil, status.Error(codes.InvalidArgument, "version is not a valid image tag")
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

// TestBinaryInstall_TriggersSupervisor: a new latestBinary (no matching Supervisor status) is reported
// installing to SCC and the Supervisor is commanded with the artefact details and the deployment id.
func TestBinaryInstall_TriggersSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Supervisor commanded with the artefact's version, payload URL, and the deployment id (the opaque
	// correlation token); the checksum is server-derived.
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
	if got := req.GetDeploymentId(); got != fmt.Sprintf("b-%d", depID) {
		t.Errorf("InstallUpdate deployment id = %q, want %q", got, fmt.Sprintf("b-%d", depID))
	}

	// installingBinary was reported to SCC: the binary deployment is now in_progress.
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("binary deployment status = %q, want in_progress", dep.Status)
	}
}

// TestBinaryInstall_AlreadyInProgress: a FailedPrecondition from the Supervisor is benign — no error.
func TestBinaryInstall_AlreadyInProgress(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{locked: true}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: expected no error from FailedPrecondition, got %v", err)
	}

	// The deployment was still reported installing before the (rejected) install attempt.
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "in_progress" {
		t.Errorf("binary deployment status = %q, want in_progress", dep.Status)
	}
}

// TestBinaryInstall_InvalidArtefactFails: a permanent rejection (InvalidArgument) from the Supervisor
// drives the deployment to a terminal failed state, so the cloud stops re-offering it, rather than
// leaving it in_progress to be retried every poll.
func TestBinaryInstall_InvalidArtefactFails(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{installReject: true}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	// The install failure is surfaced to the caller.
	if _, err := conn.Update(ctx); err == nil {
		t.Fatal("Update: want error from InvalidArgument, got nil")
	}

	// The deployment reaches a terminal failed state, with the supervisor's reason recorded.
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Errorf("binary deployment status = %q, want failed", dep.Status)
	}
	if !strings.Contains(dep.Reason, "image tag") {
		t.Errorf("binary deployment reason = %q, want it to mention the rejection", dep.Reason)
	}
}

// TestBinaryInstall_TransientFailuresCapAtThree: a Supervisor that keeps failing an install transiently
// is retried, but after maxInstallAttempts the deployment is reported permanently failed so the cloud
// stops re-offering it, rather than retrying forever.
func TestBinaryInstall_TransientFailuresCapAtThree(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{installFails: true}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env, WithBinaryInstaller(sup))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	// The first maxInstallAttempts-1 polls keep retrying: each surfaces the failure and leaves the
	// deployment in_progress, still on offer.
	for i := 1; i < maxInstallAttempts; i++ {
		if _, err := conn.Update(ctx); err == nil {
			t.Fatalf("Update %d: want transient install error, got nil", i)
		}
		if dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "in_progress" {
			t.Fatalf("after Update %d: binary deployment status = %q, want in_progress", i, dep.Status)
		}
	}

	// The maxInstallAttempts-th failure is terminal: the deployment is reported failed with the
	// attempt-count reason, and the poll no longer surfaces an error.
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update %d: want nil once the cap is reached, got %v", maxInstallAttempts, err)
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "failed" {
		t.Fatalf("binary deployment status = %q, want failed after %d attempts", dep.Status, maxInstallAttempts)
	}
	if !strings.Contains(dep.Reason, "attempts") {
		t.Errorf("binary deployment reason = %q, want it to mention the attempt cap", dep.Reason)
	}

	// A further poll has nothing to install: the failed deployment is no longer offered.
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update after cap: %v", err)
	}
	if dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID); dep.Status != "failed" {
		t.Errorf("binary deployment status = %q, want it to stay failed", dep.Status)
	}
}

// TestBinaryInstall_DisabledSupervisor: with no Supervisor wired, no install is attempted and the
// deployment is left untouched (still pending).
func TestBinaryInstall_DisabledSupervisor(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn := newConnEnv(t, env) // no WithBinaryInstaller

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "pending" {
		t.Errorf("binary deployment status = %q, want pending (Supervisor disabled)", dep.Status)
	}
}

// TestBinaryInstall_AlreadyCurrentVersion: when the artefact names the running version, the binary is
// reported successful (current) without commanding the Supervisor to install.
func TestBinaryInstall_AlreadyCurrentVersion(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	sup := dialFakeSupervisor(t, fake)
	conn := newConnEnv(t, env,
		WithBinaryInstaller(sup),
		WithBinaryVersion("9.9.9"),
	)

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "9.9.9", []byte("dummy-artefact-payload"))
	depID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 (already current)", fake.calls())
	}
	// Running the target version is reported as success: the deployment completes.
	dep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, depID)
	if dep.Status != "completed" {
		t.Errorf("binary deployment status = %q, want completed", dep.Status)
	}
}
