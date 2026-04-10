package energystoragepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedEnergyStorageApiServer

	model *Model

	readOnly bool
}

func NewModelServer(model *Model, opts ...ServerOption) *ModelServer {
	s := &ModelServer{model: model}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterEnergyStorageApiServer(server, s)
}

func (s *ModelServer) GetEnergyLevel(_ context.Context, request *GetEnergyLevelRequest) (*EnergyLevel, error) {
	return s.model.GetEnergyLevel(resource.WithReadMask(request.GetReadMask()))
}

func (s *ModelServer) PullEnergyLevel(request *PullEnergyLevelRequest, server EnergyStorageApi_PullEnergyLevelServer) error {
	for update := range s.model.PullEnergyLevel(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullEnergyLevelResponse_Change{
			Name:        request.Name,
			ChangeTime:  timestamppb.New(update.ChangeTime),
			EnergyLevel: update.Value,
		}

		err := server.Send(&PullEnergyLevelResponse{
			Changes: []*PullEnergyLevelResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

func (s *ModelServer) Charge(_ context.Context, request *ChargeRequest) (*ChargeResponse, error) {
	if s.readOnly {
		return nil, status.Errorf(codes.Unimplemented, "EnergyStorage.Charge")
	}

	level := EnergyLevel{}
	if request.GetCharge() {
		level.Flow = &EnergyLevel_Charge{}
	} else {
		level.Flow = &EnergyLevel_Discharge{}
	}
	_, err := s.model.UpdateEnergyLevel(&level, resource.WithUpdatePaths("idle", "charge", "discharge"))
	if err != nil {
		return nil, err
	}
	return &ChargeResponse{}, nil
}

type ServerOption func(s *ModelServer)

func ReadOnly() ServerOption {
	return func(s *ModelServer) {
		s.readOnly = true
	}
}
