package dataretentionpb

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Backend handles storage operations for the data retention trait.
// All operations target the same underlying store.
type Backend interface {
	// Purge deletes stored data. If before is nil, all data is removed;
	// otherwise only data recorded before *before is removed.
	// Returns the number of freed items.
	Purge(ctx context.Context, before *time.Time) (freedItems uint64, err error)
}

// Compacter may be implemented by a Backend that supports storage compaction.
type Compacter interface {
	Compact(ctx context.Context) error
}

// ModelServer implements DataRetentionApiServer and DataRetentionInfoServer backed by a Model.
type ModelServer struct {
	UnimplementedDataRetentionApiServer
	UnimplementedDataRetentionInfoServer

	model    *Model
	backend  Backend
	itemName string
}

// NewModelServer creates a ModelServer backed by the given Model and Backend.
// backend may be nil, in which case all mutating RPCs return Unimplemented.
func NewModelServer(model *Model, backend Backend, opts ...ModelServerOption) *ModelServer {
	s := &ModelServer{model: model, backend: backend}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ModelServerOption is a functional option for NewModelServer.
type ModelServerOption func(*ModelServer)

// WithItemName sets the singular item name reported by DescribeDataRetention
// (e.g. "row", "record", "file").
func WithItemName(name string) ModelServerOption {
	return func(s *ModelServer) { s.itemName = name }
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
	if s.backend == nil {
		return nil, status.Error(codes.Unimplemented, "PurgeDataRetention not supported")
	}
	var before *time.Time
	if req.Before != nil {
		t := req.Before.AsTime()
		before = &t
	}
	freed, err := s.backend.Purge(ctx, before)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "purge: %v", err)
	}
	return &PurgeDataRetentionResponse{FreedItemCount: &freed}, nil
}

// CompactDataRetention implements DataRetentionApiServer.
func (s *ModelServer) CompactDataRetention(ctx context.Context, _ *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error) {
	c, ok := s.backend.(Compacter)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "CompactDataRetention not supported")
	}
	if err := c.Compact(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "compact: %v", err)
	}
	return &CompactDataRetentionResponse{}, nil
}

// DescribeDataRetention implements DataRetentionInfoServer.
// Capabilities are derived from the backend's interface implementation.
func (s *ModelServer) DescribeDataRetention(_ context.Context, _ *DescribeDataRetentionRequest) (*DataRetentionSupport, error) {
	_, canCompact := s.backend.(Compacter)
	return &DataRetentionSupport{
		CanPurge:   s.backend != nil,
		CanCompact: canCompact,
		ItemName:   s.itemName,
	}, nil
}
