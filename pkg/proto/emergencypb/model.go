package emergencypb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	emergency *resource.Value // of *Emergency
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Emergency{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		emergency: resource.NewValue(opts...),
	}
}

func (m *Model) GetEmergency(opts ...resource.ReadOption) (*Emergency, error) {
	return m.emergency.Get(opts...).(*Emergency), nil
}

func (m *Model) UpdateEmergency(emergency *Emergency, opts ...resource.WriteOption) (*Emergency, error) {
	v, err := m.emergency.Set(emergency, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Emergency), nil
}

func (m *Model) PullEmergency(ctx context.Context, opts ...resource.ReadOption) <-chan PullEmergencyChange {
	send := make(chan PullEmergencyChange)
	recv := m.emergency.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullEmergencyChange{
				Value:      change.Value.(*Emergency),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullEmergencyChange struct {
	Value      *Emergency
	ChangeTime time.Time
}
