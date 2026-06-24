package cloud

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

// ErrNotRegistered is returned by Update, CommitInstall, and FailInstall when
// there is no active registration. Callers can treat it as "nothing to do".
var ErrNotRegistered = errors.New("cloud: no active registration")

// Connectivity represents the cloud connection lifecycle state.
type Connectivity int

const (
	Unconfigured Connectivity = iota // no registration on disk
	Connecting                       // credentials present, no successful check-in yet
	Connected                        // last check-in succeeded
	Failed                           // last check-in failed
)

// ConnState is a point-in-time snapshot of the cloud connection: the current
// registration (nil when unconfigured) together with the latest check-in health.
type ConnState struct {
	Connectivity    Connectivity
	Registration    *Registration // nil iff Connectivity == Unconfigured
	LastCheckInTime time.Time
	LastError       error
	ChangeTime      time.Time // when this snapshot was produced
}

// ConnOption configures a Conn.
type ConnOption func(*Conn)

// WithUpdaterOptions appends opts to the UpdaterOption slice forwarded to each
// newly-created DeploymentUpdater.
func WithUpdaterOptions(opts ...UpdaterOption) ConnOption {
	return func(c *Conn) { c.updaterOpts = append(c.updaterOpts, opts...) }
}

// WithClientFactory sets a factory used to build a Client for a given Registration.
// Primarily useful in tests to inject a fake transport.
func WithClientFactory(f func(Registration) Client) ConnOption {
	return func(c *Conn) { c.newClient = f }
}

// WithSoftwareUpdater enables the update channel on the shared poll check-in, dispatching the
// latestUpdate block to u. When unset the update channel is not reported and latestUpdate is ignored.
func WithSoftwareUpdater(u *SoftwareUpdater) ConnOption {
	return func(c *Conn) { c.softwareUpdater = u }
}

// Conn manages the lifecycle of a cloud connection.
// It tracks the current Registration, broadcasts ConnState changes, and owns a
// DeploymentUpdater that is rebuilt whenever the Registration changes.
// Methods are safe to call concurrently.
type Conn struct {
	// Immutable after construction — no locking required.
	regStore        RegistrationStore
	depStore        *DeploymentStore
	newClient       func(Registration) Client
	updaterOpts     []UpdaterOption
	softwareUpdater *SoftwareUpdater // nil when the update channel is disabled

	// serial is a binary semaphore (buffered channel of size 1, pre-filled).
	// Acquire with lockSerial; release with unlockSerial.
	// Held for the full duration of any operation that calls the server or
	// modifies the stores. Unlink also holds it, so it waits for any in-flight
	// operation to complete before clearing state.
	serial chan struct{}

	// updater is the active DeploymentUpdater; nil when unconfigured.
	// Protected by serial.
	updater *DeploymentUpdater

	// mu protects state. Read-only operations (State, PullState) acquire only mu,
	// never serial, so they never block on long-running server calls.
	mu    sync.Mutex
	state ConnState // protected by mu

	bus minibus.Bus[ConnState]
}

// OpenConn creates a new Conn backed by the given stores.
// depStore is used to construct a DeploymentUpdater whenever a Registration is loaded or set.
func OpenConn(ctx context.Context, regStore RegistrationStore, depStore *DeploymentStore, opts ...ConnOption) (*Conn, error) {
	serial := make(chan struct{}, 1)
	serial <- struct{}{} // initialise as unlocked
	c := &Conn{
		regStore: regStore,
		depStore: depStore,
		state:    ConnState{Connectivity: Unconfigured},
		newClient: func(r Registration) Client {
			return NewHTTPClient(r)
		},
		serial: serial,
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.start(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// start loads the persisted Registration (if any) and initialises internal state.
// If no registration exists the Conn stays in the unconfigured state.
func (c *Conn) start(ctx context.Context) error {
	reg, ok, err := c.regStore.Load(ctx)
	if err != nil {
		return fmt.Errorf("load registration: %w", err)
	}
	if !ok {
		return nil
	}
	// no locking required as start is only called during construction
	c.updater = NewDeploymentUpdater(c.depStore, c.newClient(reg), c.updaterOpts...)
	st := c.updateState(ConnState{Connectivity: Connecting, Registration: &reg})
	c.bus.Send(ctx, st)
	return nil
}

// Register persists the supplied credentials and updates internal state.
// A new DeploymentUpdater is created for the new registration.
func (c *Conn) Register(ctx context.Context, reg Registration) (ConnState, error) {
	if !c.lockSerial(ctx) {
		return ConnState{}, ctx.Err()
	}
	defer c.unlockSerial()

	// check that the new registration will actually work before saving it
	newUpdater := NewDeploymentUpdater(c.depStore, c.newClient(reg), c.updaterOpts...)
	if err := newUpdater.CheckIn(ctx); err != nil {
		return ConnState{}, &CredentialCheckError{Err: err}
	}
	if err := c.regStore.Save(ctx, reg); err != nil {
		return ConnState{}, fmt.Errorf("persist registration: %w", err)
	}

	c.updater = newUpdater
	st := c.updateState(ConnState{Connectivity: Connecting, Registration: &reg})

	c.bus.Send(ctx, st)
	return st, nil
}

// Unlink removes the persisted Registration and returns the Conn to the unconfigured state.
func (c *Conn) Unlink(ctx context.Context) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	if err := c.regStore.Clear(ctx); err != nil {
		return fmt.Errorf("clear registration: %w", err)
	}
	c.updater = nil
	st := c.updateState(ConnState{Connectivity: Unconfigured})
	c.bus.Send(ctx, st)
	return nil
}

// TestConn performs a non-mutating check-in using the current registration to
// verify connectivity and credentials. The outcome is recorded so that
// State/PullState immediately reflect the result.
// Returns ErrNotRegistered when no registration is active.
func (c *Conn) TestConn(ctx context.Context) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	u := c.updater
	if u == nil {
		return ErrNotRegistered
	}
	c.mu.Lock()
	reg := c.state.Registration
	c.mu.Unlock()

	err := u.CheckIn(ctx)
	c.recordCheckIn(reg, time.Now(), err)
	return err
}

// State returns the current connection state snapshot.
func (c *Conn) State() ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// PullState returns the current state plus a channel that receives ConnState updates.
// The channel is closed when ctx is cancelled.
func (c *Conn) PullState(ctx context.Context) (ConnState, <-chan ConnState) {
	c.mu.Lock()
	initial := c.state
	changes := c.bus.Listen(ctx)
	c.mu.Unlock()
	return initial, changes
}

// WaitConnected blocks until the connection has had a successful check-in (Connectivity == Connected),
// or ctx is done. It returns nil once connected, otherwise ctx.Err(). If the connection is already
// Connected it returns immediately.
func (c *Conn) WaitConnected(ctx context.Context) error {
	// Scope the subscription to a child ctx we cancel on return, so the bus drops our listener.
	// Otherwise, with a long-lived ctx, the leaked undrained listener blocks later check-in broadcasts.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	state, changes := c.PullState(ctx)
	if state.Connectivity == Connected {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case st, ok := <-changes:
			if !ok {
				return ctx.Err()
			}
			if st.Connectivity == Connected {
				return nil
			}
		}
	}
}

// Update performs a single shared check-in using the current registration and dispatches the
// response to both channels: the config channel (DeploymentUpdater) and, when enabled, the update
// channel (SoftwareUpdater). One check-in carries both channels' current state.
// Returns ErrNotRegistered when no registration is active.
// The check-in outcome is recorded internally so that State/PullState reflect it.
//
// needReboot reflects the config channel only; an update channel that records an install intent does
// not itself trigger a reboot here (the Supervisor restarts BOS in a later phase).
func (c *Conn) Update(ctx context.Context) (needReboot bool, err error) {
	if !c.lockSerial(ctx) {
		return false, ctx.Err()
	}
	defer c.unlockSerial()

	u := c.updater
	if u == nil {
		return false, ErrNotRegistered
	}
	c.mu.Lock()
	reg := c.state.Registration
	c.mu.Unlock()

	// Build one request carrying both channels' current state, perform a single check-in, and
	// dispatch the response to each handler.
	req, err := u.CheckInRequest(ctx)
	if err != nil {
		return false, err
	}
	if c.softwareUpdater != nil {
		if err := c.softwareUpdater.CheckInState(ctx, &req); err != nil {
			return false, err
		}
	}

	// Record the whole poll's outcome (transport plus handler errors) so State and WaitConnected
	// reflect a config/update that reached the server but then failed to download or stage, not just
	// the transport check-in.
	defer func() { c.recordCheckIn(reg, time.Now(), err) }()

	resp, err := u.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in: %w", err)
	}

	needReboot, err = u.HandleConfig(ctx, req, resp)
	if err != nil {
		return needReboot, err
	}
	if c.softwareUpdater != nil {
		if err = c.softwareUpdater.HandleUpdate(ctx, u.client, resp); err != nil {
			err = fmt.Errorf("handle update: %w", err)
			return needReboot, err
		}
	}
	return needReboot, nil
}

// CommitInstall marks the installing deployment as active and reports the result.
// Returns ErrNotRegistered when no registration is active.
func (c *Conn) CommitInstall(ctx context.Context) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	if c.updater == nil {
		return ErrNotRegistered
	}
	return c.updater.CommitInstall(ctx)
}

// FailInstall clears the installing mark and reports the failure.
// Returns ErrNotRegistered when no registration is active.
func (c *Conn) FailInstall(ctx context.Context, message string) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	if c.updater == nil {
		return ErrNotRegistered
	}
	return c.updater.FailInstall(ctx, message)
}

// ReconcileUpdate resolves an in-flight software update after the Supervisor has restarted BOS,
// reporting the outcome (currentUpdate or failedUpdate) to SCC and clearing the persisted intent.
// It is a no-op when the update channel is disabled (no SoftwareUpdater) or there is no in-flight
// update. Returns ErrNotRegistered when no registration is active, mirroring CommitInstall/FailInstall;
// the caller should keep the intent in that case (it self-corrects once cloud is registered).
func (c *Conn) ReconcileUpdate(ctx context.Context) error {
	if c.softwareUpdater == nil {
		return nil
	}
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	u := c.updater
	if u == nil {
		return ErrNotRegistered
	}
	return c.softwareUpdater.ReconcileStartup(ctx, u.client)
}

// recordCheckIn reports the outcome of a check-in made using reg.
// If reg is no longer the current registration the outcome is discarded.
func (c *Conn) recordCheckIn(reg *Registration, t time.Time, err error) {
	c.mu.Lock()
	if c.state.Registration == nil || reg == nil || c.state.Registration.ClientID != reg.ClientID {
		c.mu.Unlock()
		return
	}
	if err != nil {
		c.state.Connectivity = Failed
		c.state.LastError = err
	} else {
		c.state.Connectivity = Connected
		c.state.LastCheckInTime = t
		c.state.LastError = nil
	}
	state := c.state
	c.mu.Unlock()
	c.bus.Send(context.Background(), state)
}

// lockSerial acquires the serial semaphore, blocking until it is available or ctx is cancelled.
// Returns false if ctx is cancelled before the semaphore could be acquired.
func (c *Conn) lockSerial(ctx context.Context) bool {
	select {
	case <-c.serial:
		return true
	case <-ctx.Done():
		return false
	}
}

// unlockSerial releases the serial semaphore.
func (c *Conn) unlockSerial() {
	c.serial <- struct{}{}
}

func (c *Conn) updateState(st ConnState) ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	st.ChangeTime = time.Now()
	c.state = st
	return st
}
