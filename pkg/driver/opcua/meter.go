package opcua

import (
	"context"
	"encoding/json"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Meter implements the Smart Core Meter trait for OPC UA devices.
// It maps OPC UA variable nodes to Meter usage readings.
type Meter struct {
	meterpb.UnimplementedMeterApiServer
	meterpb.UnimplementedMeterInfoServer

	energyValue *resource.Value // *meterpb.MeterReading
	logger      *zap.Logger
	meterConfig config.MeterConfig
	scName      string
}

func readMeterConfig(raw []byte) (cfg config.MeterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newMeter(n string, config config.RawTrait, l *zap.Logger) (*Meter, error) {
	cfg, err := readMeterConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	return &Meter{
		energyValue: resource.NewValue(resource.WithInitialValue(&meterpb.MeterReading{}), resource.WithNoDuplicates()),
		logger:      l,
		meterConfig: cfg,
		scName:      n,
	}, nil
}

func (m *Meter) GetMeterReading(_ context.Context, _ *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	return m.energyValue.Get().(*meterpb.MeterReading), nil
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

func (m *Meter) handleEvent(_ context.Context, node *ua.NodeID, value any) {

	if m.meterConfig.Usage != nil && nodeIdsAreEqual(m.meterConfig.Usage.NodeId, node) {
		v, err := conv.Float32Value(value)
		if err != nil {
			m.logger.Error("failed to convert value", zap.String("device", m.scName), zap.Error(err))
			return
		}

		scaled := m.meterConfig.Usage.Scaled(v)
		usage, ok := scaled.(float32)
		if !ok {
			m.logger.Error("scaled value is not float32", zap.String("device", m.scName), zap.Any("value", scaled))
			return
		}
		_, _ = m.energyValue.Set(&meterpb.MeterReading{
			Usage:   usage,
			EndTime: timestamppb.Now(),
		})
	}
}
