package pestsense

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type pestSensor struct {
	traits.UnimplementedOccupancySensorApiServer

	id        string
	name      string
	occupancy *resource.Value // *traits.Occupancy
}

func newPestSensor(id, name string) *pestSensor {
	return &pestSensor{
		id:        id,
		name:      name,
		occupancy: resource.NewValue(resource.WithInitialValue(&traits.Occupancy{}), resource.WithNoDuplicates()),
	}
}

func (p *pestSensor) GetOccupancy(_ context.Context, _ *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	value := p.occupancy.Get()
	occupancy := value.(*traits.Occupancy)
	return occupancy, nil
}

func (p *pestSensor) PullOccupancy(_ *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	for value := range p.occupancy.Pull(server.Context()) {
		occupancy := value.Value.(*traits.Occupancy)
		err := server.Send(&traits.PullOccupancyResponse{Changes: []*traits.PullOccupancyResponse_Change{
			{
				Name:       p.name,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Occupancy:  occupancy,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
