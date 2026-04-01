package auto

import (
	"context"
	"math"
	"time"

	"math/rand/v2"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func AirTemperatureAuto(model *airtemperaturepb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(30 * time.Second)
		go func() {
			randomNumber := 18 + rand.Float64()*6
			// give each device a random set point between 18 and 24 with .05 degree accuracy
			setPoint := math.Round(randomNumber*2) / 2
			state := &airtemperaturepb.AirTemperature{
				AmbientTemperature: &typespb.Temperature{
					ValueCelsius: setPoint + (rand.Float64()*4 - 2),
				},
				TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
					TemperatureSetPoint: &typespb.Temperature{ValueCelsius: setPoint},
				},
			}
			_, _ = model.UpdateAirTemperature(state)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					state, err := model.GetAirTemperature()
					if err == nil {
						setPoint = state.GetTemperatureSetPoint().ValueCelsius
						// update the ambient to be +- 2 degrees from the set point
						state.AmbientTemperature.ValueCelsius = setPoint + (rand.Float64()*4 - 2)
						_, _ = model.UpdateAirTemperature(state, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
							Paths: []string{"ambient_temperature.value_celsius"},
						}))
					}
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{})
	return slc
}
