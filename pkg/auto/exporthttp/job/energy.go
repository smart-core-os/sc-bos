package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// EnergyJob gets the energy consumed over the previous execution interval (typically 24 hours)
type EnergyJob struct {
	BaseJob
	client     meterpb.MeterHistoryClient
	infoClient meterpb.MeterInfoClient
	Meters     []string
}

func (e *EnergyJob) Do(ctx context.Context, sendFn sender) error {
	consumption := float32(.0)

	now := time.Now().UTC()
	filterTime := now.Sub(e.PreviousExecution.UTC())

	for _, meter := range e.Meters {
		cctx, cancel := context.WithTimeout(ctx, e.Timeout.Or(defaultTimeout))

		multiplier, err := e.getUnitMultiplier(cctx, meter)

		if err != nil {
			// Summing a reading in an unknown unit as if it were kWh would quietly misreport the total.
			e.Logger.Warn("skipping meter, can't convert its reading to kWh", zap.String("meter", meter), zap.Error(err))
			cancel()
			continue
		}

		earliest, latest, err := getRecordsByTime(cctx, e.Logger, e.client.ListMeterReadingHistory, meter, now, filterTime)

		cancel()

		if err != nil {
			e.Logger.Warn("getting records by time", zap.String("meter", meter), zap.Error(err))
			continue
		}

		consumption += processMeterRecords(multiplier, earliest, latest)
	}

	roundedConsumption := float32(math.Floor(float64(consumption)))

	if roundedConsumption <= 0 {
		e.Logger.Debug("no energy consumption found, skipping post")
		return nil
	}

	body := &types.EnergyConsumption{
		Meta: types.Meta{
			Timestamp: now,
			Site:      e.Site,
		},
		TodaysEnergyConsumption: types.Float32Measure{
			Value: roundedConsumption,
			// every meter summed above was converted to kWh, meters we couldn't convert were skipped
			Units: "kWh",
		},
	}

	bytes, err := json.Marshal(body)

	if err != nil {
		return err
	}

	return sendFn(ctx, e.Url, bytes)
}

// getUnitMultiplier returns the factor that converts meter's usage readings to kWh,
// or an error if the meter can't be described or its unit isn't an energy unit we know.
func (e *EnergyJob) getUnitMultiplier(ctx context.Context, meter string) (float32, error) {
	infoResp, err := e.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: meter})
	if err != nil {
		return 0, fmt.Errorf("describe meter: %w", err)
	}
	return energyMultiplier(infoResp.GetUsageUnit())
}

func energyMultiplier(unit string) (float32, error) {
	switch dbo.UnitName(unit) {
	case dbo.UnitWattHours:
		return 1.0 / 1_000, nil
	case dbo.UnitKilowattHours:
		return 1, nil
	case dbo.UnitMegawattHours:
		return 1_000, nil
	}
	if unit == "" {
		return 0, errors.New("meter has no usage unit")
	}
	return 0, fmt.Errorf("unsupported usage unit %q", unit)
}
