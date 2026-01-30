package airquality

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

type Group struct {
	traits.UnimplementedAirQualitySensorApiServer
	client traits.AirQualitySensorApiClient
	names  []string

	logger *zap.Logger
}

func (g *Group) GetAirQuality(ctx context.Context, request *traits.GetAirQualityRequest) (*traits.AirQuality, error) {
	fns := make([]func() (*traits.AirQuality, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*traits.GetAirQualityRequest)
		request.Name = name
		fns[i] = func() (*traits.AirQuality, error) {
			return g.client.GetAirQuality(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some airquality sensors failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeAirQuality(allRes)
}

func (g *Group) PullAirQuality(request *traits.PullAirQualityRequest, server traits.AirQualitySensorApi_PullAirQualityServer) error {
	if len(g.names) == 0 {
		return status.Errorf(codes.FailedPrecondition, "zone has no air quality sensor names")
	}

	type c struct {
		name string
		val  *traits.AirQuality
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	// get air quality from each of the named devices
	for _, name := range g.names {
		request := proto.Clone(request).(*traits.PullAirQualityRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullAirQuality(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{name: request.Name, val: change.AirQuality}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetAirQuality(ctx, &traits.GetAirQualityRequest{Name: name, ReadMask: request.ReadMask})
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

	// merge all the changes into one airQuality obj and send to server
	group.Go(func() error {
		// indexes reports which index in values each name name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]*traits.AirQuality, len(g.names))

		var last *traits.AirQuality
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				r, err := mergeAirQuality(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&traits.PullAirQualityResponse{Changes: []*traits.PullAirQualityResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					AirQuality: r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeAirQuality(all []*traits.AirQuality) (*traits.AirQuality, error) {
	switch len(all) {
	case 0:
		return nil, status.Errorf(codes.FailedPrecondition, "zone has no air quality sensor names")
	case 1:
		return all[0], nil
	default:
		out := &traits.AirQuality{}
		// CO2
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.CarbonDioxideLevel == nil {
				return 0, false
			}
			return *e.CarbonDioxideLevel, true
		}); ok {
			out.CarbonDioxideLevel = &val
		}
		// VOC
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.VolatileOrganicCompounds == nil {
				return 0, false
			}
			return *e.VolatileOrganicCompounds, true
		}); ok {
			out.VolatileOrganicCompounds = &val
		}
		// AirPressure
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.AirPressure == nil {
				return 0, false
			}
			return *e.AirPressure, true
		}); ok {
			out.AirPressure = &val
		}
		// InfectionRisk
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.InfectionRisk == nil {
				return 0, false
			}
			return *e.InfectionRisk, true
		}); ok {
			out.InfectionRisk = &val
		}
		// comfort
		if val, ok := merge.Mode(all, func(e *traits.AirQuality) (traits.AirQuality_Comfort, bool) {
			if e == nil {
				return traits.AirQuality_COMFORT_UNSPECIFIED, false
			}
			return e.Comfort, true
		}); ok {
			out.Comfort = val
		}
		// IAQ Score
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.Score == nil {
				return 0, false
			}
			return *e.Score, true
		}); ok {
			out.Score = &val
		}
		// PM1
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.ParticulateMatter_1 == nil {
				return 0, false
			}
			return *e.ParticulateMatter_1, true
		}); ok {
			out.ParticulateMatter_1 = &val
		}
		// PM10
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.ParticulateMatter_10 == nil {
				return 0, false
			}
			return *e.ParticulateMatter_10, true
		}); ok {
			out.ParticulateMatter_10 = &val
		}
		// PM25
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.ParticulateMatter_25 == nil {
				return 0, false
			}
			return *e.ParticulateMatter_25, true
		}); ok {
			out.ParticulateMatter_25 = &val
		}
		// AirChangePerHour
		if val, ok := merge.Mean(all, func(e *traits.AirQuality) (float32, bool) {
			if e == nil || e.AirChangePerHour == nil {
				return 0, false
			}
			return *e.AirChangePerHour, true
		}); ok {
			out.AirChangePerHour = &val
		}
		return out, nil
	}
}
