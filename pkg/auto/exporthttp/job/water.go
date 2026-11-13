package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// WaterJob gets the water consumed over the previous execution interval (typically 24 hours)
type WaterJob struct {
	BaseJob
	client     meterpb.MeterHistoryClient
	infoClient meterpb.MeterInfoClient
	Meters     []string
}

func (w *WaterJob) Do(ctx context.Context, sendFn sender) error {
	consumption := float32(.0)

	now := time.Now().UTC()
	filterTime := now.Sub(w.PreviousExecution.UTC())

	for _, meter := range w.Meters {
		cctx, cancel := context.WithTimeout(ctx, w.Timeout.Or(defaultTimeout))

		multiplier, err := w.getUnitMultiplier(cctx, meter)

		if err != nil {
			// Summing a reading in an unknown unit as if it were litres would quietly misreport the total.
			w.Logger.Warn("skipping meter, can't convert its reading to litres", zap.String("meter", meter), zap.Error(err))
			cancel()
			continue
		}

		earliest, latest, err := getRecordsByTime(cctx, w.Logger, w.client.ListMeterReadingHistory, meter, now, filterTime)

		cancel()
		if err != nil {
			w.Logger.Warn("getting records by time", zap.String("meter", meter), zap.Error(err))
			continue
		}

		consumption += processMeterRecords(multiplier, earliest, latest)
	}

	if consumption <= 0 {
		w.Logger.Debug("no water consumption found, skipping post")
		return nil
	}

	body := &types.WaterConsumption{
		Meta: types.Meta{
			Timestamp: now,
			Site:      w.Site,
		},
		TodaysWaterConsumption: types.IntMeasure{
			Value: int32(consumption),
			// every meter summed above was converted to litres, meters we couldn't convert were skipped
			Units: "litres",
		},
	}

	bytes, err := json.Marshal(body)

	if err != nil {
		return err
	}

	return sendFn(ctx, w.Url, bytes)
}

// getUnitMultiplier returns the factor that converts meter's usage readings to litres,
// or an error if the meter can't be described or its unit isn't a volume unit we know.
func (w *WaterJob) getUnitMultiplier(ctx context.Context, meter string) (float32, error) {
	infoResp, err := w.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: meter})
	if err != nil {
		return 0, fmt.Errorf("describe meter: %w", err)
	}
	return waterMultiplier(infoResp.GetUsageUnit())
}

func waterMultiplier(unit string) (float32, error) {
	if dbo.UnitName(unit) == dbo.UnitCubicMeters {
		return 1_000, nil
	}
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "l", "litre", "litres", "liter", "liters":
		return 1, nil
	case "cm3", "cm³":
		return 1.0 / 1_000, nil
	case "":
		return 0, errors.New("meter has no usage unit")
	}
	return 0, fmt.Errorf("unsupported usage unit %q", unit)
}
