package connecttelemetry

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Connect's intake limits, see docs/connect-telemetry-ingest.md.
// Connect drops what is over them; we warn so the cause shows up on the node too.
const (
	healthMaxResources     = 5000
	healthMaxChecksPerSet  = 50
	healthMaxSetBytes      = 64 << 10
	healthMaxManifestBytes = 512 << 10 // the broker's message ceiling
	// healthSkipSetBytes is the size beyond which a set is not sent at all.
	healthSkipSetBytes = 256 << 10
)

const (
	// healthStartDelay gives the initial checks and kinds time to arrive before the first sends.
	healthStartDelay = 10 * time.Second
	// healthFirstManifestDelay is how long after start the first manifest goes, see manifestLoop.
	healthFirstManifestDelay = 5 * time.Minute
	healthMinRetry           = time.Second
	healthMaxRetry           = time.Minute
)

// healthSink is where health messages go, a *publisher outside tests.
type healthSink interface {
	// awaitConnection blocks until the connection is up, or ctx is done.
	awaitConnection(ctx context.Context) error
	// connectionUp returns a channel that is closed the next time the connection comes up.
	connectionUp() <-chan struct{}
	// publishAcked publishes at QoS 1, returning nil once the broker has acknowledged the message.
	publishAcked(ctx context.Context, topic string, payload []byte) error
}

// devicesSource streams devices, like auto.HealthCheckSource or *node.Node.
type devicesSource interface {
	PullDevices(ctx context.Context, opts ...resource.ReadOption) <-chan devicespb.DevicesChange
}

// healthPublisher publishes the health checks this node evaluates to Connect.
//
// Each resource's complete set of checks is sent whenever its hash differs from the last one the
// broker acknowledged, and again once a day. A manifest of every set's hash is sent periodically
// so Connect can find sets it lost, or holds but should not.
type healthPublisher struct {
	sink             healthSink
	checks           devicesSource // this node's own checks
	devices          devicesSource // for each resource's device type
	topicPrefix      string
	manifestInterval time.Duration
	limiter          *rate.Limiter
	logger           *zap.Logger

	mu        sync.Mutex
	resources map[string]*healthResource
	kinds     map[string]string // subject kind by name, absent means device
	queue     []string          // names that may need sending, in order
	queued    map[string]bool
	wake      chan struct{} // signalled when queue becomes non-empty
	// inflight is the set being published, if any. sendLoop publishes one at a time.
	inflight healthInflight
	// tooManyWarned avoids repeating the resource limit warning until the count drops below it.
	tooManyWarned bool
}

// healthInflight identifies a set that has been built and is being published.
type healthInflight struct {
	name, hash string
}

// healthResource is what we know about one resource's set.
type healthResource struct {
	checks           []*healthpb.HealthCheck
	kind             string
	hash             string        // of checks and kind
	ackedHash        string        // the last hash the broker acknowledged
	skippedHash      string        // the last hash too large to send
	lastStamp        time.Time     // the timestamp of the last set built
	warnedChecksHash string        // the last hash warned about for having too many checks
	warnedSizeHash   string        // the last hash warned about for being too large
	removed          bool          // the resource no longer has checks, forget it once its empty set is acknowledged
	force            bool          // send even if hash is acknowledged, for the daily resend
	slot             time.Duration // the minute of the UTC day for the daily resend
	failures         int
	retry            *time.Timer
}

func newHealthPublisher(sink healthSink, checks, devices devicesSource, topicPrefix string, manifestInterval time.Duration, maxRate float64, logger *zap.Logger) *healthPublisher {
	return &healthPublisher{
		sink:             sink,
		checks:           checks,
		devices:          devices,
		topicPrefix:      topicPrefix,
		manifestInterval: manifestInterval,
		limiter:          rate.NewLimiter(rate.Limit(maxRate), 1),
		logger:           logger,
		resources:        make(map[string]*healthResource),
		kinds:            make(map[string]string),
		queued:           make(map[string]bool),
		wake:             make(chan struct{}, 1),
	}
}

// run publishes health until ctx is done.
func (p *healthPublisher) run(ctx context.Context) error {
	defer p.stopRetries()
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return p.watchChecks(ctx) })
	grp.Go(func() error { return p.watchKinds(ctx) })
	grp.Go(func() error { return p.sendLoop(ctx) })
	grp.Go(func() error { return p.manifestLoop(ctx) })
	grp.Go(func() error { return p.timerLoop(ctx) })
	err := grp.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// watchChecks tracks this node's own checks.
func (p *healthPublisher) watchChecks(ctx context.Context) error {
	for change := range p.checks.PullDevices(ctx) {
		if change.ChangeType == typespb.ChangeType_REMOVE {
			p.removeResource(change.Id)
		} else {
			p.setChecks(change.Id, change.NewValue.GetHealthChecks())
		}
	}
	return ctx.Err()
}

// watchKinds tracks the subject kind of every resource that isn't a device.
func (p *healthPublisher) watchKinds(ctx context.Context) error {
	changes := p.devices.PullDevices(ctx,
		resource.WithReadPaths(&devicespb.Device{}, "metadata.device_type"),
		resource.WithInclude(func(_ string, item proto.Message) bool {
			return subjectKind(item.(*devicespb.Device).GetMetadata().GetDeviceType()) != subjectKindDevice
		}),
	)
	for change := range changes {
		kind := subjectKindDevice
		if change.ChangeType != typespb.ChangeType_REMOVE {
			kind = subjectKind(change.NewValue.GetMetadata().GetDeviceType())
		}
		p.setKind(change.Id, kind)
	}
	return ctx.Err()
}

func (p *healthPublisher) setChecks(name string, checks []*healthpb.HealthCheck) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.resources[name]
	if !ok {
		r = &healthResource{kind: p.kindLocked(name), slot: dailySlot(name)}
		p.resources[name] = r
		p.checkResourceCountLocked()
	}
	r.removed = false
	r.checks = checks
	r.kind = p.kindLocked(name) // ignored while removed, see setKind
	p.rehashLocked(name, r)
}

func (p *healthPublisher) removeResource(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.resources[name]
	if !ok {
		return
	}
	// Connect may hold checks for name from before we started, so always tell it the set is empty.
	r.removed = true
	r.checks = nil
	p.rehashLocked(name, r)
	p.forgetIfDoneLocked(name, r)
}

func (p *healthPublisher) setKind(name, kind string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kind == subjectKindDevice {
		delete(p.kinds, name)
	} else {
		p.kinds[name] = kind
	}
	// An empty set removes every check whatever its kind, so a removed resource needn't be resent.
	// Kinds often change as a resource is removed, a driver's SERVICE type goes with its check for example.
	if r, ok := p.resources[name]; ok && !r.removed && r.kind != kind {
		r.kind = kind
		p.rehashLocked(name, r)
	}
}

func (p *healthPublisher) kindLocked(name string) string {
	if kind, ok := p.kinds[name]; ok {
		return kind
	}
	return subjectKindDevice
}

// rehashLocked recomputes r's hash after its checks or kind changed, queueing it if it needs sending.
func (p *healthPublisher) rehashLocked(name string, r *healthResource) {
	r.hash = checkSetHash(r.kind, r.checks)
	if n := len(r.checks); n > healthMaxChecksPerSet && r.warnedChecksHash != r.hash {
		r.warnedChecksHash = r.hash
		p.logger.Warn("health set has more checks than Connect accepts, Connect will drop it",
			zap.String("resource", name), zap.Int("checks", n), zap.Int("limit", healthMaxChecksPerSet))
	}
	if needsSend(r) {
		p.enqueueLocked(name)
	}
}

func (p *healthPublisher) checkResourceCountLocked() {
	n := len(p.resources)
	switch {
	case n > healthMaxResources && !p.tooManyWarned:
		p.tooManyWarned = true
		p.logger.Warn("more resources have health checks than Connect accepts, Connect will drop some",
			zap.Int("resources", n), zap.Int("limit", healthMaxResources))
	case n <= healthMaxResources:
		p.tooManyWarned = false
	}
}

// needsSend reports whether r's current set should be sent.
func needsSend(r *healthResource) bool {
	if r.hash == r.skippedHash {
		return false
	}
	return r.force || r.hash != r.ackedHash
}

// forgetIfDoneLocked drops a removed resource once Connect has acknowledged its empty set.
func (p *healthPublisher) forgetIfDoneLocked(name string, r *healthResource) {
	if !r.removed || r.hash != r.ackedHash {
		return
	}
	if r.retry != nil {
		r.retry.Stop()
	}
	delete(p.resources, name)
	p.checkResourceCountLocked()
}

func (p *healthPublisher) enqueueLocked(name string) {
	if p.queued[name] {
		return
	}
	p.queued[name] = true
	p.queue = append(p.queue, name)
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *healthPublisher) enqueue(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enqueueLocked(name)
}

// dequeue returns the next queued name, waiting until there is one.
func (p *healthPublisher) dequeue(ctx context.Context) (string, error) {
	for {
		p.mu.Lock()
		if len(p.queue) > 0 {
			name := p.queue[0]
			p.queue[0] = ""
			p.queue = p.queue[1:]
			delete(p.queued, name)
			p.mu.Unlock()
			return name, nil
		}
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-p.wake:
		}
	}
}

// sendLoop sends queued sets, one at a time, no faster than the rate limit.
func (p *healthPublisher) sendLoop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(healthStartDelay):
	}
	for {
		name, err := p.dequeue(ctx)
		if err != nil {
			return err
		}
		if !p.needsSend(name) {
			continue
		}
		if err := p.limiter.Wait(ctx); err != nil {
			return err
		}
		// Wait before building the message, so it is current when it goes.
		if err := p.sink.awaitConnection(ctx); err != nil {
			return err
		}
		payload, hash, ok := p.buildSet(name)
		if !ok {
			continue
		}
		err = p.sink.publishAcked(ctx, healthTopic(p.topicPrefix), payload)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		p.sent(name, hash, err)
	}
}

func (p *healthPublisher) needsSend(name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.resources[name]
	return ok && needsSend(r)
}

// buildSet encodes name's current set, returning false if there is nothing to send.
func (p *healthPublisher) buildSet(name string) (payload []byte, hash string, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.resources[name]
	if !ok || !needsSend(r) {
		return nil, "", false
	}
	// Connect orders sets by timestamp to the millisecond, so keep them strictly increasing even if the clock steps back.
	stamp := time.Now().UTC().Truncate(time.Millisecond)
	if next := r.lastStamp.Add(time.Millisecond); stamp.Before(next) {
		stamp = next
	}
	payload, err := buildHealthSet(stamp, name, r.kind, r.hash, r.checks)
	if err != nil {
		p.logger.Error("failed to encode health set", zap.String("resource", name), zap.Error(err))
		r.skippedHash = r.hash
		return nil, "", false
	}
	r.lastStamp = stamp
	switch n := len(payload); {
	case n > healthSkipSetBytes:
		r.skippedHash = r.hash
		p.logger.Warn("health set too large to send, skipping until it changes",
			zap.String("resource", name), zap.Int("bytes", n), zap.Int("limit", healthSkipSetBytes))
		return nil, "", false
	case n > healthMaxSetBytes && r.warnedSizeHash != r.hash:
		r.warnedSizeHash = r.hash
		p.logger.Warn("health set is larger than Connect accepts, Connect will drop it",
			zap.String("resource", name), zap.Int("bytes", n), zap.Int("limit", healthMaxSetBytes))
	}
	p.inflight = healthInflight{name: name, hash: r.hash}
	return payload, r.hash, true
}

// sent records the outcome of sending hash for name.
func (p *healthPublisher) sent(name, hash string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inflight = healthInflight{}
	r, ok := p.resources[name]
	if !ok {
		return
	}
	if err != nil {
		// An unacknowledged publish may still land, after a late PUBACK or resent from the session,
		// so we no longer know which set the broker has: send the next even if it matches the last acknowledged.
		r.ackedHash = ""
		r.failures++
		backoff := healthMaxRetry
		if r.failures <= 6 { // beyond this the shift is past the limit, and eventually overflows
			backoff = min(healthMinRetry<<(r.failures-1), healthMaxRetry)
		}
		log := p.logger.Debug
		if r.failures == 1 { // once per streak of failures
			log = p.logger.Warn
		}
		log("failed to publish health set, will retry",
			zap.String("resource", name), zap.Int("failures", r.failures), zap.Duration("retryIn", backoff), zap.Error(err))
		if r.retry != nil {
			r.retry.Stop()
		}
		r.retry = time.AfterFunc(backoff, func() { p.enqueue(name) })
		return
	}
	r.failures = 0
	if r.retry != nil {
		r.retry.Stop()
		r.retry = nil
	}
	if hash == r.hash {
		// cleared on acknowledgement, so a failed daily resend is retried like any other send
		r.force = false
	}
	r.ackedHash = hash
	p.forgetIfDoneLocked(name, r)
	// the set may have changed while it was being sent
	if needsSend(r) {
		p.enqueueLocked(name)
	}
}

// retryNow retries every failed send straight away, with a fresh backoff.
// Called when the connection comes back, since most failures are down to it.
func (p *healthPublisher) retryNow() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for name, r := range p.resources {
		if r.retry == nil {
			continue
		}
		r.retry.Stop()
		r.retry = nil
		r.failures = 0
		p.enqueueLocked(name)
	}
}

func (p *healthPublisher) stopRetries() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.resources {
		if r.retry != nil {
			r.retry.Stop()
		}
	}
}

// manifestLoop sends a manifest every manifestInterval, the first healthFirstManifestDelay after start.
// The first waits so Connect doesn't dispose of checks that are still being created at startup,
// but not a whole interval: Connect flags a node online for 30 minutes without a manifest,
// and a restart too quick to take the node offline counts towards that.
func (p *healthPublisher) manifestLoop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(min(healthFirstManifestDelay, p.manifestInterval)):
	}
	ticker := time.NewTicker(p.manifestInterval)
	defer ticker.Stop()
	for {
		if err := p.sendManifest(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// sendManifest sends one manifest, returning an error only once ctx is done.
func (p *healthPublisher) sendManifest(ctx context.Context) error {
	if err := p.limiter.Wait(ctx); err != nil {
		return err
	}
	if err := p.sink.awaitConnection(ctx); err != nil {
		return err
	}
	payload, ok := p.buildManifest()
	if !ok {
		return nil
	}
	err := p.sink.publishAcked(ctx, healthManifestTopic(p.topicPrefix), payload)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		p.logger.Warn("failed to publish health manifest", zap.Error(err))
	}
	return nil
}

// buildManifest encodes the manifest of every resource with checks.
func (p *healthPublisher) buildManifest() ([]byte, bool) {
	p.mu.Lock()
	// stamped under mu, so every set built after this is stamped no earlier, see buildSet
	now := time.Now().Truncate(time.Millisecond)
	entries := make([]manifestEntry, 0, len(p.resources))
	for name, r := range p.resources {
		if r.removed || len(r.checks) == 0 {
			continue
		}
		// Connect judges each set stamped before the manifest against the hash listed, and shows the
		// resource as out of step if they differ, treating that as a set lost after the broker took it.
		// So list the hash of the last set built: the one being published if there is one, else the
		// acknowledged one. A set still waiting to go out, behind the rate limit say, isn't stamped yet.
		// Sets never acknowledged, whose last send failed, or skipped as too large get their current
		// hash: Connect then shows them as out of step, which is or may be true, rather than disposing
		// of checks the node still has.
		hash := r.ackedHash
		switch {
		case p.inflight.name == name:
			hash = p.inflight.hash
		case hash == "" || r.skippedHash == r.hash:
			hash = r.hash
		}
		entries = append(entries, manifestEntry{NameHash: nameHash(name), Hash: hash})
	}
	p.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool { return entries[i].NameHash < entries[j].NameHash })

	payload, err := buildHealthManifest(now, entries)
	if err != nil {
		p.logger.Error("failed to encode health manifest", zap.Error(err))
		return nil, false
	}
	if n := len(payload); n > healthMaxManifestBytes {
		p.logger.Warn("health manifest too large to send, skipping",
			zap.Int("bytes", n), zap.Int("limit", healthMaxManifestBytes), zap.Int("resources", len(entries)))
		return nil, false
	}
	return payload, true
}

// timerLoop runs the daily resends, and retries failed sends when the connection comes back.
func (p *healthPublisher) timerLoop(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	last := time.Now().UTC().Truncate(time.Minute)
	// held across iterations, so a reconnect while busy below is still seen
	connUp := p.sink.connectionUp()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-connUp:
			connUp = p.sink.connectionUp()
			p.retryNow()
		case now := <-ticker.C:
			now = now.UTC().Truncate(time.Minute)
			p.forceDue(last, now)
			last = now
		}
	}
}

// forceDue marks every resource whose daily slot is in (from, to] for resending.
func (p *healthPublisher) forceDue(from, to time.Time) {
	const day = 24 * time.Hour
	if !to.After(from) {
		return
	}
	if to.Sub(from) >= day {
		from = to.Add(-day)
	}
	fromSlot := from.Sub(from.Truncate(day))
	toSlot := to.Sub(to.Truncate(day))
	due := func(slot time.Duration) bool {
		if fromSlot < toSlot {
			return slot > fromSlot && slot <= toSlot
		}
		// the window wraps past midnight
		return slot > fromSlot || slot <= toSlot
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	for name, r := range p.resources {
		if r.removed || !due(r.slot) {
			continue
		}
		r.force = true
		if needsSend(r) {
			p.enqueueLocked(name)
		}
	}
}
