package app

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"

	bolterrors "go.etcd.io/bbolt/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// TestController_closesDatabase checks Bootstrap registers the local bolt database for cleanup, so
// that Run closes it on the way out. A leaked handle is invisible on Linux — it only shows up on
// Windows, as a TempDir cleanup failure, because Windows won't unlink an open file — so assert the
// close directly rather than relying on that symptom.
func TestController_closesDatabase(t *testing.T) {
	config := sysconf.Default()
	config.PolicyMode = sysconf.PolicyOff
	config.DataDir = t.TempDir()
	// Don't bind real ports while testing.
	config.ListenGRPC = ""
	config.ListenHTTPS = ""

	c, err := Bootstrap(t.Context(), config)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if c.Database == nil {
		t.Fatal("Bootstrap() left Database nil, nothing to assert on")
	}

	// Run executes the deferred closers as it returns; cancel immediately, we don't need it to do
	// any work first.
	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() { runErr <- c.Run(ctx) }()
	cancel()
	<-runErr

	if _, err := c.Database.Bolt().Begin(false); !errors.Is(err, bolterrors.ErrDatabaseNotOpen) {
		t.Errorf("bolt database still open after Run returned: Begin() error = %v, want %v", err, bolterrors.ErrDatabaseNotOpen)
	}
}

// TestController_protoPkgCompat tests that both versioned and unversioned proto packages are served.
// We had a bug where only dynamically registered services (i.e. traits) were served,
// but statically registered services (i.e. devices api) were not.
func TestController_protoPkgCompat(t *testing.T) {
	config := sysconf.Default()
	config.PolicyMode = sysconf.PolicyOff
	c, err := Bootstrap(t.Context(), config)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// so there's something to return
	c.Node.Announce("test-device",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterpb.NewModel()))),
		node.HasTrait(meterpb.TraitName),
	)

	bufl := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() {
		c.GRPC.Stop()
		bufl.Close()
	})
	go func() {
		err := c.GRPC.Serve(bufl)
		if err != nil {
			t.Errorf("gRPC Serve() error = %v", err)
		}
	}()

	cc, err := grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return bufl.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer cc.Close()

	// devices api (statically registered)
	devReq := &devicespb.ListDevicesRequest{}
	revRes := new(devicespb.ListDevicesResponse)
	if err := cc.Invoke(t.Context(), "/smartcore.bos.DevicesApi/ListDevices", devReq, revRes); err != nil {
		t.Errorf("unversioned ListDevices() error = %v", err)
	}
	if err := cc.Invoke(t.Context(), "/smartcore.bos.devices.v1.DevicesApi/ListDevices", devReq, revRes); err != nil {
		t.Errorf("versioned ListDevices() error = %v", err)
	}
	// trait api (dynamically registered)
	meterReq := &meterpb.GetMeterReadingRequest{Name: "test-device"}
	meterRes := new(meterpb.MeterReading)
	if err := cc.Invoke(t.Context(), "/smartcore.bos.MeterApi/GetMeterReading", meterReq, meterRes); err != nil {
		t.Errorf("unversioned GetMeterReading() error = %v", err)
	}
	if err := cc.Invoke(t.Context(), "/smartcore.bos.meter.v1.MeterApi/GetMeterReading", meterReq, meterRes); err != nil {
		t.Errorf("versioned GetMeterReading() error = %v", err)
	}
}
