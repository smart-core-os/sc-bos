package opcua

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// fakeSubscription stands in for the *opcua.Subscription that CreateSubscription hands back:
// it answers CreateMonitoredItems from a script and records what it was asked.
type fakeSubscription struct {
	mu sync.Mutex

	// statuses is the result status for a node id, StatusOK when absent.
	statuses map[string]ua.StatusCode
	// callErrs is the request-level error of each successive Monitor call, a nil entry meaning
	// the request itself succeeded. The last entry repeats once the script runs out.
	callErrs []error
	// resultCount, when non-zero, forces how many results the response carries however many
	// items were asked about, which no conformant server does but is the case we cannot tell
	// apart from a success.
	resultCount int
	// cancelWait, when non-nil, holds Cancel until it is closed. Stands in for
	// Subscription.Cancel waiting on the client's publish goroutine.
	cancelWait chan struct{}

	calls        [][]*ua.MonitoredItemCreateRequest // the items of each successive call
	timestamps   []ua.TimestampsToReturn            // what each call asked the server to stamp
	cancels      int
	cancelCtxErr error // the state of the ctx Cancel was called with
}

func (f *fakeSubscription) Monitor(_ context.Context, ts ua.TimestampsToReturn, items ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	call := len(f.calls)
	f.calls = append(f.calls, items)
	f.timestamps = append(f.timestamps, ts)
	if len(f.callErrs) > 0 {
		if call >= len(f.callErrs) {
			call = len(f.callErrs) - 1
		}
		if err := f.callErrs[call]; err != nil {
			return nil, err
		}
	}

	count := len(items)
	if f.resultCount > 0 {
		count = f.resultCount
	}
	res := &ua.CreateMonitoredItemsResponse{}
	for i := range count {
		result := &ua.MonitoredItemCreateResult{StatusCode: ua.StatusOK}
		if i < len(items) {
			if code, ok := f.statuses[items[i].ItemToMonitor.NodeID.String()]; ok {
				result.StatusCode = code
			}
			// echo the request back, so warnIfRevised has nothing to say and the logs stay
			// about whatever the test is actually doing
			result.RevisedSamplingInterval = items[i].RequestedParameters.SamplingInterval
			result.RevisedQueueSize = items[i].RequestedParameters.QueueSize
		}
		res.Results = append(res.Results, result)
	}
	return res, nil
}

func (f *fakeSubscription) Cancel(ctx context.Context) error {
	f.mu.Lock()
	f.cancels++
	f.cancelCtxErr = ctx.Err()
	wait := f.cancelWait
	f.mu.Unlock()

	if wait != nil {
		select {
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// callItems returns the node ids of each successive Monitor call.
func (f *fakeSubscription) callItems() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, 0, len(f.calls))
	for _, call := range f.calls {
		nodeIds := make([]string, 0, len(call))
		for _, item := range call {
			nodeIds = append(nodeIds, item.ItemToMonitor.NodeID.String())
		}
		out = append(out, nodeIds)
	}
	return out
}

func (f *fakeSubscription) cancelCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancels
}

// fakeFactory stands in for the CreateSubscription request, counting the subscriptions created
// and keeping the notification channel each was given.
type fakeFactory struct {
	mu     sync.Mutex
	sub    subscription
	err    error
	calls  int
	notify chan<- *opcua.PublishNotificationData
}

func (f *fakeFactory) create(_ context.Context, _ *opcua.SubscriptionParameters, notifyCh chan<- *opcua.PublishNotificationData) (subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.notify = notifyCh
	if f.err != nil {
		return nil, f.err
	}
	return f.sub, nil
}

func (f *fakeFactory) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// testInterval is an arbitrary duration for a Conn built in a test.
const testInterval = 5 * time.Second

// newTestClient builds a Client whose subscriptions come from newSub, with itemsPerRequest as
// the chunk size. Pass 0 for itemsPerRequest to take the configured default.
func newTestClient(t *testing.T, newSub subFactory, itemsPerRequest int) *Client {
	t.Helper()
	c := NewClient(nil, zaptest.NewLogger(t), config.Conn{
		SubscriptionInterval:        &jsontypes.Duration{Duration: testInterval},
		SamplingInterval:            &jsontypes.Duration{Duration: testInterval},
		RequestTimeout:              &jsontypes.Duration{Duration: testInterval},
		QueueSize:                   1,
		MaxConcurrentSubscribes:     config.DefaultMaxConcurrentSubscribes,
		MaxMonitoredItemsPerRequest: itemsPerRequest,
	})
	if newSub != nil {
		c.newSub = newSub
	}
	return c
}

// newTestSub creates a subscription on a client wired to sub, as a *pointSub so a test can
// read the handle map directly.
func newTestSub(t *testing.T, sub *fakeSubscription, itemsPerRequest int) (*pointSub, *fakeFactory) {
	t.Helper()
	factory := &fakeFactory{sub: sub}
	c := newTestClient(t, factory.create, itemsPerRequest)
	created, err := c.NewSubscription(t.Context())
	if err != nil {
		t.Fatalf("NewSubscription() = %v", err)
	}
	return created.(*pointSub), factory
}

func testNodeIds(ids ...string) []*ua.NodeID {
	out := make([]*ua.NodeID, 0, len(ids))
	for _, id := range ids {
		out = append(out, mustParseNodeID(id))
	}
	return out
}

// TestClient_NewSubscription checks the subscription is handed the buffered channel its device
// will read, and that nothing else about it is decided per point.
func TestClient_NewSubscription(t *testing.T) {
	sub := &fakeSubscription{}
	ps, factory := newTestSub(t, sub, 0)

	if got := factory.callCount(); got != 1 {
		t.Errorf("created %d subscriptions, want 1", got)
	}
	if got := cap(ps.notify); got != notifyBuffer {
		t.Errorf("notification channel capacity = %d, want %d", got, notifyBuffer)
	}
	if ps.Notifications() == nil {
		t.Error("Notifications() = nil, want the channel the server publishes to")
	}
	// no items until the device asks for them, so an unused subscription monitors nothing
	if len(sub.callItems()) != 0 {
		t.Errorf("Monitor called %d times before any node was asked for", len(sub.callItems()))
	}
}

// TestClient_NewSubscription_createFails checks a failed create reports a usable error and,
// crucially, a nil pointSubscription. The factory returns the interface, so a create that
// handed back its concrete *opcua.Subscription would produce a non-nil interface holding a nil
// pointer and panic on the first method call rather than here.
func TestClient_NewSubscription_createFails(t *testing.T) {
	factory := &fakeFactory{err: ua.StatusBadTooManySubscriptions}
	c := newTestClient(t, factory.create, 0)

	sub, err := c.NewSubscription(t.Context())
	if sub != nil {
		t.Errorf("NewSubscription() = %v, want nil", sub)
	}
	if !errors.Is(err, ua.StatusBadTooManySubscriptions) {
		t.Errorf("NewSubscription() error = %v, want it to match %v", err, ua.StatusBadTooManySubscriptions)
	}
	// the code has to survive as a code: the capacity codes are retryable, and a device that
	// classified this as permanent would never come up
	if subscribeErrIsPermanent(err) {
		t.Error("a subscription refused for capacity should be retryable")
	}
}

// TestPointSub_Monitor_perNodeResults is the property the whole design rests on: one result per
// node, in the order they were asked about, each carrying that node's own status code. A device
// asking about 50 points routinely gets a mix of codes needing opposite treatment, and it can
// only tell them apart if the code reaches it per node.
func TestPointSub_Monitor_perNodeResults(t *testing.T) {
	const good, missing, slow = "ns=2;s=Good", "ns=2;s=Missing", "ns=2;s=Slow"
	sub := &fakeSubscription{statuses: map[string]ua.StatusCode{
		missing: ua.StatusBadNodeIDUnknown,
		slow:    ua.StatusBadTimeout,
	}}
	ps, _ := newTestSub(t, sub, 0)

	results := ps.Monitor(t.Context(), testNodeIds(good, missing, slow)...)
	if len(results) != 3 {
		t.Fatalf("Monitor() returned %d results, want 3", len(results))
	}
	if got := results[0].NodeId.String(); got != good || results[0].Err != nil {
		t.Errorf("results[0] = {%s, %v}, want {%s, nil}", got, results[0].Err, good)
	}
	if got := results[1].NodeId.String(); got != missing {
		t.Errorf("results[1] node = %s, want %s", got, missing)
	}
	if !errors.Is(results[1].Err, ua.StatusBadNodeIDUnknown) {
		t.Errorf("results[1] err = %v, want it to match %v", results[1].Err, ua.StatusBadNodeIDUnknown)
	}
	if !subscribeErrIsPermanent(results[1].Err) {
		t.Error("an unknown node id should be classified permanent")
	}
	if !errors.Is(results[2].Err, ua.StatusBadTimeout) {
		t.Errorf("results[2] err = %v, want it to match %v", results[2].Err, ua.StatusBadTimeout)
	}
	if subscribeErrIsPermanent(results[2].Err) {
		t.Error("a timeout should be classified retryable")
	}
	// the fault text a device raises is this error, so it stays as it always read
	if want := "monitor " + missing + ": " + ua.StatusBadNodeIDUnknown.Error(); results[1].Err.Error() != want {
		t.Errorf("results[1] err = %q, want %q", results[1].Err.Error(), want)
	}

	// handles: the good node demultiplexes, and the refused ones are absent. A Bad item result
	// means the server created no item (Part 4, 5.13.2.2), so a handle left behind for one
	// could only ever misattribute somebody else's value.
	if nodeId, ok := ps.Node(1); !ok || nodeId.String() != good {
		t.Errorf("Node(1) = %v, %v, want %s", nodeId, ok, good)
	}
	if nodeId, ok := ps.Node(2); ok {
		t.Errorf("Node(2) = %v, want no handle for a node the server refused", nodeId)
	}
	if nodeId, ok := ps.Node(3); ok {
		t.Errorf("Node(3) = %v, want no handle for a node the server refused", nodeId)
	}
	// each item asks for the value attribute with no timestamps, as it always has
	if sub.timestamps[0] != ua.TimestampsToReturnNeither {
		t.Errorf("asked for timestamps %v, want %v", sub.timestamps[0], ua.TimestampsToReturnNeither)
	}
	if got := sub.calls[0][0].ItemToMonitor.AttributeID; got != ua.AttributeIDValue {
		t.Errorf("monitored attribute = %v, want %v", got, ua.AttributeIDValue)
	}
}

// TestPointSub_Monitor_keepsHandlesOnRequestFailure covers the other half of the handle policy.
// A request that failed says nothing about the items: the server may well have created them,
// a timeout being exactly that case, so keeping the handles means anything it did create still
// demultiplexes to the right node.
func TestPointSub_Monitor_keepsHandlesOnRequestFailure(t *testing.T) {
	const nodeA, nodeB = "ns=2;s=A", "ns=2;s=B"
	sub := &fakeSubscription{callErrs: []error{ua.StatusBadTimeout}}
	ps, _ := newTestSub(t, sub, 0)

	results := ps.Monitor(t.Context(), testNodeIds(nodeA, nodeB)...)
	if len(results) != 2 {
		t.Fatalf("Monitor() returned %d results, want 2", len(results))
	}
	for i, res := range results {
		if !errors.Is(res.Err, ua.StatusBadTimeout) {
			t.Errorf("results[%d] err = %v, want it to match %v", i, res.Err, ua.StatusBadTimeout)
		}
	}
	for handle, want := range map[uint32]string{1: nodeA, 2: nodeB} {
		if nodeId, ok := ps.Node(handle); !ok || nodeId.String() != want {
			t.Errorf("Node(%d) = %v, %v, want %s", handle, nodeId, ok, want)
		}
	}
}

// TestPointSub_Monitor_chunks checks the point list is split into requests of at most the
// configured size, and that splitting it changes nothing about the subscription: however many
// requests it takes, there is still exactly one.
func TestPointSub_Monitor_chunks(t *testing.T) {
	all := []string{"ns=2;s=1", "ns=2;s=2", "ns=2;s=3", "ns=2;s=4", "ns=2;s=5", "ns=2;s=6", "ns=2;s=7"}
	tests := []struct {
		name            string
		itemsPerRequest int
		want            [][]string
	}{
		{
			name:            "seven at three",
			itemsPerRequest: 3,
			want:            [][]string{all[0:3], all[3:6], all[6:7]},
		},
		{
			// one item per request is a legitimate thing to configure for a server that
			// refuses batches, and it is still not one subscription per point
			name:            "one at a time",
			itemsPerRequest: 1,
			want:            [][]string{all[0:1], all[1:2], all[2:3], all[3:4], all[4:5], all[5:6], all[6:7]},
		},
		{
			name:            "all in one request",
			itemsPerRequest: 50,
			want:            [][]string{all},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &fakeSubscription{}
			ps, factory := newTestSub(t, sub, tt.itemsPerRequest)

			results := ps.Monitor(t.Context(), testNodeIds(all...)...)
			if len(results) != len(all) {
				t.Fatalf("Monitor() returned %d results, want %d", len(results), len(all))
			}
			for i, res := range results {
				if res.Err != nil {
					t.Errorf("results[%d] = %v, want nil", i, res.Err)
				}
				if got := res.NodeId.String(); got != all[i] {
					t.Errorf("results[%d] node = %s, want %s", i, got, all[i])
				}
			}
			got := sub.callItems()
			if len(got) != len(tt.want) {
				t.Fatalf("Monitor made %d requests, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if len(got[i]) != len(tt.want[i]) {
					t.Errorf("request %d carried %v, want %v", i, got[i], tt.want[i])
					continue
				}
				for j := range tt.want[i] {
					if got[i][j] != tt.want[i][j] {
						t.Errorf("request %d carried %v, want %v", i, got[i], tt.want[i])
						break
					}
				}
			}
			if calls := factory.callCount(); calls != 1 {
				t.Errorf("created %d subscriptions, want 1 however many requests the items took", calls)
			}
			// every handle is distinct and resolves, which is what makes one subscription per
			// device work at all
			for handle := uint32(1); handle <= uint32(len(all)); handle++ {
				if nodeId, ok := ps.Node(handle); !ok || nodeId.String() != all[handle-1] {
					t.Errorf("Node(%d) = %v, %v, want %s", handle, nodeId, ok, all[handle-1])
				}
			}
		})
	}
}

// TestPointSub_Monitor_stopsAfterRequestFailure checks a failed request abandons the rest.
// A request-level failure says the link is down rather than that an item was refused, so the
// remaining chunks would only take their turn through the gate to learn the same thing - and
// their nodes still need reporting, or the device would think they were fine.
func TestPointSub_Monitor_stopsAfterRequestFailure(t *testing.T) {
	all := []string{"ns=2;s=1", "ns=2;s=2", "ns=2;s=3", "ns=2;s=4", "ns=2;s=5", "ns=2;s=6"}
	sub := &fakeSubscription{callErrs: []error{nil, ua.StatusBadTimeout, nil}}
	ps, _ := newTestSub(t, sub, 2)

	results := ps.Monitor(t.Context(), testNodeIds(all...)...)
	if len(results) != len(all) {
		t.Fatalf("Monitor() returned %d results, want one per node (%d)", len(results), len(all))
	}
	for i, res := range results {
		if got := res.NodeId.String(); got != all[i] {
			t.Errorf("results[%d] node = %s, want %s", i, got, all[i])
		}
		if i < 2 {
			if res.Err != nil {
				t.Errorf("results[%d] = %v, want nil for a node monitored before the failure", i, res.Err)
			}
			continue
		}
		if !errors.Is(res.Err, ua.StatusBadTimeout) {
			t.Errorf("results[%d] err = %v, want it to match %v", i, res.Err, ua.StatusBadTimeout)
		}
	}
	if got := len(sub.callItems()); got != 2 {
		t.Errorf("Monitor made %d requests, want it to stop after the one that failed (2)", got)
	}
}

// TestPointSub_Monitor_resultCountMismatch covers a response we cannot interpret. Only the
// long direction is reachable: gopcua indexes the results by request position without checking
// the length, so a short response panics inside the library before we see it.
func TestPointSub_Monitor_resultCountMismatch(t *testing.T) {
	const nodeA, nodeB = "ns=2;s=A", "ns=2;s=B"
	sub := &fakeSubscription{resultCount: 3}
	ps, _ := newTestSub(t, sub, 0)

	results := ps.Monitor(t.Context(), testNodeIds(nodeA, nodeB)...)
	if len(results) != 2 {
		t.Fatalf("Monitor() returned %d results, want 2", len(results))
	}
	for i, res := range results {
		if !errors.Is(res.Err, errUnexpectedResults) {
			t.Errorf("results[%d] err = %v, want it to match %v", i, res.Err, errUnexpectedResults)
		}
		// no status code in it, so it falls to the default: retried rather than given up on
		if subscribeErrIsPermanent(res.Err) {
			t.Errorf("results[%d] err = %v, want it classified retryable", i, res.Err)
		}
	}
}

// TestPointSub_Cancel_liveCtx checks the tear-down still happens when the ctx that got us here
// has already expired, which is the case it exists for: a Monitor that timed out may well have
// created its items anyway, and Cancel on a dead ctx returns without sending anything.
func TestPointSub_Cancel_liveCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	sub := &fakeSubscription{}
	ps, _ := newTestSub(t, sub, 0)
	ps.Cancel(ctx)

	if got := sub.cancelCount(); got != 1 {
		t.Fatalf("Cancel called %d times, want 1", got)
	}
	if sub.cancelCtxErr != nil {
		t.Errorf("Cancel called with a ctx already in state %v, want a live one", sub.cancelCtxErr)
	}
}

// TestPointSub_Cancel_drains is the test for the failure that stalls a whole connection rather
// than one device.
//
// Subscription.Cancel takes the client's subscription mutex and, when this was the last
// subscription, waits for the single client-wide publish goroutine to pause. That goroutine
// delivers with a blocking send, so if it is parked on our channel and we do not read, Cancel
// waits a whole publishing interval holding the lock every other device's subscribe needs.
// Here the fake will not finish cancelling until the publisher has got unstuck, so a Cancel
// that does not drain never returns at all.
func TestPointSub_Cancel_drains(t *testing.T) {
	published := make(chan struct{})
	sub := &fakeSubscription{cancelWait: published}
	ps, _ := newTestSub(t, sub, 0)

	// more notifications than the buffer holds, so the publisher blocks part way through
	go func() {
		defer close(published)
		for range notifyBuffer * 3 {
			ps.notify <- &opcua.PublishNotificationData{}
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		ps.Cancel(t.Context())
	}()
	// well inside orphanCancelTimeout, which is what a Cancel that does not drain waits out
	// before giving up and leaking the subscription anyway
	select {
	case <-done:
	case <-time.After(orphanCancelTimeout / 5):
		t.Fatal("Cancel is waiting on a publisher blocked sending to the channel it should be draining")
	}
	select {
	case <-published:
	default:
		t.Error("Cancel returned with the publisher still blocked, so it gave up rather than draining")
	}
	if got := sub.cancelCount(); got != 1 {
		t.Errorf("Cancel called %d times, want 1", got)
	}
}

// TestNewClient_defaults checks a Conn assembled in code rather than parsed still gets usable
// limits. A gate weight of zero would make every request block forever, and a chunk of no
// items would never monitor anything.
func TestNewClient_defaults(t *testing.T) {
	conn := config.Conn{
		SubscriptionInterval: &jsontypes.Duration{Duration: testInterval},
		SamplingInterval:     &jsontypes.Duration{Duration: testInterval},
		RequestTimeout:       &jsontypes.Duration{Duration: testInterval},
		// MaxConcurrentSubscribes and MaxMonitoredItemsPerRequest deliberately left at zero
	}
	c := NewClient(nil, zaptest.NewLogger(t), conn)
	if !c.subGate.TryAcquire(1) {
		t.Fatal("subGate would not admit a single request, so the weight defaulted to zero")
	}
	c.subGate.Release(1)
	if want := 3 * testInterval; c.reqTimeout != want {
		t.Errorf("reqTimeout = %s, want %s", c.reqTimeout, want)
	}
	if want := config.DefaultMaxMonitoredItemsPerRequest; c.itemsPerRequest != want {
		t.Errorf("itemsPerRequest = %d, want %d", c.itemsPerRequest, want)
	}
}
