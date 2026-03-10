package airtemperaturepb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	airTemperature *resource.Value
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		airTemperature: resource.NewValue(args.airTemperatureOpts...),
	}
}

func (m *Model) UpdateAirTemperature(airTemperature *traits.AirTemperature, opts ...resource.WriteOption) (*traits.AirTemperature, error) {
	res, err := m.airTemperature.Set(airTemperature, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.AirTemperature), nil
}

func (m *Model) GetAirTemperature(opts ...resource.ReadOption) (*traits.AirTemperature, error) {
	return m.airTemperature.Get(opts...).(*traits.AirTemperature), nil
}

func (m *Model) PullAirTemperature(ctx context.Context, opts ...resource.ReadOption) <-chan PullAirTemperatureChange {
	send := make(chan PullAirTemperatureChange)

	recv := m.airTemperature.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.AirTemperature)
			send <- PullAirTemperatureChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullAirTemperatureChange struct {
	Value      *traits.AirTemperature
	ChangeTime time.Time
}
