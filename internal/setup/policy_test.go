package setup

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
)

// acceptingEnrollment stands in for the node's enrollment server; reaching it means the policy allowed the call.
type acceptingEnrollment struct {
	enrollmentpb.UnimplementedEnrollmentApiServer
}

func (acceptingEnrollment) CreateEnrollment(_ context.Context, req *enrollmentpb.CreateEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	return req.GetEnrollment(), nil
}

// TestBearerCode_defaultPolicy joins the validator to the default policy the way the controller does, and checks a hub
// presenting the code can enrol the node only while the node is waiting to be set up.
func TestBearerCode_defaultPolicy(t *testing.T) {
	f := newFixture(t)
	c := f.mustOpen(t)
	code := f.shownCode()

	validators := &token.ValidatorSet{}
	validators.Append(c)
	interceptor := policy.NewInterceptor(policy.Default(false), policy.WithTokenVerifier(validators))
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()))
	enrollmentpb.RegisterEnrollmentApiServer(server, acceptingEnrollment{})
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := enrollmentpb.NewEnrollmentApiClient(conn)

	enrol := func(bearer string) codes.Code {
		ctx := context.Background()
		if bearer != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+bearer)
		}
		_, err := client.CreateEnrollment(ctx, &enrollmentpb.CreateEnrollmentRequest{})
		return status.Code(err)
	}
	expect := func(name, bearer string, want codes.Code) {
		t.Helper()
		if got := enrol(bearer); got != want {
			t.Errorf("%s: CreateEnrollment = %v, want %v", name, got, want)
		}
	}

	expect("no code", "", codes.Unauthenticated)
	expect("wrong code", "2222-2222", codes.Unauthenticated)
	expect("right code", code, codes.OK)

	f.enrolled.Store(true)
	expect("enrolled", code, codes.Unauthenticated)
	f.enrolled.Store(false)
	expect("unenrolled again", code, codes.OK)

	f.addAccount(t, "admin")
	expect("claimed", code, codes.Unauthenticated)
}
