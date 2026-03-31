package vendingpb

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// ModelServer adapts a Model to implement traits.VendingApiServer.
type ModelServer struct {
	UnimplementedVendingApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterVendingApiServer(server, m)
}

func (m *ModelServer) ListConsumables(_ context.Context, request *ListConsumablesRequest) (*ListConsumablesResponse, error) {
	pageToken := &typespb.PageToken{}
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

	result := &ListConsumablesResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &typespb.PageToken_LastResourceName{
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

func (m *ModelServer) PullConsumables(request *PullConsumablesRequest, server VendingApi_PullConsumablesServer) error {
	for change := range m.model.PullConsumables(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullConsumablesResponse{Changes: []*PullConsumablesResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) GetStock(_ context.Context, request *GetStockRequest) (*Consumable_Stock, error) {
	if request.Consumable == "" {
		return nil, status.Error(codes.InvalidArgument, "GetStockRequest.consumable empty")
	}
	stock, exists := m.model.GetStock(request.Consumable, resource.WithReadMask(request.ReadMask))
	if !exists {
		return nil, status.Errorf(codes.NotFound, "unknown consumable:%v", request.Consumable)
	}
	return stock, nil
}

func (m *ModelServer) UpdateStock(_ context.Context, request *UpdateStockRequest) (*Consumable_Stock, error) {
	return m.model.UpdateStock(request.Stock, resource.WithUpdateMask(request.UpdateMask))
}

func (m *ModelServer) PullStock(request *PullStockRequest, server VendingApi_PullStockServer) error {
	for change := range m.model.PullStock(server.Context(), request.Consumable, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullStockResponse{Changes: []*PullStockResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), Stock: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) ListInventory(_ context.Context, request *ListInventoryRequest) (*ListInventoryResponse, error) {
	pageToken := &typespb.PageToken{}
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

	result := &ListInventoryResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &typespb.PageToken_LastResourceName{
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

func (m *ModelServer) PullInventory(request *PullInventoryRequest, server VendingApi_PullInventoryServer) error {
	for change := range m.model.PullInventory(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullInventoryResponse{Changes: []*PullInventoryResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) Dispense(_ context.Context, request *DispenseRequest) (*Consumable_Stock, error) {
	if request.Consumable == "" {
		return nil, status.Error(codes.InvalidArgument, "request.consumable is absent")
	}
	return m.model.DispenseInstantly(request.Consumable, request.Quantity)
}

func (m *ModelServer) StopDispense(ctx context.Context, request *StopDispenseRequest) (*Consumable_Stock, error) {
	// always succeeds, we always dispense immediately
	return m.GetStock(ctx, &GetStockRequest{Consumable: request.Name})
}
