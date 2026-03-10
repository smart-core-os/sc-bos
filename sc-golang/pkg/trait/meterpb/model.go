package meterpb

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	meterReading *resource.Value // of *traits.MeterReading
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&traits.MeterReading{})}
	value := resource.NewValue(append(defaultOptions, opts...)...)
	// make sure start and end time are recorded
	_, _ = value.Set(&traits.MeterReading{}, resource.InterceptBefore(func(old, new proto.Message) {
		oldVal := old.(*traits.MeterReading)
		newVal := new.(*traits.MeterReading)
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

func (m *Model) GetMeterReading(opts ...resource.ReadOption) (*traits.MeterReading, error) {
	return m.meterReading.Get(opts...).(*traits.MeterReading), nil
}

func (m *Model) UpdateMeterReading(meterReading *traits.MeterReading, opts ...resource.WriteOption) (*traits.MeterReading, error) {
	res, err := m.meterReading.Set(meterReading, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.MeterReading), nil
}

// RecordReading records a new usage value, updating end time to now.
func (m *Model) RecordReading(val float32) (*traits.MeterReading, error) {
	return m.UpdateMeterReading(&traits.MeterReading{Usage: val}, resource.InterceptBefore(func(old, new proto.Message) {
		now := m.meterReading.Clock().Now()
		newVal := new.(*traits.MeterReading)
		newVal.EndTime = timestamppb.New(now)
	}))
}

// Reset resets the meter to zero, updating both start and end times to now.
func (m *Model) Reset() (*traits.MeterReading, error) {
	now := timestamppb.New(m.meterReading.Clock().Now())
	return m.UpdateMeterReading(&traits.MeterReading{Usage: 0, StartTime: now, EndTime: now},
		// force usage (which is zero) to be updated
		resource.WithUpdatePaths("usage", "start_time", "end_time"))
}

func (m *Model) PullMeterReadings(ctx context.Context, opts ...resource.ReadOption) <-chan PullMeterReadingChange {
	send := make(chan PullMeterReadingChange)

	recv := m.meterReading.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.MeterReading)
			send <- PullMeterReadingChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullMeterReadingChange struct {
	Value      *traits.MeterReading
	ChangeTime time.Time
}
