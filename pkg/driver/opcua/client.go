package opcua

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
)

// Client wraps an OPC UA client connection and manages subscriptions to variable nodes.
type Client struct {
	client *opcua.Client
	logger *zap.Logger

	interval         time.Duration // publishing interval for the subscription
	samplingInterval time.Duration // how often the server samples each monitored item
	queueSize        uint32        // server-side queue depth per monitored item
	itemsPerRequest  int           // how many monitored items go in one CreateMonitoredItems

	// subGate bounds how many requests are in flight at once across every device sharing
	// this client. A semaphore rather than a channel because Acquire is context-aware, and
	// deliberately not errgroup.SetLimit, which bounds live goroutines: the pump goroutines
	// live as long as the config does, so a limited group would leave most points
	// permanently unsubscribed.
	subGate *semaphore.Weighted
	// reqTimeout bounds one request to the server. Derived from the configured request timeout
	// rather than configured separately: a standalone value could be set below it and
	// pre-empt gopcua's own StatusBadTimeout, hiding the reason the subscribe failed.
	//
	// Per request rather than per subscribe or per device: a device's points are monitored in
	// as many requests as they need, and a whole-operation deadline scaled by that count would
	// give a 500-point device tens of minutes, long enough that a wedged request looks like a
	// working one. What bounds the whole attempt is the device's own ctx.
	reqTimeout time.Duration

	// newSub creates the underlying subscription. A field so that NewSubscription can be
	// tested at all: c.client.Subscribe needs a live *opcua.Client and a real server.
	newSub subFactory
}

// orphanCancelTimeout bounds the tear-down of a subscription we are not going to use.
const orphanCancelTimeout = 5 * time.Second

// notifyBuffer is how many notifications a device's channel holds before a send blocks.
//
// Buffered, but the send stays blocking: gopcua delivers with a blocking send from the one
// publish goroutine the whole client shares, so a device that stops reading has to be felt as
// back-pressure. Dropping instead would lose values silently, which is the failure this driver
// is least able to detect.
//
// The buffer only decouples dispatching one publishing cycle from the next request going out.
// Deliberately not scaled by item count: the channel carries whole NotificationData messages,
// and a device's items arrive as one DataChangeNotification per cycle unless the cycle exceeds
// MaxNotificationsPerPublish, which gopcua leaves at 10000.
const notifyBuffer = 4

// NewClient creates a new Client wrapper around an OPC UA client connection.
// The monitoring parameters are taken from conn, which ParseConfig has already defaulted.
func NewClient(client *opcua.Client, logger *zap.Logger, conn config.Conn) *Client {
	maxConcurrent := conn.MaxConcurrentSubscribes
	if maxConcurrent < 1 {
		// a weight of zero would make Acquire block forever. ParseConfig defaults this, so
		// only a Conn assembled in code reaches here.
		maxConcurrent = config.DefaultMaxConcurrentSubscribes
	}
	itemsPerRequest := conn.MaxMonitoredItemsPerRequest
	if itemsPerRequest < 1 {
		// a chunk of no items would never monitor anything, for the same reason
		itemsPerRequest = config.DefaultMaxMonitoredItemsPerRequest
	}
	c := &Client{
		client:           client,
		interval:         conn.SubscriptionInterval.Duration,
		samplingInterval: conn.SamplingInterval.Duration,
		queueSize:        conn.QueueSize,
		itemsPerRequest:  itemsPerRequest,
		logger:           logger,
		subGate:          semaphore.NewWeighted(int64(maxConcurrent)),
		reqTimeout:       3 * conn.RequestTimeout.Duration,
	}
	c.newSub = func(ctx context.Context, params *opcua.SubscriptionParameters, notifyCh chan<- *opcua.PublishNotificationData) (subscription, error) {
		sub, err := client.Subscribe(ctx, params, notifyCh)
		if err != nil {
			// a literal nil, not sub: returning the concrete *opcua.Subscription would make a
			// non-nil interface holding a nil pointer, and every later method call panic
			return nil, err
		}
		return sub, nil
	}
	return c
}

// subFactory creates the subscription a device monitors its points on.
// The seam that lets a test drive NewSubscription without an OPC UA server.
type subFactory func(ctx context.Context, params *opcua.SubscriptionParameters,
	notifyCh chan<- *opcua.PublishNotificationData) (subscription, error)

// subscription is the part of *opcua.Subscription that a pointSub needs.
// Narrow enough that a test can supply a fake and check the subscription is torn down on
// every failure path, which is the difference between a failed subscribe and a wedged client.
type subscription interface {
	Monitor(ctx context.Context, ts ua.TimestampsToReturn, items ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error)
	Cancel(ctx context.Context) error
}

// monitorResult is the outcome of monitoring one node. Err wraps the ua.StatusCode the server
// returned for that item, so subscribeErrIsPermanent can classify each node on its own.
type monitorResult struct {
	NodeId *ua.NodeID
	Err    error
}

// NewSubscription creates one OPC UA subscription with no monitored items on it, for one
// device to monitor all of its points on. The caller owns the result and must Cancel it once
// it stops reading Notifications, even when Monitor never succeeded for a single node.
//
// One subscription per device rather than per point because the subscription count is what
// bounds throughput: gopcua keeps a single PublishRequest in flight per client, on one
// goroutine, and a PublishResponse names one subscription, so every subscription costs a
// serialised round trip per publishing cycle.
func (c *Client) NewSubscription(ctx context.Context) (pointSubscription, error) {
	// acquire before starting the clock, queuing for a turn is not the server being slow
	if err := c.subGate.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer c.subGate.Release(1)

	// safe to cancel on return: gopcua uses this ctx only for the CreateSubscription request,
	// the Subscription it hands back is driven by the client's own publish loop
	ctx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	notify := make(chan *opcua.PublishNotificationData, notifyBuffer)
	sub, err := c.newSub(ctx, &opcua.SubscriptionParameters{Interval: c.interval}, notify)
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return &pointSub{
		client:     c,
		sub:        sub,
		notify:     notify,
		nextHandle: 1,
		nodes:      make(map[uint32]*ua.NodeID),
	}, nil
}

// pointSub is one device's subscription: the notification stream, the monitored items on it,
// and the map from the client handle on a notification back to the node it came from.
//
// A notification names the item that produced it only by the ClientHandle we chose when we
// created it (Part 4, 7.21), so with more than one item on a subscription that map is the only
// thing that says which node a value belongs to.
type pointSub struct {
	client *Client
	sub    subscription
	notify chan *opcua.PublishNotificationData

	mu         sync.RWMutex // handleEvent reads nodes while the straggler retry writes it
	nextHandle uint32       // unique within the subscription, starts at 1, never reused
	nodes      map[uint32]*ua.NodeID
}

// Notifications is the stream of everything the server publishes for this subscription.
func (s *pointSub) Notifications() <-chan *opcua.PublishNotificationData {
	return s.notify
}

// Node reports the node behind a notification's client handle, and whether we know it.
func (s *pointSub) Node(handle uint32) (*ua.NodeID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodeId, ok := s.nodes[handle]
	return nodeId, ok
}

// Monitor adds each node's value attribute to the subscription as a monitored item, in
// requests of at most the configured size, and reports an outcome for every node it was given.
//
// A total function over nodeIds by design: the caller has to be able to tell, for each node,
// whether it is delivering, worth retrying, or hopeless, and it needs the successes by name to
// clear their faults. An error carrying only the failures would leave the successes to a set
// difference.
//
// Safe to call again on a live subscription, which is how a node that failed transiently
// rejoins its siblings without tearing down items that are already delivering.
func (s *pointSub) Monitor(ctx context.Context, nodeIds ...*ua.NodeID) []monitorResult {
	results := make([]monitorResult, 0, len(nodeIds))
	for start := 0; start < len(nodeIds); start += s.client.itemsPerRequest {
		end := min(start+s.client.itemsPerRequest, len(nodeIds))
		chunk, reqErr := s.monitorChunk(ctx, nodeIds[start:end])
		results = append(results, chunk...)
		if reqErr != nil {
			// the request itself failed, which says the link is down rather than that an item
			// was refused. The remaining chunks would only take their turn through the gate to
			// learn the same thing, so report their nodes with the same error and stop.
			for _, nodeId := range nodeIds[end:] {
				results = append(results, monitorResult{NodeId: nodeId, Err: monitorErr(nodeId, reqErr)})
			}
			break
		}
	}
	return results
}

// monitorChunk creates monitored items for one request's worth of nodes. It always reports an
// outcome for every node in the chunk; a non-nil error is a request-level failure, which says
// nothing about the individual items and means the caller should stop.
func (s *pointSub) monitorChunk(ctx context.Context, nodeIds []*ua.NodeID) ([]monitorResult, error) {
	c := s.client
	// acquire before starting the clock, queuing for a turn is not the server being slow.
	// Per request rather than per device deliberately: a device holding a slot for its whole
	// point list would block every other device behind it for minutes.
	if err := c.subGate.Acquire(ctx, 1); err != nil {
		return failAll(nodeIds, err), err
	}
	defer c.subGate.Release(1)

	ctx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	reqs := make([]*ua.MonitoredItemCreateRequest, len(nodeIds))
	handles := make([]uint32, len(nodeIds))
	s.mu.Lock()
	for i, nodeId := range nodeIds {
		handles[i] = s.nextHandle
		s.nextHandle++
		s.nodes[handles[i]] = nodeId
		reqs[i] = c.itemRequest(nodeId, handles[i])
	}
	s.mu.Unlock()

	res, err := s.sub.Monitor(ctx, ua.TimestampsToReturnNeither, reqs...)
	if err != nil {
		// the handles stay in the map. We do not know whether the server created the items - a
		// timeout is exactly the case where it may well have - and keeping them means an item
		// it created behind our back still demultiplexes to the right node. A handle for an
		// item that does not exist costs nothing, since nothing ever reports under it.
		return failAll(nodeIds, err), err
	}
	// only the long direction is worth guarding: gopcua indexes res.Results by request
	// position without checking its length (subscription.go:168-177), so a response carrying
	// fewer results than we asked about panics inside the library before we see it. Reported
	// upstream rather than vendored around.
	if len(res.Results) != len(reqs) {
		c.logger.Warn("unexpected number of monitored item results",
			zap.Int("requested", len(reqs)), zap.Int("returned", len(res.Results)),
			zap.Any("results", res.Results))
		countErr := fmt.Errorf("%w, asked about %d and got %d", errUnexpectedResults, len(reqs), len(res.Results))
		return failAll(nodeIds, countErr), countErr
	}

	out := make([]monitorResult, len(nodeIds))
	var refused []uint32
	for i, nodeId := range nodeIds {
		item := res.Results[i]
		if code := item.StatusCode; statusIsBad(code) {
			// wrap the code rather than rendering it: whether retrying could ever work is
			// decided from the code, and a string loses the only thing that says so
			out[i] = monitorResult{NodeId: nodeId, Err: monitorErr(nodeId, code)}
			refused = append(refused, handles[i])
			continue
		}
		out[i] = monitorResult{NodeId: nodeId}
		c.warnIfRevised(nodeId, item)
	}
	if len(refused) > 0 {
		// a Bad item result guarantees the server created no item for it (Part 4, 5.13.2.2),
		// so its handle could only ever misattribute someone else's value. Drop it.
		s.mu.Lock()
		for _, handle := range refused {
			delete(s.nodes, handle)
		}
		s.mu.Unlock()
	}
	return out, nil
}

// Cancel tears down the subscription, draining notifications until it has.
//
// Draining is not tidiness. Subscription.Cancel takes the client's subscription mutex and,
// when this was the last subscription, signals the single client-wide publish goroutine to
// pause. That goroutine may be parked in a blocking send on our channel, so a Cancel that
// does not read waits a whole publishing interval while holding the lock every other device's
// subscribe needs.
func (s *pointSub) Cancel(ctx context.Context) {
	// WithoutCancel because the case this exists for is a ctx that has already expired, where
	// Cancel would return without sending anything and leak the subscription anyway
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), orphanCancelTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- s.sub.Cancel(ctx) }()
	for {
		select {
		case err := <-done:
			if err != nil {
				s.client.logger.Warn("failed to cancel a subscription", zap.Error(err))
			}
			return
		case <-s.notify:
			// discard: nobody is going to dispatch these, and holding up the publish
			// goroutine is the thing this exists to avoid
		}
	}
}

// itemRequest is the CreateMonitoredItems entry for one node, stamped with the client handle
// its notifications will come back under.
func (c *Client) itemRequest(nodeId *ua.NodeID, handle uint32) *ua.MonitoredItemCreateRequest {
	// deliberately not NewMonitoredItemCreateRequestWithDefaults: its 10-deep queue sampled as
	// fast as the server allows overflows on every publish cycle for a fast-sampling server,
	// and the server flags that on every value it sends us
	return &ua.MonitoredItemCreateRequest{
		ItemToMonitor: &ua.ReadValueID{
			NodeID:       nodeId,
			AttributeID:  ua.AttributeIDValue,
			DataEncoding: &ua.QualifiedName{},
		},
		MonitoringMode: ua.MonitoringModeReporting,
		RequestedParameters: &ua.MonitoringParameters{
			ClientHandle:     handle,
			DiscardOldest:    true,
			QueueSize:        c.queueSize,
			SamplingInterval: float64(c.samplingInterval.Milliseconds()),
		},
	}
}

// monitorErr renders a node's failure to be monitored, keeping a status code inside it
// matchable through errors.As so subscribeErrIsPermanent can still classify it.
func monitorErr(nodeId *ua.NodeID, err error) error {
	return fmt.Errorf("monitor %s: %w", nodeId, err)
}

// failAll reports the same failure for every node, for the cases where the request failed and
// so says nothing about the individual items.
func failAll(nodeIds []*ua.NodeID, err error) []monitorResult {
	out := make([]monitorResult, len(nodeIds))
	for i, nodeId := range nodeIds {
		out[i] = monitorResult{NodeId: nodeId, Err: monitorErr(nodeId, err)}
	}
	return out
}

// warnIfRevised reports monitoring parameters the server declined to honour.
// A server is free to revise what we ask for, typically clamping a sampling interval to its
// MinSupportedSampleRate or a queue to the depth it is willing to hold, and it tells us what
// it settled on rather than failing. That revision is the authoritative version of the
// config-time warnings in config.Conn.MonitoringWarnings, so it is worth surfacing: a queue
// revised down below a publishing cycle's worth of samples is exactly the setup that makes
// the server flag every value with the Overflow info bit.
func (c *Client) warnIfRevised(nodeId *ua.NodeID, res *ua.MonitoredItemCreateResult) {
	// floatEqual rather than !=: the revised interval is a float off the wire, and a server
	// echoing back what we asked for should not read as a revision
	requested := float64(c.samplingInterval.Milliseconds())
	if !floatEqual(res.RevisedSamplingInterval, requested) {
		c.logger.Warn("server revised the sampling interval",
			zap.Stringer("node", nodeId),
			zap.Float64("requestedMs", requested),
			zap.Float64("revisedMs", res.RevisedSamplingInterval))
	}
	if res.RevisedQueueSize != c.queueSize {
		c.logger.Warn("server revised the queue size",
			zap.Stringer("node", nodeId),
			zap.Uint32("requested", c.queueSize),
			zap.Uint32("revised", res.RevisedQueueSize))
	}
}
