package auto

// InputSelectAuto simulates a meeting room AV system input selector (e.g. an Extron or Crestron switcher).
// During business hours the active input cycles randomly through the common physical inputs found on a
// meeting room display: HDMI 1 (laptop), HDMI 2 (second laptop), Display Port (desktop), and Wireless
// (screen-share dongle). Outside business hours the input is left on HDMI 1 as a default standby.
// Updates every 30–90 minutes, matching the cadence of back-to-back meeting slots.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/inputselectpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// inputs represents the physical video inputs available on a typical meeting room display.
var inputs = []string{"HDMI 1", "HDMI 2", "Display Port", "Wireless"}

// InputSelectAuto returns a Lifecycle that drives an InputSelect model to simulate a meeting room AV switcher.
// The video_input cycles among common inputs during business hours; defaults to HDMI 1 when the room is unused.
func InputSelectAuto(model *inputselectpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Minute, 90*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				input := inputs[0] // HDMI 1 — standby default
				if factor > 0.3 {
					// A meeting is in progress; pick whichever input the presenter connected to.
					input = oneOf(inputs...)
				}
				_, _ = model.UpdateInput(
					&inputselectpb.Input{VideoInput: input},
					resource.WithUpdatePaths("video_input"),
				)
			}
			update()
			for {
				timer.Reset(durationBetween(30*time.Minute, 90*time.Minute))
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
