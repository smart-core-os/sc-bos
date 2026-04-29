package airqualitysensorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	airQuality *resource.Value
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		airQuality: resource.NewValue(args.airQualityOpts...),
	}
}

func (m *Model) UpdateAirQuality(airQuality *AirQuality, opts ...resource.WriteOption) (*AirQuality, error) {
	res, err := m.airQuality.Set(airQuality, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*AirQuality), nil
}

func (m *Model) GetAirQuality(opts ...resource.ReadOption) (*AirQuality, error) {
	return m.airQuality.Get(opts...).(*AirQuality), nil
}

func (m *Model) PullAirQuality(ctx context.Context, opts ...resource.ReadOption) <-chan PullAirQualityChange {
	send := make(chan PullAirQualityChange)

	recv := m.airQuality.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*AirQuality)
			send <- PullAirQualityChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullAirQualityChange struct {
	Value      *AirQuality
	ChangeTime time.Time
}
