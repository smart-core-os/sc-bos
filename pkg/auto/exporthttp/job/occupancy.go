package job

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
)

// OccupancyJob gets the occupancy at the current point in time
type OccupancyJob struct {
	BaseJob
	client  traits.OccupancySensorApiClient
	Sensors []string
}

func (o *OccupancyJob) Do(ctx context.Context, sendFn sender) error {
	sum := int32(0)
	hasCounted := false

	for _, sensor := range o.Sensors {
		cctx, cancel := context.WithTimeout(ctx, o.Timeout.Or(defaultTimeout))

		resp, err := o.client.GetOccupancy(cctx, &traits.GetOccupancyRequest{Name: sensor})
		cancel()

		if err != nil {
			o.Logger.Error("getting occupancy", zap.String("sensor", sensor), zap.Error(err))
			continue
		}

		// confidence value semantics can vary between driver implementations
		// 0.2 can be a bad threshold. We will assume it isn't for exporthttp
		if resp.GetConfidence() > 0.2 {
			hasCounted = true
			sum += resp.GetPeopleCount()
		}
	}

	// don't submit a reading if we aren't confident for any people counts
	if !hasCounted {
		o.Logger.Debug("no occupancy counts with sufficient confidence found, skipping post")
		return nil
	}

	body := &types.TotalOccupancy{
		Meta: types.Meta{
			Site:      o.Site,
			Timestamp: time.Now(),
		},
		TotalOccupancy: types.IntMeasure{
			Value: sum,
			Units: "People",
		},
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return sendFn(ctx, o.Url, bytes)
}
