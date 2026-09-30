package policy

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesscredentialpb"
)

func TestRedacted(t *testing.T) {
	req := &accesscredentialpb.CreateCredentialRequest{
		Name:       "paxton/cardholder/24",
		Credential: &accesscredentialpb.Credential{Type: "ProxCard", Value: "12345678"},
	}
	orig := proto.Clone(req)

	got := redacted(req)
	want := &accesscredentialpb.CreateCredentialRequest{
		Name:       "paxton/cardholder/24",
		Credential: &accesscredentialpb.Credential{Type: "ProxCard", Value: redactedString},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("redacted (-want,+got)\n%s", diff)
	}
	if diff := cmp.Diff(orig, req, protocmp.Transform()); diff != "" {
		t.Errorf("the original was modified (-want,+got)\n%s", diff)
	}

	// values in repeated fields are found too
	list := &accesscredentialpb.ListCredentialsResponse{Credentials: []*accesscredentialpb.Credential{{Value: "1"}, {Value: "2"}}}
	for _, c := range redacted(list).(*accesscredentialpb.ListCredentialsResponse).GetCredentials() {
		if c.GetValue() != redactedString {
			t.Errorf("want value redacted, got %q", c.GetValue())
		}
	}
}

// A request the policy denies is logged at debug, which the log capture keeps, so it mustn't carry credential values.
func TestInterceptor_deniedRequestLogIsRedacted(t *testing.T) {
	const value = "12345678"
	core, logs := observer.New(zapcore.DebugLevel)
	interceptor := NewInterceptor(Default(false), WithLogger(zap.New(core)))

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()))
	accesscredentialpb.NewModelServer(accesscredentialpb.NewModel(nil)).Register(server)
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

	_, err = accesscredentialpb.NewAccessCredentialApiClient(conn).CreateCredential(context.Background(), &accesscredentialpb.CreateCredentialRequest{
		Name:       "paxton/cardholder/24",
		Credential: &accesscredentialpb.Credential{Type: "card", Value: value},
	})
	if err == nil {
		t.Fatal("want the request denied, it has no credentials")
	}

	blocked := logs.FilterMessage("request blocked by policy").All()
	if len(blocked) != 1 {
		t.Fatalf("want 1 blocked request logged, got %d", len(blocked))
	}
	logged, err := json.Marshal(blocked[0].ContextMap())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logged), value) {
		t.Errorf("log contains the credential value: %s", logged)
	}
	if !strings.Contains(string(logged), "paxton/cardholder/24") {
		t.Errorf("log should still contain the rest of the request: %s", logged)
	}
}
