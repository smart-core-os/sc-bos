package paxton

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// setupController creates an EventsController backed by a test HTTP server.
// batches define the sequence of event slices returned per API call to /api/v1/events;
// once exhausted, subsequent calls return an empty slice.
func setupController(t *testing.T, batches [][]Event) (*EventsController, func()) {
	t.Helper()
	var callCount atomic.Int32

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/authorization/tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuthResponse{
			AccessToken:    "test-token",
			ExpiryDatetime: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		n := int(callCount.Add(1)) - 1
		w.Header().Set("Content-Type", "application/json")
		if n >= len(batches) {
			_ = json.NewEncoder(w).Encode([]Event{})
			return
		}
		_ = json.NewEncoder(w).Encode(batches[n])
	})

	srv := httptest.NewServer(mux)

	cli := retryablehttp.NewClient()
	cli.RetryMax = 0
	cli.Logger = nil

	d := &Driver{
		client: NewClient(cli, zap.NewNop(), config.Root{
			BaseUrl: srv.URL,
			Auth:    config.Auth{},
		}, nil),
		logger: zap.NewNop(),
	}

	return NewEventsController(d), srv.Close
}

// makeEvents returns n events in descending time order (newest first),
// matching the orderBy=eventTime DESC ordering the Paxton API returns.
func makeEvents(n int, newest time.Time) []Event {
	events := make([]Event, n)
	for i := range events {
		events[i] = Event{
			ID:               i + 1,
			EventTime:        newest.Add(-time.Duration(i) * time.Second),
			EventDescription: "Door opened",
			DeviceName:       "Door 1",
		}
	}
	return events
}

func TestListSecurityEvents_Empty(t *testing.T) {
	ec, cleanup := setupController(t, [][]Event{{}})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.SecurityEvents)
	assert.Empty(t, resp.NextPageToken)
}

func TestListSecurityEvents_FewerThanDefaultPageSize(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ec, cleanup := setupController(t, [][]Event{makeEvents(10, now)})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 10)
	// NextPageToken is always set after a non-empty batch; the caller discovers
	// end-of-pages by receiving an empty response on the next call.
	assert.NotEmpty(t, resp.NextPageToken)
}

func TestListSecurityEvents_DefaultPageSize_Is50(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ec, cleanup := setupController(t, [][]Event{makeEvents(60, now)})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 50)
	assert.NotEmpty(t, resp.NextPageToken)
}

func TestListSecurityEvents_ZeroPageSizeDefaultsTo50(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ec, cleanup := setupController(t, [][]Event{makeEvents(60, now)})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{PageSize: 0})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 50)
}

func TestListSecurityEvents_CustomPageSize(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ec, cleanup := setupController(t, [][]Event{makeEvents(20, now)})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 10)
	assert.NotEmpty(t, resp.NextPageToken)
}

func TestListSecurityEvents_MultipleAPICallsToFillPage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	batch1 := makeEvents(30, now)
	// batch2 starts from just before the oldest event in batch1
	batch2 := makeEvents(30, batch1[len(batch1)-1].EventTime.Add(-time.Second))

	ec, cleanup := setupController(t, [][]Event{batch1, batch2})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 50)
	assert.NotEmpty(t, resp.NextPageToken)
}

func TestListSecurityEvents_NextPageTokenIsOldestReturnedEventTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	events := makeEvents(60, now)

	ec, cleanup := setupController(t, [][]Event{events})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)

	// The token must be the oldest event actually returned (the 50th of 60), not the
	// oldest event fetched (the 60th); otherwise events 51-60 would be skipped next page.
	require.Len(t, resp.SecurityEvents, 50)
	expectedToken := events[49].EventTime.Format(time.RFC3339)
	assert.Equal(t, expectedToken, resp.NextPageToken)
}

func TestListSecurityEvents_DoesNotSplitEventsSharingABoundarySecond(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	events := makeEvents(60, now)
	// Force the 51st event (index 50) to share the 50th event's (index 49) second, so
	// it sits exactly at the page boundary. The second-granularity API filter would skip
	// it on the next page, so it must be drained onto this page instead.
	events[50].EventTime = events[49].EventTime

	ec, cleanup := setupController(t, [][]Event{events})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)

	// 50 up to the boundary, plus the boundary-second straggler.
	assert.Len(t, resp.SecurityEvents, 51)
	assert.Equal(t, events[49].EventTime.Format(time.RFC3339), resp.NextPageToken)
}

func TestListSecurityEvents_PageTokenContinuation(t *testing.T) {
	cutoff := time.Now().Add(-30 * time.Minute).UTC().Truncate(time.Second)
	// events older than the cutoff, simulating a second page
	secondPage := makeEvents(10, cutoff.Add(-time.Second))

	ec, cleanup := setupController(t, [][]Event{secondPage})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{
		PageToken: cutoff.Format(time.RFC3339),
	})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 10)
}

func TestListSecurityEvents_InvalidPageToken(t *testing.T) {
	ec, cleanup := setupController(t, nil)
	defer cleanup()

	_, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{
		PageToken: "not-a-valid-time",
	})
	require.Error(t, err)
	st, ok := grpcstatus.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestListSecurityEvents_InfiniteLoopGuard(t *testing.T) {
	// All events share the same timestamp; the oldest event's time equals the next timeFilter,
	// which would loop forever without the guard.
	sameTime := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	events := []Event{
		{ID: 1, EventTime: sameTime, EventDescription: "Door opened"},
		{ID: 2, EventTime: sameTime, EventDescription: "Door opened"},
		{ID: 3, EventTime: sameTime, EventDescription: "Door opened"},
	}

	ec, cleanup := setupController(t, [][]Event{events})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.SecurityEvents, 3)
}

func TestListSecurityEvents_EventFieldsAreMapped(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	events := []Event{
		{
			ID:               42,
			EventTime:        now,
			EventDescription: "Access granted",
			FirstName:        "Jane",
			Surname:          "Doe",
			Priority:         5,
		},
	}

	ec, cleanup := setupController(t, [][]Event{events})
	defer cleanup()

	resp, err := ec.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.SecurityEvents, 1)

	se := resp.SecurityEvents[0]
	assert.Equal(t, "42", se.Id)
	assert.Equal(t, "Jane Doe - Access granted", se.Description)
	assert.Equal(t, int32(5), se.Priority)
	assert.Equal(t, now.Unix(), se.SecurityEventTime.AsTime().Unix())
}
