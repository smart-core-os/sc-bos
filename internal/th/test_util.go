package th

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var Ctx = context.Background()
var StreamTimout = 500 * time.Millisecond

func CheckErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%v returned an error: %v", msg, err)
	}
}

func Dial(lis *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///test",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
