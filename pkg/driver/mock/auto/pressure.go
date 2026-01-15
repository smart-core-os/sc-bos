package auto

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/pressurepb"
	gen_pressurepb "github.com/smart-core-os/sc-bos/pkg/proto/pressurepb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func Pressure(model *pressurepb.Model) service.Lifecycle {
	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			for {
				state := &gen_pressurepb.Pressure{
					Pressure:       ptr(float32Between(0, 100)),
					TargetPressure: ptr(float32Between(0, 100)),
				}
				_, _ = model.UpdatePressure(state)

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
