package bms

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	timepb "github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// MeanOATempPatches emits MeanOATemp patches based on an exponential running mean.
// During Subscribe it will attempt to fetch historical data from the device to seed the running mean before pulling
// updated data from the device. If this fails then the initial mean will be potentially inaccurate.
type MeanOATempPatches struct {
	name          string
	apiClient     traits.AirTemperatureApiClient
	historyClient airtemperaturepb.AirTemperatureHistoryClient
	logger        *zap.Logger
}

func (m *MeanOATempPatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	defer func() {
		changes <- PatchFunc(func(s *ReadState) {
			s.MeanOATemp = nil
		})
	}()

	// The temperature as it was at each hour.
	// Used to calculate the daily mean air temperature.
	//
	// hourlyTemps[3] is a temperature recorded somewhere between [03:00, 04:00).
	// nil means we don't have a temperature for that hour yet.
	var hourlyTemps dailyTemp
	runningMean := &exponentialMean{alpha: 0.8}

	if m.historyClient != nil {
		var err error
		hourlyTemps, err = m.readHistoricalMean(ctx, runningMean)
		if err != nil {
			return err
		}
	}

	tc := make(chan *traits.AirTemperature)
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		defer close(tc)
		return m.pullDeviceChanges(ctx, tc)
	})
	grp.Go(func() error {
		return m.sendMeanChanges(ctx, tc, changes, runningMean, hourlyTemps)
	})

	return grp.Wait()
}

// readHistoricalMean calculates the mean temperature for the last 7 days.
// The mean is stored in runningMean, returning uncounted temperatures for the current day.
func (m *MeanOATempPatches) readHistoricalMean(ctx context.Context, runningMean *exponentialMean) (dailyTemp, error) {
	now := time.Now()
	startOfToday := now.Truncate(24 * time.Hour) // technically not true during DST transitions, but close enough
	startOfRange := startOfToday.Add(-7 * 24 * time.Hour)

	var dayTemp dailyTemp
	endOfCurrentDay := startOfRange.Add(24 * time.Hour)

	req := &airtemperaturepb.ListAirTemperatureHistoryRequest{
		Name: m.name,
		Period: &timepb.Period{
			StartTime: timestamppb.New(startOfRange),
			EndTime:   timestamppb.New(startOfToday.Add(24 * time.Hour)),
		},
	}
	for {
		res, err := retryForeverT(ctx, func(ctx context.Context) (*airtemperaturepb.ListAirTemperatureHistoryResponse, error) {
			res, err := m.historyClient.ListAirTemperatureHistory(ctx, req)
			if c := status.Code(err); c == codes.NotFound || c == codes.Unimplemented {
				return nil, nil
			}
			return res, err
		})
		if err != nil {
			return dailyTemp{}, err
		}
		if res == nil {
			break // device doesn't support history
		}

		for _, record := range res.AirTemperatureRecords {
			rt := record.RecordTime.AsTime()
			if !rt.Before(endOfCurrentDay) {
				// we've started a new day
				if mean, ok := dayTemp.Mean(); ok {
					runningMean.Add(mean)
				}
				dayTemp.Clear()
				endOfCurrentDay = rt.Truncate(24 * time.Hour).Add(24 * time.Hour)
			}

			if t := record.GetAirTemperature().GetAmbientTemperature(); t != nil {
				dayTemp.Set(rt, t.ValueCelsius)
			}
		}

		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	return dayTemp, nil
}

// pullDeviceChanges pulls changes from the device and sends them on out.
// Returns when ctx is done or non-recoverable error occurs talking to the device.
func (m *MeanOATempPatches) pullDeviceChanges(ctx context.Context, out chan<- *traits.AirTemperature) error {
	return pull.Changes(ctx, pull.NewFetcher(
		func(ctx context.Context, changes chan<- *traits.AirTemperature) error {
			stream, err := m.apiClient.PullAirTemperature(ctx, &traits.PullAirTemperatureRequest{Name: m.name})
			if err != nil {
				return err
			}
			for {
				msg, err := stream.Recv()
				if err != nil {
					return err
				}
				for _, change := range msg.Changes {
					if err := chans.SendContext(ctx, changes, change.AirTemperature); err != nil {
						return err
					}
				}
			}
		},
		func(ctx context.Context, changes chan<- *traits.AirTemperature) error {
			res, err := m.apiClient.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{Name: m.name})
			if err != nil {
				return err
			}
			return chans.SendContext(ctx, changes, res)
		},
	), out)
}

// sendMeanChanges reads temperatures from in and sends mean changes on out.
// runningMean and hourlyTemps can be used to seed the running mean with historical data.
// Returns either when ctx is done, or in is closed.
func (m *MeanOATempPatches) sendMeanChanges(ctx context.Context, in <-chan *traits.AirTemperature, out chan<- Patcher, runningMean *exponentialMean, hourlyTemps dailyTemp) error {
	setMean := func() error {
		err := chans.SendContext[Patcher](ctx, out, PatchFunc(func(s *ReadState) {
			s.MeanOATemp = &types.Temperature{
				ValueCelsius: runningMean.mean,
			}
		}))
		if err != nil {
			return err
		}
		return nil
	}
	if runningMean.set {
		if err := setMean(); err != nil {
			return err
		}
	}
	startOfCurrentDay := time.Now().Truncate(24 * time.Hour)
	endOfCurrentDay := startOfCurrentDay.Add(24 * time.Hour)
	for temp := range in {
		if temp == nil {
			continue
		}
		now := time.Now()
		if !now.Before(endOfCurrentDay) {
			// we've started a new day, update the running mean and send a patch
			if mean, ok := hourlyTemps.Mean(); ok {
				runningMean.Add(mean)
			}
			hourlyTemps.Clear()
			startOfCurrentDay = now.Truncate(24 * time.Hour)
			endOfCurrentDay = startOfCurrentDay.Add(24 * time.Hour)
			if err := setMean(); err != nil {
				return err
			}
		}

		hourlyTemps.Set(now, temp.AmbientTemperature.ValueCelsius)
	}

	return nil // only get here if in is closed, which means the puller has stopped
}

// dailyTemp represents 24 hourly temperature readings.
type dailyTemp [24]*float64

func (dt dailyTemp) Set(t time.Time, temp float64) {
	dt[t.Hour()] = &temp
}

// Mean returns the mean temperature for the day.
// Hours with absent values are treated as if they have the same value as the previous hour.
// Initial absent values are ignored.
func (dt dailyTemp) Mean() (float64, bool) {
	var sum float64
	var count int
	var last *float64
	for _, temp := range dt {
		if temp == nil {
			if last == nil {
				continue
			}
			temp = last
		}
		sum += *temp
		last = temp
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

func (dt dailyTemp) Clear() {
	for i := range dt {
		dt[i] = nil
	}
}

// exponentialMean calculates an exponentially weighted running mean.
type exponentialMean struct {
	alpha float64
	mean  float64
	set   bool
}

func (em *exponentialMean) Add(x float64) {
	if !em.set {
		em.mean = x
		em.set = true
		return
	}
	em.mean = em.alpha*x + (1-em.alpha)*em.mean
}
