package emergencypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedEmergencyApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterEmergencyApiServer(server, s)
}

func (s *ModelServer) GetEmergency(_ context.Context, req *GetEmergencyRequest) (*Emergency, error) {
	return s.model.GetEmergency(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateEmergency(_ context.Context, req *UpdateEmergencyRequest) (*Emergency, error) {
	return s.model.UpdateEmergency(req.Emergency, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullEmergency(request *PullEmergencyRequest, server EmergencyApi_PullEmergencyServer) error {
	for update := range s.model.PullEmergency(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullEmergencyResponse{Changes: []*PullEmergencyResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Emergency:  update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
