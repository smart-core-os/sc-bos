package supervisor_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// recordingServer is a SupervisorApiServer that records every committed version in order.
type recordingServer struct {
	supervisorpb.UnimplementedSupervisorApiServer

	mu       sync.Mutex
	versions []string
}

func (s *recordingServer) Commit(_ context.Context, req *supervisorpb.CommitRequest) (*supervisorpb.CommitResponse, error) {
	s.mu.Lock()
	s.versions = append(s.versions, req.GetVersion())
	s.mu.Unlock()
	return &supervisorpb.CommitResponse{}, nil
}

func (s *recordingServer) committed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.versions))
	copy(out, s.versions)
	return out
}

// dialRecordingServer starts a real gRPC server over a Unix socket backed by rec, dials a Client,
// and registers test cleanup. Reuses tempSockPath from client_test.go (same _test package).
func dialRecordingServer(t *testing.T, rec *recordingServer) *supervisor.Client {
	t.Helper()
	sockPath := tempSockPath(t)

	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s) = %v", sockPath, err)
	}
	srv := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(srv, rec)
	t.Cleanup(func() { srv.Stop() })
	go func() { _ = srv.Serve(lis) }()

	c, err := supervisor.Dial(sockPath, 5*time.Second)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestRunStartupCommit_CommitsOnce verifies that, with no check-in gate, RunStartupCommit issues
// exactly one Commit on startup with the given version, over a real gRPC server on a Unix socket.
// It returns once the commit completes (it is a one-shot, not a heartbeat).
func TestRunStartupCommit_CommitsOnce(t *testing.T) {
	const wantVersion = "v1.2.3-test"

	rec := &recordingServer{}
	c := dialRecordingServer(t, rec)

	err := supervisor.RunStartupCommit(context.Background(), c, wantVersion, nil, nil, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("RunStartupCommit() = %v", err)
	}

	got := rec.committed()
	if len(got) != 1 {
		t.Fatalf("want exactly 1 Commit call, got %d (%v)", len(got), got)
	}
	if got[0] != wantVersion {
		t.Errorf("Commit version = %q, want %q", got[0], wantVersion)
	}
}
