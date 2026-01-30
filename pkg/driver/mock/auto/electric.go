package auto

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb"
)

func Electric(model *electricpb.Model) service.Lifecycle {
	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			for {
				tod := float32(scale.NineToFive.Now())
				state := &traits.ElectricDemand{
					Current:     float32Between(20, 40) * tod,
					Voltage:     ptr(float32Between(238, 243)),
					PowerFactor: ptr(float32Between(0.7, 1.3)),
				}
				state.ApparentPower = ptr(state.Current * *state.Voltage)
				state.RealPower = ptr(*state.ApparentPower * *state.PowerFactor)
				state.ReactivePower = ptr(*state.ApparentPower * (1 - *state.PowerFactor))
				_, _ = model.UpdateDemand(state)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer(durationBetween(time.Minute, 30*time.Minute))
				}
			}
		}()
		return nil
	}))
	_, _ = s.Configure([]byte(`""`)) // ensure when start is called it actually starts
	return s
}
