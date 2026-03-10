package enterleavesensorpb

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelServer struct {
	traits.UnimplementedEnterLeaveSensorApiServer
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
	traits.RegisterEnterLeaveSensorApiServer(server, m)
}

func (m *ModelServer) GetEnterLeaveEvent(ctx context.Context, request *traits.GetEnterLeaveEventRequest) (*traits.EnterLeaveEvent, error) {
	return m.model.GetEnterLeaveEvent(resource.WithReadMask(request.ReadMask))
}

func (m *ModelServer) ResetEnterLeaveTotals(ctx context.Context, request *traits.ResetEnterLeaveTotalsRequest) (*traits.ResetEnterLeaveTotalsResponse, error) {
	return &traits.ResetEnterLeaveTotalsResponse{}, m.model.ResetTotals()
}

func (m *ModelServer) PullEnterLeaveEvents(request *traits.PullEnterLeaveEventsRequest, server traits.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	for change := range m.model.PullEnterLeaveEvents(server.Context(), resource.WithReadMask(request.ReadMask)) {
		err := server.Send(&traits.PullEnterLeaveEventsResponse{Changes: []*traits.PullEnterLeaveEventsResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), EnterLeaveEvent: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
