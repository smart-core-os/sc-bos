package auto

// LightAuto simulates dimmable overhead office lighting (e.g. a ceiling LED panel with DALI control).
// Brightness tracks business hours: at full intensity (~80%) during core hours, gradually dims
// during the ramp-up and wind-down periods, and drops to near-zero outside working hours.
// A small random drift (±5%) adds realistic variation from cloud cover, blind position, etc.
// Updates every 1–3 minutes, representing the dimming step rate of a typical DALI ballast.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// LightAuto returns a Lifecycle that drives a Light model to simulate office overhead lighting.
// level_percent = NineToFive * 80 + random drift of ±5, clamped to [0, 100].
func LightAuto(model *lightpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(1*time.Minute, 3*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := float32(scale.NineToFive.Now())
				// Target ~80% at peak; drift ±5% for ambient variation.
				level := factor*80 + float32Between(-5, 5)
				if level < 0 {
					level = 0
				} else if level > 100 {
					level = 100
				}
				_, _ = model.UpdateBrightness(
					&lightpb.Brightness{LevelPercent: level},
					resource.WithUpdatePaths("level_percent"),
				)
			}
			update()
			for {
				timer.Reset(durationBetween(1*time.Minute, 3*time.Minute))
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					update()
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
