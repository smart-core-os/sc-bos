package auto

import (
	"context"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func SoundSensorAuto(model *soundsensorpb.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(10 * time.Second)
		go func() {
			state := &soundsensorpb.SoundLevel{
				SoundPressureLevel: new(20 + rand.Float32()*20),
			}
			_, _ = model.UpdateSoundLevel(state)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					state, err := model.GetSoundLevel()
					if err == nil {
						state.SoundPressureLevel = new(*state.SoundPressureLevel + (rand.Float32()*4 - 2))
						_, _ = model.UpdateSoundLevel(state, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
							Paths: []string{"sound_pressure_level"},
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
