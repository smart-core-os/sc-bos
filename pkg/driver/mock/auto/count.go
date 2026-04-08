package auto

// CountAuto simulates an optical people counter or access event counter above a doorway.
// People (or events) accumulate during business hours at a rate of 1–4 per tick, scaled by
// the NineToFive factor to reflect higher footfall during core hours and a quiet overnight period.
// The counter is monotonically increasing for the lifetime of the service;
// use the ResetCount API to reset it manually when needed.
// Updates every 1–5 minutes, matching the reporting interval of a typical turnstile or door counter.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/countpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// CountAuto returns a Lifecycle that drives a Count model to simulate a doorway people counter.
// Each tick increments added by floor(factor * float32Between(1, 4)), accumulating over time.
func CountAuto(model *countpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			var totalAdded int32
			timer := time.NewTimer(durationBetween(1*time.Minute, 5*time.Minute))
			defer timer.Stop()
			for {
				timer.Reset(durationBetween(1*time.Minute, 5*time.Minute))
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					factor := float32(scale.NineToFive.Now())
					// Scale the increment by the time-of-day factor; quiet overnight.
					delta := int32(factor * float32Between(1, 4))
					if delta > 0 {
						totalAdded += delta
						_, _ = model.UpdateCount(&countpb.Count{Added: totalAdded})
					}
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
