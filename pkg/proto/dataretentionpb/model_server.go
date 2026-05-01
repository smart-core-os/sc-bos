package dataretentionpb

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// PurgeHandler is called by ModelServer to delete stored data.
// If req.Before is nil, all records are removed; otherwise only records older than req.Before.
type PurgeHandler func(ctx context.Context, req *PurgeDataRetentionRequest) (*PurgeDataRetentionResponse, error)

// CompactHandler is called by ModelServer to compact/optimise storage.
type CompactHandler func(ctx context.Context, req *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error)

// ModelServer implements DataRetentionApiServer and DataRetentionInfoServer backed by a Model.
type ModelServer struct {
	UnimplementedDataRetentionApiServer
	UnimplementedDataRetentionInfoServer

	model          *Model
	purgeHandler   PurgeHandler
	compactHandler CompactHandler
	support        *DataRetentionSupport

	subscribers atomic.Int32
}

// NewModelServer creates a ModelServer backed by the given Model.
// Provide optional functional options to configure handler and info behaviour.
func NewModelServer(model *Model, opts ...ModelServerOption) *ModelServer {
	s := &ModelServer{model: model}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ModelServerOption is a functional option for NewModelServer.
type ModelServerOption func(*ModelServer)

// WithPurgeHandler sets the handler invoked by PurgeDataRetention.
func WithPurgeHandler(h PurgeHandler) ModelServerOption {
	return func(s *ModelServer) { s.purgeHandler = h }
}

// WithCompactHandler sets the handler invoked by CompactDataRetention.
func WithCompactHandler(h CompactHandler) ModelServerOption {
	return func(s *ModelServer) { s.compactHandler = h }
}

// WithDataRetentionSupport sets the DataRetentionSupport returned by DescribeDataRetention.
func WithDataRetentionSupport(sup *DataRetentionSupport) ModelServerOption {
	return func(s *ModelServer) { s.support = sup }
}

// Register registers both services on the given gRPC server.
func (s *ModelServer) Register(server *grpc.Server) {
	RegisterDataRetentionApiServer(server, s)
	RegisterDataRetentionInfoServer(server, s)
}

// Unwrap returns the underlying Model.
func (s *ModelServer) Unwrap() any {
	return s.model
}

// GetDataRetention implements DataRetentionApiServer.
func (s *ModelServer) GetDataRetention(_ context.Context, req *GetDataRetentionRequest) (*DataRetention, error) {
	return s.model.GetDataRetention(resource.WithReadMask(req.ReadMask))
}

// PullDataRetention implements DataRetentionApiServer.
func (s *ModelServer) PullDataRetention(req *PullDataRetentionRequest, server DataRetentionApi_PullDataRetentionServer) error {
	s.subscribers.Add(1)
	defer s.subscribers.Add(-1)
	for change := range s.model.PullDataRetention(server.Context(), resource.WithReadMask(req.ReadMask), resource.WithUpdatesOnly(req.UpdatesOnly)) {
		msg := &PullDataRetentionResponse{
			Changes: []*PullDataRetentionResponse_Change{{
				Name:          req.Name,
				ChangeTime:    timestamppb.New(change.ChangeTime),
				DataRetention: change.Value,
			}},
		}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

// PurgeDataRetention implements DataRetentionApiServer.
func (s *ModelServer) PurgeDataRetention(ctx context.Context, req *PurgeDataRetentionRequest) (*PurgeDataRetentionResponse, error) {
	if s.purgeHandler == nil {
		return nil, status.Error(codes.Unimplemented, "PurgeDataRetention not supported")
	}
	return s.purgeHandler(ctx, req)
}

// CompactDataRetention implements DataRetentionApiServer.
func (s *ModelServer) CompactDataRetention(ctx context.Context, req *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error) {
	if s.compactHandler == nil {
		return nil, status.Error(codes.Unimplemented, "CompactDataRetention not supported")
	}
	return s.compactHandler(ctx, req)
}

// DescribeDataRetention implements DataRetentionInfoServer.
func (s *ModelServer) DescribeDataRetention(_ context.Context, _ *DescribeDataRetentionRequest) (*DataRetentionSupport, error) {
	if s.support == nil {
		return &DataRetentionSupport{}, nil
	}
	return s.support, nil
}
