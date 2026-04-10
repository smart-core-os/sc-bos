package ptzpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	ptz *resource.Value // of *Ptz
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Ptz{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		ptz: resource.NewValue(opts...),
	}
}

func (m *Model) GetPtz(opts ...resource.ReadOption) (*Ptz, error) {
	return m.ptz.Get(opts...).(*Ptz), nil
}

func (m *Model) UpdatePtz(ptz *Ptz, opts ...resource.WriteOption) (*Ptz, error) {
	v, err := m.ptz.Set(ptz, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Ptz), nil
}

func (m *Model) PullPtz(ctx context.Context, opts ...resource.ReadOption) <-chan PullPtzChange {
	send := make(chan PullPtzChange)
	recv := m.ptz.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullPtzChange{
				Value:      change.Value.(*Ptz),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullPtzChange struct {
	Value      *Ptz
	ChangeTime time.Time
}
