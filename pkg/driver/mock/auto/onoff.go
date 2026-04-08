package auto

// OnOffAuto simulates an office light or piece of equipment — e.g. a projector, desk lamp, or HVAC unit —
// that is switched on during business hours and off outside of them.
// A small random flip (10%) represents manual overrides, so the state is not strictly deterministic.
// Updates every 5–10 minutes, matching the cadence of a light toggled occasionally throughout the day.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// OnOffAuto returns a Lifecycle that drives an OnOff model to simulate office equipment behaviour.
// The device turns on when NineToFive > 0.3 (core business hours) and off otherwise,
// with a 10% random inversion to represent manual overrides.
func OnOffAuto(model *onoffpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(5*time.Minute, 10*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				// On during core hours; 10% chance of the opposite to simulate manual control.
				expectedOn := factor > 0.3
				if randomBool(0.1) {
					expectedOn = !expectedOn
				}
				state := onoffpb.OnOff_OFF
				if expectedOn {
					state = onoffpb.OnOff_ON
				}
				_, _ = model.UpdateOnOff(&onoffpb.OnOff{State: state})
			}
			update()
			for {
				timer.Reset(durationBetween(5*time.Minute, 10*time.Minute))
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
