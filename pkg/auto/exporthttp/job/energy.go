package job

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
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
			e.Logger.Error("getting unit multiplier", zap.String("meter", meter), zap.Error(err))
		}

		earliest, latest, err := getRecordsByTime(cctx, e.Logger, e.client.ListMeterReadingHistory, meter, now, filterTime)

		cancel()

		if err != nil {
			e.Logger.Error("getting records by time", zap.String("meter", meter), zap.Error(err))
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
			Units: "kWh",
		},
	}

	bytes, err := json.Marshal(body)

	if err != nil {
		return err
	}

	return sendFn(ctx, e.Url, bytes)
}

func (e *EnergyJob) getUnitMultiplier(ctx context.Context, meter string) (float32, error) {
	infoResp, err := e.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: meter})

	if err != nil {
		return 1, err
	}

	//  convert reading to kWh
	var multiplier float32

	switch strings.ToLower(infoResp.GetUsageUnit()) {
	case "wh":
		multiplier = 1 / 1_000
	case "mwh":
		multiplier = 1_000
	case "kwh":
		fallthrough
	default:
		multiplier = 1
	}

	return multiplier, nil
}
