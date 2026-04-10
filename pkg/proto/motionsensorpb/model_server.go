package motionsensorpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedMotionSensorApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterMotionSensorApiServer(server, s)
}

func (s *ModelServer) GetMotionDetection(_ context.Context, req *GetMotionDetectionRequest) (*MotionDetection, error) {
	return s.model.GetMotionDetection(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullMotionDetections(request *PullMotionDetectionRequest, server MotionSensorApi_PullMotionDetectionsServer) error {
	for update := range s.model.PullMotionDetections(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullMotionDetectionResponse{Changes: []*PullMotionDetectionResponse_Change{{
			Name:            request.Name,
			ChangeTime:      timestamppb.New(update.ChangeTime),
			MotionDetection: update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
