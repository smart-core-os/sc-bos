package meter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/once"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/meter/config"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

type Group struct {
	meterpb.UnimplementedMeterApiServer
	meterpb.UnimplementedMeterInfoServer
	apiClient        meterpb.MeterApiClient
	infoClient       meterpb.MeterInfoClient
	historyApiClient meterpb.MeterHistoryClient
	names            []string
	readOnly         bool

	unit     string
	unitOnce once.RetryError

	historyBackupConf *config.HistoryBackup
	now               func() time.Time

	logger *zap.Logger
}

func (g *Group) DescribeMeterReading(ctx context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	err := g.unitOnce.Do(ctx, func() error {
		if g.unit != "" {
			return nil
		}

		var err error
		for _, name := range g.names {
			ctx, cleanup := context.WithTimeout(context.Background(), 5*time.Second)
			var s *meterpb.MeterReadingSupport
			s, err = g.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: name})
			cleanup()
			if err == nil && s.UsageUnit != "" {
				g.unit = s.UsageUnit
				return nil
			}
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return &meterpb.MeterReadingSupport{
		ResourceSupport: &types.ResourceSupport{
			Readable:   true,
			Writable:   !g.readOnly,
			Observable: true,
		},
		UsageUnit: g.unit,
	}, nil
}

func (g *Group) GetMeterReading(ctx context.Context, request *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	allRes := make([]value, len(g.names))
	fns := make([]func(), len(g.names))

	countedErrs := atomic.Int32{}

	for i, name := range g.names {
		request := proto.Clone(request).(*meterpb.GetMeterReadingRequest)
		request.Name = name
		i := i
		fns[i] = func() {
			res, err := g.apiClient.GetMeterReading(ctx, request)

			if err != nil {
				countedErrs.Add(1)

				historyRes, historyErr := g.attemptHistoricalReading(ctx, name, err)

				if historyErr == nil {
					if float32(100*(countedErrs.Load()/int32(len(g.names)))) < g.historyBackupConf.PercentageOfAcceptableErrors {
						// use historical reading if available, within lookback limit, and we haven't exceeded the acceptable error percentage
						res = historyRes
						err = nil
					}
				}
			}

			allRes[i] = value{name: name, val: res, err: err}
		}
	}
	if err := run.InParallel(ctx, run.DefaultConcurrency, fns...); err != nil {
		return nil, err
	}
	return mergeMeterReading(allRes)
}

type nameToError struct {
	sync.Map // meter name key -> error value
}

func newNameToError() *nameToError {
	return &nameToError{}
}

func (n *nameToError) store(name string, v error) {
	n.Store(name, v)
}

func (n *nameToError) countErrs() int {
	count := 0
	n.Range(func(key, value any) bool {
		if value != nil && value.(error) != nil {
			count++
		}
		return true
	})

	return count
}

func (g *Group) PullMeterReadings(request *meterpb.PullMeterReadingsRequest, server meterpb.MeterApi_PullMeterReadingsServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no meter names")
	}

	changes := make(chan value)
	defer close(changes)

	// we use a map to track errors per name
	// so that we can calculate the percentage of errors across all names
	// and decide whether to use historical readings as a backup
	// based on the configured acceptable error percentage
	errs := newNameToError()

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*meterpb.PullMeterReadingsRequest)
		request.Name = name
		handleResp := func(err error) error {
			errs.store(request.Name, err)
			historyRes, historyErr := g.attemptHistoricalReading(ctx, request.Name, err)

			if historyErr != nil {
				changes <- value{name: request.Name, err: err}
				return err
			}
			if float32(100*(errs.countErrs())/len(g.names)) < g.historyBackupConf.PercentageOfAcceptableErrors {
				// use historical reading if available, within lookback limit, and we haven't exceeded the acceptable error percentage
				changes <- value{name: request.Name, val: historyRes}
				return nil
			}
			return err
		}
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- value) error {
					stream, err := g.apiClient.PullMeterReadings(ctx, request)
					if err != nil {
						errs.Store(request.Name, err)
						// if the stream fails pull.Changes should fall back to Get/Polling
						changes <- value{name: request.Name, err: err}
						return err
					}
					errs.Delete(request.Name)
					for {
						res, err := stream.Recv()
						if err != nil {
							if err2 := handleResp(err); err2 != nil {
								return err
							}
							// since the stream has errored and a history reading (if any) has been used
							// we don't delete the original error from the errs map
							continue
						}
						errs.Delete(request.Name)
						for _, change := range res.Changes {
							changes <- value{name: request.Name, val: change.MeterReading}
						}
					}
				},
				func(ctx context.Context, changes chan<- value) error {
					res, err := g.apiClient.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						if err2 := handleResp(err); err2 != nil {
							return err
						}
						return nil
					}
					changes <- value{name: request.Name, val: res}
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
		values := make([]value, len(g.names))

		var last *meterpb.MeterReading
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change
				r, err := mergeMeterReading(values)
				if err != nil {
					continue
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&meterpb.PullMeterReadingsResponse{
					Changes: []*meterpb.PullMeterReadingsResponse_Change{
						{
							Name:         request.Name,
							ChangeTime:   timestamppb.Now(),
							MeterReading: r,
						},
					},
				})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func (g *Group) attemptHistoricalReading(ctx context.Context, name string, originalErr error) (*meterpb.MeterReading, error) {
	if g.historyBackupConf == nil || g.historyBackupConf.Disabled {
		return nil, originalErr
	}

	// try to get the latest backup reading from the history table if the main read fails
	latest, historyErr := g.historyApiClient.ListMeterReadingHistory(ctx, &meterpb.ListMeterReadingHistoryRequest{
		Name:     name,
		PageSize: 1,
		OrderBy:  "record_time DESC",
	})

	if historyErr != nil || len(latest.GetMeterReadingRecords()) == 0 {
		return nil, originalErr // we forward the original error
	}

	if g.historyBackupConf.LookbackLimit != nil {
		if g.now().After(latest.GetMeterReadingRecords()[0].GetRecordTime().AsTime().Add(g.historyBackupConf.LookbackLimit.Duration)) {
			// latest history record is too old
			return nil, originalErr // we forward the original error
		}
	}

	return latest.GetMeterReadingRecords()[0].MeterReading, nil
}

type value struct {
	name string
	val  *meterpb.MeterReading
	err  error
}

func mergeMeterReading(all []value) (*meterpb.MeterReading, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no meter names")
	default:
		// we need all configured meters to be present and have values for this meter to make any sense.
		valCount := 0
		for _, v := range all {
			if v.err != nil {
				return nil, v.err
			}
			if v.val != nil {
				valCount++
			}
		}
		if valCount < len(all) {
			return nil, status.Errorf(codes.Unavailable, "collecting initial data, please try again soon (%d/%d)", valCount, len(all))
		}

		out := &meterpb.MeterReading{}
		out.Usage, _ = merge.Sum(all, func(v value) (float32, bool) {
			if v.err != nil || v.val == nil {
				return 0, false
			}
			return v.val.Usage, true
		})
		out.StartTime = merge.EarliestTimestamp(all, func(v value) *timestamppb.Timestamp {
			if v.err != nil || v.val == nil {
				return nil
			}
			return v.val.StartTime
		})
		out.EndTime = merge.LatestTimestamp(all, func(v value) *timestamppb.Timestamp {
			if v.err != nil || v.val == nil {
				return nil
			}
			return v.val.EndTime
		})
		return out, nil
	}
}
