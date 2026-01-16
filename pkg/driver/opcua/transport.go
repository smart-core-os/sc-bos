package opcua

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Transport implements the Smart Core Transport trait for OPC UA devices.
// It maps OPC UA variable nodes to Transport system properties such as position, speed, doors, and operating mode.
// Typically used for elevators, escalators, and other vertical/horizontal Transport systems.
type Transport struct {
	config.Trait
	transportpb.UnimplementedTransportApiServer
	transportpb.UnimplementedTransportInfoServer

	logger    *zap.Logger
	transport *resource.Value // *transportpb.Transport
	cfg       config.TransportConfig
	scName    string
}

func readTransportConfig(raw []byte) (cfg config.TransportConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newTransport(n string, c config.RawTrait, l *zap.Logger) (*Transport, error) {
	cfg, err := readTransportConfig(c.Raw)
	if err != nil {
		return nil, err
	}
	t := &Transport{
		logger:    l,
		transport: resource.NewValue(resource.WithInitialValue(&transportpb.Transport{}), resource.WithNoDuplicates()),
		cfg:       cfg,
		scName:    n,
	}
	// initialise the doors as we know these from the config
	tp := &transportpb.Transport{}
	for _, door := range cfg.Doors {
		tp.Doors = append(tp.Doors, &transportpb.Transport_Door{Title: door.Title})
	}
	_, _ = t.transport.Set(tp)
	return t, nil
}

func (t *Transport) GetTransport(_ context.Context, _ *transportpb.GetTransportRequest) (*transportpb.Transport, error) {
	return t.transport.Get().(*transportpb.Transport), nil
}

func (t *Transport) PullTransport(_ *transportpb.PullTransportRequest, server transportpb.TransportApi_PullTransportServer) error {
	for value := range t.transport.Pull(server.Context()) {
		transport := value.Value.(*transportpb.Transport)
		err := server.Send(&transportpb.PullTransportResponse{Changes: []*transportpb.PullTransportResponse_Change{
			{
				Name:       t.scName,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Transport:  transport,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Transport) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	old := t.transport.Get().(*transportpb.Transport)
	if t.cfg.ActualPosition != nil && nodeIdsAreEqual(t.cfg.ActualPosition.NodeId, node) {
		floor, err := conv.ToString(value)
		if err != nil {
			t.logger.Error("failed to convert ActualPosition event", zap.String("device", t.scName), zap.Error(err))
			return
		}
		old.ActualPosition = &transportpb.Transport_Location{
			Floor: floor,
		}
	}
	if t.cfg.Load != nil && nodeIdsAreEqual(t.cfg.Load.NodeId, node) {
		load, err := conv.Float32Value(value)
		if err != nil {
			t.logger.Error("failed to convert Load event", zap.String("device", t.scName), zap.Error(err))
			return
		}
		old.Load = &load
	}
	if t.cfg.MovingDirection != nil && nodeIdsAreEqual(t.cfg.MovingDirection.NodeId, node) {
		direction, err := conv.ToTraitEnum[transportpb.Transport_Direction](value, t.cfg.MovingDirection.Enum, transportpb.Transport_Direction_value)
		if err != nil {
			t.logger.Error("failed to convert MovingDirection to trait enum", zap.String("device", t.scName), zap.Error(err))
			return
		}
		old.MovingDirection = direction
	}
	if t.cfg.NextDestinations != nil {
		for i, dest := range t.cfg.NextDestinations {
			if dest.Type == config.SingleFloor && nodeIdsAreEqual(dest.Source.NodeId, node) {
				floor, err := conv.IntValue(value)
				if err != nil {
					t.logger.Error("failed to convert NextDestinations event", zap.String("device", t.scName), zap.Error(err))
					return
				}
				if i >= len(old.NextDestinations) {
					old.NextDestinations = append(old.NextDestinations, &transportpb.Transport_Location{
						Floor: strconv.Itoa(floor),
					})
				} else {
					old.NextDestinations[i] = &transportpb.Transport_Location{
						Floor: strconv.Itoa(floor),
					}
				}
			}
		}
	}
	if t.cfg.OperatingMode != nil && nodeIdsAreEqual(t.cfg.OperatingMode.NodeId, node) {
		mode, err := conv.ToTraitEnum[transportpb.Transport_OperatingMode](value, t.cfg.OperatingMode.Enum, transportpb.Transport_OperatingMode_value)
		if err != nil {
			t.logger.Error("failed to convert OperatingMode to trait enum", zap.String("device", t.scName), zap.Error(err))
			return
		}
		old.OperatingMode = mode
	}
	if t.cfg.Doors != nil {
		for i, door := range t.cfg.Doors {
			if door.Status != nil && nodeIdsAreEqual(door.Status.NodeId, node) {
				status, err := conv.ToTraitEnum[transportpb.Transport_Door_DoorStatus](value, door.Status.Enum, transportpb.Transport_Door_DoorStatus_value)
				if err != nil {
					t.logger.Error("failed to convert Door Status to trait enum", zap.String("device", t.scName), zap.Error(err))
					return
				}
				d := &transportpb.Transport_Door{
					Title: door.Title,
				}
				d.Status = status
				old.Doors[i] = d
			}
		}
	}
	if t.cfg.Speed != nil && nodeIdsAreEqual(t.cfg.Speed.NodeId, node) {
		speed, err := conv.Float32Value(value)
		if err != nil {
			t.logger.Error("failed to convert Speed event", zap.String("device", t.scName), zap.Error(err))
			return
		}
		old.Speed = &speed
	}
	_, _ = t.transport.Set(old)
}

func (t *Transport) DescribeTransport(context.Context, *transportpb.DescribeTransportRequest) (*transportpb.TransportSupport, error) {
	return &transportpb.TransportSupport{
		LoadUnit:  t.cfg.LoadUnit,
		MaxLoad:   t.cfg.MaxLoad,
		SpeedUnit: t.cfg.SpeedUnit,
	}, nil
}
