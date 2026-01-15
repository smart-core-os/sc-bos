package auto

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/fluidflowpb"
	gen_fluidflowpb "github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func FluidFlow(model *fluidflowpb.Model) service.Lifecycle {
	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			for {
				direction := oneOf(gen_fluidflowpb.FluidFlow_FLOW, gen_fluidflowpb.FluidFlow_RETURN, gen_fluidflowpb.FluidFlow_BLOCKING)

				state := &gen_fluidflowpb.FluidFlow{
					FlowRate:             ptr(float32Between(1, 100)),
					DriveFrequency:       ptr(float32Between(0, 100)),
					TargetFlowRate:       ptr(float32Between(1, 100)),
					TargetDriveFrequency: ptr(float32Between(0, 100)),
					Direction:            direction,
				}

				if direction == gen_fluidflowpb.FluidFlow_BLOCKING {
					state.FlowRate = ptr(float32(0))
					state.TargetFlowRate = ptr(float32(0))
					state.DriveFrequency = ptr(float32(0))
					state.TargetDriveFrequency = ptr(float32(0))
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
