package auto

import (
	"context"
	"time"

	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait/fanspeedpb"
)

func FanSpeed(model *fanspeedpb.Model, presets ...fanspeedpb.Preset) service.Lifecycle {
	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
			for {
				state := &traits.FanSpeed{
					Direction: oneOf(traits.FanSpeed_FORWARD, traits.FanSpeed_BACKWARD),
				}
				if len(presets) > 0 {
					// pick a new value from the presets
					preset := oneOf(presets...)
					state.Preset = preset.Name
				} else {
					// pick a random percentage
					state.Percentage = float32Between(0, 100)
				}
				_, _ = model.UpdateFanSpeed(state)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
				}
			}
		}()
		return nil
	}))
	_, _ = s.Configure([]byte(`""`)) // ensure when start is called it actually starts
	return s
}
