package meterpb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	meterReading *resource.Value // of *meterpb.MeterReading
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&MeterReading{})}
	value := resource.NewValue(append(defaultOptions, opts...)...)
	// make sure start and end time are recorded
	_, _ = value.Set(value.Get(), resource.InterceptBefore(func(old, new proto.Message) {
		oldVal := old.(*MeterReading)
		newVal := new.(*MeterReading)
		now := value.Clock().Now()
		if oldVal.StartTime == nil {
			newVal.StartTime = timestamppb.New(now)
		}
		if newVal.EndTime == nil {
			newVal.EndTime = timestamppb.New(now)
		}
	}))
	return &Model{
		meterReading: value,
	}
}

func (m *Model) GetMeterReading(opts ...resource.ReadOption) (*MeterReading, error) {
	return m.meterReading.Get(opts...).(*MeterReading), nil
}

func (m *Model) UpdateMeterReading(meterReading *MeterReading, opts ...resource.WriteOption) (*MeterReading, error) {
	res, err := m.meterReading.Set(meterReading, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*MeterReading), nil
}

// RecordReading records a new usage value, updating end time to now.
func (m *Model) RecordReading(val float32) (*MeterReading, error) {
	return m.UpdateMeterReading(&MeterReading{Usage: val}, resource.InterceptBefore(func(old, new proto.Message) {
		now := m.meterReading.Clock().Now()
		newVal := new.(*MeterReading)
		newVal.EndTime = timestamppb.New(now)
	}))
}

// Reset resets the meter to zero, updating both start and end times to now.
func (m *Model) Reset() (*MeterReading, error) {
	now := timestamppb.New(m.meterReading.Clock().Now())
	return m.UpdateMeterReading(&MeterReading{Usage: 0, StartTime: now, EndTime: now},
		// force usage (which is zero) to be updated
		resource.WithUpdatePaths("usage", "start_time", "end_time"))
}

func (m *Model) PullMeterReadings(ctx context.Context, opts ...resource.ReadOption) <-chan PullMeterReadingChange {
	return resources.PullValue[*MeterReading](ctx, m.meterReading.Pull(ctx, opts...))
}

type PullMeterReadingChange = resources.ValueChange[*MeterReading]
