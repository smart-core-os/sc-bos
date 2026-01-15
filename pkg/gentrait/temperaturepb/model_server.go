package temperaturepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	temperaturepb.UnimplementedTemperatureApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	temperaturepb.RegisterTemperatureApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetTemperature(_ context.Context, request *temperaturepb.GetTemperatureRequest) (*temperaturepb.Temperature, error) {
	return m.model.GetTemperature(resource.WithReadMask(request.ReadMask))
}

func (m *ModelServer) UpdateTemperature(_ context.Context, request *temperaturepb.UpdateTemperatureRequest) (*temperaturepb.Temperature, error) {
	return m.model.UpdateTemperature(request.Temperature, resource.WithUpdateMask(request.UpdateMask))
}

func (m *ModelServer) PullTemperature(request *temperaturepb.PullTemperatureRequest, server grpc.ServerStreamingServer[temperaturepb.PullTemperatureResponse]) error {
	for change := range m.model.PullTemperature(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &temperaturepb.PullTemperatureResponse{Changes: []*temperaturepb.PullTemperatureResponse_Change{{
			Name:        request.Name,
			ChangeTime:  timestamppb.New(change.ChangeTime),
			Temperature: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
