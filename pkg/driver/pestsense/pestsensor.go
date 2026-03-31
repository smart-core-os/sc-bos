package pestsense

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type pestSensor struct {
	occupancysensorpb.UnimplementedOccupancySensorApiServer

	id        string
	name      string
	occupancy *resource.Value // *traits.Occupancy
}

func newPestSensor(id, name string) *pestSensor {
	return &pestSensor{
		id:        id,
		name:      name,
		occupancy: resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}), resource.WithNoDuplicates()),
	}
}

func (p *pestSensor) GetOccupancy(_ context.Context, _ *occupancysensorpb.GetOccupancyRequest) (*occupancysensorpb.Occupancy, error) {
	value := p.occupancy.Get()
	occupancy := value.(*occupancysensorpb.Occupancy)
	return occupancy, nil
}

func (p *pestSensor) PullOccupancy(_ *occupancysensorpb.PullOccupancyRequest, server occupancysensorpb.OccupancySensorApi_PullOccupancyServer) error {
	for value := range p.occupancy.Pull(server.Context()) {
		occupancy := value.Value.(*occupancysensorpb.Occupancy)
		err := server.Send(&occupancysensorpb.PullOccupancyResponse{Changes: []*occupancysensorpb.PullOccupancyResponse_Change{
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
