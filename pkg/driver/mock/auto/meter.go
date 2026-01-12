package auto

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func MeterAuto(model *meter.Model) *service.Service[string] {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			lastT := time.Now()
			start := timestamppb.New(lastT)
			var value float32
			for {
				select {
				case <-ctx.Done():
					return
				case t := <-timer.C:
					tod := scale.NineToFive.At(t)
					// typical daily household usage is 8 kWh, with TOD adjustment this is close enough
					kwh := float64Between(5, 15) * tod
					value += float32(t.Sub(lastT).Hours() / 24 * kwh)
					state := meterpb.MeterReading{
						Usage:     value,
						StartTime: start,
						EndTime:   timestamppb.Now(),
					}
					_, _ = model.UpdateMeterReading(&state)
					timer = time.NewTimer(durationBetween(time.Minute, 30*time.Minute))
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{}) // call configure to ensure we load when start is called.
	return slc
}
