package opcua

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"

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
}

// NewClient creates a new Client wrapper around an OPC UA client connection.
// The monitoring parameters are taken from conn, which ParseConfig has already defaulted.
func NewClient(client *opcua.Client, logger *zap.Logger, conn config.Conn) *Client {
	return &Client{
		client:           client,
		clientHandle:     conn.ClientId,
		interval:         conn.SubscriptionInterval.Duration,
		samplingInterval: conn.SamplingInterval.Duration,
		queueSize:        conn.QueueSize,
		logger:           logger,
	}
}

// Subscribe creates an OPC UA subscription for the specified node ID and returns a channel of value changes.
// The subscription monitors the node's value attribute and sends notifications when it changes.
// Returns an error if subscription creation or monitoring setup fails.
func (c *Client) Subscribe(ctx context.Context, nodeId *ua.NodeID) (<-chan *opcua.PublishNotificationData, error) {
	notifyCh := make(chan *opcua.PublishNotificationData)
	sub, err := c.client.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: c.interval,
	}, notifyCh)
	if err != nil {
		return nil, err
	}
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
			ClientHandle:     c.clientHandle,
			DiscardOldest:    true,
			QueueSize:        c.queueSize,
			SamplingInterval: float64(c.samplingInterval.Milliseconds()),
		},
	}
	res, err := sub.Monitor(ctx, ua.TimestampsToReturnNeither, valueReq)
	if err != nil {
		return nil, err
	}
	if len(res.Results) > 1 || len(res.Results) == 0 {
		c.logger.Warn("expected one result", zap.Int("count", len(res.Results)), zap.Any("results", res.Results))
		return nil, fmt.Errorf("expected one result, got %d", len(res.Results))
	}
	if statusIsBad(res.Results[0].StatusCode) {
		return nil, fmt.Errorf("error monitoring node: %s", res.Results[0].StatusCode.Error())
	}
	c.warnIfRevised(nodeId, res.Results[0])
	return notifyCh, nil
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
