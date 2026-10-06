package supervisor_test

import (
	"context"
	"errors"
	"math"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
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

// dialRecordingServer starts a real gRPC server over a Unix socket backed by rec, connects a
// supervisorpb client, and registers test cleanup.
func dialRecordingServer(t *testing.T, rec *recordingServer) supervisorpb.SupervisorApiClient {
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

	conn, err := supervisor.Dial(sockPath)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return supervisorpb.NewSupervisorApiClient(conn)
}

// TestCommitUntilAccepted_CommitsOnce verifies CommitUntilAccepted issues exactly one Commit with the given
// version when the Supervisor accepts it, over a real gRPC server on a Unix socket.
func TestCommitUntilAccepted_CommitsOnce(t *testing.T) {
	const wantVersion = "v1.2.3-test"

	rec := &recordingServer{}
	c := dialRecordingServer(t, rec)

	if err := supervisor.CommitUntilAccepted(context.Background(), c, wantVersion, zaptest.NewLogger(t)); err != nil {
		t.Fatalf("CommitUntilAccepted() = %v, want nil", err)
	}

	got := rec.committed()
	if len(got) != 1 {
		t.Fatalf("want exactly 1 Commit call, got %d (%v)", len(got), got)
	}
	if got[0] != wantVersion {
		t.Errorf("Commit version = %q, want %q", got[0], wantVersion)
	}
}

// flakyServer fails the first failures Commit calls with code (Unavailable if unset), then records commits.
type flakyServer struct {
	recordingServer
	failures int
	code     codes.Code
	attempts []time.Time
}

func (s *flakyServer) Commit(ctx context.Context, req *supervisorpb.CommitRequest) (*supervisorpb.CommitResponse, error) {
	s.mu.Lock()
	s.attempts = append(s.attempts, time.Now())
	fail := len(s.attempts) <= s.failures
	s.mu.Unlock()
	if fail {
		code := s.code
		if code == codes.OK {
			code = codes.Unavailable
		}
		return nil, status.Error(code, "commit failed")
	}
	return s.recordingServer.Commit(ctx, req)
}

// TestCommitUntilAccepted_RetriesUntilCommitted verifies failed commits are retried until one succeeds.
func TestCommitUntilAccepted_RetriesUntilCommitted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := &flakyServer{failures: 3}
		c := supervisorpb.NewSupervisorApiClient(wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, srv))

		if err := supervisor.CommitUntilAccepted(t.Context(), c, "v1", zaptest.NewLogger(t)); err != nil {
			t.Fatalf("CommitUntilAccepted() = %v, want nil", err)
		}
		if got := len(srv.attempts); got != 4 {
			t.Errorf("Commit attempts = %d, want 4", got)
		}
		if got := srv.committed(); !slices.Equal(got, []string{"v1"}) {
			t.Errorf("committed = %v, want [v1]", got)
		}
	})
}

// TestCommitUntilAccepted_StopsWhenRejected verifies CommitUntilAccepted makes one attempt and returns the
// Supervisor's error when it rejects the request as invalid, such as an empty version.
func TestCommitUntilAccepted_StopsWhenRejected(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := &flakyServer{failures: math.MaxInt, code: codes.InvalidArgument}
		c := supervisorpb.NewSupervisorApiClient(wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, srv))

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
		defer cancel()
		err := supervisor.CommitUntilAccepted(ctx, c, "", zaptest.NewLogger(t))
		if got := status.Code(err); got != codes.InvalidArgument {
			t.Errorf("CommitUntilAccepted() = %v, want code InvalidArgument", err)
		}
		if got := len(srv.attempts); got != 1 {
			t.Errorf("Commit attempts = %d, want 1", got)
		}
	})
}

// TestCommitUntilAccepted_StopsWhenCancelled verifies CommitUntilAccepted returns ctx.Err() once ctx is done,
// even if no commit has succeeded.
func TestCommitUntilAccepted_StopsWhenCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := &flakyServer{failures: math.MaxInt}
		c := supervisorpb.NewSupervisorApiClient(wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, srv))

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
		defer cancel()
		if err := supervisor.CommitUntilAccepted(ctx, c, "v1", zaptest.NewLogger(t)); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("CommitUntilAccepted() = %v, want %v", err, context.DeadlineExceeded)
		}
		if got := srv.committed(); len(got) != 0 {
			t.Errorf("want no successful Commit, got %v", got)
		}
	})
}

// TestCommit_WaitsForSocket verifies a single Commit attempt succeeds when the Supervisor starts listening
// after the attempt begins.
func TestCommit_WaitsForSocket(t *testing.T) {
	sockPath := tempSockPath(t)
	conn, err := supervisor.Dial(sockPath)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	errc := make(chan error, 1)
	go func() {
		errc <- supervisor.Commit(t.Context(), supervisorpb.NewSupervisorApiClient(conn), "v1")
	}()

	time.Sleep(500 * time.Millisecond) // the Supervisor starts late
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s) = %v", sockPath, err)
	}
	rec := &recordingServer{}
	srv := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(srv, rec)
	t.Cleanup(srv.Stop)
	go func() { _ = srv.Serve(lis) }()

	if err := <-errc; err != nil {
		t.Fatalf("Commit() = %v, want nil", err)
	}
	if got := rec.committed(); !slices.Equal(got, []string{"v1"}) {
		t.Errorf("committed = %v, want [v1]", got)
	}
}
