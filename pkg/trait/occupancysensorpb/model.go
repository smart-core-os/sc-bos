package occupancysensorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	occupancy *resource.Value
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		occupancy: resource.NewValue(args.occupancyOpts...),
	}
}

// SetOccupancy updates the known occupancy state for this device
func (m *Model) SetOccupancy(occupancy *occupancysensorpb.Occupancy, opts ...resource.WriteOption) (*occupancysensorpb.Occupancy, error) {
	res, err := m.occupancy.Set(occupancy, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*occupancysensorpb.Occupancy), nil
}

func (m *Model) GetOccupancy(opts ...resource.ReadOption) (*occupancysensorpb.Occupancy, error) {
	return m.occupancy.Get(opts...).(*occupancysensorpb.Occupancy), nil
}

func (m *Model) PullOccupancy(ctx context.Context, opts ...resource.ReadOption) <-chan PullOccupancyChange {
	send := make(chan PullOccupancyChange)

	recv := m.occupancy.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*occupancysensorpb.Occupancy)
			send <- PullOccupancyChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}

type PullOccupancyChange struct {
	Value      *occupancysensorpb.Occupancy
	ChangeTime time.Time
}
