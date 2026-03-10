package pressurepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	pressure *resource.Value // of *pressurepb.Pressure
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Pressure{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		pressure: resource.NewValue(opts...),
	}
}

func (m *Model) GetPressure() (*Pressure, error) {
	return m.pressure.Get().(*Pressure), nil
}

func (m *Model) UpdatePressure(pressure *Pressure, opts ...resource.WriteOption) (*Pressure, error) {
	res, err := m.pressure.Set(pressure, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*Pressure), nil
}

func (m *Model) PullPressure(ctx context.Context, opts ...resource.ReadOption) <-chan PullPressureChange {
	return resources.PullValue[*Pressure](ctx, m.pressure.Pull(ctx, opts...))
}

type PullPressureChange = resources.ValueChange[*Pressure]
