package xovis

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type occupancyServer struct {
	traits.UnimplementedOccupancySensorApiServer
	bus         *minibus.Bus[PushData]
	client      *client
	multiSensor bool
	logicID     int

	pollInit sync.Once
	poll     *task.Intermittent
	polls    *minibus.Bus[LiveLogicResponse]

	OccupancyTotal *resource.Value
}

var errDataFormat = status.Error(codes.FailedPrecondition, "data received from sensor did not match expected format")

func (o *occupancyServer) GetOccupancy(ctx context.Context, request *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	res, err := getLiveLogic(o.client, o.multiSensor, o.logicID)
	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	occupancy := decodeOccupancyCounts(res.Logic.Counts)
	if occupancy == nil {
		return nil, errDataFormat
	}

	o.OccupancyTotal.Set(occupancy)
	return occupancy, nil
}

func (o *occupancyServer) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	// fetch the initial occupancy state
	res, err := getLiveLogic(o.client, o.multiSensor, o.logicID)
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}
	occupancy := decodeOccupancyCounts(res.Logic.Counts)
	if occupancy == nil {
		return errDataFormat
	}

	var lastSent *traits.Occupancy
	if !request.UpdatesOnly {
		err = server.Send(&traits.PullOccupancyResponse{
			Changes: []*traits.PullOccupancyResponse_Change{
				{
					Name:       request.Name,
					ChangeTime: timestamppb.New(res.Time),
					Occupancy:  occupancy,
				},
			},
		})
		if err != nil {
			return err
		}
		lastSent = occupancy
	}

	ctx := server.Context()
	o.doPollInit()
	polls := o.polls.Listen(ctx)
	webhooks := o.bus.Listen(ctx)

	// tell the polling logic we're interested
	_ = o.poll.Attach(ctx) // can't error

	eq := cmp.Equal()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case data, ok := <-webhooks:
			if !ok {
				return nil
			}
			if data.LogicsData == nil {
				continue
			}
			records, ok := findLogicRecords(data.LogicsData, o.logicID)
			if !ok {
				continue
			}

			var changes []*traits.PullOccupancyResponse_Change
			for _, record := range records {
				occupancy := decodeOccupancyCounts(record.Counts)
				if occupancy == nil {
					continue
				}

				if eq(lastSent, occupancy) {
					continue
				}

				changes = append(changes, &traits.PullOccupancyResponse_Change{
					Name:       request.Name,
					ChangeTime: timestamppb.New(record.To),
					Occupancy:  occupancy,
				})
			}

			err = server.Send(&traits.PullOccupancyResponse{Changes: changes})
			if err != nil {
				return err
			}
			lastSent = changes[len(changes)-1].Occupancy

		case data, ok := <-polls:
			if !ok {
				return nil
			}
			occupancy := decodeOccupancyCounts(data.Logic.Counts)
			if occupancy == nil {
				continue
			}

			if eq(lastSent, occupancy) {
				continue
			}
			err = server.Send(&traits.PullOccupancyResponse{
				Changes: []*traits.PullOccupancyResponse_Change{
					{
						Name:       request.Name,
						ChangeTime: timestamppb.New(res.Time),
						Occupancy:  occupancy,
					},
				},
			})
			if err != nil {
				return err
			}
			lastSent = occupancy
		}
	}
}

func (o *occupancyServer) doPollInit() {
	o.pollInit.Do(func() {
		o.polls = &minibus.Bus[LiveLogicResponse]{}
		o.poll = task.Poll(func(ctx context.Context) {
			res, err := getLiveLogic(o.client, o.multiSensor, o.logicID)
			if err != nil {
				// todo: log error
				return
			}
			o.polls.Send(ctx, res)

		}, 30*time.Second)
	})
}

// returns nil if the counts don't match the expected format for an occupancy logic
func decodeOccupancyCounts(counts []Count) (occupancy *traits.Occupancy) {
	if len(counts) != 1 || counts[0].Name != "balance" {
		return nil
	}
	var state traits.Occupancy_State
	if counts[0].Value > 0 {
		state = traits.Occupancy_OCCUPIED
	} else {
		state = traits.Occupancy_UNOCCUPIED
	}
	return &traits.Occupancy{
		State:       state,
		PeopleCount: int32(counts[0].Value),
	}
}
