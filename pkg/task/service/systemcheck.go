package service

import (
	"context"
	"sync"
	"time"
)

// ManagedSystemCheck is a SystemCheck whose lifetime follows a service rather than the code reporting to it.
//
// Wrap the service with Track. Each Start creates a fresh underlying check, and each Stop disposes it.
// While the service runs, the check reflects the service's own State: a config or start failure marks the check
// failed, and once the service has loaded its config the check is marked running.
// Whatever the service reports via MarkRunning and MarkFailed layers on top: once it has reported, the framework no
// longer marks the check running for it, a report stands until the next failed attempt to apply config,
// and the service's last report is restored when a framework failure clears.
//
// A ManagedSystemCheck handle stays valid for as long as the service exists, across stops and starts.
// Reports made while the service is stopped are discarded at the next Start.
type ManagedSystemCheck struct {
	newCheck func() SystemCheck

	lifeMu sync.Mutex // serialises Start and Stop of the tracked lifecycle

	mu        sync.Mutex
	run       uint64 // changes when a run starts or stops, so a superseded watcher stops writing
	check     SystemCheck
	stopWatch context.CancelFunc
	reported  bool    // the service has called MarkRunning or MarkFailed since the run started
	reportErr error   // the service's last report, nil for MarkRunning
	failed    failure // the framework failure last written to check, zero if none
}

// failure identifies one failed attempt to start or configure a service.
type failure struct {
	attempts int
	errTime  time.Time
	msg      string
}

var _ SystemCheck = (*ManagedSystemCheck)(nil)

// NewManagedSystemCheck returns a ManagedSystemCheck that calls newCheck to create each run's underlying check.
// newCheck may return nil if the check can't be created, in which case that run has no check.
func NewManagedSystemCheck(newCheck func() SystemCheck) *ManagedSystemCheck {
	return &ManagedSystemCheck{newCheck: newCheck}
}

// MarkRunning records that the service is operating correctly.
func (m *ManagedSystemCheck) MarkRunning() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reported, m.reportErr = true, nil
	if m.check != nil {
		m.check.MarkRunning()
	}
}

// MarkFailed records that the service is not operating correctly because of err.
func (m *ManagedSystemCheck) MarkFailed(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reported, m.reportErr = true, err
	if m.check != nil {
		m.check.MarkFailed(err)
	}
}

// Dispose does nothing.
//
// Deprecated: the check is disposed when the tracked service stops; there is no need to call Dispose.
func (m *ManagedSystemCheck) Dispose() {}

// Track returns l wrapped so that starting and stopping it creates and disposes m's underlying check.
func (m *ManagedSystemCheck) Track(l Lifecycle) Lifecycle {
	return &trackedLifecycle{Lifecycle: l, check: m}
}

type trackedLifecycle struct {
	Lifecycle
	check *ManagedSystemCheck
}

func (t *trackedLifecycle) Start() (State, error) {
	m := t.check
	m.lifeMu.Lock()
	defer m.lifeMu.Unlock()
	if t.Lifecycle.State().Active {
		// keep the current run's check, Start reports ErrAlreadyStarted
		return t.Lifecycle.Start()
	}
	ctx, run := m.beginRun()
	// A failed Start keeps the run, so the check shows why the service isn't running.
	// The next Stop, which Map.Delete always calls, disposes it.
	state, err := t.Lifecycle.Start()
	go m.watch(ctx, run, t.Lifecycle)
	return state, err
}

func (t *trackedLifecycle) Stop() (State, error) {
	m := t.check
	m.lifeMu.Lock()
	defer m.lifeMu.Unlock()
	// detach the watcher first, so the stop itself, which leaves State.Err as it was, isn't reported as a failure
	m.detachWatcher()
	state, err := t.Lifecycle.Stop()
	m.disposeCheck()
	return state, err
}

func (m *ManagedSystemCheck) beginRun() (context.Context, uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// a service that failed to load stops itself without Stop being called, so the last run may still be here
	m.detachWatcherLocked()
	m.disposeCheckLocked()
	m.reported, m.reportErr, m.failed = false, nil, failure{}
	m.check = m.newCheck()
	ctx, cancel := context.WithCancel(context.Background())
	m.stopWatch = cancel
	return ctx, m.run
}

func (m *ManagedSystemCheck) detachWatcher() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detachWatcherLocked()
}

func (m *ManagedSystemCheck) detachWatcherLocked() {
	m.run++
	if m.stopWatch != nil {
		m.stopWatch()
		m.stopWatch = nil
	}
}

func (m *ManagedSystemCheck) disposeCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disposeCheckLocked()
}

func (m *ManagedSystemCheck) disposeCheckLocked() {
	if m.check != nil {
		m.check.Dispose()
		m.check = nil
	}
}

func (m *ManagedSystemCheck) watch(ctx context.Context, run uint64, l Lifecycle) {
	_, changes := l.StateAndChanges(ctx)
	m.reconcile(run, l)
	for range changes {
		m.reconcile(run, l)
	}
}

// reconcile brings the check in line with l's current state.
// Notifications are only a prompt to look: the service sends them asynchronously, so they can arrive out of order.
func (m *ManagedSystemCheck) reconcile(run uint64, l Lifecycle) {
	// read outside m.mu: the service may call MarkRunning or MarkFailed while holding its own lock
	state := l.State()

	m.mu.Lock()
	defer m.mu.Unlock()
	if run != m.run || m.check == nil {
		return
	}

	// Err stays set after a retry succeeds, so it only counts while the service is stopped or waiting to retry.
	if state.Err != nil && (!state.Active || !state.NextAttemptTime.IsZero()) {
		// Mark each failed attempt once, as notifications repeat. A report the service makes after the attempt
		// stands until the next one fails, but one it makes during the attempt, like MarkRunning once connected,
		// doesn't hide that the attempt then failed.
		f := failure{attempts: state.FailedAttempts, errTime: state.LastErrTime, msg: state.Err.Error()}
		if f != m.failed {
			m.failed = f
			m.check.MarkFailed(state.Err)
		}
		return
	}

	if m.failed != (failure{}) {
		m.failed = failure{}
		if m.reported {
			if m.reportErr != nil {
				m.check.MarkFailed(m.reportErr)
			} else {
				m.check.MarkRunning()
			}
			return
		}
	}

	// Only Start clears reported, so a config reload never overwrites what the service reported.
	if !m.reported && state.Active && !state.Loading && state.Config != nil {
		m.check.MarkRunning()
	}
}
