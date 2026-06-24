package supervisor_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// tempSockPath returns a short Unix socket path safe on all platforms (macOS limit is 104 chars).
// It creates a short-lived directory under os.TempDir() and registers cleanup.
func tempSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "sup-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

// fakeServer is a minimal SupervisorApiServer that records the last call to each RPC and returns
// canned responses. It returns codes.FailedPrecondition from InstallUpdate when locked is true.
type fakeServer struct {
	supervisorpb.UnimplementedSupervisorApiServer

	locked bool // if true, InstallUpdate returns FailedPrecondition

	lastCommitVersion string
	lastInstallReq    *supervisorpb.InstallUpdateRequest
}

func (f *fakeServer) Commit(_ context.Context, req *supervisorpb.CommitRequest) (*supervisorpb.CommitResponse, error) {
	f.lastCommitVersion = req.GetVersion()
	return &supervisorpb.CommitResponse{}, nil
}

func (f *fakeServer) InstallUpdate(_ context.Context, req *supervisorpb.InstallUpdateRequest) (*supervisorpb.InstallUpdateResponse, error) {
	if f.locked {
		return nil, status.Error(codes.FailedPrecondition, "an update is already in progress")
	}
	f.lastInstallReq = req
	return &supervisorpb.InstallUpdateResponse{
		Status: &supervisorpb.UpdateStatus{
			State:   supervisorpb.UpdateStatus_DOWNLOADING,
			Version: req.GetVersion(),
		},
	}, nil
}

func (f *fakeServer) GetUpdateStatus(_ context.Context, _ *supervisorpb.GetUpdateStatusRequest) (*supervisorpb.GetUpdateStatusResponse, error) {
	return &supervisorpb.GetUpdateStatusResponse{
		Status: &supervisorpb.UpdateStatus{
			State:   supervisorpb.UpdateStatus_IDLE,
			Version: "v1.0.0",
		},
	}, nil
}

// dialTestServer starts a grpc.Server over a real Unix socket, registers fake on it,
// dials a Client, and returns the client for RPC calls and the fake for inspection.
func dialTestServer(t *testing.T, fake *fakeServer) *supervisor.Client {
	t.Helper()
	sockPath := tempSockPath(t)

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

func TestClient_Commit(t *testing.T) {
	fake := &fakeServer{}
	c := dialTestServer(t, fake)

	if err := c.Commit(context.Background(), "v1.2.3"); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if fake.lastCommitVersion != "v1.2.3" {
		t.Errorf("server received version %q, want %q", fake.lastCommitVersion, "v1.2.3")
	}
}

func TestClient_InstallUpdate(t *testing.T) {
	fake := &fakeServer{}
	c := dialTestServer(t, fake)

	if err := c.InstallUpdate(context.Background(), "v2.0.0", "https://example.com/bos.tar", "abc123"); err != nil {
		t.Fatalf("InstallUpdate() error = %v", err)
	}
	if fake.lastInstallReq.GetVersion() != "v2.0.0" {
		t.Errorf("server received version %q, want %q", fake.lastInstallReq.GetVersion(), "v2.0.0")
	}
	if fake.lastInstallReq.GetDownloadUrl() != "https://example.com/bos.tar" {
		t.Errorf("server received download_url %q", fake.lastInstallReq.GetDownloadUrl())
	}
}

func TestClient_GetUpdateStatus(t *testing.T) {
	fake := &fakeServer{}
	c := dialTestServer(t, fake)

	st, err := c.GetUpdateStatus(context.Background())
	if err != nil {
		t.Fatalf("GetUpdateStatus() error = %v", err)
	}
	if st.GetState() != supervisorpb.UpdateStatus_IDLE {
		t.Errorf("status.State = %v, want IDLE", st.GetState())
	}
}

func TestClient_InstallUpdate_alreadyInProgress(t *testing.T) {
	fake := &fakeServer{locked: true}
	c := dialTestServer(t, fake)

	err := c.InstallUpdate(context.Background(), "v3.0.0", "https://example.com/bos.tar", "def456")
	if err == nil {
		t.Fatal("InstallUpdate() expected error, got nil")
	}
	if got := status.Code(err); got != codes.FailedPrecondition {
		t.Errorf("status.Code(err) = %v, want FailedPrecondition", got)
	}
}
