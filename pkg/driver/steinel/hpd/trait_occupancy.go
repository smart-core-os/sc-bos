package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Occupancy struct {
	occupancysensorpb.UnimplementedOccupancySensorApiServer
	udmipb.UnimplementedUdmiServiceServer

	logger *zap.Logger

	client *Client

	OccupancyValue *resource.Value
}

var _ sensor = (*Occupancy)(nil)

func newOccupancySensor(client *Client, logger *zap.Logger) *Occupancy {
	return &Occupancy{
		client:         client,
		logger:         logger,
		OccupancyValue: resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}), resource.WithNoDuplicates()),
	}
}

func (o *Occupancy) GetOccupancy(_ context.Context, _ *occupancysensorpb.GetOccupancyRequest) (*occupancysensorpb.Occupancy, error) {
	response := SensorResponse{}
	if err := doGetRequest(o.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := o.GetUpdate(&response); err != nil {
		return nil, err
	}
	return o.OccupancyValue.Get().(*occupancysensorpb.Occupancy), nil
}

func (o *Occupancy) PullOccupancy(request *occupancysensorpb.PullOccupancyRequest, server occupancysensorpb.OccupancySensorApi_PullOccupancyServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := o.OccupancyValue.Pull(ctx)

	for change := range changes {
		v := change.Value.(*occupancysensorpb.Occupancy)

		err := server.Send(&occupancysensorpb.PullOccupancyResponse{
			Changes: []*occupancysensorpb.PullOccupancyResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), Occupancy: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Occupancy) GetUpdate(response *SensorResponse) error {
	peopleCount := 0

	state := occupancysensorpb.Occupancy_STATE_UNSPECIFIED
	if response.TruePresence1 {
		state = occupancysensorpb.Occupancy_OCCUPIED
	} else {
		state = occupancysensorpb.Occupancy_UNOCCUPIED
	}

	if response.ZonePeople0 > 0 {
		state = occupancysensorpb.Occupancy_OCCUPIED
		peopleCount = response.ZonePeople0
	}

	_, err := o.OccupancyValue.Set(&occupancysensorpb.Occupancy{
		State:       state,
		PeopleCount: int32(peopleCount),
	})
	if err != nil {
		return err
	}

	return nil
}

func (o *Occupancy) GetName() string {
	return "Occupancy"
}
