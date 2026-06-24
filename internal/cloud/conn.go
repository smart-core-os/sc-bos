package cloud

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

// ErrNotRegistered is returned by Update, Renew, CommitInstall, and FailInstall
// when there is no active credential. Callers can treat it as "nothing to do".
var ErrNotRegistered = errors.New("cloud: no active credential")

// Connectivity represents the cloud connection lifecycle state.
type Connectivity int

const (
	Unconfigured Connectivity = iota // no credential on disk
	Connecting                       // credential present, no successful check-in yet
	Connected                        // last check-in succeeded
	Failed                           // last check-in failed
)

// ConnState is a point-in-time snapshot of the cloud connection: the current
// credential (nil when unconfigured) together with the latest check-in health.
type ConnState struct {
	Connectivity    Connectivity
	Registration    *Registration // nil iff Connectivity == Unconfigured
	LastCheckInTime time.Time
	LastError       error
	ChangeTime      time.Time // when this snapshot was produced
}

// ConnOption configures a Conn.
type ConnOption func(*Conn)

// WithConfigUpdaterOptions appends opts to the ConfigUpdaterOption slice forwarded to each
// newly-created ConfigUpdater.
func WithConfigUpdaterOptions(opts ...ConfigUpdaterOption) ConnOption {
	return func(c *Conn) { c.updaterOpts = append(c.updaterOpts, opts...) }
}

// WithClientFactory sets a factory used to build a Client for a given Registration.
// Primarily useful in tests to inject a fake transport.
func WithClientFactory(f func(*Registration) Client) ConnOption {
	return func(c *Conn) { c.newClient = f }
}

// WithClientOptions forwards HTTPClientOptions to the default Client built for
// each Registration (e.g. WithInsecureSkipVerify for local cloudsim). Ignored when
// a custom factory is set via WithClientFactory.
func WithClientOptions(opts ...HTTPClientOption) ConnOption {
	return func(c *Conn) { c.clientOpts = append(c.clientOpts, opts...) }
}

// WithBinaryUpdater enables the binary channel on the shared poll check-in, dispatching the
// latestBinary block to u. When unset the binary channel is not reported and latestBinary is ignored.
func WithBinaryUpdater(u *BinaryUpdater) ConnOption {
	return func(c *Conn) { c.binaryUpdater = u }
}

// WithPlatform overrides the platform details reported by this Conn when it checks in.
// Defaults to runtime.GOOS and runtime.GOARCH.
func WithPlatform(platform Platform) ConnOption {
	return func(c *Conn) {
		c.platform = platform
	}
}

// Conn manages the lifecycle of a cloud connection.
// It tracks the current Registration, broadcasts ConnState changes, and owns a
// ConfigUpdater that is rebuilt whenever the Registration changes.
// Methods are safe to call concurrently.
type Conn struct {
	// Immutable after construction — no locking required.
	regStore      RegistrationStore
	depStore      *DeploymentStore
	newClient     func(*Registration) Client
	updaterOpts   []ConfigUpdaterOption
	clientOpts    []HTTPClientOption
	binaryUpdater *BinaryUpdater // nil when the binary channel is disabled
	platform      Platform       // reported to cloud

	// serial is a binary semaphore (buffered channel of size 1, pre-filled).
	// Acquire with lockSerial; release with unlockSerial.
	// Held for the full duration of any operation that calls the server or
	// modifies the stores.
	serial chan struct{}

	// client and updater are the active Client and ConfigUpdater; nil when
	// unconfigured. Protected by serial. client renews the certificate in place
	// (hot reload), so it survives a renewal — only a new enrollment replaces it.
	client  Client
	updater *ConfigUpdater

	// baseURL is the SCC API origin the active client targets (check-in/renew).
	// It defaults to the origin of the configured register URL but follows the
	// origin a successful Register enrolled against, so the client always talks to
	// the server that issued the certificate. Protected by serial.
	baseURL string

	// Counts failures for the deployment currently in progress. Protected by serial.
	failingDeployment string // which deployment failedAttempts applies to
	failedAttempts    int

	// mu protects state. Read-only operations (State, PullState) acquire only mu,
	// never serial, so they never block on long-running server calls.
	mu    sync.Mutex
	state ConnState // protected by mu

	bus minibus.Bus[ConnState]
}

// OpenConn creates a new Conn backed by the given stores. baseURL is any URL on
// the SCC API origin (e.g. the configured register URL); OpenConn reduces it to
// its scheme://host origin, to which the check-in and renew paths are appended.
// A persisted registration's own endpoint takes precedence once loaded. depStore
// is used to construct a ConfigUpdater whenever a Registration is loaded or set.
func OpenConn(ctx context.Context, regStore RegistrationStore, depStore *DeploymentStore, baseURL string, opts ...ConnOption) (*Conn, error) {
	serial := make(chan struct{}, 1)
	serial <- struct{}{} // initialise as unlocked
	c := &Conn{
		regStore: regStore,
		depStore: depStore,
		state:    ConnState{Connectivity: Unconfigured},
		serial:   serial,
		baseURL:  OriginOf(baseURL),
	}
	for _, opt := range opts {
		opt(c)
	}
	// Default client factory honours any HTTPClientOptions (e.g. insecure TLS for
	// local cloudsim) collected from ConnOptions above, and targets c.baseURL so a
	// Register against an override URL is honoured. A WithClientFactory option
	// overrides this entirely.
	if c.newClient == nil {
		c.newClient = func(cred *Registration) Client {
			return NewHTTPClient(cred, c.baseURL, c.clientOpts...)
		}
	}
	if c.platform.OS == "" {
		c.platform.OS = runtime.GOOS
	}
	if c.platform.Arch == "" {
		c.platform.Arch = runtime.GOARCH
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
	// The persisted endpoint is authoritative: it is the origin that issued the
	// certificate, so check-in and renewal target the right server after a restart
	// even if it differs from the configured default.
	if reg.APIEndpoint != "" {
		c.baseURL = reg.APIEndpoint
	}
	// no locking required as start is only called during construction
	c.client = c.newClient(reg)
	c.updater = NewConfigUpdater(c.depStore, c.client, c.updaterOpts...)
	st := c.updateState(ConnState{Connectivity: Connecting, Registration: reg})
	c.bus.Send(ctx, st)
	return nil
}

// Register persists the supplied registration and updates internal state.
// A new Client and ConfigUpdater are created for it. The registration
// carries the API endpoint it was issued against (Registration.APIEndpoint);
// that origin becomes the one the client uses for check-in and renewal, so
// those always reach the server that issued the certificate — including after a
// restart, since the endpoint is persisted with the registration.
func (c *Conn) Register(ctx context.Context, reg *Registration) (ConnState, error) {
	if !c.lockSerial(ctx) {
		return ConnState{}, ctx.Err()
	}
	defer c.unlockSerial()

	if reg.APIEndpoint != "" {
		c.baseURL = reg.APIEndpoint
	}

	// check that the new registration will actually work before saving it
	newClient := c.newClient(reg)
	newUpdater := NewConfigUpdater(c.depStore, newClient, c.updaterOpts...)
	if err := newUpdater.CheckIn(ctx); err != nil {
		return ConnState{}, &CredentialCheckError{Err: err}
	}
	if err := c.regStore.Save(ctx, reg); err != nil {
		return ConnState{}, fmt.Errorf("persist registration: %w", err)
	}

	closeIdle(c.client)
	c.client = newClient
	c.updater = newUpdater
	st := c.updateState(ConnState{Connectivity: Connecting, Registration: reg})

	c.bus.Send(ctx, st)
	return st, nil
}

// Renew exchanges the current certificate for a fresh one over the authenticated
// connection, persists it, and hot-swaps it into the live client without
// restarting. Returns ErrNotRegistered when no credential is active.
func (c *Conn) Renew(ctx context.Context) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	if c.client == nil {
		return ErrNotRegistered
	}
	newCred, err := c.client.Renew(ctx)
	if err != nil {
		return err
	}
	// persist before swapping, so a crash never loses the new credential
	if err := c.regStore.Save(ctx, newCred); err != nil {
		return fmt.Errorf("persist renewed credential: %w", err)
	}
	c.client.SetRegistration(newCred)

	c.mu.Lock()
	c.state.Registration = newCred
	c.state.ChangeTime = time.Now()
	c.mu.Unlock()

	// Renewal alone doesn't tell us the connection is healthy — and if we were in
	// Failed, the stale error must be cleared. A check-in with the new certificate
	// refreshes Connectivity and LastError so a successful renew can't leave a
	// stale Failed status behind. The renewal itself succeeded regardless, so a
	// check-in failure here is recorded for display but not returned as an error.
	checkErr := c.updater.CheckIn(ctx)
	c.recordCheckIn(newCred, time.Now(), checkErr)
	return nil
}

// Unlink removes the persisted Registration and returns the Conn to the unconfigured state.
func (c *Conn) Unlink(ctx context.Context) error {
	if !c.lockSerial(ctx) {
		return ctx.Err()
	}
	defer c.unlockSerial()

	if err := c.regStore.Clear(ctx); err != nil {
		return fmt.Errorf("clear credential: %w", err)
	}
	closeIdle(c.client)
	c.client = nil
	c.updater = nil
	st := c.updateState(ConnState{Connectivity: Unconfigured})
	c.bus.Send(ctx, st)
	return nil
}

// closeIdle releases any pooled idle connections held by a client that is being
// replaced, so its transport's keep-alive sockets are not retained until GC.
// No-op for clients (e.g. test fakes) that do not manage a connection pool.
func closeIdle(client Client) {
	if c, ok := client.(interface{ CloseIdleConnections() }); ok {
		c.CloseIdleConnections()
	}
}

// TestConn performs a non-mutating check-in using the current credential to
// verify connectivity. The outcome is recorded so that State/PullState
// immediately reflect the result.
// Returns ErrNotRegistered when no credential is active.
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
	cred := c.state.Registration
	c.mu.Unlock()

	err := u.CheckIn(ctx)
	c.recordCheckIn(cred, time.Now(), err)
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
// response to both channels: the config channel (ConfigUpdater) and, when enabled, the update
// channel (BinaryUpdater). One check-in carries both channels' current state.
// Returns ErrNotRegistered when no credential is active.
// The check-in outcome is recorded internally so that State/PullState reflect it.
//
// needReboot reflects the config channel only; the binary channel that records an install intent does
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
	cred := c.state.Registration
	c.mu.Unlock()

	// record whatever happens in this check-in
	defer func() { c.recordCheckIn(cred, time.Now(), err) }()

	// Build the combined request: the config stream's progress, this node's platform, and the binary
	// version it runs. Perform one check-in, then dispatch the response to each handler.
	// Note: this doesn't include binary update progress, that happens in a separate check-in based on info
	// from the supervisor.
	req, err := u.CheckInRequest(ctx)
	if err != nil {
		return false, err
	}
	req.Running.Platform = new(c.platform)
	if c.binaryUpdater != nil {
		req.Running.Binary = c.binaryUpdater.runningBinary()
	}

	resp, err := u.client.CheckIn(ctx, req)
	if err != nil {
		return false, fmt.Errorf("check-in: %w", err)
	}

	// both channels report their status to the cloud first
	// as part of this process, we also find out which channel(s) have an update to install
	cfg, err := u.reportConfig(ctx, resp)
	if err != nil {
		return false, fmt.Errorf("handle config: %w", err)
	}
	needReboot = cfg.inFlight

	var bin installState
	if c.binaryUpdater != nil {
		supStatus, serr := c.binaryUpdater.updateStatus(ctx)
		if serr != nil {
			return needReboot, fmt.Errorf("get supervisor update status: %w", serr)
		}
		if bin, err = c.binaryUpdater.reportBinary(ctx, u.client, resp, supStatus); err != nil {
			return needReboot, fmt.Errorf("handle update: %w", err)
		}
	}

	// process any update available, prioritising config
	switch {
	case cfg.inFlight || bin.inFlight:
		// An install is already running; let it settle before starting another.
	case cfg.startable:
		needReboot, err = u.installConfig(ctx, resp.LatestConfig)
		if err = c.capInstall(ctx, u.client, resp.LatestConfig.Deployment.ID, err); err != nil {
			return needReboot, fmt.Errorf("handle config: %w", err)
		}
	case bin.startable:
		err = c.binaryUpdater.installBinary(ctx, u.client, resp.LatestBinary)
		if err = c.capInstall(ctx, u.client, resp.LatestBinary.Deployment.ID, err); err != nil {
			return needReboot, fmt.Errorf("handle update: %w", err)
		}
	}
	return needReboot, nil
}

// how many transient install failures to permit before giving up on a deployment permanently.
const maxInstallAttempts = 3

// capInstall records the outcome of an install attempt for install attempt capping.
//
// If the installation was successful (based on err == nil), clears the failedAttempts counter
// and returns nil.
//
// If the installation failed but the failure cap hasn't been reached yet, increments the counter
// and returns the error unchanged.
//
// # Otherwise, when
//
// It returns
// err unchanged while retries remain, so a still-failing check-in is recorded as failed; on the
// maxInstallAttempts-th consecutive failure it reports the deployment permanently failed and returns nil
// so the deployment stops being offered and the connection can recover.
func (c *Conn) capInstall(ctx context.Context, client Client, deploymentID string, err error) error {
	if err == nil {
		c.failingDeployment, c.failedAttempts = "", 0
		return nil
	}
	if deploymentID != c.failingDeployment {
		c.failingDeployment, c.failedAttempts = deploymentID, 0
	}
	c.failedAttempts++
	if c.failedAttempts < maxInstallAttempts {
		return err
	}
	c.failingDeployment, c.failedAttempts = "", 0
	reason := fmt.Sprintf("install failed after %d attempts", maxInstallAttempts)
	if _, rerr := client.CheckIn(ctx, CheckInRequest{Progress: []ProgressReport{
		{DeploymentID: deploymentID, State: ProgressFailed, Reason: reason}}}); rerr != nil {
		return errors.Join(err, rerr)
	}
	return nil
}

// CommitInstall marks the installing deployment as active and reports the result.
// Returns ErrNotRegistered when no credential is active.
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
// Returns ErrNotRegistered when no credential is active.
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

// recordCheckIn reports the outcome of a check-in made using cred.
// If cred is no longer the current credential (by node id) the outcome is discarded.
func (c *Conn) recordCheckIn(cred *Registration, t time.Time, err error) {
	c.mu.Lock()
	if c.state.Registration == nil || cred == nil || c.state.Registration.NodeID() != cred.NodeID() {
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
