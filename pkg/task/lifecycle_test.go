package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLifecycle_CurrentState(t *testing.T) {
	wait := 250 * time.Millisecond
	t.Run("create", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.assertCurrentState(StatusInactive, 0)
	})
	t.Run("start", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.startWithin(wait)
		lt.assertCurrentState(StatusActive, wait)
	})
	t.Run("stop", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.stopWithin(wait)
		lt.assertCurrentState(StatusInactive, wait)
	})
	t.Run("start,stop", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.startWithin(wait)
		lt.assertCurrentState(StatusActive, wait)
		lt.stopWithin(wait)
		lt.assertCurrentState(StatusInactive, wait)
	})
	t.Run("start,configure", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.startWithin(wait)
		lt.assertCurrentState(StatusActive, wait)
		lt.applyConfigSleep(time.Millisecond)
		lt.configureWithin("foo", wait)
		lt.assertCurrentState(StatusLoading, wait)
		lt.assertCurrentState(StatusActive, wait)
	})
	t.Run("start,configure,stop", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.startWithin(wait)
		lt.assertCurrentState(StatusActive, wait)
		lt.applyConfigSleep(time.Millisecond)
		lt.configureWithin("foo", wait)
		lt.assertCurrentState(StatusLoading, wait)
		lt.assertCurrentState(StatusActive, wait)
		lt.stopWithin(wait)
		lt.assertCurrentState(StatusInactive, wait)
	})

	t.Run("configure", func(t *testing.T) {
		t.Skip("Configure before start is not currently supported")
		lt := newLifecycleTester(t)
		lt.configureWithin("foo", wait)
		lt.assertCurrentState(StatusInactive, wait)
	})
	t.Run("configure,configure", func(t *testing.T) {
		t.Skip("Configure before start is not currently supported")
		lt := newLifecycleTester(t)
		lt.configureWithin("foo", wait)
		lt.assertCurrentState(StatusInactive, wait)
		lt.configureWithin("foo2", wait)
		lt.assertCurrentState(StatusInactive, wait)
	})

	t.Run("start,configure,error", func(t *testing.T) {
		lt := newLifecycleTester(t)
		lt.startWithin(wait)
		lt.applyConfigError(errors.New("expected"))
		lt.configureWithin("foo", wait)
		lt.assertCurrentState(StatusError, wait)
	})
}

type lifecycleTester struct {
	*testing.T
	*Lifecycle[string]

	applyConfigSetup []applyConfigSetup
	applyConfigCalls []ctxConfig
}

type ctxConfig struct {
	ctx    context.Context
	config string
}

type applyConfigSetup struct {
	sleep time.Duration
	err   error
}

func newLifecycleTester(t *testing.T) *lifecycleTester {
	lt := &lifecycleTester{T: t}
	lt.Lifecycle = NewLifecycle(lt.applyConfig)
	lt.ReadConfig = func(bytes []byte) (string, error) {
		return string(bytes), nil
	}
	return lt
}

func (lt *lifecycleTester) prepareApplyConfig(setup applyConfigSetup) {
	lt.applyConfigSetup = append(lt.applyConfigSetup, setup)
}

func (lt *lifecycleTester) applyConfigSleep(sleep time.Duration) {
	lt.prepareApplyConfig(applyConfigSetup{sleep: sleep})
}

func (lt *lifecycleTester) applyConfigError(err error) {
	lt.prepareApplyConfig(applyConfigSetup{err: err})
}

func (lt *lifecycleTester) applyConfig(ctx context.Context, config string) error {
	lt.applyConfigCalls = append(lt.applyConfigCalls, ctxConfig{ctx: ctx, config: config})
	if len(lt.applyConfigSetup) > 0 {
		setup := lt.applyConfigSetup[0]
		lt.applyConfigSetup = lt.applyConfigSetup[1:]
		if setup.sleep > 0 {
			select {
			case <-time.After(setup.sleep):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if setup.err != nil {
			return setup.err
		}
	}
	return nil
}

func (lt *lifecycleTester) startWithin(wait time.Duration) {
	lt.Helper()

	ctx, stop := context.WithTimeout(context.Background(), wait)
	defer stop()
	if err := lt.Start(ctx); err != nil {
		lt.Fatalf("Start err: %s", err)
	}
}

func (lt *lifecycleTester) configureWithin(config string, wait time.Duration) {
	lt.Helper()

	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		err = lt.Configure([]byte(config))
	}()

	select {
	case <-done:
		if err != nil {
			lt.Fatalf("Configure err: %v", err)
		}
		return // success
	case <-time.After(wait):
		lt.Fatalf("Configure timeout after %s", wait)
	}
}

func (lt *lifecycleTester) stopWithin(wait time.Duration) {
	lt.Helper()

	stopped := make(chan struct{})
	var stopErr error

	go func() {
		defer close(stopped)
		stopErr = lt.Stop()
	}()

	select {
	case <-stopped:
		if stopErr != nil {
			lt.Fatalf("Stop err: %v", stopErr)
		}
		return // success
	case <-time.After(wait):
		lt.Fatalf("Stop timeout after %s", wait)
	}
}

func (lt *lifecycleTester) assertCurrentState(want Status, wait time.Duration) {
	lt.Helper()

	ctx, stop := context.WithTimeout(context.Background(), wait)
	defer stop()
	got := lt.Lifecycle.CurrentState()
	for got != want {
		if err := lt.Lifecycle.WaitForStateChange(ctx, got); err != nil {
			lt.Fatalf("CurrentState want %s, got timeout waiting %s", want, wait)
		}
		got = lt.Lifecycle.CurrentState()
	}

	if got != want {
		lt.Fatalf("CurrentState want %s, got %s", want, got)
	}
}
