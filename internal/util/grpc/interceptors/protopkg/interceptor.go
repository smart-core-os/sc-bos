// Package protopkg aids with the migration from unversioned to versioned gRPC APIs.
// This package will be temporary while releases before the migration are maintained.
// The primary mechanism for the migration is a gRPC interceptor that translates between the old and new package structure.
package protopkg

import (
	"context"

	"google.golang.org/grpc"
)

// Interceptor provides mechanisms for forward and backward compatibility between unversioned and versioned (v1) gRPC APIs.
// The UnaryInterceptor and StreamInterceptor methods adjust interceptor and unknown service handling for middleware and dynamic services.
// The ServiceRegistrar and UnknownServiceHandler methods allow directly registered services to also be translated.
//
// Example usage
//
//	compatInterceptor := protopkg.NewNewToOldInterceptor()
//	grpcServer := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(compatInterceptor.UnaryInterceptor()),
//	    grpc.ChainStreamInterceptor(compatInterceptor.StreamInterceptor()),
//	    // other interceptors and options...
//	    grpc.UnknownServiceHandler(compatInterceptor.UnknownServiceHandler(yourUnknownServiceHandler)),
//	)
//	grpcRegistrar := compatInterceptor.ServiceRegistrar(grpcServer)
//	yourpackage.RegisterYourServiceServer(grpcRegistrar, yourServiceImpl)
//	// go grpcServer.Serve(...)
type Interceptor struct {
	translateFn func(string) string
}

// NewNewToOldInterceptor creates an interceptor that translates new-style paths to old-style.
// Example: /smartcore.bos.meter.v1.MeterApi/GetMeterReading -> /smartcore.bos.MeterApi/GetMeterReading
func NewNewToOldInterceptor() *Interceptor {
	return &Interceptor{translateFn: newToOld}
}

// NewOldToNewInterceptor creates an interceptor that translates old-style paths to new-style.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> /smartcore.bos.meter.v1.MeterApi/GetMeterReading
func NewOldToNewInterceptor() *Interceptor {
	return &Interceptor{translateFn: oldToNew}
}

// UnaryInterceptor returns a gRPC unary server interceptor.
func (i *Interceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return unaryInterceptor(i.translateFn)
}

// StreamInterceptor returns a gRPC stream server interceptor.
func (i *Interceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return streamInterceptor(i.translateFn)
}

// unaryInterceptor creates a unary interceptor that translates paths using the given function.
func unaryInterceptor(translateFn func(string) string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		translated := translateFn(info.FullMethod)
		if translated != info.FullMethod {
			info.FullMethod = translated
			ctx = contextWithMethod(ctx, translated)
		}
		return handler(ctx, req)
	}
}

// streamInterceptor creates a stream interceptor that translates paths using the given function.
func streamInterceptor(translateFn func(string) string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		translated := translateFn(info.FullMethod)
		if translated != info.FullMethod {
			info.FullMethod = translated
			ss = &wrappedServerStream{
				ServerStream: ss,
				ctx:          contextWithMethod(ss.Context(), translated),
			}
		}

		return handler(srv, ss)
	}
}

func contextWithMethod(ctx context.Context, method string) context.Context {
	return grpc.NewContextWithServerTransportStream(ctx, &methodOverride{
		ServerTransportStream: grpc.ServerTransportStreamFromContext(ctx),
		method:                method,
	})
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

type methodOverride struct {
	grpc.ServerTransportStream
	method string
}

func (m *methodOverride) Method() string {
	return m.method
}
