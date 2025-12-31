package opcua

import (
	"context"
	"errors"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
)

// device represents an OPC UA device that subscribes to variable nodes and updates trait implementations.
// It manages subscriptions for a single logical device and routes OPC UA events to the appropriate trait handlers.
// The type is intentionally unexported as it's an internal implementation detail of the driver.
type device struct {
	conf   *config.Device
	logger *zap.Logger
	client *Client

	electric  *Electric
	meter     *Meter
	transport *Transport
	udmi      *Udmi

	systemName string
	faultCheck *healthpb.FaultCheck
}

// newDevice creates a new device instance for the given configuration.
// Trait implementations (Electric, Meter, Transport, udmi) must be assigned separately before calling run.
func newDevice(conf *config.Device, logger *zap.Logger, client *Client, systemName string, check *healthpb.FaultCheck) *device {
	return &device{
		client:     client,
		conf:       conf,
		systemName: systemName,
		faultCheck: check,
		logger:     logger,
	}
}

// subscribe creates OPC UA subscriptions for all configured variables and spawns goroutines to handle events.
// If a subscription fails, it logs the error and continues with remaining variables.
// The method blocks until the context is cancelled or all subscriptions fail.
func (d *device) subscribe(ctx context.Context) error {
	grp, ctx := errgroup.WithContext(ctx)
	for _, point := range d.conf.Variables {
		pointName := point.ParsedNodeId
		c, err := d.client.Subscribe(ctx, pointName)
		if err != nil {
			d.logger.Error("failed to subscribe to point", zap.Stringer("point", pointName), zap.Error(err))
			raiseConfigFault("Failed to subscribe to point "+pointName.String(), d.systemName, d.faultCheck)
			continue
		}
		grp.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case event := <-c:
					if event == nil {
						continue
					}
					d.handleEvent(ctx, event, pointName)
				}
			}
		})
	}
	return grp.Wait()
}

// handleEvent processes OPC UA subscription events and routes them to trait handlers.
// It handles both DataChangeNotification (variable value changes) and EventNotificationList (OPC UA events).
// Values with non-OK status codes are logged as warnings and not passed to trait handlers.
func (d *device) handleEvent(ctx context.Context, event *opcua.PublishNotificationData, node *ua.NodeID) {
	switch x := event.Value.(type) {
	case *ua.DataChangeNotification:
		for _, item := range x.MonitoredItems {
			if item.Value == nil ||
				item.Value.Value == nil {
				continue
			}

			if errors.Is(item.Value.Status, ua.StatusOK) {
				clearPointFault(node.String(), d.systemName, d.faultCheck)
				value := item.Value.Value.Value()
				d.handleTraitEvent(ctx, node, value)
			} else {
				raisePointFault(node.String(), item.Value.Status.Error(), d.systemName, d.faultCheck)
				d.logger.Warn("error monitoring node", zap.Stringer("node", node), zap.String("code", item.Value.Status.Error()))
			}
		}

	case *ua.EventNotificationList:
		for _, item := range x.Events {
			for _, field := range item.EventFields {
				if errors.Is(field.StatusCode(), ua.StatusOK) {
					value := field.Value()
					clearPointFault(node.String(), d.systemName, d.faultCheck)
					d.handleTraitEvent(ctx, node, value)
				} else {
					raisePointFault(node.String(), field.StatusCode().Error(), d.systemName, d.faultCheck)
					d.logger.Warn("error monitoring node", zap.Stringer("node", node), zap.String("code", field.StatusCode().Error()))
				}
			}
		}

	default:
		d.logger.Warn("unhandled event", zap.Any("energyValue", event.Value))
	}
}

// handleTraitEvent dispatches an OPC UA value change to all configured trait handlers.
// Each trait handler is responsible for checking if the node ID matches its configuration.
func (d *device) handleTraitEvent(ctx context.Context, node *ua.NodeID, value any) {

	if d.electric != nil {
		d.electric.handleElectricEvent(node, value)
	}
	if d.meter != nil {
		d.meter.handleMeterEvent(node, value)
	}
	if d.transport != nil {
		d.transport.handleTransportEvent(node, value)
	}
	if d.udmi != nil {
		d.udmi.sendUdmiMessage(ctx, node, value)
	}
}

// nodeIdsAreEqual compares a string node ID with a ua.NodeID for equality.
// Returns true if n is non-nil and its string representation matches nodeId.
func nodeIdsAreEqual(nodeId string, n *ua.NodeID) bool {
	return n != nil && nodeId == n.String()
}
