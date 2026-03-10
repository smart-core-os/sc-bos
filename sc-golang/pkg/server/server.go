package server

import (
	"context"
	"net"
	"net/url"

	"github.com/smart-core-os/sc-api/go/info"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
	infoServer   *InfoServer
	auth         *AuthProvider
	ctx          context.Context
	logger       *zap.Logger
}

func NewServer(ctx context.Context, auth *AuthProvider, logger *zap.Logger) *Server {
	// create gRPC server
	grpcServer := grpc.NewServer(
		grpc.Creds(auth.Creds),
		grpc.UnaryInterceptor(auth.UnaryInterceptor),
	)

	// create gRPC health server
	healthServer := health.NewServer()

	// register gRPC server with health and reflection apis
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	// create SC info server
	infoServer := NewInfoServer(logger)
	info.RegisterInfoServer(grpcServer, infoServer)

	return &Server{
		grpcServer,
		healthServer,
		infoServer,
		auth,
		ctx,
		logger,
	}
}

func (s *Server) Startup(address *url.URL) chan error {
	// config bind port
	lis, err := net.Listen(address.Scheme, address.Host)
	if err != nil {
		s.logger.Fatal("could not bind to tcp port", zap.Error(err))
	}
	s.logger.Debug("Server started", zap.String("address", address.String()))

	// setup graceful shutdown
	done := make(chan error)

	// start gRPC server
	go func() { done <- s.grpcServer.Serve(lis) }()

	return done
}

func (s *Server) Shutdown() {
	s.logger.Debug("Server shutting down")
	s.grpcServer.GracefulStop()
	s.healthServer.Shutdown()
	s.infoServer.Shutdown()
}

func (s *Server) RegisterDevice(device *info.Device) {
	s.infoServer.AddDevice(device)
}
