package policy_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
)

type inspectingHub struct {
	hubpb.UnimplementedHubApiServer
}

func (inspectingHub) InspectHubNode(context.Context, *hubpb.InspectHubNodeRequest) (*hubpb.HubNodeInspection, error) {
	return &hubpb.HubNodeInspection{}, nil
}

// TestDefault_InspectHubNode_setupCode checks the default policy sees the setup code in a real request, as the
// interceptor encodes it, so an anonymous caller can't make the hub pass codes on to a node.
func TestDefault_InspectHubNode_setupCode(t *testing.T) {
	interceptor := policy.NewInterceptor(policy.Default(false))
	t.Cleanup(func() { _ = interceptor.Close() })
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()))
	hubpb.RegisterHubApiServer(server, inspectingHub{})
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
	client := hubpb.NewHubApiClient(conn)

	inspect := func(code string) codes.Code {
		_, err := client.InspectHubNode(context.Background(), &hubpb.InspectHubNodeRequest{
			Node:      &hubpb.HubNode{Address: "ac-01:23557"},
			SetupCode: code,
		})
		return status.Code(err)
	}
	if got := inspect(""); got != codes.OK {
		t.Errorf("anonymous InspectHubNode without a code = %v, want OK", got)
	}
	if got := inspect("ABCD-EFGH"); got != codes.Unauthenticated {
		t.Errorf("anonymous InspectHubNode with a code = %v, want Unauthenticated", got)
	}
}
