package temperaturepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	temperature *resource.Value // of *temperaturepb.Temperature
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&temperaturepb.Temperature{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		temperature: resource.NewValue(opts...),
	}
}

func (m *Model) GetTemperature(opts ...resource.ReadOption) (*temperaturepb.Temperature, error) {
	return m.temperature.Get(opts...).(*temperaturepb.Temperature), nil
}

func (m *Model) UpdateTemperature(temperature *temperaturepb.Temperature, opts ...resource.WriteOption) (*temperaturepb.Temperature, error) {
	res, err := m.temperature.Set(temperature, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*temperaturepb.Temperature), nil
}

func (m *Model) PullTemperature(ctx context.Context, opts ...resource.ReadOption) <-chan PullTemperatureChange {
	return resources.PullValue[*temperaturepb.Temperature](ctx, m.temperature.Pull(ctx, opts...))
}

type PullTemperatureChange = resources.ValueChange[*temperaturepb.Temperature]
