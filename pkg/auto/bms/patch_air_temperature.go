package bms

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// AirTemperaturePatches contributes patches for changing the state based on occupancy sensor readings.
type AirTemperaturePatches struct {
	name   string
	client traits.AirTemperatureApiClient
	logger *zap.Logger
}

func (o *AirTemperaturePatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	// remove our signal when we shouldn't be contributing anymore
	defer func() {
		changes <- clearAirTemperatureTransition(o.name)
	}()
	return pull.Changes[Patcher](ctx, o, changes, pull.WithLogger(o.logger))
}

func (o *AirTemperaturePatches) Pull(ctx context.Context, changes chan<- Patcher) error {
	stream, err := o.client.PullAirTemperature(ctx, &traits.PullAirTemperatureRequest{Name: o.name})
	if err != nil {
		return err
	}

	for {
		change, err := stream.Recv()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- (*pullAirTemperatureTransition)(change):
		}
	}
}

func (o *AirTemperaturePatches) Poll(ctx context.Context, changes chan<- Patcher) error {
	res, err := o.client.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{Name: o.name})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- getAirTemperaturePatcher{o.name, res}:
		return nil
	}
}

type pullAirTemperatureTransition traits.PullAirTemperatureResponse

func (o *pullAirTemperatureTransition) Patch(s *ReadState) {
	r := (*traits.PullAirTemperatureResponse)(o)

	for _, change := range r.Changes {
		v := s.AirTemperature[change.Name]
		v.set(s.Now(), change.AirTemperature, nil)
		s.AirTemperature[change.Name] = v
	}
}

type getAirTemperaturePatcher struct {
	name string
	res  *traits.AirTemperature
}

func (g getAirTemperaturePatcher) Patch(s *ReadState) {
	v := s.AirTemperature[g.name]
	v.set(s.Now(), g.res, nil)
	s.AirTemperature[g.name] = v
}

type clearAirTemperatureTransition string

func (c clearAirTemperatureTransition) Patch(s *ReadState) {
	delete(s.AirTemperature, string(c))
}
