package policy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/auth/token"
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

func TestIsAuditExcluded(t *testing.T) {
	tests := []struct {
		name    string
		service string
		method  string
		want    bool
	}{
		// noisy RPCs that must be excluded from the audit log
		{"reflection v1", "grpc.reflection.v1.ServerReflection", "ServerReflectionInfo", true},
		{"reflection v1alpha", "grpc.reflection.v1alpha.ServerReflection", "ServerReflectionInfo", true},
		{"hub test", "smartcore.bos.hub.v1.HubApi", "TestHubNode", true},
		{"history create", "smartcore.bos.history.v1.HistoryAdminApi", "CreateHistoryRecord", true},
		// legitimately-audited writes that must NOT be excluded
		{"enrollment test", "smartcore.bos.enrollment.v1.EnrollmentApi", "TestEnrollment", false},
		{"function test", "smartcore.bos.emergency.v1.EmergencyApi", "StartFunctionTest", false},
		{"enroll hub node", "smartcore.bos.hub.v1.HubApi", "EnrollHubNode", false},
		{"access grant", "smartcore.bos.tenants.v1.TenantApi", "CreateAccessGrant", false},
		// same method name on a different service must NOT be excluded
		{"other TestHubNode", "example.other.v1.OtherApi", "TestHubNode", false},
		{"other CreateHistoryRecord", "example.other.v1.OtherApi", "CreateHistoryRecord", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuditExcluded(tt.service, tt.method); got != tt.want {
				t.Errorf("isAuditExcluded(%q, %q) = %v, want %v", tt.service, tt.method, got, tt.want)
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
allow if input.request.name == "allow"
`,
	"smartcore.bos.onoff.v1.OnOffApi.rego": `package smartcore.bos.onoff.v1.OnOffApi

# This rule allows people to turn any device on (but not off)
allow if {
	input.method == "UpdateOnOff"
	input.request.onOff.state == "ON"
}
`,
	"http.rego": `package http

allow if input.method == "GET"
allow if input.path == "/foo"
`,
}

// A write from a non-gRPC ingress is recorded with the ingress detail and no invented principal.
func TestInterceptor_AuditIngress(t *testing.T) {
	sink := &captureSink{}
	interceptor := NewInterceptor(AllowAll, WithAuditSink(sink))

	interceptor.AuditIngress(IngressEntry{
		Ingress: "mqtt",
		Peer:    "tcp://broker:1883",
		Service: "smartcore.bos.udmi.v1.UdmiService",
		Method:  "OnMessage",
		Fields: map[string]string{
			"topic":  "site/dev1/config",
			"target": "dev1",
			// must not be able to override the standard fields
			"outcome": "allowed",
			"subject": "someone",
			"service": "spoofed.Service",
			"ingress": "grpc",
		},
	})
	interceptor.Close() // drain the async audit queue before asserting

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(msgs))
	}
	msg := msgs[0]
	if msg.Level != logpb.Level_LEVEL_INFO {
		t.Errorf("level = %v, want INFO", msg.Level)
	}
	for key, want := range map[string]string{
		"service": "smartcore.bos.udmi.v1.UdmiService",
		"method":  "OnMessage",
		"ingress": "mqtt",
		"peer":    "tcp://broker:1883",
		"outcome": "ok",
		"topic":   "site/dev1/config",
		"target":  "dev1",
		// no principal is knowable for a write from another ingress
		"subject": "",
		"name":    "",
		"cert":    "false",
		"token":   "false",
	} {
		if got := msg.Fields[key]; got != want {
			t.Errorf("field %q = %q, want %q", key, got, want)
		}
	}
}

// A failed ingress write is still recorded, and at WARN so it stands out.
func TestInterceptor_AuditIngress_Failed(t *testing.T) {
	sink := &captureSink{}
	interceptor := NewInterceptor(AllowAll, WithAuditSink(sink))

	interceptor.AuditIngress(IngressEntry{
		Ingress: "mqtt",
		Service: "smartcore.bos.udmi.v1.UdmiService",
		Method:  "OnMessage",
		Err:     status.Error(codes.Internal, "boom"),
	})
	interceptor.Close()

	msgs := sink.all()
	if len(msgs) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(msgs))
	}
	if got := msgs[0].Fields["outcome"]; got != "failed" {
		t.Errorf("outcome = %q, want %q", got, "failed")
	}
	if msgs[0].Level != logpb.Level_LEVEL_WARN {
		t.Errorf("level = %v, want WARN", msgs[0].Level)
	}
}

// Without an audit sink, or without an interceptor at all, AuditIngress does nothing.
func TestInterceptor_AuditIngress_NoSink(t *testing.T) {
	e := IngressEntry{Ingress: "mqtt", Service: "s", Method: "m"}
	interceptor := NewInterceptor(AllowAll)
	interceptor.AuditIngress(e)
	interceptor.Close()

	var nilInterceptor *Interceptor
	nilInterceptor.AuditIngress(e)
}

// Writes can race Close during shutdown, e.g. MQTT handlers still running after the gRPC server
// has stopped. They must be dropped rather than sent on the closed audit queue.
func TestInterceptor_AuditAfterClose(t *testing.T) {
	for range 20 {
		sink := &captureSink{}
		interceptor := NewInterceptor(AllowAll, WithAuditSink(sink))
		e := IngressEntry{Ingress: "mqtt", Service: "s", Method: "m"}

		stop := make(chan struct{})
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for {
					select {
					case <-stop:
						return
					default:
						interceptor.AuditIngress(e)
					}
				}
			})
		}
		time.Sleep(time.Millisecond) // let the writers get going before closing
		interceptor.Close()
		interceptor.AuditIngress(e) // after Close, from this goroutine too
		close(stop)
		wg.Wait()
	}
}

// A proxied call from another node carries a token this node can't verify, but the caller is
// authenticated by its client certificate. That is expected, so it must not be logged as an error.
func TestInterceptor_GRPC_BadTokenWithValidCert(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	interceptor := NewInterceptor(AllowAll,
		WithLogger(zap.New(core)),
		WithTokenVerifier(token.NeverValid(jose.ErrCryptoFailure)),
	)
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "gateway"}}
	ctx := badTokenContext(credentials.TLSInfo{State: tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
		VerifiedChains:   [][]*x509.Certificate{{cert}},
	}})

	if _, err := interceptor.checkPolicyGrpc(ctx, nil, nil, StreamAttributes{}); err != nil {
		t.Fatalf("checkPolicyGrpc: %v", err)
	}
	if n := logs.FilterLevelExact(zap.DebugLevel).FilterMessageSnippet("token failed verification").Len(); n != 1 {
		t.Errorf("got %d debug token failure entries, want 1", n)
	}
	for _, e := range logs.All() {
		if e.Level >= zap.WarnLevel {
			t.Errorf("unexpected %v entry: %q %v", e.Level, e.Message, e.ContextMap())
		}
	}
	entries := logs.FilterMessageSnippet("token failed verification").All()
	if len(entries) == 1 {
		if got := entries[0].ContextMap()["certSubject"]; got != "CN=gateway" {
			t.Errorf("certSubject = %v, want %q", got, "CN=gateway")
		}
	}
}

// A bad token from a caller with no client certificate is a client problem, so it is a warning.
func TestInterceptor_GRPC_BadTokenWithoutCert(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	interceptor := NewInterceptor(AllowAll,
		WithLogger(zap.New(core)),
		WithTokenVerifier(token.NeverValid(jose.ErrCryptoFailure)),
	)
	ctx := badTokenContext(nil)

	if _, err := interceptor.checkPolicyGrpc(ctx, nil, nil, StreamAttributes{}); err != nil {
		t.Fatalf("checkPolicyGrpc: %v", err)
	}
	if n := logs.FilterLevelExact(zap.ErrorLevel).Len(); n != 0 {
		t.Errorf("got %d error entries, want 0", n)
	}
	warns := logs.FilterLevelExact(zap.WarnLevel).FilterMessageSnippet("token failed verification").All()
	if len(warns) != 1 {
		t.Fatalf("got %d warn token failure entries, want 1", len(warns))
	}
	fields := warns[0].ContextMap()
	for key, want := range map[string]string{
		"peer":    "192.0.2.1:1234",
		"service": "smartcore.bos.onoff.v1.OnOffApi",
		"method":  "GetOnOff",
	} {
		if got := fields[key]; got != want {
			t.Errorf("field %q = %v, want %q", key, got, want)
		}
	}
}

// An HTTP request with a bad token is rejected as unauthenticated, not as a server error.
func TestInterceptor_HTTP_BadToken(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	interceptor := NewInterceptor(AllowAll,
		WithLogger(zap.New(core)),
		WithTokenVerifier(token.NeverValid(jose.ErrCryptoFailure)),
	)
	server := httptest.NewTLSServer(interceptor.HTTPInterceptor(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler called for a request with a bad token")
	})))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer x")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if n := logs.FilterLevelExact(zap.ErrorLevel).Len(); n != 0 {
		t.Errorf("got %d error entries, want 0", n)
	}
	warns := logs.FilterLevelExact(zap.WarnLevel).FilterMessageSnippet("token failed verification").All()
	if len(warns) != 1 {
		t.Fatalf("got %d warn token failure entries, want 1", len(warns))
	}
	if got := warns[0].ContextMap()["path"]; got != "/foo" {
		t.Errorf("path = %v, want %q", got, "/foo")
	}
}

// badTokenContext returns a server-side context for a GetOnOff call carrying a bearer token,
// from a peer with the given auth info.
func badTokenContext(authInfo credentials.AuthInfo) context.Context {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(),
		methodStream{method: "/smartcore.bos.onoff.v1.OnOffApi/GetOnOff"})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr:     &net.TCPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 1234},
		AuthInfo: authInfo,
	})
	return metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer x"))
}

// methodStream is a grpc.ServerTransportStream that only reports its method.
type methodStream struct {
	grpc.ServerTransportStream
	method string
}

func (s methodStream) Method() string { return s.method }
