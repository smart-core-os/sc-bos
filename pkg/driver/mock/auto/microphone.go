package auto

// MicrophoneAuto simulates a boundary or ceiling microphone in a conference room (e.g. a Shure MXA ceiling array).
// Gain varies between 50–80% during business hours to represent different meeting dynamics
// (a full-room presentation vs. a small huddle), with a 15% chance of being muted at any sample point.
// Outside business hours the microphone is muted and gain is set to zero.
// Updates every 5–15 minutes, matching the interval at which a room's audio configuration might change.
//
// AudioLevel.Gain is expressed as a percentage (0–100) per the SmartCore types convention.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/microphonepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// MicrophoneAuto returns a Lifecycle that drives a Microphone model to simulate a conference room mic.
// During business hours gain is 50–80% with a 15% mute probability; outside hours gain is 0 and muted.
func MicrophoneAuto(model *microphonepb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(5*time.Minute, 15*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				var gain float32
				var muted bool
				if factor > 0.3 {
					// Active meeting: gain between 50–80%; occasionally muted (presenter on hold, etc.)
					gain = float32Between(50, 80)
					muted = randomBool(0.15)
				} else {
					// Room unoccupied: mic silenced.
					gain = 0
					muted = true
				}
				_, _ = model.UpdateGain(
					&typespb.AudioLevel{Gain: gain, Muted: muted},
					resource.WithUpdatePaths("gain", "muted"),
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
