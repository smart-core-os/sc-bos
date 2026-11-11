package comm

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/gobacnet/encoding"
	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
)

// gobacnet waits for a reply until the request ctx is done.
// A device that never replies, or replies with an Abort (which gobacnet drops), leaves a request
// with no deadline blocked forever. These tests pin that behaviour and show Client caps it.

func TestClient_ReadProperty_noReply(t *testing.T) {
	testClientReadPropertyHang(t, func(*net.UDPConn, *net.UDPAddr, uint8) {})
}

func TestClient_ReadProperty_abort(t *testing.T) {
	testClientReadPropertyHang(t, func(conn *net.UDPConn, from *net.UDPAddr, invokeID uint8) {
		// BVLC unicast | NPDU v1, no control | APDU abort, server, invokeID, reason 4 (segmentation not supported)
		abort := []byte{0x81, 0x0a, 0x00, 0x09, 0x01, 0x00, 0x71, invokeID, 0x04}
		_, _ = conn.WriteToUDP(abort, from)
	})
}

func testClientReadPropertyHang(t *testing.T, reply func(conn *net.UDPConn, from *net.UDPAddr, invokeID uint8)) {
	device := newStubDevice(t, reply)
	raw := newLoopbackClient(t)

	t.Run("uncapped", func(t *testing.T) {
		// Before SCB-1510 every driver call went straight to gobacnet with the caller's ctx.
		// The ctx here has no deadline, so we can only show the call is still blocked after a while.
		ctx, cancel := context.WithCancel(t.Context())
		errs := make(chan error, 1)
		go func() {
			_, err := raw.ReadProperty(ctx, device.dev, readPresentValue())
			errs <- err
		}()
		select {
		case err := <-errs:
			t.Fatalf("ReadProperty returned without a deadline, gobacnet may now handle this itself: %v", err)
		case <-time.After(500 * time.Millisecond):
		}
		cancel()
		select {
		case <-errs:
		case <-time.After(5 * time.Second):
			t.Fatal("ReadProperty didn't return after ctx was cancelled")
		}
	})

	t.Run("capped", func(t *testing.T) {
		const timeout = 200 * time.Millisecond
		client := NewClient(raw, timeout)
		start := time.Now()
		_, err := client.ReadProperty(t.Context(), device.dev, readPresentValue())
		elapsed := time.Since(start)

		if elapsed > timeout+2*time.Second {
			t.Errorf("ReadProperty took %s, want about %s", elapsed, timeout)
		}
		var timeoutErr *RequestTimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Fatalf("want RequestTimeoutError, got %T %v", err, err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("error must not look like the caller's ctx expired: %v", err)
		}
		if got := status.Code(err); got != codes.DeadlineExceeded {
			t.Errorf("status code = %v, want %v", got, codes.DeadlineExceeded)
		}
	})

	t.Run("caller deadline first", func(t *testing.T) {
		client := NewClient(raw, time.Minute)
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()
		_, err := client.ReadProperty(ctx, device.dev, readPresentValue())
		var timeoutErr *RequestTimeoutError
		if errors.As(err, &timeoutErr) {
			t.Fatalf("caller's own deadline should not be reported as a request timeout: %v", err)
		}
		if err == nil {
			t.Fatal("want an error")
		}
	})

	if device.requests() == 0 {
		t.Fatal("stub device received no requests")
	}
}

func readPresentValue() bactypes.ReadPropertyData {
	return bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID:         bactypes.ObjectID{Type: objecttype.AnalogValue, Instance: 1},
			Properties: []bactypes.Property{{ID: property.PresentValue, ArrayIndex: bactypes.ArrayAll}},
		},
	}
}

type stubDevice struct {
	dev  bactypes.Device
	reqs chan struct{}
}

func (d *stubDevice) requests() int {
	return len(d.reqs)
}

// newStubDevice listens on loopback as a BACnet device, calling reply with the invoke ID of each confirmed request.
func newStubDevice(t *testing.T, reply func(conn *net.UDPConn, from *net.UDPAddr, invokeID uint8)) *stubDevice {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	d := &stubDevice{
		dev: bactypes.Device{
			ID:      bactypes.ObjectID{Type: objecttype.Device, Instance: 1},
			Addr:    bactypes.UDPToAddress(conn.LocalAddr().(*net.UDPAddr)),
			MaxApdu: 1476,
		},
		reqs: make(chan struct{}, 1000),
	}
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			var (
				bvlc bactypes.BVLC
				npdu bactypes.NPDU
				apdu bactypes.APDU
			)
			dec := encoding.NewDecoder(buf[:n])
			if dec.BVLC(&bvlc) != nil || dec.NPDU(&npdu) != nil {
				continue
			}
			_ = dec.APDU(&apdu) // the service data doesn't matter, only the invoke ID
			select {
			case d.reqs <- struct{}{}:
			default:
			}
			reply(conn, from, apdu.InvokeId)
		}
	}()
	return d
}

// newLoopbackClient returns a gobacnet client bound to an ephemeral port on the loopback interface.
func newLoopbackClient(t *testing.T) *gobacnet.Client {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		client, err := gobacnet.NewClient(iface.Name, 0, gobacnet.WithLogLevel(logrus.PanicLevel))
		if err != nil {
			continue
		}
		t.Cleanup(client.Close)
		return client
	}
	t.Skip("no usable IPv4 loopback interface")
	return nil
}
