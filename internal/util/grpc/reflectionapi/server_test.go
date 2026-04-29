package reflectionapi

import (
	"context"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
)

// type names we know exist as part of the OnOff trait apis
var onOffTypes = []string{
	"smartcore.bos.onoff.v1.OnOffApi",
	"smartcore.bos.onoff.v1.OnOffInfo",
	"smartcore.bos.onoff.v1.GetOnOffRequest",
	"smartcore.bos.onoff.v1.DescribeOnOffRequest",
	"smartcore.bos.onoff.v1.OnOff",
	"smartcore.bos.onoff.v1.PullOnOffResponse.Change",
}

func TestServer(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)

	apiClient := nodeClient(t, func(s grpc.ServiceRegistrar) {
		onoffpb.RegisterOnOffApiServer(s, &onoffpb.UnimplementedOnOffApiServer{})
	})
	infoClient := nodeClient(t, func(s grpc.ServiceRegistrar) {
		onoffpb.RegisterOnOffInfoServer(s, &onoffpb.UnimplementedOnOffInfoServer{})
	})

	gwClient, rs := gwServer(t)
	t.Run("before register", func(t *testing.T) {
		testNoDynamicAPIs(t, ctx, gwClient)
	})

	// register the dynamic services and re-test
	rs.Add(apiClient)
	rs.Add(infoClient)
	t.Run("after register", func(t *testing.T) {
		testDynamicAPIs(t, ctx, gwClient)
	})

	// remove the dynamic services and re-test
	rs.Remove(apiClient)
	rs.Remove(infoClient)
	t.Run("after remove", func(t *testing.T) {
		testNoDynamicAPIs(t, ctx, gwClient)
	})
}

func testDynamicAPIs(t *testing.T, ctx context.Context, client *grpc.ClientConn) {
	reflectionClient := reflectionpb.NewServerReflectionClient(client)
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	reflectionStream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		t.Fatalf("reflectionClient.ServerReflectionInfo() = %v", err)
	}

	// check the new services exist
	services, err := ListServices(reflectionStream)
	if err != nil {
		t.Fatalf("ListServices(reflectionStream) = %v", err)
	}
	wantServices := []*reflectionpb.ServiceResponse{
		{Name: "grpc.reflection.v1.ServerReflection"},
		{Name: "grpc.reflection.v1alpha.ServerReflection"},
		{Name: "smartcore.bos.onoff.v1.OnOffApi"},
		{Name: "smartcore.bos.onoff.v1.OnOffInfo"},
	}
	if diff := cmp.Diff(services, wantServices, protocmp.Transform()); diff != "" {
		t.Errorf("ListServices(reflectionStream) mismatch (-want +got):\n%s", diff)
	}

	// check the new types are available
	for _, symbol := range onOffTypes {
		_, err = FileContainingSymbol(reflectionStream, symbol)
		if err != nil {
			t.Errorf("FileContainingSymbol(%s) = %v", symbol, err)
		}
	}
}

func testNoDynamicAPIs(t *testing.T, ctx context.Context, client *grpc.ClientConn) {
	// first check that we don't know about the dynamic services
	reflectionClient := reflectionpb.NewServerReflectionClient(client)
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	reflectionStream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		t.Fatalf("reflectionClient.ServerReflectionInfo() = %v", err)
	}
	services, err := ListServices(reflectionStream)
	if err != nil {
		t.Fatalf("ListServices(reflectionStream) = %v", err)
	}
	wantServices := []*reflectionpb.ServiceResponse{
		{Name: "grpc.reflection.v1.ServerReflection"},
		{Name: "grpc.reflection.v1alpha.ServerReflection"},
	}
	if diff := cmp.Diff(services, wantServices, protocmp.Transform()); diff != "" {
		t.Errorf("ListServices(reflectionStream) mismatch (-want +got):\n%s", diff)
	}

	// then check that types from the dynamic services are not available
	for _, symbol := range onOffTypes {
		_, err = FileContainingSymbol(reflectionStream, symbol)
		if status.Code(err) != codes.NotFound {
			t.Errorf("FileContainingSymbol(%s) = %v; want NotFound", symbol, err)
		}
	}

}

func nodeClient(t *testing.T, register func(grpc.ServiceRegistrar)) *grpc.ClientConn {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	reflection.Register(server)
	register(server)

	t.Cleanup(func() { server.Stop() })
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("server.Serve(nodeClient) = %v", err)
		}
	}()

	client, err := bufConn(lis)
	if err != nil {
		t.Fatalf("nodeClient failed to create client = %v", err)
	}
	return client
}

func gwServer(t *testing.T) (*grpc.ClientConn, *Server) {
	gwLis := bufconn.Listen(1024 * 1024)
	gwServer := grpc.NewServer()

	reflectionServer := NewServer(gwServer)
	reflectionServer.Register(gwServer)

	// Go generated types all register with protoregistry.GlobalFiles during init,
	// this causes issues for our tests because simply importing the traits package
	// causes all these types to be available to the reflection API.
	// One solution is to build separate binaries,
	// one that imports the traits package for the nodes,
	// and one that doesn't for the gateway.
	// This is a lot of work for a unit test, and is covered by e2e tests in the proxy package.
	// For now, we just remove the global files from the reflection server.
	reflectionServer.descResolver.Remove(protoregistry.GlobalFiles)

	t.Cleanup(func() { gwServer.Stop() })
	go func() {
		if err := gwServer.Serve(gwLis); err != nil {
			t.Errorf("gwServer.Serve(gwLis) = %v", err)
		}
	}()

	client, err := bufConn(gwLis)
	if err != nil {
		t.Fatalf("gwServer failed to create client = %v", err)
	}
	return client, reflectionServer
}

func bufConn(buf *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return buf.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
