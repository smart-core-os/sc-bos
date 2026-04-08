package auto

// SpeakerAuto simulates a ceiling loudspeaker in a conference room (e.g. a Bose DS series or similar).
// Volume varies between 30–70% during business hours to represent different presentation and huddle scenarios.
// A 10% random mute chance represents a presenter who has paused audio or a room in between calls.
// Outside business hours the speaker is muted and gain is set to zero.
// Updates every 5–15 minutes, matching the cadence of room changeovers between meetings.
//
// AudioLevel.Gain is expressed as a percentage (0–100) per the SmartCore types convention.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/speakerpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// SpeakerAuto returns a Lifecycle that drives a Speaker model to simulate a conference room loudspeaker.
// During business hours volume is 30–70% with a 10% mute probability; outside hours volume is 0 and muted.
func SpeakerAuto(model *speakerpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(5*time.Minute, 15*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				var gain float32
				var muted bool
				if factor > 0.3 {
					// Active meeting: volume 30–70%; occasionally muted between presentations.
					gain = float32Between(30, 70)
					muted = randomBool(0.10)
				} else {
					// Room unoccupied: speaker silenced.
					gain = 0
					muted = true
				}
				_, _ = model.UpdateVolume(
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
