package cloud

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/internal/supervisor"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
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

// TestConn_UpdateDoesNotLockWhileReadingSupervisor verifies other users of the connection aren't blocked while
// a check-in waits on the Supervisor.
func TestConn_UpdateDoesNotLockWhileReadingSupervisor(t *testing.T) {
	sup := &blockingSupervisor{entered: make(chan struct{}), release: make(chan struct{})}
	installer := supervisorpb.NewSupervisorApiClient(wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, sup))
	conn := openConnWithInstaller(t, &scriptedClient{}, installer)

	updateDone := make(chan error, 1)
	go func() {
		_, err := conn.Update(t.Context())
		updateDone <- err
	}()
	<-sup.entered // Update is waiting on the Supervisor

	testDone := make(chan error, 1)
	go func() { testDone <- conn.TestConn(t.Context()) }()
	select {
	case err := <-testDone:
		if err != nil {
			t.Errorf("TestConn() = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("TestConn blocked while Update waited on the Supervisor")
	}

	close(sup.release)
	if err := <-updateDone; err != nil {
		t.Errorf("Update() = %v, want nil", err)
	}
}

// blockingSupervisor answers GetUpdateStatus only once release is closed, closing entered when the first call
// arrives.
type blockingSupervisor struct {
	supervisorpb.UnimplementedSupervisorApiServer
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockingSupervisor) GetUpdateStatus(ctx context.Context, _ *supervisorpb.GetUpdateStatusRequest) (*supervisorpb.GetUpdateStatusResponse, error) {
	s.once.Do(func() { close(s.entered) })
	select {
	case <-s.release:
		return &supervisorpb.GetUpdateStatusResponse{Status: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_IDLE}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
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
