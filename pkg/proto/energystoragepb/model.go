package energystoragepb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	energyLevel *resource.Value // of traits.EnergyLevel
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		energyLevel: resource.NewValue(args.energyLevelOpts...),
	}
}

func (m *Model) GetEnergyLevel(opts ...resource.ReadOption) (*EnergyLevel, error) {
	res := m.energyLevel.Get(opts...)
	return res.(*EnergyLevel), nil
}

func (m *Model) UpdateEnergyLevel(energyLevel *EnergyLevel, opts ...resource.WriteOption) (*EnergyLevel, error) {
	res, err := m.energyLevel.Set(energyLevel, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*EnergyLevel), nil
}

type PullEnergyLevelChange struct {
	Value      *EnergyLevel
	ChangeTime time.Time
}

func (m *Model) PullEnergyLevel(ctx context.Context, opts ...resource.ReadOption) <-chan PullEnergyLevelChange {
	send := make(chan PullEnergyLevelChange)

	recv := m.energyLevel.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			demand := change.Value.(*EnergyLevel)
			send <- PullEnergyLevelChange{
				Value:      demand,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}
