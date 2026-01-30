package job

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
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
			w.Logger.Error("getting unit multiplier", zap.String("meter", meter), zap.Error(err))
		}

		earliest, latest, err := getRecordsByTime(cctx, w.Logger, w.client.ListMeterReadingHistory, meter, now, filterTime)

		cancel()
		if err != nil {
			w.Logger.Error("getting records by time", zap.String("meter", meter), zap.Error(err))
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
			Units: "litres",
		},
	}

	bytes, err := json.Marshal(body)

	if err != nil {
		return err
	}

	return sendFn(ctx, w.Url, bytes)
}

func (w *WaterJob) getUnitMultiplier(ctx context.Context, meter string) (float32, error) {
	infoResp, err := w.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: meter})

	if err != nil {
		return 1, err
	}

	// convert reading to litres
	var multiplier float32

	switch infoResp.GetUsageUnit() {
	case "cm3": // TODO: these strings may need correcting I tried guessing them
		multiplier = 1 / 1_000_000
	case "m3":
		fallthrough
	case "litres":
		fallthrough
	default:
		multiplier = 1
	}

	return multiplier, nil
}
