package hailpb

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer adapts a Model to implement traits.HailApiServer.
type ModelServer struct {
	traits.UnimplementedHailApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterHailApiServer(server, m)
}

func (m *ModelServer) CreateHail(_ context.Context, request *traits.CreateHailRequest) (*traits.Hail, error) {
	hail := request.Hail
	if hail.State == traits.Hail_STATE_UNSPECIFIED {
		hail.State = traits.Hail_CALLED
	}
	return m.model.CreateHail(hail)
}

func (m *ModelServer) GetHail(_ context.Context, request *traits.GetHailRequest) (*traits.Hail, error) {
	hail, exists := m.model.GetHail(request.Id, resource.WithReadMask(request.ReadMask))
	if !exists {
		return nil, status.Errorf(codes.NotFound, "id:%v", request.Id)
	}
	return hail, nil
}

func (m *ModelServer) UpdateHail(_ context.Context, request *traits.UpdateHailRequest) (*traits.Hail, error) {
	return m.model.UpdateHail(request.Hail, resource.WithUpdateMask(request.UpdateMask))
}

func (m *ModelServer) DeleteHail(_ context.Context, request *traits.DeleteHailRequest) (*traits.DeleteHailResponse, error) {
	_, err := m.model.DeleteHail(request.Id, resource.WithAllowMissing(request.AllowMissing))
	return &traits.DeleteHailResponse{}, err
}

func (m *ModelServer) PullHail(request *traits.PullHailRequest, server traits.HailApi_PullHailServer) error {
	for change := range m.model.PullHail(server.Context(), request.Id, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullHailResponse{Changes: []*traits.PullHailResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), Hail: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) ListHails(_ context.Context, request *traits.ListHailsRequest) (*traits.ListHailsResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedItems := m.model.ListHails(resource.WithReadMask(request.ReadMask))
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedItems), func(i int) bool {
			return sortedItems[i].Id > lastKey
		})
	}

	result := &traits.ListHailsResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedItems[upperBound-1].Id,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Hails = sortedItems[nextIndex:upperBound]
	return result, nil
}

func (m *ModelServer) PullHails(request *traits.PullHailsRequest, server traits.HailApi_PullHailsServer) error {
	for change := range m.model.PullHails(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullHailsResponse{Changes: []*traits.PullHailsResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
