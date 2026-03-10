package airqualitysensorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
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

func (m *Model) UpdateAirQuality(airQuality *traits.AirQuality, opts ...resource.WriteOption) (*traits.AirQuality, error) {
	res, err := m.airQuality.Set(airQuality, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.AirQuality), nil
}

func (m *Model) GetAirQuality(opts ...resource.ReadOption) (*traits.AirQuality, error) {
	return m.airQuality.Get(opts...).(*traits.AirQuality), nil
}

func (m *Model) PullAirQuality(ctx context.Context, opts ...resource.ReadOption) <-chan PullAirQualityChange {
	send := make(chan PullAirQualityChange)

	recv := m.airQuality.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.AirQuality)
			send <- PullAirQualityChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullAirQualityChange struct {
	Value      *traits.AirQuality
	ChangeTime time.Time
}
