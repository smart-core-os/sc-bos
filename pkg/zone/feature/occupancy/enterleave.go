package occupancy

import (
	"context"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/math2"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

type sla struct {
	// cantFail is the set of names that can't fail
	cantFail map[string]struct{}
	// percentageOfAcceptableFailures see config.EnterLeaveOccupancySensorSLA for definition
	percentageOfAcceptableFailures float64
	// errs is keyed by name, value is the error
	errs *sync.Map
}

type enterLeave struct {
	traits.UnimplementedOccupancySensorApiServer
	client traits.EnterLeaveSensorApiClient
	names  []string
	sla    *sla

	model *occupancysensorpb.Model

	logger *zap.Logger
}

func (e *enterLeave) GetOccupancy(ctx context.Context, request *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	fns := make([]func() (*traits.EnterLeaveEvent, error), 0, len(e.names))
	for _, name := range e.names {
		name := name
		fns = append(fns, run.TagError(name, func() (*traits.EnterLeaveEvent, error) {
			res, err := e.client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: name})
			if err != nil {
				e.storeErr(name, err)
				return nil, err
			}
			e.removeErr(name)
			return res, nil
		}))
	}
	all, errs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	if len(errs) > 0 {
		if e.logger != nil {
			e.logger.Warn("some enter leave occupancy sensors failed to get", zap.Errors("errors", errs))
		}
	}
	if e.groupErrored() {
		return nil, e.mergeErrors()
	}
	return e.update(all)
}

func (e *enterLeave) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	if len(e.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no occupancy sensor names")
	}

	type c struct {
		name string
		val  *traits.EnterLeaveEvent
	}
	changes := make(chan c, len(e.names))
	group, ctx := errgroup.WithContext(server.Context())

	// for each name fetch the enter leave events and push them onto changes chan
	for _, name := range e.names {
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := e.client.PullEnterLeaveEvents(ctx, &traits.PullEnterLeaveEventsRequest{Name: name})
					if err != nil {
						if e.logger != nil {
							e.logger.Error("failed to pull enter leave events", zap.String("name", name), zap.Error(err))
						}
						e.storeErr(name, err)

						if e.groupErrored() {
							return e.mergeErrors()
						}
						return err
					}
					e.removeErr(name)

					for {
						res, err := stream.Recv()
						if err != nil {
							e.storeErr(name, err)
							return err
						}
						e.removeErr(name)

						for _, change := range res.Changes {
							changes <- c{name: name, val: change.EnterLeaveEvent}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := e.client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: name})
					if err != nil {
						return err
					}
					changes <- c{name: name, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	// for each change in changes merge the results into updates on e.model
	group.Go(func() error {
		// indexes reports which index in values each name name has
		indexes := make(map[string]int, len(e.names))
		for i, name := range e.names {
			indexes[name] = i
		}
		values := make([]*traits.EnterLeaveEvent, len(e.names))
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				_, err := e.update(values)
				if err != nil {
					return err
				}
			}
		}
	})

	// pull changes from e.model and send them to server
	group.Go(func() error {
		for change := range e.model.PullOccupancy(ctx, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
			msg := &traits.PullOccupancyResponse{Changes: []*traits.PullOccupancyResponse_Change{
				{Name: request.Name, Occupancy: change.Value, ChangeTime: timestamppb.New(change.ChangeTime)},
			}}
			if err := server.Send(msg); err != nil {
				return err
			}
		}
		return nil
	})

	return group.Wait()
}

func (e *enterLeave) storeErr(name string, err error) {
	if e.sla != nil {
		e.sla.errs.Store(name, err)
	}
}

func (e *enterLeave) removeErr(name string) {
	if e.sla != nil {
		e.sla.errs.Delete(name)
	}
}

func (e *enterLeave) mergeErrors() error {
	if e.sla == nil {
		return nil
	}

	var combined error
	e.sla.errs.Range(func(k, v interface{}) bool {
		if e, ok := v.(error); ok && e != nil {
			combined = multierr.Append(combined, e)
		}
		return true
	})
	return combined
}

// groupErrored returns true if the number of errors exceeds the SLA's percentageOfAcceptableFailures
// or if any of the errors are from sensors that can't fail
func (e *enterLeave) groupErrored() bool {
	if e.sla == nil {
		return false
	}

	failed := false
	totalErrors := 0
	e.sla.errs.Range(func(k, v interface{}) bool {
		if _, found := e.sla.cantFail[k.(string)]; found {
			failed = true
			return false // fail on first non-permitted error
		}

		if e, ok := v.(error); ok && e != nil {
			totalErrors++
		}

		return true
	})
	if failed {
		return true
	}

	return float64(100*totalErrors/len(e.names)) > e.sla.percentageOfAcceptableFailures
}

func (e *enterLeave) update(all []*traits.EnterLeaveEvent) (*traits.Occupancy, error) {
	return e.model.SetOccupancy(e.mergeEnterLeaveEvents(all),
		resource.InterceptAfter(func(old, new proto.Message) {
			oldVal, newVal := old.(*traits.Occupancy), new.(*traits.Occupancy)
			if oldVal.State != newVal.State {
				newVal.StateChangeTime = timestamppb.Now()
			}
			if newVal.StateChangeTime == nil {
				newVal.StateChangeTime = oldVal.StateChangeTime
			}
		}),
	)
}

func (e *enterLeave) mergeEnterLeaveEvents(all []*traits.EnterLeaveEvent) *traits.Occupancy {
	res := &traits.Occupancy{}
	for _, event := range all {
		if event == nil {
			continue
		}
		res.State = traits.Occupancy_UNOCCUPIED // so it's not unspecified, overridden later once we have a full count
		if event.EnterTotal != nil && event.LeaveTotal != nil {
			res.PeopleCount += math2.Max(*event.EnterTotal-*event.LeaveTotal, 0)
		}
	}

	if res.PeopleCount > 0 {
		res.State = traits.Occupancy_OCCUPIED
	}
	return res
}
