package job

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
)

// AirQualityJob gets average Co2 level at current point in time
type AirQualityJob struct {
	BaseJob
	client  traits.AirQualitySensorApiClient
	Sensors []string
}

func (a *AirQualityJob) Do(ctx context.Context, sendFn sender) error {
	sum := float32(0)
	count := 0

	for _, sensor := range a.Sensors {
		cctx, cancel := context.WithTimeout(ctx, a.Timeout.Or(defaultTimeout))

		resp, err := a.client.GetAirQuality(cctx, &traits.GetAirQualityRequest{Name: sensor})

		cancel()
		if err != nil {
			a.Logger.Error("getting air quality", zap.String("sensor", sensor), zap.Error(err))
			continue
		}

		count++
		sum += *resp.CarbonDioxideLevel
	}

	if count == 0 {
		return errors.Wrap(errNoSensorsRetrieved, "getting air quality")
	}

	average := sum / float32(count)

	body := &types.AverageCo2{
		Meta: types.Meta{
			Site:      a.Site,
			Timestamp: time.Now(),
		},
		AverageCo2: types.Float32Measure{
			Value: float32(math.Floor(float64(average))),
			Units: "ppm",
		},
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return sendFn(ctx, a.Url, bytes)
}
