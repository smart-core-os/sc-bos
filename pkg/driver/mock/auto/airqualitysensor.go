package auto

import (
	"context"
	"time"

	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
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

func GetAirQualityState() *traits.AirQuality {
	co2 := 400 + rand.Float32()*200
	voc := rand.Float32() * 0.3
	ap := 1000 + rand.Float32()*26.5
	ir := rand.Float32() * 30
	score := 80 + rand.Float32()*20
	pm1 := rand.Float32() * 15
	pm25 := rand.Float32() * 12
	pm10 := rand.Float32() * 54
	return &traits.AirQuality{
		CarbonDioxideLevel:       &co2,
		VolatileOrganicCompounds: &voc,
		AirPressure:              &ap,
		Comfort:                  0,
		InfectionRisk:            &ir,
		Score:                    &score,
		ParticulateMatter_1:      &pm1,
		ParticulateMatter_25:     &pm25,
		ParticulateMatter_10:     &pm10,
	}
}
