package enterleavesensorpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelServer struct {
	enterleavesensorpb.UnimplementedEnterLeaveSensorApiServer
	model *Model
}

// NewModelServer converts a Model into a type implementing traits.EnterLeaveSensorApiServer.
func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	enterleavesensorpb.RegisterEnterLeaveSensorApiServer(server, m)
}

func (m *ModelServer) GetEnterLeaveEvent(ctx context.Context, request *enterleavesensorpb.GetEnterLeaveEventRequest) (*enterleavesensorpb.EnterLeaveEvent, error) {
	return m.model.GetEnterLeaveEvent(resource.WithReadMask(request.ReadMask))
}

func (m *ModelServer) ResetEnterLeaveTotals(ctx context.Context, request *enterleavesensorpb.ResetEnterLeaveTotalsRequest) (*enterleavesensorpb.ResetEnterLeaveTotalsResponse, error) {
	return &enterleavesensorpb.ResetEnterLeaveTotalsResponse{}, m.model.ResetTotals()
}

func (m *ModelServer) PullEnterLeaveEvents(request *enterleavesensorpb.PullEnterLeaveEventsRequest, server enterleavesensorpb.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	for change := range m.model.PullEnterLeaveEvents(server.Context(), resource.WithReadMask(request.ReadMask)) {
		err := server.Send(&enterleavesensorpb.PullEnterLeaveEventsResponse{Changes: []*enterleavesensorpb.PullEnterLeaveEventsResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), EnterLeaveEvent: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
