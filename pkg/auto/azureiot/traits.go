package azureiot

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// pullDevice calls pullTraits for each SCDeviceConfig.
func (a *Auto) pullDevice(ctx context.Context, dst chan<- proto.Message, device DeviceConfig) error {
	grp, ctx := errgroup.WithContext(ctx)
	if device.Name != "" {
		grp.Go(func() error {
			return a.pullTraits(ctx, dst, device.SCDeviceConfig)
		})
	}
	for _, childCfg := range device.Children {
		childCfg := childCfg
		grp.Go(func() error {
			return a.pullTraits(ctx, dst, childCfg)
		})
	}
	return grp.Wait()
}

// pullTraits publishes all configured trait changes to dst, returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullTraits(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	logger := a.services.Logger.With(zap.String("device", device.Name))

	handleErr := func(t trait.Name, err error) error {
		if device.IgnoreUnknownTraits {
			logger.Warn("ignoring unknown trait", zap.Stringer("trait", t), zap.Error(err))
			return nil
		}
		return fmt.Errorf("trait %q: %w", t, err)
	}

	grp, ctx := errgroup.WithContext(ctx)
	for _, tn := range device.Traits {
		tn := tn
		switch tn {
		case trait.AirQualitySensor:
			grp.Go(func() error {
				return handleErr(tn, a.pullAirQuality(ctx, dst, device))
			})
		case trait.AirTemperature:
			grp.Go(func() error {
				return handleErr(tn, a.pullAirTemperature(ctx, dst, device))
			})
		case trait.BrightnessSensor:
			grp.Go(func() error {
				return handleErr(tn, a.pullAmbientBrightness(ctx, dst, device))
			})
		case trait.EnterLeaveSensor:
			grp.Go(func() error {
				return handleErr(tn, a.pullEnterLeave(ctx, dst, device))
			})
		case trait.Light:
			grp.Go(func() error {
				return handleErr(tn, a.pullBrightness(ctx, dst, device))
			})
		case meter.TraitName:
			grp.Go(func() error {
				return handleErr(tn, a.pullMeterReadings(ctx, dst, device))
			})
		case trait.OccupancySensor:
			grp.Go(func() error {
				return handleErr(tn, a.pullOccupancy(ctx, dst, device))
			})
		default:
			if device.IgnoreUnknownTraits {
				logger.Warn("ignoring unsupported trait", zap.Stringer("trait", tn))
				continue
			}
			return fmt.Errorf("unsupported trait %q", tn)
		}
	}
	return grp.Wait()
}

// pullAirQuality publishes device's air quality changes (as *traits.PullAirQualityResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullAirQuality(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewAirQualitySensorApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullAirQualityResponse_Change) error {
		ss, err := client.PullAirQuality(ctx, &traits.PullAirQualityRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullAirQualityResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullAirQualityResponse_Change) error {
		msg, err := client.GetAirQuality(ctx, &traits.GetAirQualityRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullAirQualityResponse_Change{
			Name:       device.Name,
			ChangeTime: timestamppb.Now(),
			AirQuality: msg,
		})
	}
	reduce := func(cs []*traits.PullAirQualityResponse_Change) proto.Message {
		return &traits.PullAirQualityResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullAirTemperature publishes device's air temperature changes (as *traits.PullAirTemperatureResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullAirTemperature(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewAirTemperatureApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullAirTemperatureResponse_Change) error {
		ss, err := client.PullAirTemperature(ctx, &traits.PullAirTemperatureRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullAirTemperatureResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullAirTemperatureResponse_Change) error {
		msg, err := client.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullAirTemperatureResponse_Change{
			Name:           device.Name,
			ChangeTime:     timestamppb.Now(),
			AirTemperature: msg,
		})
	}
	reduce := func(cs []*traits.PullAirTemperatureResponse_Change) proto.Message {
		return &traits.PullAirTemperatureResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullAmbientBrightness publishes device's ambient brightness changes (as *traits.PullAmbientBrightnessResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullAmbientBrightness(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewBrightnessSensorApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullAmbientBrightnessResponse_Change) error {
		ss, err := client.PullAmbientBrightness(ctx, &traits.PullAmbientBrightnessRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullAmbientBrightnessResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullAmbientBrightnessResponse_Change) error {
		msg, err := client.GetAmbientBrightness(ctx, &traits.GetAmbientBrightnessRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullAmbientBrightnessResponse_Change{
			Name:              device.Name,
			ChangeTime:        timestamppb.Now(),
			AmbientBrightness: msg,
		})
	}
	reduce := func(cs []*traits.PullAmbientBrightnessResponse_Change) proto.Message {
		return &traits.PullAmbientBrightnessResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullBrightness publishes device's brightness changes (as *traits.PullBrightnessResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullBrightness(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewLightApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullBrightnessResponse_Change) error {
		ss, err := client.PullBrightness(ctx, &traits.PullBrightnessRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullBrightnessResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullBrightnessResponse_Change) error {
		msg, err := client.GetBrightness(ctx, &traits.GetBrightnessRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullBrightnessResponse_Change{
			Name:       device.Name,
			ChangeTime: timestamppb.Now(),
			Brightness: msg,
		})
	}
	reduce := func(cs []*traits.PullBrightnessResponse_Change) proto.Message {
		return &traits.PullBrightnessResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullMeterReadings publishes device's meter readings changes (as *meterpb.PullMeterReadingsResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullMeterReadings(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, meterpb.NewMeterApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *meterpb.PullMeterReadingsResponse_Change) error {
		ss, err := client.PullMeterReadings(ctx, &meterpb.PullMeterReadingsRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*meterpb.PullMeterReadingsResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *meterpb.PullMeterReadingsResponse_Change) error {
		msg, err := client.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &meterpb.PullMeterReadingsResponse_Change{
			Name:         device.Name,
			ChangeTime:   timestamppb.Now(),
			MeterReading: msg,
		})
	}
	reduce := func(cs []*meterpb.PullMeterReadingsResponse_Change) proto.Message {
		return &meterpb.PullMeterReadingsResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullOccupancy publishes device's occupancy changes (as *traits.PullOccupancyResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullOccupancy(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewOccupancySensorApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullOccupancyResponse_Change) error {
		ss, err := client.PullOccupancy(ctx, &traits.PullOccupancyRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullOccupancyResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullOccupancyResponse_Change) error {
		msg, err := client.GetOccupancy(ctx, &traits.GetOccupancyRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullOccupancyResponse_Change{
			Name:       device.Name,
			ChangeTime: timestamppb.Now(),
			Occupancy:  msg,
		})
	}
	reduce := func(cs []*traits.PullOccupancyResponse_Change) proto.Message {
		return &traits.PullOccupancyResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// pullEnterLeave publishes device's EnterLeave changes (as *traits.PullEnterLeaveEventsResponse) to dst,
// returning when ctx is done or a non-recoverable error occurs.
func (a *Auto) pullEnterLeave(ctx context.Context, dst chan<- proto.Message, device SCDeviceConfig) error {
	client, err := grpcClient(a, traits.NewEnterLeaveSensorApiClient, device)
	if err != nil {
		return err
	}

	pullFunc := func(ctx context.Context, stream chan<- *traits.PullEnterLeaveEventsResponse_Change) error {
		ss, err := client.PullEnterLeaveEvents(ctx, &traits.PullEnterLeaveEventsRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return pullStreamChanges[*traits.PullEnterLeaveEventsResponse](ctx, stream, ss)
	}
	pollFunc := func(ctx context.Context, stream chan<- *traits.PullEnterLeaveEventsResponse_Change) error {
		msg, err := client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: device.Name})
		if err != nil {
			return err
		}
		return chans.SendContext(ctx, stream, &traits.PullEnterLeaveEventsResponse_Change{
			Name:            device.Name,
			ChangeTime:      timestamppb.Now(),
			EnterLeaveEvent: msg,
		})
	}
	reduce := func(cs []*traits.PullEnterLeaveEventsResponse_Change) proto.Message {
		return &traits.PullEnterLeaveEventsResponse{Changes: cs}
	}
	delay := device.PollInterval.Or(DefaultPollInterval)

	return doPull(ctx, dst, pullFunc, pollFunc, reduce, delay)
}

// grpcClient returns a new client of type T.
// The client used will be backed by:
//  1. a local device if dev.RemoteNode is nil
//  2. a cached connection if available keyed by dev.RemoteDevice.Host
//  3. a new connection to dev.RemoteDevice.Host
//
// grpcClient is safe to call from multiple goroutines.
func grpcClient[T any](a *Auto, f func(connInterface grpc.ClientConnInterface) T, dev SCDeviceConfig) (client T, _ error) {
	// use local client config
	rn := dev.RemoteNode
	if rn == nil {
		return f(a.services.Node.ClientConn()), nil
	}

	a.connsMu.Lock()
	defer a.connsMu.Unlock()
	if conn, ok := a.conns[rn.Host]; ok {
		return f(conn), nil
	}

	tlsConfig, err := rn.TLSConfig.Read("", a.services.ClientTLSConfig)
	if err != nil {
		return client, err
	}
	var opts []grpc.DialOption
	if tlsConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
	conn, err := grpc.NewClient(rn.Host, opts...)
	if err != nil {
		return client, err
	}
	a.conns[rn.Host] = conn
	return f(conn), nil
}

// doPull pulls changes from pullFunc or pollFunc, collates them via reduce, and publishes them to dst.
func doPull[C any](ctx context.Context, dst chan<- proto.Message, pullFunc, pollFunc func(context.Context, chan<- C) error, reduce func(cs []C) proto.Message, delay time.Duration) error {
	stream := make(chan C)
	wrap := func(f func(context.Context, chan<- C) error) func(context.Context) error {
		return func(ctx context.Context) error {
			return f(ctx, stream)
		}
	}

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		return pull.OrPoll(ctx, wrap(pullFunc), wrap(pollFunc), pull.WithPollDelay(delay))
	})
	grp.Go(func() error {
		return publishStreamChanges(ctx, dst, stream, reduce, delay)
	})
	return grp.Wait()
}

// pullStreamChanges receives messages from stream, expands them into individual changes, and sends them to dst.
func pullStreamChanges[M interface{ GetChanges() []C }, S interface{ Recv() (M, error) }, C any](ctx context.Context, dst chan<- C, stream S) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, c := range msg.GetChanges() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case dst <- c:
			}
		}
	}
}

// publishStreamChanges receives messages from stream, collates them into a single message, and sends it to dst.
func publishStreamChanges[C any, P proto.Message](ctx context.Context, dst chan<- P, stream <-chan C, pf func(cs []C) P, delay time.Duration) error {
	var changes []C
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case c := <-stream:
			changes = append(changes, c)
		case <-ticker.C:
			if len(changes) == 0 {
				continue
			}
			msg := pf(changes)
			changes = changes[:0] // keep the capacity, but empty the slice
			select {
			case <-ctx.Done():
				return ctx.Err()
			case dst <- msg:
			}
		}
	}
}
