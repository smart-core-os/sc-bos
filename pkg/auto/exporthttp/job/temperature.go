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

// TemperatureJob gets average air temperature in Celsius at current point in time
type TemperatureJob struct {
	BaseJob
	client  traits.AirTemperatureApiClient
	Sensors []string
}

func (t *TemperatureJob) Do(ctx context.Context, sendFn sender) error {
	sum := .0
	count := 0

	for _, sensor := range t.Sensors {
		cctx, cancel := context.WithTimeout(ctx, t.Timeout.Or(defaultTimeout))

		resp, err := t.client.GetAirTemperature(cctx, &traits.GetAirTemperatureRequest{Name: sensor})
		cancel()

		if err != nil {
			t.Logger.Error("getting air temperature", zap.String("sensor", sensor), zap.Error(err))
			continue
		}
		count++

		sum += resp.GetAmbientTemperature().GetValueCelsius()
	}

	if count == 0 {
		return errors.Wrap(errNoSensorsRetrieved, "getting air temperature")
	}

	average := sum / float64(count)

	body := &types.AverageTemperature{
		Meta: types.Meta{
			Site:      t.Site,
			Timestamp: time.Now(),
		},
		AverageTemperature: types.Float64Measure{
			Value: math.Round(average*10) / 10,
			Units: "°C",
		},
	}

	bytes, err := json.Marshal(body)

	if err != nil {
		return err
	}

	return sendFn(ctx, t.Url, bytes)
}
