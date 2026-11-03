package opcua

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Meter implements the Smart Core Meter trait for OPC UA devices.
// It maps OPC UA variable nodes to Meter usage readings.
//
// A reading is only served as current while the subscription vouches for it (see
// pointLiveness). Without that, Get reads the usage node live, and reports Unavailable when it
// cannot.
type Meter struct {
	meterpb.UnimplementedMeterApiServer
	meterpb.UnimplementedMeterInfoServer

	energyValue *resource.Value // *meterpb.MeterReading
	logger      *zap.Logger
	meterConfig config.MeterConfig
	scName      string

	reader pointReader
	// usageNode is the parsed usage node, for reading it live. nil when the config has no
	// usage or its node id does not parse, and then there is nothing to read.
	usageNode *ua.NodeID

	// usageKey is the configured usage node id, which live is keyed by. Empty with no usage.
	usageKey string
	live     *pointLiveness
}

func readMeterConfig(raw []byte) (cfg config.MeterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newMeter(n string, config config.RawTrait, reader pointReader, l *zap.Logger) (*Meter, error) {
	cfg, err := readMeterConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	m := &Meter{
		energyValue: resource.NewValue(resource.WithInitialValue(&meterpb.MeterReading{}), resource.WithNoDuplicates()),
		logger:      l,
		meterConfig: cfg,
		scName:      n,
		reader:      reader,
	}
	if cfg.Usage != nil {
		m.usageKey = cfg.Usage.NodeId
		m.usageNode = parseLiveNode(cfg.Usage.NodeId)
		m.live = newPointLiveness(readerConnected(reader), m.usageKey)
	} else {
		m.live = newPointLiveness(nil)
	}
	return m, nil
}

// GetMeterReading returns the meter's current reading, stamped now.
// Stamped now rather than when the value last changed, because a monitored item the server is
// sampling says the value is current at every sample, and consumers that de-duplicate on the
// timestamp would otherwise drop every reading of a steady meter.
func (m *Meter) GetMeterReading(ctx context.Context, _ *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	if m.vouched() {
		reading := m.energyValue.Get().(*meterpb.MeterReading)
		return &meterpb.MeterReading{Usage: reading.Usage, EndTime: timestamppb.Now()}, nil
	}
	return m.readLive(ctx)
}

// vouched reports whether the subscription stands behind the cached reading.
func (m *Meter) vouched() bool {
	return m.live.vouched(m.usageKey)
}

// readLive reads the usage node from the server, for a Get the subscription cannot vouch for.
func (m *Meter) readLive(ctx context.Context) (*meterpb.MeterReading, error) {
	if m.usageNode == nil || m.reader == nil {
		return nil, status.Error(codes.Unavailable, "meter usage cannot be read")
	}
	gen := m.live.beginRead(m.usageKey)
	dvs, err := m.reader.ReadValues(ctx, m.usageNode)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "read meter usage: %v", err)
	}
	raw, err := liveValue(dvs[0])
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "meter usage %v", err)
	}
	usage, err := m.usage(raw)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "meter usage: %v", err)
	}

	now := timestamppb.Now()
	m.live.readGood(m.usageKey, gen, func() {
		cached := m.energyValue.Get().(*meterpb.MeterReading)
		// only on a change, so Pull still emits once per value with the time it changed
		if cached.EndTime == nil || cached.Usage != usage {
			_, _ = m.energyValue.Set(&meterpb.MeterReading{Usage: usage, EndTime: now})
		}
	})
	return &meterpb.MeterReading{Usage: usage, EndTime: now}, nil
}

func (m *Meter) PullMeterReadings(_ *meterpb.PullMeterReadingsRequest, server meterpb.MeterApi_PullMeterReadingsServer) error {
	for value := range m.energyValue.Pull(server.Context()) {
		err := server.Send(&meterpb.PullMeterReadingsResponse{Changes: []*meterpb.PullMeterReadingsResponse_Change{
			{
				Name:         m.scName,
				ChangeTime:   timestamppb.New(value.ChangeTime),
				MeterReading: m.energyValue.Get().(*meterpb.MeterReading),
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Meter) DescribeMeterReading(context.Context, *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	return &meterpb.MeterReadingSupport{
		UsageUnit: m.meterConfig.Unit,
	}, nil
}

func (m *Meter) isUsage(node *ua.NodeID) bool {
	return m.meterConfig.Usage != nil && nodeIdsAreEqual(m.meterConfig.Usage.NodeId, node)
}

// usage converts and scales a raw usage value.
func (m *Meter) usage(value any) (float32, error) {
	v, err := conv.Float32Value(value)
	if err != nil {
		return 0, fmt.Errorf("convert value: %w", err)
	}
	scaled := m.meterConfig.Usage.Scaled(v)
	usage, ok := scaled.(float32)
	if !ok {
		return 0, fmt.Errorf("scaled value %v is not float32", scaled)
	}
	return usage, nil
}

func (m *Meter) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	if !m.isUsage(node) {
		return
	}
	usage, err := m.usage(value)
	if err != nil {
		// the server sent a value we cannot use, so the cached one is no longer its value
		m.live.unusable(m.usageKey)
		m.logger.Error("failed to convert value", zap.String("device", m.scName), zap.Error(err))
		return
	}
	m.live.good(m.usageKey, func() {
		_, _ = m.energyValue.Set(&meterpb.MeterReading{
			Usage:   usage,
			EndTime: timestamppb.Now(),
		})
	})
}

func (m *Meter) pointBad(node *ua.NodeID) { m.live.unusable(node.String()) }
func (m *Meter) connectionSuspect()       { m.live.connectionSuspect() }
func (m *Meter) subscriptionLost()        { m.live.subscriptionLost() }
