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

type Device struct {
	conf   *config.Device
	logger *zap.Logger
	client *Client

	electric  *Electric
	meter     *Meter
	transport *Transport
	udmi      *Udmi

	faultCheck *healthpb.FaultCheck
}

func newDevice(device *config.Device, logger *zap.Logger, client *Client, check *healthpb.FaultCheck) *Device {

	return &Device{
		client:     client,
		conf:       device,
		faultCheck: check,
		logger:     logger,
	}
}

func (d *Device) subscribe(ctx context.Context) error {

	grp, ctx := errgroup.WithContext(ctx)
	for _, point := range d.conf.Variables {
		pointName := point.ParsedNodeId
		c, err := d.client.Subscribe(ctx, pointName)
		if err != nil {
			d.logger.Error("failed to subscribe to point", zap.Stringer("point", pointName), zap.Error(err))
			raiseConfigFault("Failed to subscribe to point "+pointName.String(), d.faultCheck)
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

func (d *Device) handleEvent(ctx context.Context, event *opcua.PublishNotificationData, node *ua.NodeID) {
	switch x := event.Value.(type) {
	case *ua.DataChangeNotification:
		for _, item := range x.MonitoredItems {
			if item.Value == nil ||
				item.Value.Value == nil {
				continue
			}

			if errors.Is(item.Value.Status, ua.StatusOK) {
				clearPointFault(node.String(), d.faultCheck)
				value := item.Value.Value.Value()
				d.handleTraitEvent(ctx, node, value)
			} else {
				raisePointFault(node.String(), item.Value.Status.Error(), d.faultCheck)
				d.logger.Warn("error monitoring node", zap.Stringer("node", node), zap.String("code", item.Value.Status.Error()))
			}
		}

	case *ua.EventNotificationList:
		for _, item := range x.Events {
			for _, field := range item.EventFields {
				if errors.Is(field.StatusCode(), ua.StatusOK) {
					value := field.Value()
					clearPointFault(node.String(), d.faultCheck)
					d.handleTraitEvent(ctx, node, value)
				} else {
					raisePointFault(node.String(), field.StatusCode().Error(), d.faultCheck)
					d.logger.Warn("error monitoring node", zap.Stringer("node", node), zap.String("code", field.StatusCode().Error()))
				}
			}
		}

	default:
		d.logger.Warn("unhandled event", zap.Any("energyValue", event.Value))
	}
}

func (d *Device) handleTraitEvent(ctx context.Context, node *ua.NodeID, value any) {

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

func nodeIdsAreEqual(nodeId string, n *ua.NodeID) bool {
	return n != nil && nodeId == n.String()
}
