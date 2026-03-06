package wrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/smart-core-os/sc-golang/internal/testproto"
)

func TestWrapper_Unary(t *testing.T) {
	srv := &testServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, srv)
	client := testproto.NewTestApiClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("a", "avalue", "b", "bvalue"))

	t.Run("success", func(t *testing.T) {
		var header metadata.MD
		resp, err := client.Unary(ctx, &testproto.UnaryRequest{Msg: "hello"}, grpc.Header(&header))
		if err != nil {
			t.Fatalf("client.Unary(_, _) = _, %v; want _, nil", err)
		}
		if resp.Msg != "hello" {
			t.Errorf("resp.Msg = %q; want %q", resp.Msg, "hello")
		}
		expectMD := metadata.Pairs("a", "avalue", "b", "bvalue")
		if !maps.EqualFunc(header, expectMD, slices.Equal[[]string]) {
			t.Errorf("header = %v; want %v", header, expectMD)
		}
	})

	t.Run("error", func(t *testing.T) {
		_, err := client.Unary(ctx, &testproto.UnaryRequest{SimulateError: "foobar"})
		statusErr, ok := status.FromError(err)
		if !ok {
			t.Fatalf("client.Unary(...) did not return a status error - got %v", err)
		}
		if want := codes.Aborted; statusErr.Code() != want {
			t.Errorf("statusErr.Code() = %v; want %v", statusErr.Code(), want)
		}
		if want := "foobar"; statusErr.Message() != want {
			t.Errorf("statusErr.Message() = %q; want %q", statusErr.Message(), want)
		}
	})
}

// tests invoking a unary handler as a stream - this is needed to support proxies which handle all
// calls as streams
func TestWrapper_UnaryAsStream(t *testing.T) {
	src := &testServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, src)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("a", "avalue", "b", "bvalue"))

	method := fmt.Sprintf("/%s/Unary", testproto.TestApi_ServiceDesc.ServiceName)
	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    "Unary",
		ClientStreams: false,
		ServerStreams: false,
	}, method)
	if err != nil {
		t.Fatalf("conn.NewStream(_, _, _) = _, %v; want _, nil", err)
	}

	// send request
	err = stream.SendMsg(&testproto.UnaryRequest{Msg: "hello"})
	if err != nil {
		t.Fatalf("stream.SendMsg(_) = %v; want nil", err)
	}
	err = stream.CloseSend()
	if err != nil {
		t.Fatalf("stream.CloseSend() = %v; want nil", err)
	}
	// receive response
	res := &testproto.UnaryResponse{}
	err = stream.RecvMsg(res)
	if err != nil {
		t.Fatalf("stream.RecvMsg(_) = %v; want nil", err)
	}
	expect := &testproto.UnaryResponse{Msg: "hello"}
	if diff := cmp.Diff(expect, res, protocmp.Transform()); diff != "" {
		t.Errorf("stream.RecvMsg(_) mismatch (-want +got):\n%s", diff)
	}
	expectMD := metadata.Pairs("a", "avalue", "b", "bvalue")
	md, err := stream.Header()
	if err != nil {
		t.Errorf("stream.Header() = _, %v; want _, nil", err)
	}
	if diff := cmp.Diff(expectMD, md); diff != "" {
		t.Errorf("stream.Header() mismatch (-want +got):\n%s", diff)
	}

	// stream should be closed now, because unary calls send only one response
	err = stream.RecvMsg(res)
	if !errors.Is(err, io.EOF) {
		t.Errorf("stream not closed - expected EOF, got %v", err)
	}
}

// tests that the wrapper passes server-streaming calls through correctly, including metadata
func TestWrapper_ServerStream(t *testing.T) {
	srv := &testServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, srv)
	client := testproto.NewTestApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("success", func(t *testing.T) {
		// we expect the server to resend this metadata back to us
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("a", "avalue", "b", "bvalue"))
		stream, err := client.ServerStream(ctx, &testproto.ServerStreamRequest{NumRes: 3})
		if err != nil {
			t.Fatalf("client.ServerStream(_, _) = _, %v; want _, nil", err)
		}

		var received int32
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Fatalf("stream.Recv() = _, %v; want _, nil", err)
			}
			if resp.Counter != received {
				t.Errorf("resp.Counter = %v; want %v", resp.Counter, received)
			}
			received++
		}
		if received != 3 {
			t.Errorf("received = %v; want 3", received)
		}
		md, err := stream.Header()
		if err != nil {
			t.Errorf("stream.Header() = _, %v; want _, nil", err)
		}
		expectMD := metadata.Pairs("a", "avalue", "b", "bvalue")
		if diff := cmp.Diff(expectMD, md); diff != "" {
			t.Errorf("stream.Header() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("error", func(t *testing.T) {
		stream, err := client.ServerStream(ctx, &testproto.ServerStreamRequest{SimulateError: "foobar"})
		if err != nil {
			t.Fatalf("client.ServerStream(_, _) = _, %v; want _, nil", err)
		}
		_, err = stream.Recv()
		statusErr, ok := status.FromError(err)
		if !ok {
			t.Fatalf("stream.Recv() did not return a status error - got %v", err)
		}
		if want := codes.Aborted; statusErr.Code() != want {
			t.Errorf("statusErr.Code() = %v; want %v", statusErr.Code(), want)
		}
		if want := "foobar"; statusErr.Message() != want {
			t.Errorf("statusErr.Message() = %q; want %q", statusErr.Message(), want)
		}
	})
}

func TestWrapper_ClientStream(t *testing.T) {
	srv := &testServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, srv)
	client := testproto.NewTestApiClient(conn)

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("hello", "world"))
	t.Run("success", func(t *testing.T) {
		stream, err := client.ClientStream(ctx)
		if err != nil {
			t.Fatalf("client.ClientStream(_) = _, %v; want _, nil", err)
		}
		for _, msg := range []string{"a", "b", "c"} {
			err = stream.Send(&testproto.ClientStreamRequest{Msg: msg})
			if err != nil {
				t.Fatalf("stream.Send(%q) = %v; want nil", msg, err)
			}
		}
		res, err := stream.CloseAndRecv()
		if err != nil {
			t.Fatalf("stream.CloseAndRecv() = _, %v; want _, nil", err)
		}
		if want := "abc"; res.Msg != want {
			t.Errorf("res.Msg = %q; want %q", res.Msg, want)
		}
		md, err := stream.Header()
		if err != nil {
			t.Errorf("stream.Header() = _, %v; want _, nil", err)
		}
		expectMD := metadata.Pairs("hello", "world")
		if diff := cmp.Diff(expectMD, md); diff != "" {
			t.Errorf("stream.Header() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("error", func(t *testing.T) {
		stream, err := client.ClientStream(ctx)
		if err != nil {
			t.Fatalf("client.ClientStream(_) = _, %v; want _, nil", err)
		}
		err = stream.Send(&testproto.ClientStreamRequest{SimulateError: "foobar"})
		if err != nil {
			t.Fatalf("stream.Send(_) = %v; want nil", err)
		}
		_, err = stream.CloseAndRecv()
		statusErr, ok := status.FromError(err)
		if !ok {
			t.Fatalf("stream.CloseAndRecv() did not return a status error - got %v", err)
		}
		if want := codes.Aborted; statusErr.Code() != want {
			t.Errorf("statusErr.Code() = %v; want %v", statusErr.Code(), want)
		}
		if want := "foobar"; statusErr.Message() != want {
			t.Errorf("statusErr.Message() = %q; want %q", statusErr.Message(), want)
		}
	})
}

func TestWrapper_BidiStream(t *testing.T) {
	srv := &testServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, srv)
	client := testproto.NewTestApiClient(conn)

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("hello", "world"))

	t.Run("success", func(t *testing.T) {
		stream, err := client.BidiStream(ctx)
		if err != nil {
			t.Fatalf("client.BidiStream(_) = _, %v; want _, nil", err)
		}

		// each message should be echoed back to us
		// after the first message, we can check the metadata
		var metadataVerified bool
		for _, msg := range []string{"a", "b", "c"} {
			err = stream.Send(&testproto.BidiStreamRequest{Msg: msg})
			if err != nil {
				t.Fatalf("stream.Send(%q) = %v; want nil", msg, err)
			}

			resp, err := stream.Recv()
			if err != nil {
				t.Fatalf("stream.Recv() = _, %v; want _, nil", err)
			}
			if resp.Msg != msg {
				t.Errorf("resp.Msg = %q; want %q", resp.Msg, msg)
			}

			if !metadataVerified {
				md, err := stream.Header()
				if err != nil {
					t.Errorf("stream.Header() = _, %v; want _, nil", err)
				}
				expectMD := metadata.Pairs("hello", "world")
				if diff := cmp.Diff(expectMD, md); diff != "" {
					t.Errorf("stream.Header() mismatch (-want +got):\n%s", diff)
				}
				metadataVerified = true
			}
		}
		// signal to the server that we are done
		err = stream.CloseSend()
		if err != nil {
			t.Errorf("stream.CloseSend() = %v; want nil", err)
		}
		// drain the stream
		var extra int
		for {
			_, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Errorf("stream.Recv() = _, %v; want _, nil", err)
			}
			extra++
		}
		if extra != 0 {
			t.Errorf("extra messages received: %d", extra)
		}
	})

	t.Run("error", func(t *testing.T) {
		stream, err := client.BidiStream(ctx)
		if err != nil {
			t.Fatalf("client.BidiStream(_) = _, %v; want _, nil", err)
		}

		err = stream.Send(&testproto.BidiStreamRequest{SimulateError: "foobar"})
		if err != nil {
			t.Fatalf("stream.Send(_) = %v; want nil", err)
		}
		_, err = stream.Recv()
		statusErr, ok := status.FromError(err)
		if !ok {
			t.Fatalf("stream.Recv() did not return a status error - got %v", err)
		}
		if want := codes.Aborted; statusErr.Code() != want {
			t.Errorf("statusErr.Code() = %v; want %v", statusErr.Code(), want)
		}
		if want := "foobar"; statusErr.Message() != want {
			t.Errorf("statusErr.Message() = %q; want %q", statusErr.Message(), want)
		}
	})
}

type testServer struct {
	testproto.UnimplementedTestApiServer
}

func (s *testServer) Unary(ctx context.Context, req *testproto.UnaryRequest) (*testproto.UnaryResponse, error) {
	if err := copyMDUnary(ctx); err != nil {
		return nil, err
	}

	if req.SimulateError != "" {
		return nil, status.Error(codes.Aborted, req.SimulateError)
	}

	// echo request to response
	return &testproto.UnaryResponse{
		Msg: req.Msg,
	}, nil
}

func (s *testServer) ServerStream(req *testproto.ServerStreamRequest, srv testproto.TestApi_ServerStreamServer) error {
	if err := copyMDStream(srv.Context(), srv); err != nil {
		return err
	}

	if req.SimulateError != "" {
		return status.Error(codes.Aborted, req.SimulateError)
	}

	// client requests a certain number of responses, we send them, counting up
	for i := int32(0); i < req.NumRes; i++ {
		err := srv.Send(&testproto.ServerStreamResponse{
			Counter: i,
		})
		if err != nil {
			log.Printf("ServerStream server Send error: %v", err)
			return err
		}
	}
	return nil
}

func (s *testServer) ClientStream(srv testproto.TestApi_ClientStreamServer) error {
	if err := copyMDStream(srv.Context(), srv); err != nil {
		return err
	}

	// concatenate all client requests into a single response
	var msg strings.Builder
	for {
		req, err := srv.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			log.Printf("ClientStream server Recv error: %v", err)
			return err
		}
		if req.SimulateError != "" {
			return status.Error(codes.Aborted, req.SimulateError)
		}
		msg.WriteString(req.Msg)
	}

	err := srv.SendAndClose(&testproto.ClientStreamResponse{
		Msg: msg.String(),
	})
	if err != nil {
		log.Printf("ClientStream server SendAndClose error: %v", err)
	}
	return err
}

func (s *testServer) BidiStream(srv testproto.TestApi_BidiStreamServer) error {
	if err := copyMDStream(srv.Context(), srv); err != nil {
		return err
	}

	// echo all client requests back to the client
	for {
		req, err := srv.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			log.Printf("BidiStream server Recv error: %v", err)
			return err
		}
		if req.SimulateError != "" {
			return status.Error(codes.Aborted, req.SimulateError)
		}
		err = srv.Send(&testproto.BidiStreamResponse{
			Msg: req.Msg,
		})
		if err != nil {
			log.Printf("BidiStream server Send error: %v", err)
			return err
		}
	}
	return nil
}

func copyMDStream(ctx context.Context, stream grpc.ServerStream) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	log.Printf("copy metadata to response: %v", md)
	return stream.SendHeader(md)
}

func copyMDUnary(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	log.Printf("copy metadata to response: %v", md)
	return grpc.SetHeader(ctx, md)
}
