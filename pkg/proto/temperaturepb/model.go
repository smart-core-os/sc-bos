package temperaturepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	temperature *resource.Value // of *temperaturepb.Temperature
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Temperature{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		temperature: resource.NewValue(opts...),
	}
}

func (m *Model) GetTemperature(opts ...resource.ReadOption) (*Temperature, error) {
	return m.temperature.Get(opts...).(*Temperature), nil
}

func (m *Model) UpdateTemperature(temperature *Temperature, opts ...resource.WriteOption) (*Temperature, error) {
	res, err := m.temperature.Set(temperature, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*Temperature), nil
}

func (m *Model) PullTemperature(ctx context.Context, opts ...resource.ReadOption) <-chan PullTemperatureChange {
	return resources.PullValue[*Temperature](ctx, m.temperature.Pull(ctx, opts...))
}

type PullTemperatureChange = resources.ValueChange[*Temperature]
