package hvac

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	airtemperaturepb.UnimplementedAirTemperatureApiServer
	client   airtemperaturepb.AirTemperatureApiClient
	names    []string
	readOnly bool

	logger *zap.Logger
}

func (g *Group) GetAirTemperature(ctx context.Context, request *airtemperaturepb.GetAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	fns := make([]func() (*airtemperaturepb.AirTemperature, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*airtemperaturepb.GetAirTemperatureRequest)
		request.Name = name
		fns[i] = func() (*airtemperaturepb.AirTemperature, error) {
			return g.client.GetAirTemperature(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some hvacs failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeAirTemperature(allRes)
}

func (g *Group) UpdateAirTemperature(ctx context.Context, request *airtemperaturepb.UpdateAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	if g.readOnly {
		return nil, status.Errorf(codes.FailedPrecondition, "read-only")
	}
	fns := make([]func() (*airtemperaturepb.AirTemperature, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*airtemperaturepb.UpdateAirTemperatureRequest)
		request.Name = name
		fns[i] = func() (*airtemperaturepb.AirTemperature, error) {
			return g.client.UpdateAirTemperature(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some hvacs failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeAirTemperature(allRes)
}

func (g *Group) PullAirTemperature(request *airtemperaturepb.PullAirTemperatureRequest, server airtemperaturepb.AirTemperatureApi_PullAirTemperatureServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no hvac names")
	}

	type c struct {
		name string
		val  *airtemperaturepb.AirTemperature
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*airtemperaturepb.PullAirTemperatureRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullAirTemperature(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{name: request.Name, val: change.AirTemperature}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetAirTemperature(ctx, &airtemperaturepb.GetAirTemperatureRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						return err
					}
					changes <- c{name: request.Name, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	group.Go(func() error {
		// indexes reports which index in values each name name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]*airtemperaturepb.AirTemperature, len(g.names))

		var last *airtemperaturepb.AirTemperature
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				r, err := mergeAirTemperature(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&airtemperaturepb.PullAirTemperatureResponse{Changes: []*airtemperaturepb.PullAirTemperatureResponse_Change{{
					Name:           request.Name,
					ChangeTime:     timestamppb.Now(),
					AirTemperature: r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeAirTemperature(all []*airtemperaturepb.AirTemperature) (*airtemperaturepb.AirTemperature, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no hvac names")
	case 1:
		return all[0], nil
	default:
		out := &airtemperaturepb.AirTemperature{}
		// TemperatureGoal
		if setPoint, ok := merge.Mean(all, func(e *airtemperaturepb.AirTemperature) (float64, bool) {
			switch t := e.GetTemperatureGoal().(type) { // note: Get is e-nil safe
			case *airtemperaturepb.AirTemperature_TemperatureSetPoint:
				return t.TemperatureSetPoint.ValueCelsius, true
			default:
				return 0, false
			}
		}); ok {
			out.TemperatureGoal = &airtemperaturepb.AirTemperature_TemperatureSetPoint{TemperatureSetPoint: &typespb.Temperature{ValueCelsius: setPoint}}
		}
		// AmbientTemperature
		if val, ok := merge.Mean(all, func(e *airtemperaturepb.AirTemperature) (float64, bool) {
			if e == nil || e.AmbientTemperature == nil {
				return 0, false
			}
			return e.AmbientTemperature.ValueCelsius, true
		}); ok {
			out.AmbientTemperature = &typespb.Temperature{ValueCelsius: val}
		}
		// AmbientHumidity
		if val, ok := merge.Mean(all, func(e *airtemperaturepb.AirTemperature) (float32, bool) {
			if e == nil || e.AmbientHumidity == nil {
				return 0, false
			}
			return *e.AmbientHumidity, true
		}); ok {
			out.AmbientHumidity = &val
		}
		// DewPoint
		if val, ok := merge.Mean(all, func(e *airtemperaturepb.AirTemperature) (float64, bool) {
			if e == nil || e.DewPoint == nil {
				return 0, false
			}
			return e.DewPoint.ValueCelsius, true
		}); ok {
			out.DewPoint = &typespb.Temperature{ValueCelsius: val}
		}
		// can't average the mode, if they're all the same use it
		for _, temp := range all {
			if temp == nil {
				continue
			}
			if out.Mode == 0 {
				out.Mode = temp.Mode
				continue
			}
			if out.Mode == temp.Mode {
				continue
			}
			if temp.Mode == 0 {
				continue
			}

			// not all modes are the same
			out.Mode = 0
			break
		}
		return out, nil
	}
}
