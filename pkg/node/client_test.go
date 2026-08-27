package node

import (
	"context"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/interceptors"
	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// udmiStub stands in for a driver's UdmiService, which actuates real hardware in OnMessage.
type udmiStub struct {
	udmipb.UnimplementedUdmiServiceServer
	mu       sync.Mutex
	messages []*udmipb.MqttMessage
}

func (s *udmiStub) OnMessage(_ context.Context, req *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, req.Message)
	return &udmipb.OnMessageResponse{Name: req.Name}, nil
}

func (s *udmiStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

type captureSink struct {
	mu   sync.Mutex
	msgs []*logpb.LogMessage
}

func (s *captureSink) Write(msg *logpb.LogMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
}

func (s *captureSink) all() []*logpb.LogMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*logpb.LogMessage(nil), s.msgs...)
}

// auditedNode returns a Node whose in-process connection is audited, plus the sink and the
// interceptor (Close it to drain the async audit queue before asserting).
func auditedNode(t *testing.T, name string, srv udmipb.UdmiServiceServer) (*Node, *captureSink, *policy.Interceptor) {
	t.Helper()
	sink := &captureSink{}
	interceptor := policy.NewInterceptor(policy.AllowAll, policy.WithAuditSink(sink))
	n := New("test", nodeopts.WithClientConnWrapper(interceptor.AuditClientConn))
	n.Announce(name, HasServer(udmipb.RegisterUdmiServiceServer, srv))
	return n, sink, interceptor
}

// A write arriving over MQTT reaches the driver through the node's in-process connection, which
// bypasses the gRPC server's interceptor chain. It must still be audited: the broker supplies no
// smart-core identity, so these are the writes we can least otherwise account for.
func TestNode_ClientConn_AuditsInProcessWrite(t *testing.T) {
	srv := &udmiStub{}
	n, sink, interceptor := auditedNode(t, "dev-1", srv)
	client := udmipb.NewUdmiServiceClient(n.ClientConn())
	ctx := context.Background()

	_, err := client.OnMessage(ctx, &udmipb.OnMessageRequest{
		Name:    "dev-1",
		Message: &udmipb.MqttMessage{Topic: "site/dev-1/config", Payload: `{"pointset":{"points":{}}}`},
	})
	if err != nil {
		t.Fatalf("OnMessage: %v", err)
	}
	// the write really did reach the service; we are not just auditing a rejected call
	if got := srv.count(); got != 1 {
		t.Fatalf("service saw %d messages, want 1", got)
	}
	interceptor.Close()

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(msgs))
	}
	fields := msgs[0].Fields
	if got := fields["method"]; got != "OnMessage" {
		t.Errorf("method = %q, want %q", got, "OnMessage")
	}
	if got := fields["ingress"]; got != "loopback" {
		t.Errorf("ingress = %q, want %q", got, "loopback")
	}
	// the broker gives us no principal, so the entry must not imply one
	if got := fields["subject"]; got != "" {
		t.Errorf("subject = %q, want empty", got)
	}
}

// Reads over the in-process connection are the bulk of its traffic and must not be audited.
func TestNode_ClientConn_IgnoresInProcessRead(t *testing.T) {
	n, sink, interceptor := auditedNode(t, "dev-1", &udmiStub{})
	client := udmipb.NewUdmiServiceClient(n.ClientConn())

	// GetExportMessage is unimplemented by the stub; the call fails, but classification as a
	// read happens regardless of outcome, so nothing should be recorded either way.
	_, _ = client.GetExportMessage(context.Background(), &udmipb.GetExportMessageRequest{Name: "dev-1"})
	interceptor.Close()

	if got := len(sink.all()); got != 0 {
		t.Errorf("got %d audit entries, want 0", got)
	}
}

// External calls reach routed services through the server's unknown-service handler, not through
// ClientConn, so the two audit points must not both fire for one call.
func TestNode_ClientConn_NoDoubleAuditForExternalCall(t *testing.T) {
	srv := &udmiStub{}
	n, sink, interceptor := auditedNode(t, "dev-1", srv)

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			interceptors.CorrectStreamInfo(n),
			interceptor.GRPCStreamingInterceptor(),
		),
		grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
		grpc.UnknownServiceHandler(n.ServerHandler()),
	)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	t.Cleanup(func() { lis.Close(); server.Stop() })

	conn, err := grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	_, err = udmipb.NewUdmiServiceClient(conn).OnMessage(context.Background(), &udmipb.OnMessageRequest{
		Name:    "dev-1",
		Message: &udmipb.MqttMessage{Topic: "site/dev-1/config", Payload: "{}"},
	})
	if err != nil {
		t.Fatalf("OnMessage: %v", err)
	}
	interceptor.Close()

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(msgs))
	}
	// the entry came from the server interceptor, which knows the peer, not from the loopback wrapper
	if got := msgs[0].Fields["ingress"]; got != "" {
		t.Errorf("ingress = %q, want empty (server-path entry)", got)
	}
	if got := msgs[0].Fields["outcome"]; got != "allowed" {
		t.Errorf("outcome = %q, want %q", got, "allowed")
	}
}
