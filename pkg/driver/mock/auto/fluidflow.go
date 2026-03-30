package auto

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func FluidFlow(model *fluidflowpb.Model) service.Lifecycle {
	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			for {
				direction := oneOf(fluidflowpb.FluidFlow_FLOW, fluidflowpb.FluidFlow_RETURN, fluidflowpb.FluidFlow_BLOCKING)

				state := &fluidflowpb.FluidFlow{
					FlowRate:             new(float32Between(1, 100)),
					DriveFrequency:       new(float32Between(0, 100)),
					TargetFlowRate:       new(float32Between(1, 100)),
					TargetDriveFrequency: new(float32Between(0, 100)),
					Direction:            direction,
				}

				if direction == fluidflowpb.FluidFlow_BLOCKING {
					state.FlowRate = new(float32(0))
					state.TargetFlowRate = new(float32(0))
					state.DriveFrequency = new(float32(0))
					state.TargetDriveFrequency = new(float32(0))
				}

				_, _ = model.UpdateFluidFlow(state)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer(durationBetween(time.Minute, 30*time.Minute))
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) { return string(data), nil }))
	_, _ = s.Configure([]byte{}) // ensure when start is called it actually starts
	return s
}
