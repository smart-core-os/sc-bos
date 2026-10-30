package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestManagedSystemCheck(t *testing.T) {
	errBadConfig := errors.New("bad config")
	errConnLost := errors.New("connection lost")

	t.Run("never reports", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(nil))
			tt.start()
			tt.wantStatus("") // no config yet, so nothing to say
			tt.configure("ok")
			tt.wantStatus(statusRunning)
			tt.stop()
		})
	})

	t.Run("unspecified while loading", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			tt := newSystemCheckTester(t, func(ctx context.Context, _ string) error {
				<-release
				return nil
			})
			tt.start()
			tt.configure("ok")
			tt.wantStatus("")
			close(release)
			tt.wantStatus(statusRunning)
			tt.stop()
		})
	})

	t.Run("config error then fix", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(map[string]error{"bad": errBadConfig}))
			tt.start()
			tt.configure("bad")
			tt.wantStatus(errBadConfig.Error())

			// the failed apply stopped the service, so it needs starting again
			tt.configure("ok")
			tt.wantStatus(errBadConfig.Error())
			tt.start()
			tt.wantStatus(statusRunning)
			tt.wantChecks(2)
			tt.stop()
		})
	})

	t.Run("parse error without config", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(nil))
			tt.start()
			_, _ = tt.lifecycle.Configure([]byte(unparseable))
			tt.wantStatus(errUnparseable.Error())
			tt.stop()
		})
	})

	t.Run("retry with sticky Err", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			attempts := 0
			tt := newSystemCheckTester(t, func(ctx context.Context, _ string) error {
				attempts++
				if attempts <= 2 {
					return errBadConfig
				}
				return nil
			}, WithRetry[string](RetryWithInitialDelay(time.Second), RetryWithFactor(1)))
			tt.start()
			tt.configure("ok")
			tt.wantStatus(errBadConfig.Error())
			time.Sleep(2 * time.Second)
			tt.wantStatus(statusRunning)
			if err := tt.lifecycle.State().Err; err == nil {
				t.Fatalf("State().Err = nil, want the last retry's error, which this test relies on")
			}
			tt.stop()
			// the stop leaves Err set, which mustn't be reported as a failure
			tt.wantLastStatus(statusRunning)
		})
	})

	t.Run("report between retries", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(map[string]error{"bad": errBadConfig}),
				WithRetry[string](RetryWithInitialDelay(time.Second), RetryWithFactor(1)))
			tt.start()
			tt.configure("bad")
			tt.wantStatus(errBadConfig.Error())

			// stands while the service waits to retry
			tt.check.MarkFailed(errConnLost)
			tt.wantStatus(errConnLost.Error())

			// the next attempt failing is newer
			time.Sleep(time.Second)
			tt.wantStatus(errBadConfig.Error())
			tt.stop()
		})
	})

	t.Run("running reported during a failing attempt", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// like drivers that mark running once connected, then fail later in the same attempt
			var check *ManagedSystemCheck
			tt := newSystemCheckTester(t, func(ctx context.Context, _ string) error {
				check.MarkRunning()
				return errBadConfig
			}, WithRetry[string](RetryWithInitialDelay(time.Second), RetryWithFactor(1)))
			check = tt.check
			tt.start()
			tt.configure("ok")
			tt.wantStatus(errBadConfig.Error())
			for range 3 {
				time.Sleep(time.Second)
				tt.wantStatus(errBadConfig.Error())
			}
			tt.stop()
		})
	})

	t.Run("report restored when error clears", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var check *ManagedSystemCheck
			attempts := 0
			tt := newSystemCheckTester(t, func(ctx context.Context, _ string) error {
				attempts++
				if attempts == 1 {
					check.MarkFailed(errConnLost)
					return errBadConfig
				}
				return nil
			}, WithRetry[string](RetryWithInitialDelay(time.Second)))
			check = tt.check
			tt.start()
			tt.configure("ok")
			tt.wantStatus(errBadConfig.Error())
			time.Sleep(time.Second)
			tt.wantStatus(errConnLost.Error())
			tt.stop()
		})
	})

	t.Run("report survives reload", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(nil))
			tt.start()
			tt.configure("ok")
			tt.wantStatus(statusRunning)
			tt.check.MarkFailed(errConnLost)
			tt.wantStatus(errConnLost.Error())
			tt.configure("ok2")
			tt.wantStatus(errConnLost.Error())
			tt.stop()
		})
	})

	t.Run("stop and start", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(nil))
			tt.start()
			tt.configure("ok")
			tt.wantStatus(statusRunning)
			tt.stop()
			tt.wantChecks(1)
			if tt.checks.live() != nil {
				t.Fatalf("check not disposed on stop")
			}

			// reports while stopped are dropped
			tt.check.MarkFailed(errConnLost)
			tt.wantLastStatus(statusRunning)

			tt.start()
			tt.wantChecks(2)
			tt.wantStatus(statusRunning)
			tt.stop()
		})
	})

	t.Run("start while started", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(nil))
			tt.start()
			tt.configure("ok")
			tt.check.MarkFailed(errConnLost)
			if _, err := tt.lifecycle.Start(); !errors.Is(err, ErrAlreadyStarted) {
				t.Fatalf("Start() error = %v, want %v", err, ErrAlreadyStarted)
			}
			tt.wantChecks(1)
			tt.wantStatus(errConnLost.Error())
			tt.stop()
		})
	})

	t.Run("stopped after failing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tt := newSystemCheckTester(t, applyByName(map[string]error{"bad": errBadConfig}))
			tt.start()
			tt.configure("bad")
			tt.wantStatus(errBadConfig.Error())
			if _, err := tt.lifecycle.Stop(); !errors.Is(err, ErrAlreadyStopped) {
				t.Fatalf("Stop() error = %v, want %v", err, ErrAlreadyStopped)
			}
			synctest.Wait()
			if tt.checks.live() != nil {
				t.Fatalf("check not disposed on stop")
			}
		})
	})

	t.Run("no underlying check", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			m := NewManagedSystemCheck(func() SystemCheck { return nil })
			l := m.Track(New(applyByName(nil), WithParser(parseString)))
			_, _ = l.Start()
			_, _ = l.Configure([]byte("ok"))
			m.MarkFailed(errConnLost)
			m.MarkRunning()
			synctest.Wait()
			_, _ = l.Stop()
		})
	})
}

func TestManagedSystemCheck_concurrent(t *testing.T) {
	tt := newSystemCheckTester(t, applyByName(map[string]error{"bad": errors.New("bad config")}),
		WithRetry[string](RetryWithInitialDelay(time.Millisecond)))
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			for j := range 200 {
				switch (i + j) % 6 {
				case 0:
					_, _ = tt.lifecycle.Start()
				case 1:
					_, _ = tt.lifecycle.Stop()
				case 2:
					_, _ = tt.lifecycle.Configure([]byte("ok"))
				case 3:
					_, _ = tt.lifecycle.Configure([]byte("bad"))
				case 4:
					tt.check.MarkFailed(errors.New("connection lost"))
				case 5:
					tt.check.MarkRunning()
				}
			}
		})
	}
	wg.Wait()
	_, _ = tt.lifecycle.Stop()
	if c := tt.checks.live(); c != nil {
		t.Fatalf("check %d not disposed after the final stop", c.id)
	}
}

const statusRunning = "running"

const unparseable = "unparseable"

var errUnparseable = errors.New("unparseable config")

func parseString(data []byte) (string, error) {
	if string(data) == unparseable {
		return "", errUnparseable
	}
	return string(data), nil
}

// applyByName returns an ApplyFunc that fails with errs[cfg], succeeding for configs not in errs.
func applyByName(errs map[string]error) ApplyFunc[string] {
	return func(ctx context.Context, cfg string) error {
		return errs[cfg]
	}
}

type systemCheckTester struct {
	t         *testing.T
	check     *ManagedSystemCheck
	checks    *fakeChecks
	lifecycle Lifecycle
}

func newSystemCheckTester(t *testing.T, apply ApplyFunc[string], opts ...Option[string]) *systemCheckTester {
	t.Helper()
	checks := &fakeChecks{t: t}
	check := NewManagedSystemCheck(checks.new)
	opts = append([]Option[string]{WithParser(parseString)}, opts...)
	lifecycle := check.Track(New(apply, opts...))
	// stops a service a failed test left running, which synctest would otherwise report as leaked goroutines
	t.Cleanup(func() { _, _ = lifecycle.Stop() })
	return &systemCheckTester{
		t:         t,
		check:     check,
		checks:    checks,
		lifecycle: lifecycle,
	}
}

func (tt *systemCheckTester) start() {
	tt.t.Helper()
	if _, err := tt.lifecycle.Start(); err != nil {
		tt.t.Fatalf("Start() error = %v", err)
	}
}

func (tt *systemCheckTester) configure(cfg string) {
	tt.t.Helper()
	if _, err := tt.lifecycle.Configure([]byte(cfg)); err != nil {
		tt.t.Fatalf("Configure(%q) error = %v", cfg, err)
	}
}

// stop stops the service, which may already have stopped itself after failing.
func (tt *systemCheckTester) stop() {
	tt.t.Helper()
	if _, err := tt.lifecycle.Stop(); err != nil && !errors.Is(err, ErrAlreadyStopped) {
		tt.t.Fatalf("Stop() error = %v", err)
	}
	synctest.Wait()
}

// wantStatus waits for the service to settle and checks the live check's status.
func (tt *systemCheckTester) wantStatus(want string) {
	tt.t.Helper()
	synctest.Wait()
	c := tt.checks.live()
	if c == nil {
		tt.t.Fatalf("no live check, want status %q", want)
	}
	if got := c.getStatus(); got != want {
		tt.t.Fatalf("status = %q, want %q", got, want)
	}
}

// wantLastStatus checks the status of the most recently created check, live or not.
func (tt *systemCheckTester) wantLastStatus(want string) {
	tt.t.Helper()
	synctest.Wait()
	if got := tt.checks.last().getStatus(); got != want {
		tt.t.Fatalf("last check status = %q, want %q", got, want)
	}
}

func (tt *systemCheckTester) wantChecks(n int) {
	tt.t.Helper()
	if got := tt.checks.count(); got != n {
		tt.t.Fatalf("created %d checks, want %d", got, n)
	}
}

// fakeChecks creates fakeCheck values, failing the test if a check is created while another is live.
type fakeChecks struct {
	t      *testing.T
	mu     sync.Mutex
	checks []*fakeCheck
}

func (f *fakeChecks) new() SystemCheck {
	if c := f.live(); c != nil {
		f.t.Errorf("check %d created while check %d is live", len(f.checks), c.id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c := &fakeCheck{id: len(f.checks)}
	f.checks = append(f.checks, c)
	return c
}

func (f *fakeChecks) live() *fakeCheck {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.checks {
		if !c.isDisposed() {
			return c
		}
	}
	return nil
}

func (f *fakeChecks) last() *fakeCheck {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.checks[len(f.checks)-1]
}

func (f *fakeChecks) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.checks)
}

// fakeCheck records what it was last marked as: empty if never, statusRunning, or the failure's error text.
type fakeCheck struct {
	id       int
	mu       sync.Mutex
	status   string
	disposed bool
}

func (c *fakeCheck) MarkRunning() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = statusRunning
}

func (c *fakeCheck) MarkFailed(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = err.Error()
}

func (c *fakeCheck) Dispose() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disposed = true
}

func (c *fakeCheck) getStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *fakeCheck) isDisposed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disposed
}
