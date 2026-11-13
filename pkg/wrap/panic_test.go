package wrap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/testproto"
)

// panicServer panics when asked to, via the simulate_error field, and otherwise behaves as testServer.
type panicServer struct {
	testServer
}

func (s *panicServer) Unary(ctx context.Context, req *testproto.UnaryRequest) (*testproto.UnaryResponse, error) {
	if req.SimulateError == "panic" {
		panic("unary handler panic")
	}
	return s.testServer.Unary(ctx, req)
}

func (s *panicServer) ServerStream(req *testproto.ServerStreamRequest, srv testproto.TestApi_ServerStreamServer) error {
	if req.SimulateError == "panic" {
		// send something first so the panic happens part-way through the stream
		if err := srv.Send(&testproto.ServerStreamResponse{Counter: 0}); err != nil {
			return err
		}
		panic("stream handler panic")
	}
	return s.testServer.ServerStream(req, srv)
}

func observePanics(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.ErrorLevel)
	SetPanicLogger(zap.New(core))
	t.Cleanup(func() { SetPanicLogger(nil) })
	return logs
}

func checkPanicLogged(t *testing.T, logs *observer.ObservedLogs, method string) {
	t.Helper()
	// the stream is closed after the recovery handler logs, so the entry is already there
	entries := logs.FilterField(zap.String("method", method)).All()
	if len(entries) != 1 {
		t.Fatalf("got %d panic log entries for %s, want 1 (all: %v)", len(entries), method, logs.All())
	}
	if _, ok := entries[0].ContextMap()["stack"]; !ok {
		t.Errorf("panic log entry has no stack: %v", entries[0].ContextMap())
	}
}

func checkHandlerPanicErr(t *testing.T, err error) {
	t.Helper()
	if code := status.Code(err); code != codes.Internal {
		t.Fatalf("got error %v, want code %v", err, codes.Internal)
	}
	if msg := status.Convert(err).Message(); msg != "internal error" {
		t.Errorf("got message %q, want the panic value kept server-side", msg)
	}
}

func TestWrapper_Unary_panic(t *testing.T) {
	logs := observePanics(t)
	client := testproto.NewTestApiClient(ServerToClient(testproto.TestApi_ServiceDesc, &panicServer{}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Unary(ctx, &testproto.UnaryRequest{SimulateError: "panic"})
	checkHandlerPanicErr(t, err)
	checkPanicLogged(t, logs, fmt.Sprintf("/%s/Unary", testproto.TestApi_ServiceDesc.ServiceName))

	// the wrapper is still usable after a panic
	res, err := client.Unary(ctx, &testproto.UnaryRequest{Msg: "after"})
	if err != nil {
		t.Fatalf("Unary after panic: %v", err)
	}
	if res.Msg != "after" {
		t.Errorf("Unary after panic got %q, want %q", res.Msg, "after")
	}
}

func TestWrapper_UnaryAsStream_panic(t *testing.T) {
	logs := observePanics(t)
	conn := ServerToClient(testproto.TestApi_ServiceDesc, &panicServer{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	method := fmt.Sprintf("/%s/Unary", testproto.TestApi_ServiceDesc.ServiceName)
	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{StreamName: "Unary"}, method)
	if err != nil {
		t.Fatalf("NewStream: %v", err)
	}
	if err := stream.SendMsg(&testproto.UnaryRequest{SimulateError: "panic"}); err != nil {
		t.Fatalf("SendMsg: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}
	err = stream.RecvMsg(&testproto.UnaryResponse{})
	checkHandlerPanicErr(t, err)
	checkPanicLogged(t, logs, method)
}

func TestWrapper_ServerStream_panic(t *testing.T) {
	logs := observePanics(t)
	client := testproto.NewTestApiClient(ServerToClient(testproto.TestApi_ServiceDesc, &panicServer{}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.ServerStream(ctx, &testproto.ServerStreamRequest{SimulateError: "panic"})
	if err != nil {
		t.Fatalf("ServerStream: %v", err)
	}
	res, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv: %v", err)
	}
	if res.Counter != 0 {
		t.Errorf("first Recv got counter %d, want 0", res.Counter)
	}
	_, err = stream.Recv()
	checkHandlerPanicErr(t, err)
	checkPanicLogged(t, logs, fmt.Sprintf("/%s/ServerStream", testproto.TestApi_ServiceDesc.ServiceName))

	// the wrapper is still usable after a panic
	stream, err = client.ServerStream(ctx, &testproto.ServerStreamRequest{NumRes: 2})
	if err != nil {
		t.Fatalf("ServerStream after panic: %v", err)
	}
	for i := range 2 {
		if _, err := stream.Recv(); err != nil {
			t.Fatalf("Recv %d after panic: %v", i, err)
		}
	}
}

func TestCallHandler_noLogger(t *testing.T) {
	SetPanicLogger(nil)
	err := callHandler("/test/Method", func() error { panic("boom") })
	checkHandlerPanicErr(t, err)
}
