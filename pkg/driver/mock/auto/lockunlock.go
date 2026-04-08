package auto

// LockUnlockAuto simulates an electronic strike lock on an office access door.
// The door is unlocked during business hours and re-locked outside of them,
// representing scheduled access control on a front or floor-entry door.
// Updates every 5–15 minutes so state transitions are visible but not noisy.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/lockunlockpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// LockUnlockAuto returns a Lifecycle that drives a LockUnlock model to simulate an access-controlled door.
// The door unlocks when NineToFive > 0.3 (business hours) and locks otherwise.
func LockUnlockAuto(model *lockunlockpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(5*time.Minute, 15*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				position := lockunlockpb.LockUnlock_LOCKED
				if factor > 0.3 {
					position = lockunlockpb.LockUnlock_UNLOCKED
				}
				_, _ = model.UpdateLockUnlock(&lockunlockpb.LockUnlock{Position: position})
			}
			update()
			for {
				timer.Reset(durationBetween(5*time.Minute, 15*time.Minute))
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
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
