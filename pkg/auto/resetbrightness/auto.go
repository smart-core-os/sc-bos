// Package resetbrightness is an automation that resets the brightness of one of more lights after some delay.
// The auto will monitor a number of lights brightness to see when they enter into an off-normal state, i.e. they are turned on.
// After some delay of either entering this state, or after some change, the auto will adjust the light into the reset state.
//
// As an example, the auto can be configured to treat a light as off-normal if its brightness is greater than 10%,
// when the light enters this state a timer is started to reset the light to 0% brightness after 15 minutes.
package resetbrightness

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/resetbrightness/config"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

const AutoName = "resetbrightness"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &impl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

type impl struct {
	*service.Service[config.Root]
	auto.Services
}

func (a *impl) applyConfig(ctx context.Context, cfg config.Root) error {
	grp, ctx := errgroup.WithContext(ctx)

	// pull brightness from all the devices, notify via readEvents
	readEvents := make(chan func(rs *readState))
	lightClient := traits.NewLightApiClient(a.Node.ClientConn())
	for _, device := range cfg.Devices {
		grp.Go(func() error {
			return pull.Changes(ctx, brightnessPuller{
				client: lightClient,
				name:   device,
				normal: cfg.Normal,
			}, readEvents)
		})
	}

	// apply changes to the read state, notify of reset times via resetTimes
	resetTimes := make(chan time.Time) // sending a zero time means no reset
	grp.Go(func() error {
		defer close(resetTimes)
		// if we get a lot of readState updates in one go we can skip
		// processing the state until things quiet down, then we
		// process the settled state once.
		// These chans do this.
		readStateUpdates := make(chan struct{})
		defer close(readStateUpdates)
		readStateUpdated := minibus.DropExcess(readStateUpdates)

		rs := &readState{
			Brightness: make(map[string]value),
		}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-readEvents:
				event(rs)
				err := chans.SendContext(ctx, readStateUpdates, struct{}{})
				if err != nil {
					return err
				}
			case <-readStateUpdated:
				var t time.Time
				for _, v := range rs.Brightness {
					switch cfg.TimerStart {
					case config.TimerStartAfterEnter:
						cmp := v.EnterAt
						if cmp.IsZero() {
							continue
						}
						if t.IsZero() || cmp.Before(t) {
							t = cmp
						}
					case config.TimerStartAfterChange:
						cmp := v.UpdateAt
						if cmp.IsZero() {
							continue
						}
						if t.IsZero() || t.Before(cmp) {
							t = cmp
						}
					}
				}
				if err := chans.SendContext(ctx, resetTimes, t); err != nil {
					return err
				}
			}
		}
	})

	// reset the brightness of the devices
	grp.Go(func() error {
		timer := time.NewTimer(math.MaxInt64)
		timer.Stop()
		var timerT time.Time

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case t := <-resetTimes:
				if t.IsZero() {
					timer.Stop()
					timerT = time.Time{}
					continue
				}
				if timerT.IsZero() || t.Before(timerT) {
					timerT = t
					delay := t.Sub(time.Now()) + cfg.ResetDelay.Or(config.DefaultResetDelay)
					if delay <= 0 {
						// assume the reset has already happened
						timer.Stop()
						continue
					}
					timer.Reset(delay)
				}
			case <-timer.C:
				var errs error
				for _, device := range cfg.Devices {
					req := &traits.UpdateBrightnessRequest{
						Name:       device,
						Brightness: &traits.Brightness{LevelPercent: cfg.ResetState},
					}
					_, err := lightClient.UpdateBrightness(ctx, req)
					if err != nil {
						errs = multierr.Append(errs, fmt.Errorf("update %q: %w", device, err))
					}
				}
				if errs != nil {
					a.Logger.Error("failed to reset brightness", zap.Errors("error", multierr.Errors(errs)))
				}
			}
		}
	})
	return nil
}

type readState struct {
	Brightness map[string]value
}

func (rs *readState) SetBrightness(device string, now time.Time, v *traits.Brightness, normal config.StateRange) {
	if v == nil {
		delete(rs.Brightness, device)
		return
	}
	val := rs.Brightness[device]
	val.V = v

	var minNorm, maxNorm float32
	if normal.Min != nil {
		minNorm = *normal.Min
	}
	if normal.Max != nil {
		maxNorm = *normal.Max
	}
	if v.LevelPercent < minNorm || v.LevelPercent > maxNorm {
		val.UpdateAt = now
		if val.EnterAt.IsZero() {
			val.EnterAt = now
		}
	} else {
		val.EnterAt = time.Time{}
		val.UpdateAt = time.Time{}
	}
	rs.Brightness[device] = val
}

type value struct {
	V *traits.Brightness
	// Times when V left the normal state.
	// EnterAt is when V first left normal, UpdateAt is the time of the most recent update to V that was off-normal.
	EnterAt, UpdateAt time.Time
}

type brightnessPuller struct {
	client traits.LightApiClient
	name   string
	normal config.StateRange
}

func (b brightnessPuller) Pull(ctx context.Context, changes chan<- func(rs *readState)) error {
	stream, err := b.client.PullBrightness(ctx, &traits.PullBrightnessRequest{Name: b.name})
	if err != nil {
		return fmt.Errorf("pull %q: %w", b.name, err)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("pull recv %q: %w", b.name, err)
		}
		for _, change := range resp.GetChanges() {
			t := change.GetChangeTime().AsTime()
			if t.IsZero() {
				t = time.Now()
			}
			err := chans.SendContext(ctx, changes, func(rs *readState) {
				rs.SetBrightness(b.name, t, change.GetBrightness(), b.normal)
			})
			if err != nil {
				return err
			}
		}
	}
}

func (b brightnessPuller) Poll(ctx context.Context, changes chan<- func(rs *readState)) error {
	res, err := b.client.GetBrightness(ctx, &traits.GetBrightnessRequest{Name: b.name})
	if err != nil {
		return fmt.Errorf("get %q: %w", b.name, err)
	}
	return chans.SendContext(ctx, changes, func(rs *readState) {
		rs.SetBrightness(b.name, time.Now(), res, b.normal)
	})
}
