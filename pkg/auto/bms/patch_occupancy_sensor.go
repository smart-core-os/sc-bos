package bms

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// OccupancySensorPatches contributes patches for changing the state based on occupancy sensor readings.
type OccupancySensorPatches struct {
	name   string
	client traits.OccupancySensorApiClient
	logger *zap.Logger
}

func (o *OccupancySensorPatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	// remove our signal when we shouldn't be contributing anymore
	defer func() {
		changes <- clearOccupancyTransition(o.name)
	}()
	return pull.Changes[Patcher](ctx, o, changes, pull.WithLogger(o.logger))
}

func (o *OccupancySensorPatches) Pull(ctx context.Context, changes chan<- Patcher) error {
	stream, err := o.client.PullOccupancy(ctx, &traits.PullOccupancyRequest{Name: o.name})
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
		case changes <- (*pullOccupancyTransition)(change):
		}
	}
}

func (o *OccupancySensorPatches) Poll(ctx context.Context, changes chan<- Patcher) error {
	res, err := o.client.GetOccupancy(ctx, &traits.GetOccupancyRequest{Name: o.name})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- getOccupancyPatcher{o.name, res}:
		return nil
	}
}

type pullOccupancyTransition traits.PullOccupancyResponse

func (o *pullOccupancyTransition) Patch(s *ReadState) {
	r := (*traits.PullOccupancyResponse)(o)

	for _, change := range r.Changes {
		if change.Occupancy.StateChangeTime == nil {
			change.Occupancy.StateChangeTime = timestamppb.New(s.Now())
		}
		v := s.Occupancy[change.Name]
		v.set(s.Now(), change.Occupancy, nil)
		s.Occupancy[change.Name] = v
	}
}

type getOccupancyPatcher struct {
	name string
	res  *traits.Occupancy
}

func (g getOccupancyPatcher) Patch(s *ReadState) {
	if g.res.StateChangeTime == nil {
		g.res.StateChangeTime = timestamppb.New(s.Now())
	}
	v := s.Occupancy[g.name]
	v.set(s.Now(), g.res, nil)
	s.Occupancy[g.name] = v
}

type clearOccupancyTransition string

func (c clearOccupancyTransition) Patch(s *ReadState) {
	delete(s.Occupancy, string(c))
}
