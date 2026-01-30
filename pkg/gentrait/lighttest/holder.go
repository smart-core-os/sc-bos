package lighttest

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/lightingtestpb"
)

var errNotEnabled = status.Error(codes.FailedPrecondition, "lighttest - not enabled")

type Holder struct {
	lightingtestpb.UnimplementedLightingTestApiServer

	client lightingtestpb.LightingTestApiClient
	mu     sync.Mutex
}

func (h *Holder) Register(server *grpc.Server) {
	lightingtestpb.RegisterLightingTestApiServer(server, h)
}

func (h *Holder) Fill(client lightingtestpb.LightingTestApiClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = client
}

func (h *Holder) Empty() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client = nil
}

func (h *Holder) GetLightHealth(ctx context.Context, request *lightingtestpb.GetLightHealthRequest) (*lightingtestpb.LightHealth, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.GetLightHealth(ctx, request)
}

func (h *Holder) ListLightHealth(ctx context.Context, request *lightingtestpb.ListLightHealthRequest) (*lightingtestpb.ListLightHealthResponse, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.ListLightHealth(ctx, request)
}

func (h *Holder) ListLightEvents(ctx context.Context, request *lightingtestpb.ListLightEventsRequest) (*lightingtestpb.ListLightEventsResponse, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.ListLightEvents(ctx, request)
}

func (h *Holder) GetReportCSV(ctx context.Context, request *lightingtestpb.GetReportCSVRequest) (*lightingtestpb.ReportCSV, error) {
	c, err := h.getClient()
	if err != nil {
		return nil, err
	}
	return c.GetReportCSV(ctx, request)
}

func (h *Holder) getClient() (lightingtestpb.LightingTestApiClient, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client == nil {
		return nil, errNotEnabled
	}
	return h.client, nil
}
