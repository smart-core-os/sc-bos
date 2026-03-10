package server

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type AuthProvider struct {
	Creds  credentials.TransportCredentials
	logger *zap.Logger
}

func NewAuthProvider(creds credentials.TransportCredentials, logger *zap.Logger) *AuthProvider {
	return &AuthProvider{
		Creds:  creds,
		logger: logger,
	}
}

func (a *AuthProvider) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	a.logger.Debug("Unary interceptor:", zap.String("method", info.FullMethod))
	return handler(ctx, req)
}
