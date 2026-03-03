package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestLifecycle(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		tt := newLifecycleTester(t)
		tt.tick()
		gotState, err := tt.sub.Start()
		tt.assertNoErr(err)
		tt.assertNoApply()

		wantState := State{
			Active:           true,
			LastInactiveTime: tt.lastTick(),
			LastActiveTime:   tt.now,
		}
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)
	})
	t.Run("start,start", func(t *testing.T) {
		tt := newLifecycleTester(t)
		_, _ = tt.sub.Start()
		_, err := tt.sub.Start()
		tt.assertErr(ErrAlreadyStarted, err)
		tt.assertCurrentState(State{
			Active:           true,
			LastInactiveTime: tt.now,
			LastActiveTime:   tt.now,
		})
	})

	t.Run("configure", func(t *testing.T) {
		tt := newLifecycleTester(t)
		tt.tick()
		gotState, err := tt.sub.Configure([]byte("hello"))
		tt.assertNoErr(err)
		tt.assertNoApply()

		wantState := State{
			LastInactiveTime: tt.lastTick(),
			LastConfigTime:   tt.now,
			Config:           []byte("hello"),
		}
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)
	})

	t.Run("configure,configure", func(t *testing.T) {
		tt := newLifecycleTester(t)
		tt.tick()
		gotState, err := tt.sub.Configure([]byte("hello"))
		tt.assertNoErr(err)
		tt.assertNoApply()

		wantState := State{
			LastInactiveTime: tt.lastTick(),
			LastConfigTime:   tt.now,
			Config:           []byte("hello"),
		}
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)

		tt.tick()
		gotState, err = tt.sub.Configure([]byte("World"))
		tt.assertNoErr(err)
		tt.assertNoApply()

		wantState.LastConfigTime = tt.now
		wantState.Config = []byte("World")
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)
	})

	t.Run("configure,error", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantErr := errors.New("expected parse err")
		tt.setupParseErr(wantErr)
		gotState, gotErr := tt.sub.Configure([]byte("hello"))
		tt.assertNoErr(gotErr)
		wantState := State{
			LastInactiveTime: tt.now,
			Err:              wantErr,
			LastErrTime:      tt.now,
		}
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)
	})

	t.Run("configure,start", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.LastConfigTime = tt.now
		wantState.Config = []byte("hello")
		_, _ = tt.sub.Configure([]byte("hello"))

		tt.tick()
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		wantState.Loading = true
		wantState.LastLoadingStartTime = tt.now
		unblock := tt.setupApply().withTick().blockUntilCall()
		gotState, err := tt.sub.Start()
		tt.assertNoErr(err)
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)

		wantState.Loading = false
		wantState.LastLoadingEndTime = tt.nextTick()

		unblock()
		tt.waitForState(wantState, 100*time.Millisecond)
		tt.assertNextApplyConfig("hello")
	})

	t.Run("start,configure", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		_, _ = tt.sub.Start()

		tt.tick()
		wantState.Config = []byte("hello")
		wantState.LastConfigTime = tt.now
		wantState.Loading = true
		wantState.LastLoadingStartTime = tt.now
		unblock := tt.setupApply().withTick().blockUntilCall()
		gotState, err := tt.sub.Configure([]byte("hello"))
		tt.assertNoErr(err)
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)

		wantState.Loading = false
		wantState.LastLoadingEndTime = tt.nextTick()

		unblock()
		tt.waitForState(wantState, 100*time.Millisecond)
		tt.assertNextApplyConfig("hello")
	})

	t.Run("start,configure,configure,error", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		wantState.Config = []byte("hello")
		wantState.LastConfigTime = tt.now
		wantState.Loading = true
		wantState.LastLoadingStartTime = tt.now
		unblock := tt.setupApply().blockUntilCall()
		tt.Cleanup(unblock) // we want to stay blocked, but still clean up at the end of the test
		_, _ = tt.sub.Start()
		_, _ = tt.sub.Configure([]byte("hello"))

		_, gotErr := tt.sub.Configure([]byte("world"))
		tt.assertErr(ErrAlreadyLoading, gotErr)
		tt.assertCurrentState(wantState)
	})

	t.Run("configure,start,error", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.LastConfigTime = tt.now
		wantState.Config = []byte("hello")
		_, _ = tt.sub.Configure([]byte("hello"))

		tt.tick()
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		wantState.Loading = true
		wantState.LastLoadingStartTime = tt.now
		wantErr := errors.New("expected apply error")
		unblock := tt.setupApply().withTick().withErr(wantErr).blockUntilCall()
		gotState, err := tt.sub.Start()
		tt.assertNoErr(err)
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)

		wantState.Loading = false
		wantState.LastLoadingEndTime = tt.nextTick()
		wantState.Active = false
		wantState.LastInactiveTime = tt.nextTick()
		wantState.Err = wantErr
		wantState.LastErrTime = tt.nextTick()
		wantState.FailedAttempts = 1

		unblock()
		tt.waitForState(wantState, 100*time.Millisecond)
		tt.assertNextApplyConfig("hello")
	})

	t.Run("start,configure,error", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		_, _ = tt.sub.Start()

		tt.tick()
		wantState.Config = []byte("hello")
		wantState.LastConfigTime = tt.now
		wantState.Loading = true
		wantState.LastLoadingStartTime = tt.now
		wantErr := errors.New("expected apply error")
		unblock := tt.setupApply().withTick().withErr(wantErr).blockUntilCall()
		gotState, err := tt.sub.Configure([]byte("hello"))
		tt.assertNoErr(err)
		tt.assertCurrentState(wantState)
		tt.testState(wantState, gotState)

		wantState.Loading = false
		wantState.LastLoadingEndTime = tt.nextTick()
		wantState.Active = false
		wantState.LastInactiveTime = tt.nextTick()
		wantState.Err = wantErr
		wantState.LastErrTime = tt.nextTick()
		wantState.FailedAttempts = 1

		unblock()
		tt.waitForState(wantState, 100*time.Millisecond)
		tt.assertNextApplyConfig("hello")
	})

	t.Run("stop", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		_, err := tt.sub.Stop()
		tt.assertErr(ErrAlreadyStopped, err)
		tt.assertCurrentState(wantState)
	})

	t.Run("start,stop", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.Active = true
		wantState.LastActiveTime = tt.now
		_, _ = tt.sub.Start()

		tt.tick()
		wantState.Active = false
		wantState.LastInactiveTime = tt.now
		gotState, err := tt.sub.Stop()
		tt.assertNoErr(err)
		tt.assertNoApply()
		tt.testState(wantState, gotState)
		tt.assertCurrentState(wantState)
	})

	t.Run("configure,stop", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.LastInactiveTime = tt.now

		tt.tick()
		wantState.Config = []byte("hello")
		wantState.LastConfigTime = tt.now
		_, _ = tt.sub.Configure([]byte("hello"))

		tt.tick()
		_, err := tt.sub.Stop()
		tt.assertErr(ErrAlreadyStopped, err)
		tt.assertCurrentState(wantState)
	})

	t.Run("configure,start,stop", func(t *testing.T) {
		tt := newLifecycleTester(t)
		wantState := State{}
		wantState.Active = true
		wantState.Config = []byte("hello")
		wantState.LastInactiveTime = tt.now
		wantState.LastActiveTime = tt.now
		wantState.LastLoadingStartTime = tt.now
		wantState.LastLoadingEndTime = tt.now
		wantState.LastConfigTime = tt.now
		unblock := tt.setupApply().blockUntilCall()
		_, _ = tt.sub.Configure(wantState.Config)
		_, _ = tt.sub.Start()

		unblock()
		tt.waitForState(wantState, 100*time.Millisecond)

		tt.tick()
		wantState.Active = false
		wantState.LastInactiveTime = tt.now
		gotState, err := tt.sub.Stop()
		tt.assertNoErr(err)
		tt.testState(wantState, gotState)
		tt.assertCurrentState(wantState)
		tt.assertNextApplyContextCancelled(time.Millisecond)
	})

	t.Run("retry", func(t *testing.T) {
		l := newRetryLogger(t)
		tt := newLifecycleTester(t, WithRetry[string](RetryWithLogger(l.Log), RetryWithInitialDelay(10*time.Millisecond), RetryWithMinDelay(10*time.Millisecond)))
		applyErr := errors.New("expected apply error")
		tt.setupApply().withErr(applyErr)
		unblock := tt.setupApply().withTick().blockUntilCall()

		wantState := State{}
		wantState.Active = true
		wantState.Loading = true
		wantState.Config = []byte("hello")
		wantState.LastInactiveTime = tt.now
		wantState.LastActiveTime = tt.now
		wantState.LastLoadingStartTime = tt.now
		wantState.LastConfigTime = tt.now
		wantState.NextAttemptTime = tt.now.Add(10 * time.Millisecond) // initial delay
		wantState.Err = applyErr
		wantState.FailedAttempts = 1
		_, _ = tt.sub.Configure([]byte("hello"))
		_, _ = tt.sub.Start()

		tt.waitForState(wantState, 100*time.Millisecond)
		l.assertLog(RetryContext{
			Attempt: 1,
			Err:     applyErr,
			Delay:   10 * time.Millisecond,
			T0:      tt.now,
		})
		tt.assertNextApplyContextCancelled(time.Millisecond)

		unblock()
		wantState.Loading = false
		wantState.NextAttemptTime = time.Time{}
		wantState.LastLoadingEndTime = tt.nextTick()
		tt.waitForState(wantState, 100*time.Millisecond)
	})
}

type lifecycleTester struct {
	*testing.T
	sub *Service[string]

	now time.Time

	m          sync.Mutex
	applyCalls []applyCall

	parseSetup []parseSetup
	applySetup []applySetup
}

type applyCall struct {
	ctx    context.Context
	config string
}

type parseSetup struct {
	err error
}

type applySetup struct {
	wait <-chan struct{}
	tick bool
	err  error
}

func (a *applySetup) withTick() *applySetup {
	a.tick = true
	return a
}

func (a *applySetup) withErr(err error) *applySetup {
	a.err = err
	return a
}

// blockUntilCall causes the next apply call to block until the returned func is invoked.
func (a *applySetup) blockUntilCall() func() {
	ch := make(chan struct{})
	a.wait = ch

	closed := false
	return func() {
		if !closed {
			close(ch)
		}
	}
}

func newLifecycleTester(t *testing.T, opts ...Option[string]) *lifecycleTester {
	tt := &lifecycleTester{
		T:   t,
		now: time.UnixMilli(0), // make sure time isn't the zero time
	}
	opts = append([]Option[string]{
		WithNow[string](func() time.Time { return tt.now }),
		WithParser(func(data []byte) (string, error) {
			if len(tt.parseSetup) > 0 {
				setup := tt.parseSetup[0]
				tt.parseSetup = tt.parseSetup[1:]
				if setup.err != nil {
					return "", setup.err
				}
			}
			return string(data), nil
		}),
	}, opts...)
	s := New[string](func(ctx context.Context, config string) error {
		tt.pushApplyCall(applyCall{ctx, config})
		if setup, ok := tt.tryPopApplySetup(); ok {
			if setup.wait != nil {
				<-setup.wait
			}
			if setup.tick {
				tt.tick()
			}
			if setup.err != nil {
				return setup.err
			}
		}
		return nil
	},
		opts...,
	)
	tt.sub = s
	return tt
}

func (tt *lifecycleTester) pushApplyCall(ac applyCall) {
	tt.m.Lock()
	tt.applyCalls = append(tt.applyCalls, ac)
	tt.m.Unlock()
}

func (tt *lifecycleTester) popApplyCall() applyCall {
	tt.m.Lock()
	defer tt.m.Unlock()
	if len(tt.applyCalls) == 0 {
		tt.Fatalf("Expecting at least one apply call")
	}
	ac := tt.applyCalls[0]
	tt.applyCalls = tt.applyCalls[1:]
	return ac
}

func (tt *lifecycleTester) tryPopApplySetup() (applySetup, bool) {
	tt.m.Lock()
	defer tt.m.Unlock()
	if len(tt.applySetup) == 0 {
		return applySetup{}, false
	}
	setup := tt.applySetup[0]
	tt.applySetup = tt.applySetup[1:]
	return setup, true
}

// tick adds 1 second to the current time.
func (tt *lifecycleTester) tick() {
	tt.m.Lock()
	tt.now = tt.now.Add(time.Second)
	tt.m.Unlock()
}

// lastTick returns the time before tick was called.
func (tt *lifecycleTester) lastTick() time.Time {
	tt.m.Lock()
	defer tt.m.Unlock()
	return tt.now.Add(-time.Second)
}

// nextTick returns the time after tick is next called.
func (tt *lifecycleTester) nextTick() time.Time {
	tt.m.Lock()
	defer tt.m.Unlock()
	return tt.now.Add(time.Second)
}

func (tt *lifecycleTester) setupParseErr(err error) {
	tt.m.Lock()
	tt.parseSetup = append(tt.parseSetup, parseSetup{err: err})
	tt.m.Unlock()
}

func (tt *lifecycleTester) setupApply() *applySetup {
	tt.m.Lock()
	defer tt.m.Unlock()
	tt.applySetup = append(tt.applySetup, applySetup{})
	return &tt.applySetup[len(tt.applySetup)-1]
}

func (tt *lifecycleTester) assertNoErr(err error) {
	tt.Helper()
	if err != nil {
		tt.Fatalf("Expecting no error, got %v", err)
	}
}

func (tt *lifecycleTester) assertErr(wantErr, gotErr error) {
	tt.Helper()
	if !errors.Is(gotErr, wantErr) {
		tt.Fatalf("Expecting error, want %v, got %v", wantErr, gotErr)
	}
}

func (tt *lifecycleTester) assertNoApply() {
	tt.Helper()
	if len(tt.applyCalls) > 0 {
		tt.Fatalf("Expecting no apply calls, got %d", len(tt.applyCalls))
	}
}

func (tt *lifecycleTester) assertNextApplyConfig(config string) {
	tt.Helper()
	a := tt.popApplyCall()
	if a.config != config {
		tt.Fatalf("Apply call config want %s, got %s", config, a.config)
	}
}

func (tt *lifecycleTester) assertNextApplyContextCancelled(wait time.Duration) {
	tt.Helper()
	a := tt.popApplyCall()

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		tt.Fatalf("Timeout waiting for context cancellation")
	case <-a.ctx.Done():
		return
	}
}

func (tt *lifecycleTester) waitForState(want State, wait time.Duration) {
	tt.Helper()
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	timer := time.NewTimer(wait)
	defer timer.Stop()

	gotState, stateChanges := tt.sub.StateAndChanges(ctx)
	if diff := stateDiff(want, gotState); diff == "" {
		return // state is already equal
	}
	lastState := gotState
	for {
		select {
		case <-timer.C:
			tt.Fatalf("Timeout waiting for state, diff with last (-want, +got)\n%s", stateDiff(want, lastState))
		case gotState := <-stateChanges:
			if diff := stateDiff(want, gotState); diff == "" {
				return // state is already equal
			}
			lastState = gotState
		}
	}
}

func (tt *lifecycleTester) assertCurrentState(state State) {
	tt.Helper()
	tt.testState(state, tt.sub.State())
}

func (tt *lifecycleTester) testState(want, got State) {
	tt.Helper()
	if diff := stateDiff(want, got); diff != "" {
		tt.Fatalf("State (-want, +got)\n%s", diff)
	}
}

func stateDiff(want, got State) string {
	byteStringTransformer := func(a []byte) string {
		return string(a)
	}
	return cmp.Diff(want, got,
		cmp.Transformer("byteSliceToString", byteStringTransformer),
		cmpopts.EquateErrors(),
	)
}

func newRetryLogger(t *testing.T) *testRetryLogger {
	return &testRetryLogger{T: t}
}

type testRetryLogger struct {
	*testing.T
	logs []RetryContext
}

func (l *testRetryLogger) Log(ctx RetryContext) {
	l.logs = append(l.logs, ctx)
}

func (l *testRetryLogger) assertLog(want RetryContext) {
	l.Helper()
	if len(l.logs) == 0 {
		l.Fatalf("Expecting at least one retry log")
	}
	got := l.logs[0]
	l.logs = l.logs[1:]
	if diff := cmp.Diff(want, got, cmpopts.EquateErrors()); diff != "" {
		l.Fatalf("Retry log (-want, +got)\n%s", diff)
	}
}
