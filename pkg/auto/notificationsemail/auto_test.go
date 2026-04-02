package notificationsemail

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// mockAlertServer is a minimal AlertApiServer that returns a configurable error or
// response for ListAlerts.
type mockAlertServer struct {
	alertpb.UnimplementedAlertApiServer
	responses []*alertpb.ListAlertsResponse
	err       error
	callCount int
}

func (m *mockAlertServer) ListAlerts(_ context.Context, _ *alertpb.ListAlertsRequest) (*alertpb.ListAlertsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.callCount < len(m.responses) {
		r := m.responses[m.callCount]
		m.callCount++
		return r, nil
	}
	return &alertpb.ListAlertsResponse{}, nil
}

func newMockAlertClient(srv alertpb.AlertApiServer) alertpb.AlertApiClient {
	conn := wrap.ServerToClient(alertpb.AlertApi_ServiceDesc, srv)
	return alertpb.NewAlertApiClient(conn)
}

func newTestAutoImpl() *autoImpl {
	logger, _ := zap.NewDevelopment()
	return &autoImpl{Services: auto.Services{Logger: logger}}
}

func TestGetAlertsInLastMonth_FirstPageError(t *testing.T) {
	a := newTestAutoImpl()
	client := newMockAlertClient(&mockAlertServer{err: errors.New("rpc unavailable")})
	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	// Before the fix this panics with a nil dereference on res.Alerts.
	result := a.getAlertsInLastMonth(context.Background(), client, "test", now)
	if result == nil {
		// nil slice is acceptable
	}
	if len(result) != 0 {
		t.Errorf("expected empty result on RPC error, got %d alerts", len(result))
	}
}

func TestGetAlertsInLastMonth_PaginationError(t *testing.T) {
	a := newTestAutoImpl()
	firstPage := &alertpb.ListAlertsResponse{
		NextPageToken: "page2",
		Alerts:        []*alertpb.Alert{{Id: "alert1"}},
	}
	secondCallErr := errors.New("rpc error on page 2")
	client := newMockAlertClient(&sequentialMockServer{
		first:     firstPage,
		secondErr: secondCallErr,
	})

	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	result := a.getAlertsInLastMonth(context.Background(), client, "test", now)
	// Should not panic; result may be empty or contain first page alerts
	_ = result
}

// sequentialMockServer returns firstPage on the first call and secondErr on the second.
type sequentialMockServer struct {
	alertpb.UnimplementedAlertApiServer
	first     *alertpb.ListAlertsResponse
	secondErr error
	calls     int
}

func (s *sequentialMockServer) ListAlerts(_ context.Context, _ *alertpb.ListAlertsRequest) (*alertpb.ListAlertsResponse, error) {
	s.calls++
	if s.calls == 1 {
		return s.first, nil
	}
	return nil, s.secondErr
}
