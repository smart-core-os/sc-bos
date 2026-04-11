package auto

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func AirQualitySensorAuto(model *airqualitysensorpb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				s := GetAirQualityState()
				_, _ = model.UpdateAirQuality(s)
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

func GetAirQualityState() *airqualitysensorpb.AirQuality {
	score := float32(rand.Int32N(100))
	comfort := airqualitysensorpb.AirQuality_COMFORTABLE
	if score < 70 {
		comfort = airqualitysensorpb.AirQuality_UNCOMFORTABLE
	}
	return &airqualitysensorpb.AirQuality{
		CarbonDioxideLevel:       new(rand.Float32() * 1000),
		VolatileOrganicCompounds: new(rand.Float32()),
		AirPressure:              new(rand.Float32() * 1200),
		ParticulateMatter_1:      new(rand.Float32() * 1000),
		ParticulateMatter_25:     new(rand.Float32() * 1000),
		ParticulateMatter_10:     new(rand.Float32() * 1000),
		Comfort:                  comfort,
		InfectionRisk:            new(float32(rand.Int32N(100))),
		Score:                    &score,
	}
}
