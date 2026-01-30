package auto

import (
	"context"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/openclosepb"
)

func OpenClose(model *openclosepb.Model) service.Lifecycle {
	resistances := maps.Values(traits.OpenClosePosition_Resistance_value)
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
			for {
				presets := model.ListPresets()
				if len(presets) > 0 {
					preset := oneOf(presets...)
					_, _ = model.UpdatePositions(&traits.OpenClosePositions{Preset: preset})
				} else {
					state := &traits.OpenClosePosition{
						OpenPercent: float32(rand.Intn(100 + 1)),
						Resistance:  traits.OpenClosePosition_Resistance(resistances[rand.Intn(len(resistances))]),
					}
					_, _ = model.UpdatePosition(state, resource.WithCreateIfAbsent())
				}

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
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
