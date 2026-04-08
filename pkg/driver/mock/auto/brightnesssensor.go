package auto

import (
	"context"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func BrightnessSensorAuto(model *brightnesssensorpb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(10 * time.Second)
		go func() {
			initialLux := float32(300 + rand.Float32()*200) // Initial lux between 300 and 500
			state := &brightnesssensorpb.AmbientBrightness{BrightnessLux: initialLux}
			_, _ = model.UpdateAmbientBrightness(state)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					state, err := model.GetAmbientBrightness()
					if err == nil {
						newLux := state.BrightnessLux + (rand.Float32()*20 - 10) // ±10 lux drift
						if newLux < 0 {
							newLux = 0
						}
						state.BrightnessLux = newLux
						_, _ = model.UpdateAmbientBrightness(state, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
							Paths: []string{"brightness_lux"},
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
