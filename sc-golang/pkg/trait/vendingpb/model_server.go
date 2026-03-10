package vendingpb

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

// ModelServer adapts a Model to implement traits.VendingApiServer.
type ModelServer struct {
	traits.UnimplementedVendingApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterVendingApiServer(server, m)
}

func (m *ModelServer) ListConsumables(_ context.Context, request *traits.ListConsumablesRequest) (*traits.ListConsumablesResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedItems := m.model.ListConsumables(resource.WithReadMask(request.ReadMask))
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedItems), func(i int) bool {
			return sortedItems[i].Name > lastKey
		})
	}

	result := &traits.ListConsumablesResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedItems[upperBound-1].Name,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Consumables = sortedItems[nextIndex:upperBound]
	return result, nil
}

func (m *ModelServer) PullConsumables(request *traits.PullConsumablesRequest, server traits.VendingApi_PullConsumablesServer) error {
	for change := range m.model.PullConsumables(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullConsumablesResponse{Changes: []*traits.PullConsumablesResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) GetStock(_ context.Context, request *traits.GetStockRequest) (*traits.Consumable_Stock, error) {
	if request.Consumable == "" {
		return nil, status.Error(codes.InvalidArgument, "GetStockRequest.consumable empty")
	}
	stock, exists := m.model.GetStock(request.Consumable, resource.WithReadMask(request.ReadMask))
	if !exists {
		return nil, status.Errorf(codes.NotFound, "unknown consumable:%v", request.Consumable)
	}
	return stock, nil
}

func (m *ModelServer) UpdateStock(_ context.Context, request *traits.UpdateStockRequest) (*traits.Consumable_Stock, error) {
	return m.model.UpdateStock(request.Stock, resource.WithUpdateMask(request.UpdateMask))
}

func (m *ModelServer) PullStock(request *traits.PullStockRequest, server traits.VendingApi_PullStockServer) error {
	for change := range m.model.PullStock(server.Context(), request.Consumable, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullStockResponse{Changes: []*traits.PullStockResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), Stock: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) ListInventory(_ context.Context, request *traits.ListInventoryRequest) (*traits.ListInventoryResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedItems := m.model.ListInventory(resource.WithReadMask(request.ReadMask))
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedItems), func(i int) bool {
			return sortedItems[i].Consumable > lastKey
		})
	}

	result := &traits.ListInventoryResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedItems[upperBound-1].Consumable,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Inventory = sortedItems[nextIndex:upperBound]
	return result, nil
}

func (m *ModelServer) PullInventory(request *traits.PullInventoryRequest, server traits.VendingApi_PullInventoryServer) error {
	for change := range m.model.PullInventory(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullInventoryResponse{Changes: []*traits.PullInventoryResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) Dispense(_ context.Context, request *traits.DispenseRequest) (*traits.Consumable_Stock, error) {
	if request.Consumable == "" {
		return nil, status.Error(codes.InvalidArgument, "request.consumable is absent")
	}
	return m.model.DispenseInstantly(request.Consumable, request.Quantity)
}

func (m *ModelServer) StopDispense(ctx context.Context, request *traits.StopDispenseRequest) (*traits.Consumable_Stock, error) {
	// always succeeds, we always dispense immediately
	return m.GetStock(ctx, &traits.GetStockRequest{Consumable: request.Name})
}
