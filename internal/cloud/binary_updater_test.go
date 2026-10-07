package cloud

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

// TestConn_SupervisorBusyWaitsForSocket verifies the start-up status check succeeds when the Supervisor
// starts listening after the check begins.
func TestConn_SupervisorBusyWaitsForSocket(t *testing.T) {
	sockPath, installer := dialLateSupervisor(t)
	conn := openConnWithInstaller(t, &scriptedClient{}, installer)

	type result struct {
		busy bool
		err  error
	}
	resc := make(chan result, 1)
	go func() {
		busy, err := conn.supervisorBusy(t.Context())
		resc <- result{busy, err}
	}()

	time.Sleep(500 * time.Millisecond) // the Supervisor starts late
	serveFakeSupervisor(t, sockPath, &fakeSupervisor{status: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_INSTALLING}})

	res := <-resc
	if res.err != nil {
		t.Fatalf("supervisorBusy() = %v, want nil", res.err)
	}
	if !res.busy {
		t.Error("supervisorBusy() = false, want true while INSTALLING")
	}
}

// TestConn_UpdateFailsFastWithoutSupervisor verifies a check-in doesn't wait for a missing Supervisor socket,
// so it doesn't hold up other users of the connection.
func TestConn_UpdateFailsFastWithoutSupervisor(t *testing.T) {
	_, installer := dialLateSupervisor(t) // never listens
	conn := openConnWithInstaller(t, &scriptedClient{}, installer)

	_, err := conn.Update(t.Context())
	if got := status.Code(err); got != codes.Unavailable {
		t.Errorf("Update() = %v, want code Unavailable", err)
	}
}

// dialLateSupervisor returns a socket path with nothing listening on it yet, and a Supervisor client for it.
func dialLateSupervisor(t *testing.T) (string, supervisorpb.SupervisorApiClient) {
	t.Helper()
	dir, err := os.MkdirTemp("", "sup-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "s.sock")

	cc, err := supervisor.Dial(sockPath)
	if err != nil {
		t.Fatalf("supervisor.Dial() = %v", err)
	}
	t.Cleanup(func() { _ = cc.Close() })
	return sockPath, supervisorpb.NewSupervisorApiClient(cc)
}

// serveFakeSupervisor serves fake on sockPath until the test ends.
func serveFakeSupervisor(t *testing.T, sockPath string, fake *fakeSupervisor) {
	t.Helper()
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s) = %v", sockPath, err)
	}
	srv := grpc.NewServer()
	supervisorpb.RegisterSupervisorApiServer(srv, fake)
	t.Cleanup(srv.Stop)
	go func() { _ = srv.Serve(lis) }()
}

// openConnWithInstaller opens a registered Conn that checks in through client and, when installer is not nil,
// reads update status from installer.
func openConnWithInstaller(t *testing.T, client Client, installer supervisorpb.SupervisorApiClient) *Conn {
	t.Helper()
	regStore, depStore := newStores(t)
	if err := regStore.Save(context.Background(), testRegistration(t, "node-a")); err != nil {
		t.Fatalf("save registration: %v", err)
	}
	connOpts := []ConnOption{WithClientFactory(func(*Registration) Client { return client })}
	if installer != nil {
		connOpts = append(connOpts, WithBinaryUpdater(NewBinaryUpdater(WithBinaryInstaller(installer))))
	}
	conn, err := OpenConn(context.Background(), regStore, depStore, "", connOpts...)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return conn
}
