package auto

import (
	"context"
	"time"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func TemperatureAuto(model *temperaturepb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(30 * time.Second)
		go func() {

			initialTemp := 12 + rand.Float64()*3
			state := &gen.Temperature{
				Measured: &types.Temperature{
					ValueCelsius: initialTemp,
				},
			}
			_, _ = model.UpdateTemperature(state)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					state, err := model.GetTemperature()
					if err == nil {

						currentTemp := state.GetMeasured().GetValueCelsius()
						variation := rand.Float64()*2 - 1
						state.Measured.ValueCelsius = currentTemp + variation
						_, _ = model.UpdateTemperature(state, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
							Paths: []string{"measured.value_celsius"},
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
