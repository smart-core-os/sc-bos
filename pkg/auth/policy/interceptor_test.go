package policy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait/onoffpb"
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
	traits.RegisterOnOffApiServer(server, onoffpb.NewModelServer(onoffpb.NewModel()))
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

	client := traits.NewOnOffApiClient(conn)

	// check simple name based auth, global for all smartcore.* apis
	_, err = client.GetOnOff(ctx, &traits.GetOnOffRequest{Name: "allow"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = client.GetOnOff(ctx, &traits.GetOnOffRequest{Name: "deny"})
	if err == nil {
		t.Error("expected error")
	}
	if c := status.Code(err); c != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}

	// check action based auth, specific to this trait
	_, err = client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: "any", OnOff: &traits.OnOff{State: traits.OnOff_ON}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: "any", OnOff: &traits.OnOff{State: traits.OnOff_OFF}})
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

func TestInterceptor_AuditLogger(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	auditLog := zap.New(core)

	lis := bufconn.Listen(1024 * 1024)
	interceptor := NewInterceptor(allowAll{}, WithAuditLogger(auditLog))
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
		grpc.ChainStreamInterceptor(interceptor.GRPCStreamingInterceptor()),
	)
	traits.RegisterOnOffApiServer(server, onoffpb.NewModelServer(onoffpb.NewModel()))
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
	client := traits.NewOnOffApiClient(conn)

	// GetOnOff must NOT produce an audit entry.
	_, err = client.GetOnOff(ctx, &traits.GetOnOffRequest{Name: "x"})
	if err != nil {
		t.Fatalf("GetOnOff: %v", err)
	}
	if n := logs.Len(); n != 0 {
		t.Errorf("GetOnOff: expected 0 audit entries, got %d", n)
	}

	// UpdateOnOff must produce exactly one audit entry with the expected fields.
	_, err = client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: "x", OnOff: &traits.OnOff{State: traits.OnOff_ON}})
	if err != nil {
		t.Fatalf("UpdateOnOff: %v", err)
	}
	if n := logs.Len(); n != 1 {
		t.Errorf("UpdateOnOff: expected 1 audit entry, got %d", n)
	} else {
		entry := logs.All()[0]
		if entry.Message != "write" {
			t.Errorf("expected message %q, got %q", "write", entry.Message)
		}
		if v := entry.ContextMap()["outcome"]; v != "allowed" {
			t.Errorf("expected outcome %q, got %v", "allowed", v)
		}
		if v := entry.ContextMap()["method"]; v != "UpdateOnOff" {
			t.Errorf("expected method %q, got %v", "UpdateOnOff", v)
		}
	}
}

// allowAll is a Policy that permits every request.
type allowAll struct{}

func (allowAll) EvalPolicy(_ context.Context, _ string, _ Attributes) (rego.ResultSet, error) {
	return rego.ResultSet{{Expressions: []*rego.ExpressionValue{{Value: true}}}}, nil
}

var regoFiles = map[string]string{
	"smartcore.rego": `package smartcore

# This simple rule allows any request whose name is "allow", all other requests are denied
allow {
	input.request.name == "allow"
}
`,
	"smartcore.traits.OnOffApi.rego": `package smartcore.traits.OnOffApi

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
