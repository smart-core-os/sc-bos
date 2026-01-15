package app

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

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
		node.HasTrait(meter.TraitName,
			node.WithClients(meterpb.WrapApi(meter.NewModelServer(meter.NewModel()))),
		),
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
