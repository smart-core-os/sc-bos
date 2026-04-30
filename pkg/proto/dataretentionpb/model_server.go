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

// ClearHandler is called by ModelServer to clear all records from the data retention resource.
type ClearHandler func(ctx context.Context, req *ClearDataRetentionRequest) (*ClearDataRetentionResponse, error)

// DeleteOldHandler is called by ModelServer to delete records older than the given retention period.
type DeleteOldHandler func(ctx context.Context, req *DeleteOldDataRetentionRequest) (*DeleteOldDataRetentionResponse, error)

// CompactHandler is called by ModelServer to compact/optimise storage.
type CompactHandler func(ctx context.Context, req *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error)

// SpringCleanHandler is called by ModelServer to delete old records and compact storage in one operation.
type SpringCleanHandler func(ctx context.Context, req *SpringCleanDataRetentionRequest) (*SpringCleanDataRetentionResponse, error)

// ModelServer implements DataRetentionApiServer and DataRetentionInfoServer backed by a Model.
type ModelServer struct {
	UnimplementedDataRetentionApiServer
	UnimplementedDataRetentionInfoServer

	model            *Model
	clearHandler     ClearHandler
	deleteOldHandler DeleteOldHandler
	compactHandler   CompactHandler
	springCleanHandler SpringCleanHandler
	support          *DataRetentionSupport

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

// WithClearHandler sets the handler invoked by ClearDataRetention.
func WithClearHandler(h ClearHandler) ModelServerOption {
	return func(s *ModelServer) { s.clearHandler = h }
}

// WithDeleteOldHandler sets the handler invoked by DeleteOldDataRetention.
func WithDeleteOldHandler(h DeleteOldHandler) ModelServerOption {
	return func(s *ModelServer) { s.deleteOldHandler = h }
}

// WithCompactHandler sets the handler invoked by CompactDataRetention.
func WithCompactHandler(h CompactHandler) ModelServerOption {
	return func(s *ModelServer) { s.compactHandler = h }
}

// WithSpringCleanHandler sets the handler invoked by SpringCleanDataRetention.
func WithSpringCleanHandler(h SpringCleanHandler) ModelServerOption {
	return func(s *ModelServer) { s.springCleanHandler = h }
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

// HasSubscribers reports whether any PullDataRetention streams are currently active.
// Polling goroutines can use this to skip work when no one is subscribed.
func (s *ModelServer) HasSubscribers() bool {
	return s.subscribers.Load() > 0
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

// ClearDataRetention implements DataRetentionApiServer.
func (s *ModelServer) ClearDataRetention(ctx context.Context, req *ClearDataRetentionRequest) (*ClearDataRetentionResponse, error) {
	if s.clearHandler == nil {
		return nil, status.Error(codes.Unimplemented, "ClearDataRetention not supported")
	}
	return s.clearHandler(ctx, req)
}

// DeleteOldDataRetention implements DataRetentionApiServer.
func (s *ModelServer) DeleteOldDataRetention(ctx context.Context, req *DeleteOldDataRetentionRequest) (*DeleteOldDataRetentionResponse, error) {
	if s.deleteOldHandler == nil {
		return nil, status.Error(codes.Unimplemented, "DeleteOldDataRetention not supported")
	}
	return s.deleteOldHandler(ctx, req)
}

// CompactDataRetention implements DataRetentionApiServer.
func (s *ModelServer) CompactDataRetention(ctx context.Context, req *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error) {
	if s.compactHandler == nil {
		return nil, status.Error(codes.Unimplemented, "CompactDataRetention not supported")
	}
	return s.compactHandler(ctx, req)
}

// SpringCleanDataRetention implements DataRetentionApiServer.
func (s *ModelServer) SpringCleanDataRetention(ctx context.Context, req *SpringCleanDataRetentionRequest) (*SpringCleanDataRetentionResponse, error) {
	if s.springCleanHandler == nil {
		return nil, status.Error(codes.Unimplemented, "SpringCleanDataRetention not supported")
	}
	return s.springCleanHandler(ctx, req)
}

// DescribeDataRetention implements DataRetentionInfoServer.
func (s *ModelServer) DescribeDataRetention(_ context.Context, _ *DescribeDataRetentionRequest) (*DataRetentionSupport, error) {
	if s.support == nil {
		return &DataRetentionSupport{}, nil
	}
	return s.support, nil
}
