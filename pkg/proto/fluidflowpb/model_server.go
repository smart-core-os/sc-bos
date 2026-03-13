package fluidflowpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type ModelServer struct {
	UnimplementedFluidFlowApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}
func (m *ModelServer) Register(server *grpc.Server) {
	RegisterFluidFlowApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetFluidFlow(_ context.Context, _ *GetFluidFlowRequest) (*FluidFlow, error) {
	return m.model.GetFluidFlow()
}

func (m *ModelServer) PullFluidFlow(request *PullFluidFlowRequest, server FluidFlowApi_PullFluidFlowServer) error {
	for change := range m.model.PullFluidFlow(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &PullFluidFlowResponse{Changes: []*PullFluidFlowResponse_Change{
			{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				Flow:       change.Value,
			},
		}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) UpdateFluidFlow(_ context.Context, request *UpdateFluidFlowRequest) (*FluidFlow, error) {
	return m.model.UpdateFluidFlow(request.Flow, resource.WithUpdateMask(request.UpdateMask))
}
