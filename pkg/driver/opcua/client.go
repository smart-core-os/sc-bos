package opcua

import (
	"context"
	"fmt"
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
	clientHandle     uint32

	// subGate bounds how many subscribes are in flight at once across every device sharing
	// this client. A semaphore rather than a channel because Acquire is context-aware, and
	// deliberately not errgroup.SetLimit, which bounds live goroutines: the pump goroutines
	// live as long as the config does, so a limited group would leave most points
	// permanently unsubscribed.
	subGate *semaphore.Weighted
	// subTimeout bounds one whole Subscribe call. Derived from the configured request timeout
	// rather than configured separately: a standalone value could be set below it and
	// pre-empt gopcua's own StatusBadTimeout, hiding the reason the subscribe failed. Three
	// requests' worth covers CreateSubscription plus CreateMonitoredItems with room to spare.
	subTimeout time.Duration
}

// orphanCancelTimeout bounds the tear-down of a subscription we are not going to use.
const orphanCancelTimeout = 5 * time.Second

// NewClient creates a new Client wrapper around an OPC UA client connection.
// The monitoring parameters are taken from conn, which ParseConfig has already defaulted.
func NewClient(client *opcua.Client, logger *zap.Logger, conn config.Conn) *Client {
	maxConcurrent := conn.MaxConcurrentSubscribes
	if maxConcurrent < 1 {
		// a weight of zero would make Acquire block forever. ParseConfig defaults this, so
		// only a Conn assembled in code reaches here.
		maxConcurrent = config.DefaultMaxConcurrentSubscribes
	}
	return &Client{
		client:           client,
		clientHandle:     conn.ClientId,
		interval:         conn.SubscriptionInterval.Duration,
		samplingInterval: conn.SamplingInterval.Duration,
		queueSize:        conn.QueueSize,
		logger:           logger,
		subGate:          semaphore.NewWeighted(int64(maxConcurrent)),
		subTimeout:       3 * conn.RequestTimeout.Duration,
	}
}

// subscription is the part of *opcua.Subscription that monitorPoint needs.
// Narrow enough that a test can supply a fake and check the subscription is torn down on
// every failure path, which is the difference between a failed subscribe and a wedged client.
type subscription interface {
	Monitor(ctx context.Context, ts ua.TimestampsToReturn, items ...*ua.MonitoredItemCreateRequest) (*ua.CreateMonitoredItemsResponse, error)
	Cancel(ctx context.Context) error
}

// Subscribe creates an OPC UA subscription for the specified node ID and returns a channel of value changes.
// The subscription monitors the node's value attribute and sends notifications when it changes.
// Returns an error if subscription creation or monitoring setup fails, wrapping the server's
// status code where there is one so the caller can tell a failure worth retrying from one that
// never will be.
//
// Only so many calls run at once, and each is given a deadline, because both defaults were
// effectively unbounded: every device subscribing to every point the moment a config loads is
// what makes a slow server answer StatusBadTimeout.
func (c *Client) Subscribe(ctx context.Context, nodeId *ua.NodeID) (<-chan *opcua.PublishNotificationData, error) {
	// acquire before starting the clock, queuing for a turn is not the server being slow
	if err := c.subGate.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer c.subGate.Release(1)

	// safe to cancel on return: gopcua uses this ctx only for the two requests below, the
	// Subscription it hands back is driven by the client's own publish loop
	ctx, cancel := context.WithTimeout(ctx, c.subTimeout)
	defer cancel()

	notifyCh := make(chan *opcua.PublishNotificationData)
	sub, err := c.client.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: c.interval,
	}, notifyCh)
	if err != nil {
		return nil, fmt.Errorf("create subscription for %s: %w", nodeId, err)
	}
	if err := c.monitorPoint(ctx, sub, nodeId); err != nil {
		return nil, err
	}
	return notifyCh, nil
}

// monitorPoint adds nodeId's value attribute to sub as a monitored item.
//
// It cancels sub on every failure path. That is not tidiness: by the time we get here the
// subscription is live on the server and registered with the client, and gopcua delivers
// notifications with a blocking send on the single client-wide publish goroutine. Leaving one
// behind holding a channel nobody reads - which is exactly what a Monitor timeout means, since
// the server may well have created the item anyway - stops every device on this connection
// from receiving values, and fills the client's resume channel until every later Subscribe
// blocks forever too. The ok flag keeps that true for any failure path added later.
func (c *Client) monitorPoint(ctx context.Context, sub subscription, nodeId *ua.NodeID) error {
	ok := false
	defer func() {
		if !ok {
			c.cancelSub(ctx, sub, nodeId)
		}
	}()

	// deliberately not NewMonitoredItemCreateRequestWithDefaults: its 10-deep queue sampled as
	// fast as the server allows overflows on every publish cycle for a fast-sampling server,
	// and the server flags that on every value it sends us
	valueReq := &ua.MonitoredItemCreateRequest{
		ItemToMonitor: &ua.ReadValueID{
			NodeID:       nodeId,
			AttributeID:  ua.AttributeIDValue,
			DataEncoding: &ua.QualifiedName{},
		},
		MonitoringMode: ua.MonitoringModeReporting,
		RequestedParameters: &ua.MonitoringParameters{
			ClientHandle:  c.clientHandle,
			DiscardOldest: true,
			QueueSize:     c.queueSize,
			// exact rather than truncating because config.Conn rejects any interval that
			// isn't a whole number of milliseconds, which ParseConfig has already checked
			SamplingInterval: float64(c.samplingInterval.Milliseconds()),
		},
	}
	res, err := sub.Monitor(ctx, ua.TimestampsToReturnNeither, valueReq)
	if err != nil {
		return fmt.Errorf("monitor %s: %w", nodeId, err)
	}
	if len(res.Results) != 1 {
		c.logger.Warn("expected one result", zap.Int("count", len(res.Results)), zap.Any("results", res.Results))
		return fmt.Errorf("monitor %s: %w, got %d", nodeId, errUnexpectedResults, len(res.Results))
	}
	if code := res.Results[0].StatusCode; statusIsBad(code) {
		// wrap the code rather than rendering it: whether retrying could ever work is decided
		// from the code, and a string loses the only thing that says so
		return fmt.Errorf("monitor %s: %w", nodeId, code)
	}
	c.warnIfRevised(nodeId, res.Results[0])
	ok = true
	return nil
}

// cancelSub tears down a subscription we are not going to hand back to a caller.
func (c *Client) cancelSub(ctx context.Context, sub subscription, nodeId *ua.NodeID) {
	// WithoutCancel because the case this exists for is a ctx that has already expired, where
	// Cancel would return without sending anything and leak the subscription anyway
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), orphanCancelTimeout)
	defer cancel()
	if err := sub.Cancel(ctx); err != nil {
		c.logger.Warn("failed to cancel an unused subscription",
			zap.Stringer("node", nodeId), zap.Error(err))
	}
}

// warnIfRevised reports monitoring parameters the server declined to honour.
// A server is free to revise what we ask for, typically clamping a sampling interval to its
// MinSupportedSampleRate or a queue to the depth it is willing to hold, and it tells us what
// it settled on rather than failing. That revision is the authoritative version of the
// config-time warnings in config.Conn.MonitoringWarnings, so it is worth surfacing: a queue
// revised down below a publishing cycle's worth of samples is exactly the setup that makes
// the server flag every value with the Overflow info bit. A revision that can only help,
// meanwhile, is worth a record but not an operator's attention.
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
	// only a shallower queue than we asked for is a warning: that is the one that overflows.
	// A server is entitled to hand back a deeper queue than requested, typically its own
	// minimum depth, and a deeper queue only discards fewer samples. Warning on it would mean
	// a line per monitored item on every config load for a server behaving perfectly well.
	switch {
	case res.RevisedQueueSize < c.queueSize:
		c.logger.Warn("server revised the queue size down",
			zap.Stringer("node", nodeId),
			zap.Uint32("requested", c.queueSize),
			zap.Uint32("revised", res.RevisedQueueSize))
	case res.RevisedQueueSize > c.queueSize:
		c.logger.Debug("server revised the queue size up",
			zap.Stringer("node", nodeId),
			zap.Uint32("requested", c.queueSize),
			zap.Uint32("revised", res.RevisedQueueSize))
	}
}
