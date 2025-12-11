package allocationpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	gen.UnimplementedAllocationApiServer
	Model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		Model: model,
	}
}

func (m *ModelServer) Register(server *grpc.Server) {
	gen.RegisterAllocationApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.Model
}

func (m *ModelServer) GetAllocation(ctx context.Context, request *gen.GetAllocationRequest) (*gen.Allocation, error) {
	allocations := m.Model.ListAllocations()
	for _, a := range allocations.Allocations {
		if a.GetId() != "" && a.GetId() == request.GetAllocationId() {
			return a, nil
		}
	}

	return nil, status.Error(codes.NotFound, "allocation not found")
}

func (m *ModelServer) UpdateAllocation(ctx context.Context, request *gen.UpdateAllocationRequest) (*gen.Allocation, error) {
	m.Model.UpdateAllocation(request.Allocation, resource.WithUpdateMask(request.UpdateMask))
	return request.Allocation, nil
}

func (m *ModelServer) PullAllocations(request *gen.PullAllocationsRequest, server gen.AllocationApi_PullAllocationsServer) error {
	allocations := m.Model.ListAllocationResources()
	for _, a := range allocations {
		allocation, ok := a.Get().(*gen.Allocation)
		if !ok {
			continue
		}
		if allocation.GetId() != "" && allocation.GetId() == request.GetAllocationId() {
			changes := a.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly))
			for change := range changes {
				select {
				case <-server.Context().Done():
					return server.Context().Err()
				default:
				}
				msg := &gen.PullAllocationsResponse{
					Changes: []*gen.PullAllocationsResponse_Change{
						{
							Name:       request.Name,
							ChangeTime: timestamppb.New(change.ChangeTime),
							Allocation: change.Value.(*gen.Allocation),
						},
					},
				}
				if err := server.Send(msg); err != nil {
					return err
				}
			}

			break
		}
	}
	return status.Error(codes.NotFound, "allocation not found")
}

func (m *ModelServer) ListAllocatableResources(ctx context.Context, request *gen.ListAllocatableResourcesRequest) (*gen.ListAllocatableResourcesResponse, error) {
	return m.Model.ListAllocations(), nil
}
