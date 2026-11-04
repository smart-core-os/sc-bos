// Package testtarget runs a node for a hub to enrol in tests.
// The node serves the enrollment and metadata APIs over TLS. With policy on it also checks requests against the
// default policy, accepting its setup code as the controller does.
package testtarget

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/internal/setup"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

// Code is the setup code of every Target with policy on.
const Code = "ABCD-EFGH"

// Name is the name a Target reports in its metadata.
const Name = "test-target"

// Call records a request a Target received.
type Call struct {
	Method string // full gRPC method name
	Bearer bool   // whether the request carried an authorization header
}

// Target is a node waiting to be enrolled.
type Target struct {
	Addr string

	enrollment atomic.Pointer[enrollmentpb.Enrollment]

	mu    sync.Mutex
	calls []Call
}

// New starts a Target that stops when t ends.
// With policyOn the target refuses requests the default policy denies, and has nobody claimed it, so it accepts Code.
func New(t testing.TB, policyOn bool) *Target {
	t.Helper()
	tgt := &Target{}

	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatal(err)
	}
	cert, roots, err := pki.SelfSignedSource(key).Certs()
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig := pki.TLSServerConfig(pki.DirectSource(cert, roots))

	interceptors := []grpc.UnaryServerInterceptor{tgt.record}
	if policyOn {
		interceptors = append(interceptors, newPolicy(t, tgt).GRPCUnaryInterceptor())
	}
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	enrollmentpb.RegisterEnrollmentApiServer(server, &enrollmentServer{tgt: tgt})
	metadatapb.RegisterMetadataApiServer(server, metadataServer{})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)
	tgt.Addr = lis.Addr().String()
	return tgt
}

// newPolicy returns the default policy with tgt's setup codes as the only way to present a token.
func newPolicy(t testing.TB, tgt *Target) *policy.Interceptor {
	t.Helper()
	dir := t.TempDir()
	codeFile := filepath.Join(dir, "code.txt")
	if err := os.WriteFile(codeFile, []byte(Code+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewMemoryStore(zap.NewNop())
	t.Cleanup(func() { _ = accounts.Close() })
	setupCodes, err := setup.Open(context.Background(), setup.Config{
		Dir:      filepath.Join(dir, "setup"),
		CodeFile: codeFile,
		Accounts: accounts,
		Enrolled: func() bool { return tgt.Enrollment() != nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	validators := &token.ValidatorSet{}
	validators.Append(setupCodes)
	interceptor := policy.NewInterceptor(policy.Default(false), policy.WithTokenVerifier(validators))
	t.Cleanup(func() { _ = interceptor.Close() })
	return interceptor
}

// Enrollment returns the enrollment the target accepted, or nil if it is not enrolled.
func (t *Target) Enrollment() *enrollmentpb.Enrollment {
	return t.enrollment.Load()
}

// Calls returns the requests the target has received, in order, whether or not policy allowed them.
func (t *Target) Calls() []Call {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Call(nil), t.calls...)
}

func (t *Target) record(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	t.mu.Lock()
	t.calls = append(t.calls, Call{Method: info.FullMethod, Bearer: len(md.Get("authorization")) > 0})
	t.mu.Unlock()
	return handler(ctx, req)
}

// Authority returns a CA a hub can sign enrollments with.
func Authority(t testing.TB) pki.Source {
	t.Helper()
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatal(err)
	}
	cert, roots, err := pki.SelfSignedSourceT(key, &x509.Certificate{
		Subject:               pkix.Name{CommonName: "test-hub-ca"},
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}, pki.WithExpireAfter(time.Hour)).Certs()
	if err != nil {
		t.Fatal(err)
	}
	return pki.DirectSource(cert, roots)
}

// enrollmentServer behaves like the controller's: one enrollment, and NotFound until there is one.
type enrollmentServer struct {
	enrollmentpb.UnimplementedEnrollmentApiServer
	tgt *Target
}

func (s *enrollmentServer) GetEnrollment(context.Context, *enrollmentpb.GetEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	if en := s.tgt.Enrollment(); en != nil {
		return en, nil
	}
	return nil, status.Error(codes.NotFound, "not enrolled")
}

func (s *enrollmentServer) CreateEnrollment(_ context.Context, req *enrollmentpb.CreateEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	if !s.tgt.enrollment.CompareAndSwap(nil, req.GetEnrollment()) {
		return nil, status.Error(codes.AlreadyExists, "already enrolled")
	}
	return req.GetEnrollment(), nil
}

type metadataServer struct {
	metadatapb.UnimplementedMetadataApiServer
}

func (metadataServer) GetMetadata(context.Context, *metadatapb.GetMetadataRequest) (*metadatapb.Metadata, error) {
	return &metadatapb.Metadata{Name: Name}, nil
}
