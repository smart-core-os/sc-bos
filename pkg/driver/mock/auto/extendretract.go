package auto

// ExtendRetractAuto simulates automated solar-shading blinds on an office window.
// The blinds retract (0%) before the work day and after dusk to maximise daylight,
// then extend progressively during core hours (up to 70%) to reduce glare and solar heat gain.
// A small random drift (±5%) represents the stepwise nature of motorised blind actuators.
// Updates every 10–30 minutes, matching the typical blind-adjustment interval in a BMS schedule.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/extendretractpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// ExtendRetractAuto returns a Lifecycle that drives an ExtendRetract model to simulate window blinds.
// extend_percent = NineToFive * 70 + random drift of ±5, clamped to [0, 100].
// 0% = blind fully retracted (open); 100% = blind fully extended (closed).
func ExtendRetractAuto(model *extendretractpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(10*time.Minute, 30*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := float32(scale.NineToFive.Now())
				// Blinds extend up to 70% at peak to reduce solar glare; retract outside hours.
				pct := factor*70 + float32Between(-5, 5)
				if pct < 0 {
					pct = 0
				} else if pct > 100 {
					pct = 100
				}
				_, _ = model.UpdateExtension(
					&extendretractpb.Extension{ExtendPercent: pct},
					resource.WithUpdatePaths("extend_percent"),
				)
			}
			update()
			for {
				timer.Reset(durationBetween(10*time.Minute, 30*time.Minute))
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
