package auto

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait/enterleavesensorpb"
)

func EnterLeaveAuto(model *enterleavesensorpb.Model) *service.Service[string] {
	occupant := []*traits.EnterLeaveEvent_Occupant{
		nil,
		{DisplayName: "Scott Lang", Ids: map[string]string{"card": "1234567890"}},
		{DisplayName: "Hope Van Dyne", Ids: map[string]string{"card": "0987654321"}},
		{DisplayName: "Janet Van Dyne", Ids: map[string]string{"card": "1234567890"}},
	}

	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			enterCount := int32(0)
			leaveCount := int32(0)

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				chance := rand.Float32()
				direction := traits.EnterLeaveEvent_DIRECTION_UNSPECIFIED
				if chance < 0.5 {
					enterCount++
					direction = traits.EnterLeaveEvent_ENTER
				} else {
					leaveCount++
					direction = traits.EnterLeaveEvent_LEAVE
				}

				enterLeave := &traits.EnterLeaveEvent{
					Direction:  direction,
					Occupant:   occupant[rand.IntN(len(occupant))],
					EnterTotal: &enterCount,
					LeaveTotal: &leaveCount,
				}

				_ = model.CreateEnterLeaveEvent(enterLeave)

				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) { return string(data), nil }))
	_, _ = slc.Configure([]byte{}) // call configure to ensure we load when start is called.

	return slc
}
