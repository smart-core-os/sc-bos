// Package occupancyemail provides an automation that creates a digest email of occupancy statistics.
// The automation periodically uses the OccupancySensorHistoryApi to fetch occupancy records, analyses them,
// formats an email using html/template, and sends it to some recipients using smtp.
package occupancyemail

// NOTE: There's an e2e test in cmd/tools/test-occupancyemail/main.go

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	timepb "github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/occupancyemail/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const AutoName = "occupancyemail"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &autoImpl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

type autoImpl struct {
	*service.Service[config.Root]
	auto.Services
}

func (a *autoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	logger := a.Logger
	logger = logger.With(zap.String("snmp.host", cfg.Destination.Host), zap.Int("snmp.port", cfg.Destination.Port))

	ohClient := occupancysensorpb.NewOccupancySensorHistoryClient(a.Node.ClientConn())
	sendTime := cfg.Destination.SendTime
	now := cfg.Now
	if now == nil {
		now = a.Now
	}
	if now == nil {
		now = time.Now
	}

	go func() {
		t := now()
		for {
			next := sendTime.Next(t)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
				// Use the time we were planning on running instead of the current time.
				// We do this to make output more predictable
				t = next
			}

			attrs := Attrs{
				Now:   t,
				Stats: []Stats{{Source: cfg.Source}},
			}
			stats := &attrs.Stats[0]
			days := make(map[time.Time]OccupancyStats) // the time.Time key should be at 00:00 for the day

			rangeStart := t.Add(-7 * 24 * time.Hour)
			ohReq := &occupancysensorpb.ListOccupancyHistoryRequest{
				Name: cfg.Source.Name,
				Period: &timepb.Period{
					StartTime: timestamppb.New(rangeStart),
					EndTime:   timestamppb.New(t),
				},
			}
			for {
				ohResp, err := retryT(ctx, func(ctx context.Context) (*occupancysensorpb.ListOccupancyHistoryResponse, error) {
					return ohClient.ListOccupancyHistory(ctx, ohReq)
				})
				if err != nil {
					logger.Warn("failed to fetch occupancy history", zap.Error(err))
					break
				}
				for _, r := range ohResp.GetOccupancyRecords() {
					if pc := r.GetOccupancy().GetPeopleCount(); pc > stats.Last7Days.MaxPeopleCount {
						stats.Last7Days.MaxPeopleCount = pc
					}
					day := startOfDay(r.GetRecordTime().AsTime().In(t.Location()))
					if pc := r.GetOccupancy().GetPeopleCount(); pc > days[day].MaxPeopleCount {
						s := days[day]
						s.MaxPeopleCount = pc
						days[day] = s
					}
				}
				ohReq.PageToken = ohResp.GetNextPageToken()
				if ohReq.PageToken == "" {
					break
				}
			}

			// process days into stats
			for dt := startOfDay(rangeStart); dt.Before(t); dt = startOfDay(dt.Add(30 * time.Hour)) {
				stats.Days = append(stats.Days, DayStats{
					Date:           dt,
					OccupancyStats: days[dt],
				})
			}

			err := retry(ctx, func(ctx context.Context) error {
				return sendEmail(cfg.Destination, attrs)
			})
			if err != nil {
				logger.Warn("failed to send email", zap.Error(err))
			} else {
				logger.Info("email sent")
			}
		}
	}()

	return nil
}

func retry(ctx context.Context, f func(context.Context) error) error {
	return task.Run(ctx, func(ctx context.Context) (task.Next, error) {
		return 0, f(ctx)
	}, task.WithBackoff(10*time.Second, 10*time.Minute), task.WithRetry(40))
}

func retryT[T any](ctx context.Context, f func(context.Context) (T, error)) (T, error) {
	var t T
	err := retry(ctx, func(ctx context.Context) error {
		var err error
		t, err = f(ctx)
		return err
	})
	return t, err
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
