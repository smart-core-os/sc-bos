package helvarnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

const testPirAddr = "1.2.3.4"

// fakeRouter answers HelvarNet input-state queries (command 151) over TCP, as a router would.
type fakeRouter struct {
	t  *testing.T
	ln net.Listener

	mu     sync.Mutex
	state  int
	silent bool
	conns  []net.Conn
}

func newFakeRouter(t *testing.T) *fakeRouter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	f := &fakeRouter{t: t, ln: ln}
	go f.accept()
	t.Cleanup(f.stop)
	return f
}

func (f *fakeRouter) accept() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		f.mu.Lock()
		f.conns = append(f.conns, conn)
		f.mu.Unlock()
		go f.serve(conn)
	}
}

func (f *fakeRouter) serve(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		req := string(buf[:n])
		if !strings.HasPrefix(req, ">V:1,C:151,@") || !strings.HasSuffix(req, "#") {
			continue
		}
		f.mu.Lock()
		state, silent := f.state, f.silent
		f.mu.Unlock()
		if silent {
			continue
		}
		resp := fmt.Sprintf("?%s=%d#", req[1:len(req)-1], state)
		if _, err := conn.Write([]byte(resp)); err != nil {
			return
		}
	}
}

func (f *fakeRouter) setState(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = n
}

func (f *fakeRouter) setSilent(silent bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.silent = silent
}

// stop closes the listener and every open connection, so later dials are refused.
func (f *fakeRouter) stop() {
	_ = f.ln.Close()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conns {
		_ = c.Close()
	}
	f.conns = nil
}

func (f *fakeRouter) addr() *net.TCPAddr {
	return f.ln.Addr().(*net.TCPAddr)
}

func newTestPir(t *testing.T, router *fakeRouter) *Pir {
	t.Helper()
	bufSize := 1024
	cfg := &config.Root{
		ConnectTimeout:     &jsontypes.Duration{Duration: time.Second},
		RxTimeout:          &jsontypes.Duration{Duration: 200 * time.Millisecond},
		SendPacketTimeout:  &jsontypes.Duration{Duration: time.Second},
		RetrySleepDuration: &jsontypes.Duration{Duration: time.Millisecond},
		RxBufferSize:       &bufSize,
	}
	client := newTcpClient(router.addr(), zap.NewNop(), cfg)
	t.Cleanup(func() {
		client.mu.Lock()
		defer client.mu.Unlock()
		client.close()
	})
	return newPir(client, zap.NewNop(), &config.Device{
		Name:        "pir-1",
		Address:     testPirAddr,
		TopicPrefix: "site/pir-1",
	})
}

func refresh(t *testing.T, p *Pir) {
	t.Helper()
	if _, err := p.refreshOccupancyStatus(context.Background()); err != nil {
		t.Fatalf("refreshOccupancyStatus: %v", err)
	}
}

func occupancyPoint(t *testing.T, m *udmipb.MqttMessage) string {
	t.Helper()
	var points udmi.PointsEvent
	if err := json.Unmarshal([]byte(m.GetPayload()), &points); err != nil {
		t.Fatalf("payload %q is not a pointset: %v", m.GetPayload(), err)
	}
	v, ok := points["OccupancyStatus"]
	if !ok {
		t.Fatalf("payload %q has no OccupancyStatus point", m.GetPayload())
	}
	s, _ := v.PresentValue.(string)
	return s
}

// pullExports opens a PullExportMessages stream over the gRPC service and relays what arrives.
func pullExports(t *testing.T, p *Pir) <-chan *udmipb.PullExportMessagesResponse {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client := udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, p))
	stream, err := client.PullExportMessages(ctx, &udmipb.PullExportMessagesRequest{})
	if err != nil {
		t.Fatalf("PullExportMessages: %v", err)
	}
	ch := make(chan *udmipb.PullExportMessagesResponse, 10)
	go func() {
		defer close(ch)
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			ch <- msg
		}
	}()
	return ch
}

func expectExport(t *testing.T, ch <-chan *udmipb.PullExportMessagesResponse, wantState string) {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatalf("stream closed, want %s", wantState)
		}
		if got, want := msg.GetName(), "pir-1"; got != want {
			t.Errorf("name = %q, want %q", got, want)
		}
		if got, want := msg.GetMessage().GetTopic(), "site/pir-1/event/pointset/points"; got != want {
			t.Errorf("topic = %q, want %q", got, want)
		}
		if got := occupancyPoint(t, msg.GetMessage()); got != wantState {
			t.Errorf("OccupancyStatus = %q, want %q", got, wantState)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("no export arrived, want %s", wantState)
	}
}

// quietWindow outlasts a failed read against a silent router (3 tries of RxTimeout), so a
// stream's opening read has finished before we conclude it sent nothing.
const quietWindow = 1500 * time.Millisecond

func expectNoExport(t *testing.T, ch <-chan *udmipb.PullExportMessagesResponse, wait time.Duration) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("unexpected export: %v", msg)
	case <-time.After(wait):
	}
}

func TestPir_PullExportMessages(t *testing.T) {
	router := newFakeRouter(t)
	p := newTestPir(t, router)

	// the device has never answered, so there is nothing true to say
	router.setSilent(true)
	stream := pullExports(t, p)
	expectNoExport(t, stream, quietWindow)

	router.setSilent(false)
	router.setState(1)
	refresh(t, p)
	expectExport(t, stream, "OCCUPIED")

	router.setState(0)
	refresh(t, p)
	expectExport(t, stream, "UNOCCUPIED")

	// a poll that sees no change sends nothing
	refresh(t, p)
	expectNoExport(t, stream, 200*time.Millisecond)

	// a new subscriber is sent a reading the device has just confirmed
	late := pullExports(t, p)
	expectExport(t, late, "UNOCCUPIED")
	expectNoExport(t, late, 200*time.Millisecond)

	// a change found by a new subscriber's opening read goes out once on each stream
	router.setState(1)
	later := pullExports(t, p)
	expectExport(t, later, "OCCUPIED")
	expectExport(t, stream, "OCCUPIED")
	expectNoExport(t, later, 200*time.Millisecond)
	expectNoExport(t, stream, 200*time.Millisecond)

	// the device stops answering with a reading still cached: resubscribing, as the udmi auto
	// does when its export task retries, must not replay that reading as if it were current
	router.setSilent(true)
	resub := pullExports(t, p)
	expectNoExport(t, resub, quietWindow)
	expectNoExport(t, stream, 200*time.Millisecond)
}

func TestPir_GetExportMessage(t *testing.T) {
	t.Run("live reading", func(t *testing.T) {
		router := newFakeRouter(t)
		p := newTestPir(t, router)

		router.setState(1)
		m, err := p.GetExportMessage(context.Background(), &udmipb.GetExportMessageRequest{})
		if err != nil {
			t.Fatalf("GetExportMessage: %v", err)
		}
		if got := occupancyPoint(t, m); got != "OCCUPIED" {
			t.Errorf("OccupancyStatus = %q, want OCCUPIED", got)
		}
		if got, want := m.GetTopic(), "site/pir-1/event/pointset/points"; got != want {
			t.Errorf("topic = %q, want %q", got, want)
		}

		// no poll in between: the answer must come from the device, not the cache
		router.setState(0)
		m, err = p.GetExportMessage(context.Background(), &udmipb.GetExportMessageRequest{})
		if err != nil {
			t.Fatalf("GetExportMessage: %v", err)
		}
		if got := occupancyPoint(t, m); got != "UNOCCUPIED" {
			t.Errorf("OccupancyStatus = %q, want UNOCCUPIED", got)
		}
	})

	unavailable := []struct {
		name string
		fail func(*fakeRouter)
		// whether the PIR had a good reading before the router stopped answering
		readFirst bool
	}{
		{name: "silent before any read", fail: func(r *fakeRouter) { r.setSilent(true) }},
		{name: "silent after a good read", fail: func(r *fakeRouter) { r.setSilent(true) }, readFirst: true},
		{name: "refused before any read", fail: (*fakeRouter).stop},
		{name: "refused after a good read", fail: (*fakeRouter).stop, readFirst: true},
	}
	for _, tt := range unavailable {
		t.Run(tt.name, func(t *testing.T) {
			router := newFakeRouter(t)
			p := newTestPir(t, router)
			if tt.readFirst {
				router.setState(1)
				refresh(t, p)
			}
			tt.fail(router)

			m, err := p.GetExportMessage(context.Background(), &udmipb.GetExportMessageRequest{})
			if got := status.Code(err); got != codes.Unavailable {
				t.Fatalf("GetExportMessage = %v, %v; want code Unavailable", m, err)
			}
		})
	}

	t.Run("cancelled", func(t *testing.T) {
		router := newFakeRouter(t)
		p := newTestPir(t, router)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := p.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{})
		if got := status.Code(err); got != codes.Canceled {
			t.Fatalf("GetExportMessage err = %v, want code Canceled", err)
		}
	})
}
