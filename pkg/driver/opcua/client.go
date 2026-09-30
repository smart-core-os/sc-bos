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

// warnIfRevised logs monitoring parameters the server revised rather than honoured.
// Servers may clamp the sampling interval to their MinSupportedSampleRate or change the queue
// depth; a queue revised below a publishing cycle's worth of samples overflows every cycle.
func (c *Client) warnIfRevised(nodeId *ua.NodeID, res *ua.MonitoredItemCreateResult) {
	// the revised interval is a float off the wire, so compare with a tolerance
	requested := float64(c.samplingInterval.Milliseconds())
	if !floatEqual(res.RevisedSamplingInterval, requested) {
		c.logger.Warn("server revised the sampling interval",
			zap.Stringer("node", nodeId),
			zap.Float64("requestedMs", requested),
			zap.Float64("revisedMs", res.RevisedSamplingInterval))
	}
	// only a shallower queue can overflow; a deeper one is harmless, so log it at debug
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
