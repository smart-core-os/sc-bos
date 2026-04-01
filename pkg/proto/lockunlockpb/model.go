package lockunlockpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	lockUnlock *resource.Value // of *LockUnlock
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&LockUnlock{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		lockUnlock: resource.NewValue(opts...),
	}
}

func (m *Model) GetLockUnlock(opts ...resource.ReadOption) (*LockUnlock, error) {
	return m.lockUnlock.Get(opts...).(*LockUnlock), nil
}

func (m *Model) UpdateLockUnlock(lockUnlock *LockUnlock, opts ...resource.WriteOption) (*LockUnlock, error) {
	v, err := m.lockUnlock.Set(lockUnlock, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*LockUnlock), nil
}

func (m *Model) PullLockUnlock(ctx context.Context, opts ...resource.ReadOption) <-chan PullLockUnlockChange {
	send := make(chan PullLockUnlockChange)
	recv := m.lockUnlock.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullLockUnlockChange{
				Value:      change.Value.(*LockUnlock),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullLockUnlockChange struct {
	Value      *LockUnlock
	ChangeTime time.Time
}
