package interceptors

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoverUnary(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	interceptor := RecoverUnary(zap.New(core))

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"},
		func(ctx context.Context, req any) (any, error) {
			panic("boom")
		})
	checkRecovered(t, err, logs)

	// handlers that don't panic are passed through
	res, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"},
		func(ctx context.Context, req any) (any, error) {
			return "ok", status.Error(codes.NotFound, "nope")
		})
	if res != "ok" || status.Code(err) != codes.NotFound {
		t.Errorf("got (%v, %v), want the handler's own result", res, err)
	}
}

func TestRecoverStream(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	interceptor := RecoverStream(zap.New(core))

	err := interceptor(nil, &fakeServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/test/Stream"},
		func(srv any, stream grpc.ServerStream) error {
			panic("boom")
		})
	checkRecovered(t, err, logs)
}

func checkRecovered(t *testing.T, err error, logs *observer.ObservedLogs) {
	t.Helper()
	if code := status.Code(err); code != codes.Internal {
		t.Fatalf("got error %v, want code %v", err, codes.Internal)
	}
	if msg := status.Convert(err).Message(); msg != "internal error" {
		t.Errorf("got message %q, want the panic value kept server-side", msg)
	}
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d log entries, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["panic"] != "boom" {
		t.Errorf("logged panic %v, want %q", fields["panic"], "boom")
	}
	if _, ok := fields["stack"]; !ok {
		t.Errorf("log entry has no stack: %v", fields)
	}
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}
