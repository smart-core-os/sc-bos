package cloud

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// TestBinaryUpdater_UpdateStatusWaitsForSocket verifies reading the Supervisor's status succeeds when the
// Supervisor starts listening after the read begins.
func TestBinaryUpdater_UpdateStatusWaitsForSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "sup-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "s.sock")

	conn, err := supervisor.Dial(sockPath)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	u := NewBinaryUpdater(WithBinaryInstaller(supervisorpb.NewSupervisorApiClient(conn)))

	type result struct {
		st  *supervisorpb.UpdateStatus
		err error
	}
	resc := make(chan result, 1)
	go func() {
		st, err := u.updateStatus(t.Context())
		resc <- result{st, err}
	}()

	time.Sleep(500 * time.Millisecond) // the Supervisor starts late
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s) = %v", sockPath, err)
	}
	fake := &fakeSupervisor{status: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_INSTALLING}}
	srv := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(srv, fake)
	t.Cleanup(srv.Stop)
	go func() { _ = srv.Serve(lis) }()

	res := <-resc
	if res.err != nil {
		t.Fatalf("updateStatus() = %v, want nil", res.err)
	}
	if got := res.st.GetState(); got != supervisorpb.UpdateStatus_INSTALLING {
		t.Errorf("state = %v, want INSTALLING", got)
	}
}
