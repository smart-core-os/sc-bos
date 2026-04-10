package auto

// ButtonAuto simulates an office push-button — e.g. a light switch, doorbell, or room service call button.
// During business hours a press is generated every 30–120 seconds. Each press is modelled as:
//  1. The button state transitions to PRESSED with a StateChangeTime.
//  2. After a 150 ms hold (a typical finger press), the state returns to UNPRESSED and a gesture is recorded.
//
// 80% of presses are single clicks (CLICK, Count=1); 20% are double clicks (CLICK, Count=2).
// Outside business hours no presses are generated and the button stays UNPRESSED.

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// ButtonAuto returns a Lifecycle that drives a Button model to simulate office push-button presses.
// Press events occur every 30–120 s during business hours; each press lasts ~150 ms before release.
func ButtonAuto(model *buttonpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 120*time.Second))
			defer timer.Stop()
			for {
				timer.Reset(durationBetween(30*time.Second, 120*time.Second))
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					factor := scale.NineToFive.Now()
					if factor <= 0.3 {
						// Outside business hours: no button activity.
						continue
					}
					// Simulate the button being pressed.
					pressTime := time.Now()
					_, _ = model.UpdateButtonState(&buttonpb.ButtonState{
						State:           buttonpb.ButtonState_PRESSED,
						StateChangeTime: timestamppb.New(pressTime),
					})
					// Hold the button for ~150 ms, then release.
					select {
					case <-ctx.Done():
						return
					case <-time.After(150 * time.Millisecond):
					}
					// 80% single click, 20% double click.
					count := int32(1)
					if randomBool(0.2) {
						count = 2
					}
					releaseTime := time.Now()
					_, _ = model.UpdateButtonState(&buttonpb.ButtonState{
						State:           buttonpb.ButtonState_UNPRESSED,
						StateChangeTime: timestamppb.New(releaseTime),
						MostRecentGesture: &buttonpb.ButtonState_Gesture{
							Kind:      buttonpb.ButtonState_Gesture_CLICK,
							Count:     count,
							StartTime: timestamppb.New(pressTime),
							EndTime:   timestamppb.New(releaseTime),
						},
					})
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
