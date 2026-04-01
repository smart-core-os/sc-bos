package extendretractpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	extension *resource.Value // of *Extension
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Extension{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		extension: resource.NewValue(opts...),
	}
}

func (m *Model) GetExtension(opts ...resource.ReadOption) (*Extension, error) {
	return m.extension.Get(opts...).(*Extension), nil
}

func (m *Model) UpdateExtension(extension *Extension, opts ...resource.WriteOption) (*Extension, error) {
	v, err := m.extension.Set(extension, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Extension), nil
}

func (m *Model) PullExtensions(ctx context.Context, opts ...resource.ReadOption) <-chan PullExtensionsChange {
	send := make(chan PullExtensionsChange)
	recv := m.extension.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullExtensionsChange{
				Value:      change.Value.(*Extension),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullExtensionsChange struct {
	Value      *Extension
	ChangeTime time.Time
}
