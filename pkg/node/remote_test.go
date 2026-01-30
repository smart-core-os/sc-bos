package node

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

func TestDialChan(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	t.Cleanup(done)
	address := make(chan string) // no buffer means we block sending until DialChan decides to receive

	remote := DialChan(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := remote.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect err %v", err)
	}

	client := healthpb.NewHealthClient(conn)

	// first run some tests before we've announced any addresses

	_, err = client.Check(ctx, &healthpb.HealthCheckRequest{Service: "test"})
	if err == nil {
		t.Fatalf("Expecting error when no address given")
	}

	l1, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Error listening %v", err)
	}

	addr1 := l1.Addr()

	grpcServer := grpc.NewServer()
	t.Cleanup(grpcServer.Stop)

	impl := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, impl)

	go func() {
		grpcServer.Serve(l1)
	}()
	err = awaitServing(ctx, addr1.String())
	if err != nil {
		t.Fatalf("Error waiting for server start %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	err = awaitSend(ctx2, address, addr1.String())
	if err != nil {
		t.Fatalf("Error waiting for send %v", err)
	}

	// wait a little, we know the address chan has been drained but we don't know the address has been passed on to the
	// grpc innards yet
	<-time.After(100 * time.Millisecond)

	impl.SetServingStatus("test", healthpb.HealthCheckResponse_SERVING)
	status, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "test"})
	if err != nil {
		t.Fatalf("Error from health check call %v", err)
	}
	if status.Status != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("Bad response from health check call %v", status)
	}
}

func awaitServing(ctx context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	return client.WaitForReady(ctx, conn)
}

func awaitSend[T any](ctx context.Context, ch chan<- T, v T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- v:
		return nil
	}
}
