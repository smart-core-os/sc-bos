package router

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/trait/onoffpb"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/wrap"
)

func TestStreamHandler(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	// downstream nodes
	n1Client, err := newNode(t, "n1")
	if err != nil {
		t.Fatalf("newNode(n1) = %v", err)
	}
	n2Client, err := newNode(t, "n2")
	if err != nil {
		t.Fatalf("newNode(n2) = %v", err)
	}

	reg := New()
	registerService(reg, traits.OnOffApi_ServiceDesc.ServiceName, n1Client, n2Client)

	proxyServer := grpc.NewServer(grpc.UnknownServiceHandler(StreamHandler(reg)))
	proxyLis := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() { proxyServer.Stop() })
	go func() {
		if err := proxyServer.Serve(proxyLis); err != nil {
			t.Errorf("proxyServer.Serve(proxyLis) = %v", err)
		}
	}()

	proxyConn, err := bufConn(proxyLis)
	if err != nil {
		t.Fatalf("bufConn(proxyLis) = %v", err)
	}
	t.Cleanup(func() { proxyConn.Close() })

	onOffClient := traits.NewOnOffApiClient(proxyConn)
	t.Run("downstream", func(t *testing.T) {
		testDownstream(t, ctx, onOffClient, "n1")
		testDownstream(t, ctx, onOffClient, "n2")
	})

	t.Run("unknown key", func(t *testing.T) {
		// known api, unknown key
		_, err = onOffClient.GetOnOff(ctx, &traits.GetOnOffRequest{Name: "missing"})
		if code := status.Code(err); code != codes.NotFound {
			t.Fatalf("onOffClient.GetOnOff(missing) want NotFound, got: %v", err)
		}
	})

	t.Run("unknown api", func(t *testing.T) {
		// unknown api
		client2 := traits.NewOnOffInfoClient(proxyConn)
		_, err = client2.DescribeOnOff(ctx, &traits.DescribeOnOffRequest{Name: "n1"})
		if code := status.Code(err); code != codes.Unimplemented {
			t.Fatalf("client2.DescribeOnOff(n1) want Unimplemented, got: %v", err)
		}
	})
}

// tests that grpc server interceptors work properly with the stream handler
func TestStreamHandler_Interceptors(t *testing.T) {
	deviceName := "foobar"
	model := onoffpb.NewModel(onoffpb.WithInitialOnOff(&traits.OnOff{State: traits.OnOff_OFF}))
	modelServer := onoffpb.NewModelServer(model)
	modelServerConn := wrap.ServerToClient(traits.OnOffApi_ServiceDesc, modelServer)
	srvDesc := serviceDescriptor(traits.OnOffApi_ServiceDesc.ServiceName)

	// a StreamHandler that always directs to modelServerConn
	handler := StreamHandler(methodResolverFunc(func(fullName string) (Method, error) {
		_, methodName, ok := parseMethod(fullName)
		if !ok {
			return Method{}, ErrMissingMethod
		}
		method := srvDesc.Methods().ByName(protoreflect.Name(methodName))
		if method == nil {
			return Method{}, ErrUnknownService
		}

		return Method{
			Desc: method,
			Resolver: ConnResolverFunc(func(mr MsgRecver) (grpc.ClientConnInterface, error) {
				return modelServerConn, nil
			}),
		}, nil
	}))

	// interceptors that expect the requests to have descriptors matching the right types
	// also check that we can extract the device name from the request
	// (because all request go to the UnknownServiceHandler, they are all treated as streams so we don't need a
	// unary interceptor)
	var expectDesc protoreflect.Descriptor
	streamInterceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		cb := func(msg any) error {
			reflectReq := msg.(proto.Message).ProtoReflect()
			reqDesc := reflectReq.Descriptor()
			if reqDesc != expectDesc {
				t.Errorf("unaryInterceptor: got message with desc %s, want %s", reqDesc.FullName(), expectDesc.FullName())
			}
			if name, ok := extractStringField(reflectReq, "name"); !ok || name != deviceName {
				t.Errorf("unaryInterceptor: got name %q, want %q", name, deviceName)
			}
			return nil
		}
		return handler(srv, &streamRequestInterceptor{ServerStream: ss, cb: cb})
	}

	server := grpc.NewServer(
		grpc.UnknownServiceHandler(handler),
		grpc.StreamInterceptor(streamInterceptor),
	)
	defer server.Stop()
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("server.Serve() = %v", err)
		}
	}()
	conn, err := bufConn(lis)
	if err != nil {
		t.Fatalf("bufConn error %v", err)
	}
	client := traits.NewOnOffApiClient(conn)

	expectDesc = (&traits.GetOnOffRequest{}).ProtoReflect().Descriptor()
	_, err = client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: deviceName})
	if err != nil {
		t.Errorf("client.GetOnOff() = %v", err)
	}

	expectDesc = (&traits.PullOnOffRequest{}).ProtoReflect().Descriptor()
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.PullOnOff(ctx, &traits.PullOnOffRequest{Name: deviceName})
	if err != nil {
		t.Errorf("client.PullOnOff() = %v", err)
	}
	_, err = stream.Recv()
	if err != nil {
		t.Errorf("stream.Recv() = %v", err)
	}
	cancel()
}

type methodResolverFunc func(fullName string) (Method, error)

func (f methodResolverFunc) ResolveMethod(fullName string) (Method, error) {
	return f(fullName)
}

type streamRequestInterceptor struct {
	grpc.ServerStream
	cb func(msg any) error
}

func (s *streamRequestInterceptor) RecvMsg(m any) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	return s.cb(m)
}

func testDownstream(t *testing.T, ctx context.Context, client traits.OnOffApiClient, name string) {
	ctx, stop := context.WithTimeout(ctx, time.Second)
	defer stop() // also cancels the stream

	stream, err := client.PullOnOff(ctx, &traits.PullOnOffRequest{Name: name})
	if err != nil {
		t.Fatalf("client.PullOnOff(%s) = %v", name, err)
	}
	assertNextEvent := func(s *traits.OnOff) {
		event, err := stream.Recv()
		if err != nil {
			t.Fatalf("stream.Recv(%s) = %v", name, err)
		}
		// clear timestamps to make comparison easier
		for i := range event.Changes {
			event.Changes[i].ChangeTime = nil
		}
		wantEvent := &traits.PullOnOffResponse{Changes: []*traits.PullOnOffResponse_Change{
			{Name: name, OnOff: s},
		}}
		if diff := cmp.Diff(event, wantEvent, protocmp.Transform()); diff != "" {
			t.Fatalf("stream.Recv(%s) mismatch (-want +got):\n%s", name, diff)
		}
	}

	// ensure the stream is open
	// note: we must Recv on the stream before calling update, otherwise we race with the server invocation.
	assertNextEvent(&traits.OnOff{State: traits.OnOff_OFF})
	// and matches initial state
	res, err := client.GetOnOff(ctx, &traits.GetOnOffRequest{Name: name})
	if err != nil {
		t.Fatalf("client.GetOnOff(%s) = %v", name, err)
	}
	if diff := cmp.Diff(res, &traits.OnOff{State: traits.OnOff_OFF}, protocmp.Transform()); diff != "" {
		t.Fatalf("client.GetOnOff(%s) mismatch (-want +got):\n%s", name, diff)
	}

	// update state
	res, err = client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: name, OnOff: &traits.OnOff{State: traits.OnOff_ON}})
	if err != nil {
		t.Fatalf("client.UpdateOnOff(%s) = %v", name, err)
	}
	if diff := cmp.Diff(res, &traits.OnOff{State: traits.OnOff_ON}, protocmp.Transform()); diff != "" {
		t.Fatalf("client.UpdateOnOff(%s) mismatch (-want +got):\n%s", name, diff)
	}

	// check stream emitted the update
	assertNextEvent(&traits.OnOff{State: traits.OnOff_ON})
}

func registerService(reg *Router, service string, clients ...*grpc.ClientConn) {
	supportService(reg, service)
	for i, client := range clients {
		name := fmt.Sprintf("n%d", i+1)
		err := reg.AddRoute(service, name, client)
		if err != nil {
			panic(err)
		}
	}
}

func supportService(reg *Router, service string) {
	if reg.GetService(service) != nil {
		return
	}
	desc := serviceDescriptor(service)
	s, err := NewRoutedService(desc, "name")
	if err != nil {
		// register as unrouted instead
		s = NewUnroutedService(desc)
	}
	_ = reg.AddService(s)
}

func newNode(t *testing.T, name string) (*grpc.ClientConn, error) {
	lis := bufconn.Listen(1024 * 1024)
	s := nodeServer(name)
	c, err := bufConn(lis)

	t.Cleanup(func() { c.Close() })
	t.Cleanup(func() { s.Stop() })

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("%s.Serve() = %v", name, err)
		}
	}()
	return c, err
}

// nodeServer returns a *grpc.Server that implements the OnOffApi service that responds to the given name.
func nodeServer(name string) *grpc.Server {
	r := New() // use a router to force NOT_FOUND for unknown names
	supportService(r, traits.OnOffApi_ServiceDesc.ServiceName)
	m := onoffpb.NewModel(onoffpb.WithInitialOnOff(&traits.OnOff{State: traits.OnOff_OFF}))
	err := r.AddRoute("", name, wrap.ServerToClient(traits.OnOffApi_ServiceDesc, onoffpb.NewModelServer(m)))
	if err != nil {
		panic(err)
	}
	return grpc.NewServer(grpc.UnknownServiceHandler(StreamHandler(r)))
}

func bufConn(buf *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.NewClient("localhost:0",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return buf.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func serviceDescriptor(name string) protoreflect.ServiceDescriptor {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		panic(err)
	}
	sd, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		panic("not a service descriptor")
	}
	return sd
}

func extractStringField(msg protoreflect.Message, field string) (string, bool) {
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return "", false
	}
	return msg.Get(fd).String(), true
}
