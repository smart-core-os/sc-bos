package storagepb

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/storagepb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// AdminHandler is called by ModelServer to perform an administrative action on the storage resource.
type AdminHandler func(ctx context.Context, action storagepb.StorageAdminAction) (*storagepb.PerformStorageAdminResponse, error)

// ModelServer implements StorageApiServer, StorageAdminApiServer, and StorageInfoServer
// backed by a Model.
type ModelServer struct {
	storagepb.UnimplementedStorageApiServer
	storagepb.UnimplementedStorageAdminApiServer
	storagepb.UnimplementedStorageInfoServer

	model        *Model
	adminHandler AdminHandler
	support      *storagepb.StorageSupport

	subscribers atomic.Int32
}

// NewModelServer creates a ModelServer backed by the given Model.
// Provide optional functional options to configure admin and info behaviour.
func NewModelServer(model *Model, opts ...ModelServerOption) *ModelServer {
	s := &ModelServer{model: model}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ModelServerOption is a functional option for NewModelServer.
type ModelServerOption func(*ModelServer)

// WithAdminHandler sets the handler invoked by PerformStorageAdmin.
func WithAdminHandler(h AdminHandler) ModelServerOption {
	return func(s *ModelServer) { s.adminHandler = h }
}

// WithStorageSupport sets the StorageSupport returned by DescribeStorage.
func WithStorageSupport(sup *storagepb.StorageSupport) ModelServerOption {
	return func(s *ModelServer) { s.support = sup }
}

// Register registers all three services on the given gRPC server.
func (s *ModelServer) Register(server *grpc.Server) {
	storagepb.RegisterStorageApiServer(server, s)
	storagepb.RegisterStorageAdminApiServer(server, s)
	storagepb.RegisterStorageInfoServer(server, s)
}

// Unwrap returns the underlying Model.
func (s *ModelServer) Unwrap() any {
	return s.model
}

// GetStorage implements StorageApiServer.
func (s *ModelServer) GetStorage(_ context.Context, req *storagepb.GetStorageRequest) (*storagepb.Storage, error) {
	return s.model.GetStorage(resource.WithReadMask(req.ReadMask))
}

// HasSubscribers reports whether any PullStorage streams are currently active.
// Polling goroutines can use this to skip work when no one is subscribed.
func (s *ModelServer) HasSubscribers() bool {
	return s.subscribers.Load() > 0
}

// PullStorage implements StorageApiServer.
func (s *ModelServer) PullStorage(req *storagepb.PullStorageRequest, server storagepb.StorageApi_PullStorageServer) error {
	s.subscribers.Add(1)
	defer s.subscribers.Add(-1)
	for change := range s.model.PullStorage(server.Context(), resource.WithReadMask(req.ReadMask), resource.WithUpdatesOnly(req.UpdatesOnly)) {
		msg := &storagepb.PullStorageResponse{
			Changes: []*storagepb.PullStorageResponse_Change{{
				Name:       req.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				Storage:    change.Value,
			}},
		}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

// PerformStorageAdmin implements StorageAdminApiServer.
func (s *ModelServer) PerformStorageAdmin(ctx context.Context, req *storagepb.PerformStorageAdminRequest) (*storagepb.PerformStorageAdminResponse, error) {
	if s.adminHandler == nil {
		return nil, status.Error(codes.Unimplemented, "PerformStorageAdmin not supported")
	}
	return s.adminHandler(ctx, req.Action)
}

// DescribeStorage implements StorageInfoServer.
func (s *ModelServer) DescribeStorage(_ context.Context, _ *storagepb.DescribeStorageRequest) (*storagepb.StorageSupport, error) {
	if s.support == nil {
		return &storagepb.StorageSupport{}, nil
	}
	return s.support, nil
}
