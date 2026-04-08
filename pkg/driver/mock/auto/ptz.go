package auto

// PtzAuto simulates a pan-tilt-zoom camera in an office — e.g. a security camera or a conference room
// VC camera (such as a Logitech Rally or Poly Studio). During business hours the camera cycles
// through named preset positions representing different office viewpoints (door, desk area, whiteboard,
// and an overview wide shot). Outside business hours it parks at the "overview" preset.
// Updates every 5–15 minutes, representing a guard tour or an auto-framing transition between calls.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// ptzPresets represents named camera positions typically configured on an office PTZ camera.
var ptzPresets = []string{
	"door",       // facing the room entrance — useful for access monitoring
	"desk-area",  // covering the main desk cluster
	"whiteboard", // framed on the presentation whiteboard
	"overview",   // wide shot of the whole room — used as the parked/idle position
}

// PtzAuto returns a Lifecycle that drives a Ptz model to simulate an office PTZ camera.
// During business hours the camera cycles through preset positions every 5–15 minutes;
// outside hours it parks at the "overview" preset.
func PtzAuto(model *ptzpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(5*time.Minute, 15*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				preset := "overview" // parked position outside business hours
				if factor > 0.3 {
					preset = oneOf(ptzPresets...)
				}
				_, _ = model.UpdatePtz(
					&ptzpb.Ptz{Preset: preset},
					resource.WithUpdatePaths("preset"),
				)
			}
			update()
			for {
				timer.Reset(durationBetween(5*time.Minute, 15*time.Minute))
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
