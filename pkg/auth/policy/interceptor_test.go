package policy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"sync"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
)

func TestInterceptor_GRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)

	compiler, err := ast.CompileModules(regoFiles)
	if err != nil {
		t.Fatal(err)
	}
	interceptor := NewInterceptor(&static{compiler: compiler})
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
		grpc.ChainStreamInterceptor(interceptor.GRPCStreamingInterceptor()),
	)
	onoffpb.RegisterOnOffApiServer(server, onoffpb.NewModelServer(onoffpb.NewModel()))
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("server stopped with error: %v", err)
		}
	}()

	t.Cleanup(func() {
		if err := lis.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
		server.Stop()
	})

	ctx := context.Background()
	conn, err := grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}

	client := onoffpb.NewOnOffApiClient(conn)

	// check simple name based auth, global for all smartcore.* apis
	_, err = client.GetOnOff(ctx, &onoffpb.GetOnOffRequest{Name: "allow"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = client.GetOnOff(ctx, &onoffpb.GetOnOffRequest{Name: "deny"})
	if err == nil {
		t.Error("expected error")
	}
	if c := status.Code(err); c != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}

	// check action based auth, specific to this trait
	_, err = client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "any", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_ON}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "any", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}})
	if err == nil {
		t.Error("expected error")
	}
	if c := status.Code(err); c != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}
}

func TestInterceptor_HTTP(t *testing.T) {
	compiler, err := ast.CompileModules(regoFiles)
	if err != nil {
		t.Fatal(err)
	}
	interceptor := NewInterceptor(&static{compiler: compiler})

	server := httptest.NewTLSServer(interceptor.HTTPInterceptor(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// this handler should be called only for requests that are allowed by the policy
		writer.WriteHeader(http.StatusOK)
	})))
	defer server.Close()
	client := server.Client()

	check := func(method, path string, expectedStatus int) {
		req, err := http.NewRequest(method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != expectedStatus {
			t.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
		}
	}

	// all GET requests are allowed
	check(http.MethodGet, "/foo", http.StatusOK)
	check(http.MethodGet, "/bar", http.StatusOK)
	// POST requests are only allowed for /foo
	check(http.MethodPost, "/foo", http.StatusOK)
	check(http.MethodPost, "/bar", http.StatusUnauthorized)
}

func TestIsWriteMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		// read-only prefixes — must NOT be audited
		{"GetOnOff", false},
		{"GetBrightness", false},
		{"PullOnOff", false},
		{"PullDevices", false},
		{"DescribeOnOff", false},
		{"DescribeService", false},
		{"ListDevices", false},
		{"ListAlerts", false},
		{"ListHubNodes", false},
		// mutating methods that used to be missed — must be audited
		{"AcknowledgeAlert", true},
		{"UnacknowledgeAlert", true},
		{"ResolveAlert", true},
		{"EnrollHubNode", true},
		{"RenewHubNode", true},
		{"ForgetHubNode", true},
		{"RotateAccountClientSecret", true},
		{"SaveQRCredential", true},
		{"AddToGroup", true},
		{"RemoveFromGroup", true},
		{"StartFunctionTest", true},
		{"StopEmergencyTest", true},
		{"TestEnrollment", true},
		// standard mutating prefixes
		{"CreateAccessGrant", true},
		{"UpdateOnOff", true},
		{"DeleteAlert", true},
		{"SetBrightness", true},
		{"BatchUpdateDevices", true},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isWriteMethod(tt.method); got != tt.want {
				t.Errorf("isWriteMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestInterceptor_AuditSink(t *testing.T) {
	sink := &captureSink{}

	lis := bufconn.Listen(1024 * 1024)
	interceptor := NewInterceptor(AllowAll, WithAuditSink(sink))
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
		grpc.ChainStreamInterceptor(interceptor.GRPCStreamingInterceptor()),
	)
	onoffpb.RegisterOnOffApiServer(server, onoffpb.NewModelServer(onoffpb.NewModel()))
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	t.Cleanup(func() { lis.Close(); server.Stop() })

	ctx := context.Background()
	conn, err := grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := onoffpb.NewOnOffApiClient(conn)

	// GetOnOff must NOT produce an audit entry.
	_, err = client.GetOnOff(ctx, &onoffpb.GetOnOffRequest{Name: "x"})
	if err != nil {
		t.Fatalf("GetOnOff: %v", err)
	}
	if n := len(sink.all()); n != 0 {
		t.Errorf("GetOnOff: expected 0 audit entries, got %d", n)
	}

	// UpdateOnOff must produce exactly one audit entry with the expected fields.
	_, err = client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "x", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_ON}})
	if err != nil {
		t.Fatalf("UpdateOnOff: %v", err)
	}
	interceptor.Close() // drain the async audit queue before asserting
	msgs := sink.all()
	if n := len(msgs); n != 1 {
		t.Errorf("UpdateOnOff: expected 1 audit entry, got %d", n)
	} else {
		msg := msgs[0]
		if msg.Message != "write" {
			t.Errorf("message = %q, want %q", "write", msg.Message)
		}
		if v := msg.Fields["outcome"]; v != "allowed" {
			t.Errorf("outcome = %q, want %q", "allowed", v)
		}
		if v := msg.Fields["method"]; v != "UpdateOnOff" {
			t.Errorf("method = %q, want %q", "UpdateOnOff", v)
		}
	}
}

func TestInterceptor_AuditSink_DeniedWrite(t *testing.T) {
	sink := &captureSink{}

	lis := bufconn.Listen(1024 * 1024)
	interceptor := NewInterceptor(denyAll{}, WithAuditSink(sink))
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
		grpc.ChainStreamInterceptor(interceptor.GRPCStreamingInterceptor()),
	)
	onoffpb.RegisterOnOffApiServer(server, onoffpb.NewModelServer(onoffpb.NewModel()))
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	t.Cleanup(func() { lis.Close(); server.Stop() })

	ctx := context.Background()
	conn, err := grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := onoffpb.NewOnOffApiClient(conn)

	_, err = client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "x", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_ON}})
	if err == nil {
		t.Fatal("expected UpdateOnOff to be denied")
	}
	interceptor.Close() // drain the async audit queue before asserting
	msgs := sink.all()
	if n := len(msgs); n != 1 {
		t.Errorf("expected 1 audit entry for denied write, got %d", n)
	} else if v := msgs[0].Fields["outcome"]; v != "denied" {
		t.Errorf("outcome = %q, want %q", v, "denied")
	}
}

// captureSink is a test-only AuditSink that records every message it receives.
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

// denyAll is a Policy that rejects every request.
type denyAll struct{}

func (denyAll) EvalPolicy(_ context.Context, _ string, _ Attributes) (rego.ResultSet, error) {
	return rego.ResultSet{{Expressions: []*rego.ExpressionValue{{Value: false}}}}, nil
}

var regoFiles = map[string]string{
	"smartcore.rego": `package smartcore

# This simple rule allows any request whose name is "allow", all other requests are denied
allow {
	input.request.name == "allow"
}
`,
	"smartcore.bos.onoff.v1.OnOffApi.rego": `package smartcore.bos.onoff.v1.OnOffApi

# This rule allows people to turn any device on (but not off)
allow {
	input.method == "UpdateOnOff"
	input.request.onOff.state == "ON"
}
`,
	"http.rego": `package http

allow { input.method == "GET" }
allow { input.path == "/foo" }
`,
}
