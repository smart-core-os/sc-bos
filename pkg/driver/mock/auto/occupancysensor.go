package auto

import (
	"context"
	"math"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func OccupancySensorAuto(model *occupancysensorpb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				tod := scale.NineToFive.Now()
				// Two-stage: first decide if anyone is present (probability = tod/2,
				// so even at peak hours there's ~50% chance of 0), then pick a count.
				var peopleCount int32
				if randomBool(tod * 0.50) {
					peopleCount = int32(math.Round(float64Between(1, 200) * tod))
				}
				occupancy := &occupancysensorpb.Occupancy{PeopleCount: peopleCount}
				if peopleCount == 0 {
					occupancy.State = oneOf(occupancysensorpb.Occupancy_UNOCCUPIED, occupancysensorpb.Occupancy_IDLE)
				} else {
					occupancy.State = occupancysensorpb.Occupancy_OCCUPIED
				}
				_, _ = model.SetOccupancy(occupancy, resource.WithUpdatePaths("state", "people_count"))
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{}) // call configure to ensure we load when start is called.
	return slc
}
