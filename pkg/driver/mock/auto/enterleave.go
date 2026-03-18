package auto

import (
	"context"
	"math/rand/v2"
	"time"

	gen_enterleavesensorpb "github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func EnterLeaveAuto(model *gen_enterleavesensorpb.Model) *service.Service[string] {
	occupant := []*gen_enterleavesensorpb.EnterLeaveEvent_Occupant{
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
				direction := gen_enterleavesensorpb.EnterLeaveEvent_DIRECTION_UNSPECIFIED
				if chance < 0.5 {
					enterCount++
					direction = gen_enterleavesensorpb.EnterLeaveEvent_ENTER
				} else {
					leaveCount++
					direction = gen_enterleavesensorpb.EnterLeaveEvent_LEAVE
				}

				enterLeave := &gen_enterleavesensorpb.EnterLeaveEvent{
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
