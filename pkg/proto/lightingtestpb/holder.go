package lightingtestpb

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errNotEnabled = status.Error(codes.FailedPrecondition, "lighttest - not enabled")

type Holder struct {
	UnimplementedLightingTestApiServer

	client LightingTestApiClient
	mu     sync.Mutex
}

func (h *Holder) Register(server *grpc.Server) {
	RegisterLightingTestApiServer(server, h)
}

func (h *Holder) Fill(client LightingTestApiClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = client
}

func (h *Holder) Empty() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = nil
}

func (h *Holder) GetLightHealth(ctx context.Context, request *GetLightHealthRequest) (*LightHealth, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.GetLightHealth(ctx, request)
}

func (h *Holder) ListLightHealth(ctx context.Context, request *ListLightHealthRequest) (*ListLightHealthResponse, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.ListLightHealth(ctx, request)
}

func (h *Holder) ListLightEvents(ctx context.Context, request *ListLightEventsRequest) (*ListLightEventsResponse, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.ListLightEvents(ctx, request)
}

func (h *Holder) GetReportCSV(ctx context.Context, request *GetReportCSVRequest) (*ReportCSV, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.GetReportCSV(ctx, request)
}

func (h *Holder) getClient() (LightingTestApiClient, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client == nil {
		return nil, errNotEnabled
	}
	return h.client, nil
}
