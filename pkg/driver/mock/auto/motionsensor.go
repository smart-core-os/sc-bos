package auto

// MotionSensorAuto simulates a passive infrared (PIR) motion sensor mounted in an office room.
// Motion is detected probabilistically: up to 50% chance during peak business hours,
// dropping to ~5% overnight and on weekends to represent occasional out-of-hours movement.
// Updates every 30 seconds, matching the typical reporting interval of a commercial PIR sensor.

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/motionsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// MotionSensorAuto returns a Lifecycle that drives a MotionSensor model to simulate office room occupancy.
// Detection probability scales with NineToFive: factor*0.5 during the day, a fixed 0.05 overnight.
func MotionSensorAuto(model *motionsensorpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				// Overnight background probability is 5% (security rounds, cleaners, etc.)
				probability := 0.05 + factor*0.45
				state := motionsensorpb.MotionDetection_NOT_DETECTED
				if randomBool(probability) {
					state = motionsensorpb.MotionDetection_DETECTED
				}
				_, _ = model.SetMotionDetection(&motionsensorpb.MotionDetection{
					State:           state,
					StateChangeTime: timestamppb.Now(),
				})
			}
			update()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
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
