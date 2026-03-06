package electricpb

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

//go:generate protomod protoc -- -I../../.. --go_out=paths=source_relative:../../../ --go-grpc_out=paths=source_relative:../../../ pkg/trait/electricpb/memory_settings.proto

func (s *ModelServer) UpdateDemand(_ context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	return s.model.UpdateDemand(request.Demand, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) CreateMode(_ context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() != "" {
		return nil, status.Errorf(codes.InvalidArgument, "id '%v' should be empty", request.GetMode().GetId())
	}

	return s.model.CreateMode(request.Mode)
}

func (s *ModelServer) UpdateMode(_ context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	return s.model.UpdateMode(request.Mode, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) DeleteMode(_ context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	// start by validating things
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.model.DeleteMode(request.Id, resource.WithAllowMissing(request.AllowMissing))
	return &emptypb.Empty{}, err
}
