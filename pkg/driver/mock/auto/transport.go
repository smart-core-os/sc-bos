package auto

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func getLocation(f int) *transportpb.Transport_Location {

	floorName := fmt.Sprintf("%2d", f)
	if f == 0 {
		floorName = "GF"
	}
	return &transportpb.Transport_Location{
		Floor: floorName,
		Title: floorName,
	}
}

func getNextDestination(maxFloor int) *transportpb.Transport_Location {
	f := 1 + rand.Int()%maxFloor
	return getLocation(f)
}

func initTransport(load *float32) *transportpb.Transport {
	return &transportpb.Transport{
		ActualPosition: &transportpb.Transport_Location{
			Floor: "GF",
			Id:    "GF",
			Title: "Ground Floor",
		},
		NextDestinations: []*transportpb.Transport_Location{
			{
				Floor: "GF",
				Id:    "GF",
				Title: "Ground Floor",
			},
		},
		MovingDirection: transportpb.Transport_NO_DIRECTION,
		Load:            load,
		Doors: []*transportpb.Transport_Door{
			{
				Deck:   0,
				Title:  "Main Door",
				Status: transportpb.Transport_Door_CLOSED,
			},
		},
	}
}

func openCloseDoor(t *transportpb.Transport, nextDest *transportpb.Transport_Location, load float32) {
	t.MovingDirection = transportpb.Transport_NO_DIRECTION
	if t.Doors[0].Status == transportpb.Transport_Door_CLOSED {
		t.Doors[0].Status = transportpb.Transport_Door_OPENING
	} else if t.Doors[0].Status == transportpb.Transport_Door_OPENING {
		t.Doors[0].Status = transportpb.Transport_Door_OPEN
	} else if t.Doors[0].Status == transportpb.Transport_Door_OPEN {
		t.Doors[0].Status = transportpb.Transport_Door_CLOSING
		t.Load = &load
	} else if t.Doors[0].Status == transportpb.Transport_Door_CLOSING {
		t.Doors[0].Status = transportpb.Transport_Door_CLOSED
		t.NextDestinations[0] = nextDest
	}
}

func TransportAuto(model *transport.Model, maxFloor int) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(2 * time.Second)
		go func() {
			gfMaxWait := 3
			gfCurrentWait := 0
			currentFloor := 0
			load := float32(0)
			t := initTransport(&load)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// a basic simulation of a lift moving up and down.
					// if we are on the ground floor, choose a random floor between 1 and maxFloor and
					// and move to that floor, travelling at 1 floor per interval.
					if t.ActualPosition.Floor == "GF" && t.NextDestinations[0].Floor == "GF" {
						t.MovingDirection = transportpb.Transport_NO_DIRECTION
						if gfCurrentWait >= gfMaxWait {
							// we have waited long enough, move to a new floor
							load = float32(rand.Int() % 1000)
							openCloseDoor(t, getNextDestination(maxFloor), load)
						} else {
							gfCurrentWait++
						}
					} else if t.ActualPosition.Floor == t.NextDestinations[0].Floor { // we have arrived
						t.MovingDirection = transportpb.Transport_NO_DIRECTION
						gfCurrentWait = 0
						load = 0
						openCloseDoor(t, getLocation(0), load)
					} else { // we are on the move,
						if t.NextDestinations[0].Floor == "GF" {
							t.MovingDirection = transportpb.Transport_DOWN
							currentFloor--
							t.ActualPosition = getLocation(currentFloor)
						} else {
							t.MovingDirection = transportpb.Transport_UP
							currentFloor++
							t.ActualPosition = getLocation(currentFloor)
						}
					}
					_, _ = model.UpdateTransport(t)
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{})
	return slc
}
