package auto

import (
	"context"
	"math"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

func OccupancySensorAuto(model *occupancysensorpb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				tod := scale.NineToFive.Now()
				peopleCount := int32(math.Round(tod * float64Between(0, 10)))
				occupancy := &traits.Occupancy{PeopleCount: peopleCount}
				if peopleCount == 0 {
					occupancy.State = oneOf(traits.Occupancy_UNOCCUPIED, traits.Occupancy_IDLE)
				} else {
					occupancy.State = traits.Occupancy_OCCUPIED
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
