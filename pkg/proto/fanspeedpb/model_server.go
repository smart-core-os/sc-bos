package fanspeedpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// ModelServer adapts a Model as a traits.FanSpeedApiServer.
type ModelServer struct {
	UnimplementedFanSpeedApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterFanSpeedApiServer(server, s)
}

func (s *ModelServer) GetFanSpeed(_ context.Context, request *GetFanSpeedRequest) (*FanSpeed, error) {
	return s.model.FanSpeed(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateFanSpeed(_ context.Context, request *UpdateFanSpeedRequest) (*FanSpeed, error) {
	return s.model.UpdateFanSpeed(request.FanSpeed, resource.InterceptBefore(func(old, new proto.Message) {
		if request.Relative {
			oldVal := old.(*FanSpeed)
			newVal := new.(*FanSpeed)
			newVal.Percentage += oldVal.Percentage
			newVal.PresetIndex += oldVal.PresetIndex
			// todo: should we support setting the preset relatively if we're between presets?
		}
	}))
}

func (s *ModelServer) PullFanSpeed(request *PullFanSpeedRequest, server FanSpeedApi_PullFanSpeedServer) error {
	for change := range s.model.PullFanSpeed(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullFanSpeedResponse{Changes: []*PullFanSpeedResponse_Change{
			{Name: request.Name, FanSpeed: change.Value, ChangeTime: timestamppb.New(change.ChangeTime)},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ModelServer) ReverseFanSpeedDirection(ctx context.Context, request *ReverseFanSpeedDirectionRequest) (*FanSpeed, error) {
	// TODO implement me
	panic("implement me")
}
