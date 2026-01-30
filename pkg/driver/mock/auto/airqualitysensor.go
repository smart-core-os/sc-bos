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
	co2 := rand.Float32() * 1000
	voc := rand.Float32()
	ap := rand.Float32() * 1200
	ir := float32(rand.Int31n(100))
	score := float32(rand.Int31n(100))
	return &traits.AirQuality{
		CarbonDioxideLevel:       &co2,
		VolatileOrganicCompounds: &voc,
		AirPressure:              &ap,
		Comfort:                  0,
		InfectionRisk:            &ir,
		Score:                    &score,
	}
}
