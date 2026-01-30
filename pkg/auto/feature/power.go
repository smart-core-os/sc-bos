package feature

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/util/times"
	"github.com/smart-core-os/sc-bos/pkg/auto/runstate"
	"github.com/smart-core-os/sc-bos/pkg/util/state"
)

type TurnOffScreensOutsideWorkingHours struct {
	sm     *state.Manager[runstate.RunState]
	smOnce sync.Once

	offBefore, onBefore time.Duration // relative to the start of a day
	disabledWeekdays    map[time.Weekday]struct{}

	screens     []string
	powerClient traits.OnOffApiClient

	logger *zap.Logger
	now    func() time.Time

	runningCtx context.Context
	stop       context.CancelFunc
}

func (t *TurnOffScreensOutsideWorkingHours) Start(_ context.Context) error {
	t.state().Update(runstate.Starting)
	t.runningCtx, t.stop = context.WithCancel(context.Background())

	go func() {
		t.state().Update(runstate.Running)
		defer t.state().Update(runstate.Stopped)
		now := t.now()

		for {
			// control devices based on the state they should be in now
			powerState := t.expectedPowerState(now)
			ctx, stop := context.WithTimeout(t.runningCtx, 15*time.Second)
			err := t.setScreenPower(ctx, powerState)
			if err != nil {
				t.logger.Error("Setting power was not completely successful", zap.Error(err))
			}
			stop() // clean up resources if the ctx timeout wasn't triggered

			// sleep until we expect the state should change
			wakeTime := t.nextWakeTime(now)
			now = t.now()
			if wakeTime.Before(now) {
				continue
			}
			waker := time.NewTimer(wakeTime.Sub(now))
			select {
			case <-t.runningCtx.Done():
				waker.Stop()
				return
			case now = <-waker.C:
			}
		}
	}()
	return nil
}

func (t *TurnOffScreensOutsideWorkingHours) Stop() error {
	if t.stop != nil {
		t.stop()
	}
	return nil
}

func (t *TurnOffScreensOutsideWorkingHours) WaitForStateChange(ctx context.Context, sourceState runstate.RunState) error {
	return t.state().WaitForStateChange(ctx, sourceState)
}

func (t *TurnOffScreensOutsideWorkingHours) CurrentState() runstate.RunState {
	return t.state().CurrentState()
}

func (t *TurnOffScreensOutsideWorkingHours) state() *state.Manager[runstate.RunState] {
	t.smOnce.Do(func() {
		t.sm = state.NewManager(runstate.Idle)
	})
	return t.sm
}

func (t *TurnOffScreensOutsideWorkingHours) setScreenPower(ctx context.Context, state traits.OnOff_State) error {
	responses := make(chan error)
	count := 0
	for _, screen := range t.screens {
		screen := screen
		count++
		go func() {
			_, err := t.powerClient.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: screen, OnOff: &traits.OnOff{State: state}})
			responses <- err
		}()
	}

	var err error
	for range count {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-responses:
			err = multierr.Append(err, res)
		}
	}
	return err
}

func (t *TurnOffScreensOutsideWorkingHours) expectedPowerState(now time.Time) traits.OnOff_State {
	if _, off := t.disabledWeekdays[now.Weekday()]; off {
		return traits.OnOff_OFF
	}
	offBefore, onBefore := t.offTime(now)
	if now.Before(offBefore) {
		return traits.OnOff_OFF
	}
	if now.Before(onBefore) {
		return traits.OnOff_ON
	}
	return traits.OnOff_OFF
}

func (t *TurnOffScreensOutsideWorkingHours) nextWakeTime(now time.Time) time.Time {
	if _, off := t.disabledWeekdays[now.Weekday()]; !off {
		offBefore, onBefore := t.offTime(now)
		if now.Before(offBefore) {
			return offBefore
		}
		if now.Before(onBefore) {
			return onBefore
		}
	}
	nextOnDay := times.NextWeekday(now, func(wd time.Weekday) bool { _, ok := t.disabledWeekdays[wd]; return !ok })
	offBefore, _ := t.offTime(nextOnDay)
	if !now.Before(offBefore) {
		panic(fmt.Errorf("there should always be a next time! now=%v, onBetween=[%v,%v), disabled=%v",
			now, t.offBefore, t.onBefore, t.disabledWeekdays))
	}
	return offBefore
}

func (t *TurnOffScreensOutsideWorkingHours) offTime(now time.Time) (offBefore, onBefore time.Time) {
	startOfDay := times.StartOfDay(now)
	return startOfDay.Add(t.offBefore), startOfDay.Add(t.onBefore)
}
