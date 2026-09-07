package opcua

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// fakeSubscription stands in for the *opcua.Subscription that CreateSubscription hands back,
// recording whether it was cancelled.
type fakeSubscription struct {
	res *ua.CreateMonitoredItemsResponse
	err error

	cancels int
	// cancelCtxErr is the state of the ctx Cancel was called with, which has to be live even
	// when the ctx that got us here has already expired.
	cancelCtxErr error
}

func (f *fakeSubscription) Monitor(_ context.Context, _ ua.TimestampsToReturn, _ ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.res, nil
}

func (f *fakeSubscription) Cancel(ctx context.Context) error {
	f.cancels++
	f.cancelCtxErr = ctx.Err()
	return nil
}

func oneResult(code ua.StatusCode) *ua.CreateMonitoredItemsResponse {
	return &ua.CreateMonitoredItemsResponse{
		Results: []*ua.MonitoredItemCreateResult{{StatusCode: code}},
	}
}

// TestClient_monitorPoint_cancelsOrphans is the test for the failure that wedges a whole
// connection rather than a single point.
//
// By the time monitorPoint runs, the subscription is live on the server and registered with
// the client, and gopcua delivers notifications with a blocking send on one client-wide
// publish goroutine. A subscribe that bails out without cancelling leaves that subscription
// holding a channel nobody will ever read, which stops every device sharing the connection
// from receiving values. So every failure path has to cancel, and the only path that must not
// is the one that hands the subscription back.
func TestClient_monitorPoint_cancelsOrphans(t *testing.T) {
	tests := []struct {
		name string
		sub  *fakeSubscription
		// wantErrIs, when set, must match the returned error through errors.Is. The status
		// code has to survive as a code rather than a string, or the caller cannot tell a
		// failure worth retrying from one that never will be.
		wantErrIs   error
		wantCancels int
	}{
		{
			name:        "Monitor fails",
			sub:         &fakeSubscription{err: ua.StatusBadTimeout},
			wantErrIs:   ua.StatusBadTimeout,
			wantCancels: 1,
		},
		{
			name:        "no results",
			sub:         &fakeSubscription{res: &ua.CreateMonitoredItemsResponse{}},
			wantErrIs:   errUnexpectedResults,
			wantCancels: 1,
		},
		{
			name: "more results than items",
			sub: &fakeSubscription{res: &ua.CreateMonitoredItemsResponse{
				Results: []*ua.MonitoredItemCreateResult{{StatusCode: ua.StatusOK}, {StatusCode: ua.StatusOK}},
			}},
			wantErrIs:   errUnexpectedResults,
			wantCancels: 1,
		},
		{
			name:        "bad result status",
			sub:         &fakeSubscription{res: oneResult(ua.StatusBadNodeIDUnknown)},
			wantErrIs:   ua.StatusBadNodeIDUnknown,
			wantCancels: 1,
		},
		{
			name:        "success keeps the subscription",
			sub:         &fakeSubscription{res: oneResult(ua.StatusOK)},
			wantCancels: 0,
		},
		{
			// a Good code carrying the Overflow info bit is still a successful monitor
			name:        "good result with info bits keeps the subscription",
			sub:         &fakeSubscription{res: oneResult(ua.StatusCode(0x480))},
			wantCancels: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t)
			err := c.monitorPoint(t.Context(), tt.sub, mustParseNodeID("ns=2;s=Tag1"))

			if tt.wantErrIs == nil {
				if err != nil {
					t.Errorf("monitorPoint() = %v, want nil", err)
				}
			} else if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("monitorPoint() = %v, want it to match %v", err, tt.wantErrIs)
			}
			if tt.sub.cancels != tt.wantCancels {
				t.Errorf("Cancel called %d times, want %d", tt.sub.cancels, tt.wantCancels)
			}
		})
	}
}

// TestClient_monitorPoint_cancelsAfterDeadline checks the orphan is still cancelled when the
// ctx that got us here has already expired, which is the case the cleanup exists for: a
// Monitor that timed out may well have created the item on the server anyway, and Cancel on a
// dead ctx would return without sending anything and leak it.
func TestClient_monitorPoint_cancelsAfterDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	c := newTestClient(t)
	sub := &fakeSubscription{err: ua.StatusBadTimeout}
	if err := c.monitorPoint(ctx, sub, mustParseNodeID("ns=2;s=Tag1")); err == nil {
		t.Fatal("monitorPoint() = nil, want an error")
	}
	if sub.cancels != 1 {
		t.Fatalf("Cancel called %d times, want 1", sub.cancels)
	}
	if sub.cancelCtxErr != nil {
		t.Errorf("Cancel called with a ctx already in state %v, want a live one", sub.cancelCtxErr)
	}
}

// TestNewClient_subGateDefaulted checks a Conn assembled in code rather than parsed still gets
// a usable gate. A weight of zero would make every Subscribe block forever.
func TestNewClient_subGateDefaulted(t *testing.T) {
	conn := config.Conn{
		SubscriptionInterval: &jsontypes.Duration{Duration: testInterval},
		SamplingInterval:     &jsontypes.Duration{Duration: testInterval},
		RequestTimeout:       &jsontypes.Duration{Duration: testInterval},
		// MaxConcurrentSubscribes deliberately left at its zero value
	}
	c := NewClient(nil, zaptest.NewLogger(t), conn)
	if !c.subGate.TryAcquire(1) {
		t.Fatal("subGate would not admit a single subscribe, so the weight defaulted to zero")
	}
	c.subGate.Release(1)
	if want := 3 * testInterval; c.subTimeout != want {
		t.Errorf("subTimeout = %s, want %s", c.subTimeout, want)
	}
}

// testInterval is an arbitrary duration for a Conn built in a test.
const testInterval = 5 * time.Second

func newTestClient(t *testing.T) *Client {
	t.Helper()
	return &Client{logger: zaptest.NewLogger(t)}
}
