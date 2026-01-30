package allocationpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	allocationpb.UnimplementedAllocationApiServer
	Model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		Model: model,
	}
}

func (m *ModelServer) Register(server *grpc.Server) {
	allocationpb.RegisterAllocationApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.Model
}

func (m *ModelServer) GetAllocation(ctx context.Context, request *allocationpb.GetAllocationRequest) (*allocationpb.Allocation, error) {
	allocation := m.Model.GetAllocation()
	return allocation, nil
}

func (m *ModelServer) UpdateAllocation(ctx context.Context, request *allocationpb.UpdateAllocationRequest) (*allocationpb.Allocation, error) {
	m.Model.UpdateAllocation(request.Allocation, resource.WithUpdateMask(request.UpdateMask))
	return request.Allocation, nil
}

func (m *ModelServer) PullAllocation(request *allocationpb.PullAllocationRequest, server allocationpb.AllocationApi_PullAllocationServer) error {
	for change := range m.Model.PullAllocation(server.Context(), resource.WithReadMask(request.ReadMask)) {
		msg := &allocationpb.PullAllocationResponse{Changes: []*allocationpb.PullAllocationResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(change.ChangeTime),
			Allocation: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
