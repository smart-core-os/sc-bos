package electricpb

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-api/go/types"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// ModelServer is an implementation of ElectricApiServer and MemorySettingsApiServer backed by a Model.
type ModelServer struct {
	UnimplementedElectricApiServer
	UnimplementedMemorySettingsApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterElectricApiServer(server, s)
	RegisterMemorySettingsApiServer(server, s)
}

func (s *ModelServer) GetDemand(_ context.Context, request *GetDemandRequest) (*ElectricDemand, error) {
	return s.model.Demand(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) PullDemand(request *PullDemandRequest, server ElectricApi_PullDemandServer) error {
	for update := range s.model.PullDemand(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullDemandResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Demand:     update.Value,
		}

		err := server.Send(&PullDemandResponse{
			Changes: []*PullDemandResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func (s *ModelServer) GetActiveMode(_ context.Context, request *GetActiveModeRequest) (*ElectricMode, error) {
	return s.model.ActiveMode(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateActiveMode(_ context.Context, request *UpdateActiveModeRequest) (*ElectricMode, error) {
	mode := request.GetActiveMode()
	// hydrate the mode using the list of known modes (by id)
	id := mode.GetId()
	if id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Id should be present during update")
	}

	return s.model.ChangeActiveMode(id)
}

func (s *ModelServer) ClearActiveMode(_ context.Context, _ *ClearActiveModeRequest) (*ElectricMode, error) {
	return s.model.ChangeToNormalMode()
}

func (s *ModelServer) PullActiveMode(request *PullActiveModeRequest, server ElectricApi_PullActiveModeServer) error {
	for event := range s.model.PullActiveMode(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullActiveModeResponse_Change{
			Name:       request.Name,
			ActiveMode: event.ActiveMode,
			ChangeTime: timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&PullActiveModeResponse{
			Changes: []*PullActiveModeResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

func (s *ModelServer) ListModes(_ context.Context, request *ListModesRequest) (*ListModesResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedModes := s.model.Modes(resource.WithReadMask(request.ReadMask))
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedModes), func(i int) bool {
			return sortedModes[i].Id > lastKey
		})
	}

	result := &ListModesResponse{
		TotalSize: int32(len(sortedModes)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedModes) {
		upperBound = len(sortedModes)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedModes[upperBound-1].Id,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Modes = sortedModes[nextIndex:upperBound]
	return result, nil
}

func (s *ModelServer) PullModes(request *PullModesRequest, server ElectricApi_PullModesServer) error {
	for change := range s.model.PullModes(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullModesResponse{Changes: []*PullModesResponse_Change{
			{
				Name:       request.Name,
				Type:       change.Type,
				NewValue:   change.NewValue,
				OldValue:   change.OldValue,
				ChangeTime: timestamppb.New(change.ChangeTime),
			},
		}})

		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
