package sc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/smart-core-os/gobacnet/encoding"
	"github.com/smart-core-os/gobacnet/enum/pdutype"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
	"go.uber.org/zap/zaptest"
	"nhooyr.io/websocket"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

// fakeHub is a minimal BACnet/SC hub: it completes the connect handshake and
// answers any Encapsulated-NPDU (assumed to be a Who-Is) with an I-Am from a
// fixed device.
func fakeHub(t *testing.T, hubVMAC, deviceVMAC VMAC, instance uint32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{subprotocolHub},
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if typ != websocket.MessageBinary {
				continue
			}
			msg, err := decodeBVLC(data)
			if err != nil {
				continue
			}
			switch msg.Function {
			case bvlcConnectRequest:
				accept := bvlcMessage{
					Function:  bvlcConnectAccept,
					MessageID: msg.MessageID,
					Payload: connectInfo{
						VMAC:          hubVMAC,
						MaxBVLCLength: 1497,
						MaxNPDULength: 1497,
					}.encode(),
				}
				_ = conn.Write(ctx, websocket.MessageBinary, accept.encode())
			case bvlcEncapsulatedNPDU:
				resp := bvlcMessage{
					Function:  bvlcEncapsulatedNPDU,
					MessageID: 1,
					HasOrig:   true,
					Orig:      deviceVMAC,
					HasDest:   true,
					Dest:      msg.Orig,
					Payload:   buildIAm(instance),
				}
				_ = conn.Write(ctx, websocket.MessageBinary, resp.encode())
			case bvlcHeartbeatRequest:
				ack := bvlcMessage{Function: bvlcHeartbeatACK, MessageID: msg.MessageID}
				_ = conn.Write(ctx, websocket.MessageBinary, ack.encode())
			}
		}
	}))
}

func buildIAm(instance uint32) []byte {
	enc := encoding.NewEncoder()
	enc.NPDU(bactypes.NPDU{
		Version:  bactypes.ProtocolVersion,
		Priority: bactypes.Normal,
	})
	_ = enc.APDU(bactypes.APDU{
		DataType:           pdutype.UnconfirmedServiceRequest,
		UnconfirmedService: bactypes.ServiceUnconfirmedIAm,
	})
	_ = enc.IAm(bactypes.IAm{
		ID:           bactypes.ObjectID{Type: objecttype.Device, Instance: bactypes.ObjectInstance(instance)},
		MaxApdu:      1476,
		Segmentation: bactypes.Enumerated(3),
		Vendor:       42,
	})
	return enc.Bytes()
}

func TestClientWhoIsOverFakeHub(t *testing.T) {
	const instance = 1234
	deviceVMAC := VMAC{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01}
	hubVMAC := VMAC{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}

	srv := fakeHub(t, hubVMAC, deviceVMAC, instance)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	cfg := config.SecureConnect{
		PrimaryHubURI:     wsURL,
		ConnectTimeout:    config.Duration{Duration: 2 * time.Second},
		HeartbeatInterval: config.Duration{Duration: time.Hour}, // suppress heartbeats during the test
	}
	client, err := NewClient(cfg, 0, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := client.WhoIs(ctx, instance, instance)
	if err != nil {
		t.Fatalf("WhoIs: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d: %+v", len(devices), devices)
	}
	dev := devices[0]
	if dev.ID.Instance != instance {
		t.Errorf("device instance = %d, want %d", dev.ID.Instance, instance)
	}
	if dev.Vendor != 42 {
		t.Errorf("device vendor = %d, want 42", dev.Vendor)
	}
	if got := addressToVMAC(dev.Addr); got != deviceVMAC {
		t.Errorf("device addr vmac = %v, want %v", got, deviceVMAC)
	}
}
