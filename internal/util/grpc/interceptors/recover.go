package interceptors

import (
	"context"
	"runtime/debug"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoverUnary returns a unary interceptor that turns a handler panic into a codes.Internal error, logging the
// panic and its stack to logger.
// The panic value isn't returned to the caller, as it may describe server internals.
// Chain it first so it also covers panics in the interceptors after it.
func RecoverUnary(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return grpc_recovery.UnaryServerInterceptor(recoveryHandler(logger))
}

// RecoverStream is the streaming equivalent of RecoverUnary.
func RecoverStream(logger *zap.Logger) grpc.StreamServerInterceptor {
	return grpc_recovery.StreamServerInterceptor(recoveryHandler(logger))
}

func recoveryHandler(logger *zap.Logger) grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, p any) error {
		// still inside the deferred recover, so the stack includes the frames that panicked
		method, _ := grpc.Method(ctx)
		logger.Error("recovered from handler panic",
			zap.String("method", method), zap.Any("panic", p), zap.ByteString("stack", debug.Stack()))
		return status.Error(codes.Internal, "internal error")
	})
}
