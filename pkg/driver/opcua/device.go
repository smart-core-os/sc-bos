package opcua

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const (
	// subscribeRetryInitial is the delay before the first retry of a failed subscribe.
	subscribeRetryInitial = 2 * time.Second
	// subscribeRetryMax caps the retry delay. A device behind a server that is down for the
	// night should still come back on its own, without asking for it every couple of seconds
	// all night.
	subscribeRetryMax = 5 * time.Minute
	// subscribeStagger bounds the random delay before a device's first subscribe attempt, so
	// that a config load spreads its opening requests instead of firing them all at once.
	subscribeStagger = 500 * time.Millisecond
	// resubscribeGrace is how long a subscription has to survive before we treat it as having
	// demonstrably worked, and so start the retry backoff over when it later dies.
	resubscribeGrace = time.Minute
)

// subscriber creates the OPC UA subscription a device monitors its points on.
// It is the one method of *Client that a device needs, kept narrow so tests can drive the
// retry behaviour without an OPC UA server.
type subscriber interface {
	NewSubscription(ctx context.Context) (pointSubscription, error)
}

// pointSubscription is one device's subscription: the notifications the server publishes for
// it, the monitored items on it, and the map from a notification's client handle back to the
// node it came from. *pointSub is the implementation.
type pointSubscription interface {
	Monitor(ctx context.Context, nodeIds ...*ua.NodeID) []monitorResult
	Notifications() <-chan *opcua.PublishNotificationData
	Node(handle uint32) (*ua.NodeID, bool)
	Cancel(ctx context.Context)
}

// nodeLookup resolves the client handle on a notification back to its node. One item per
// subscription made this a closure variable; a device's whole point list on one makes it a
// lookup. *pointSub.Node satisfies it.
type nodeLookup func(handle uint32) (*ua.NodeID, bool)

// device represents an OPC UA device that subscribes to variable nodes and updates trait implementations.
// It manages the subscription for a single logical device and routes OPC UA events to the appropriate trait handlers.
// The type is intentionally unexported as it's an internal implementation detail of the driver.
type device struct {
	conf   *config.Device
	logger *zap.Logger
	client subscriber

	eventHandlers []EventHandler

	faultCheck  *healthpb.FaultCheck
	systemCheck service.SystemCheck
	points      *pointHealth

	// maxStagger bounds the random delay before the device's first subscribe attempt.
	// A field so tests can zero it, which makes the retry timings the only thing on the clock.
	maxStagger time.Duration

	// unknownHandles counts notifications we could not attribute to a node, for the thinning
	// in logUnknownHandle. Only ever touched from the goroutine pumping this device.
	unknownHandles int
}

// newDevice creates a new device instance for the given configuration.
// Trait implementations (Electric, Meter, Transport, udmi) must be assigned separately before calling run.
func newDevice(conf *config.Device, logger *zap.Logger, client subscriber, check *healthpb.FaultCheck, systemCheck service.SystemCheck) *device {
	return &device{
		client:      client,
		conf:        conf,
		faultCheck:  check,
		logger:      logger,
		systemCheck: systemCheck,
		points:      newPointHealth(check),
		maxStagger:  subscribeStagger,
	}
}

// subscribe monitors every configured variable on one subscription and pumps its notifications
// to the trait handlers, retrying while there is anything left that might come good.
// It blocks until ctx is cancelled.
//
// A point the server says can never be monitored is given up on and reported as a device
// config fault; every other failure is retried, because the alternative - what this replaced -
// abandoned a working point for the lifetime of the config over a single slow answer.
func (d *device) subscribe(ctx context.Context) error {
	if d.stagger(ctx) {
		err := task.Run(ctx, d.deviceTask(),
			task.WithRetry(task.RetryUnlimited),
			task.WithBackoff(subscribeRetryInitial, subscribeRetryMax),
			// no WithTimeout: Runner.Step cancels the attempt ctx as soon as the Task
			// returns, and this Task only returns when the subscription dies, so a
			// per-attempt deadline would sever a live subscription on a timer. Client puts a
			// deadline on each request instead.
			// no WithErrorLogger either, logSubscribeFailure thins the logging out.
		)
		// with RetryUnlimited the only way task.Run gives up while its ctx is still live is a
		// device with no point left that could ever work, so this is that device being
		// abandoned
		if err != nil && ctx.Err() == nil {
			d.logger.Error("stopped monitoring device, it has no point that can be monitored",
				zap.String("device", d.conf.Name), zap.Error(err))
		}
	}
	// park rather than return. Returning would let grp.Wait in applyConfig return, which it
	// reads as the config being finished and takes as its cue to close the client every other
	// device is still using.
	<-ctx.Done()
	return ctx.Err()
}

// stagger waits a random part of maxStagger, reporting false if ctx ended first.
//
// Every device subscribes the moment a config loads, so without this a controller with a few
// dozen devices opens with a burst the server answers by timing requests out. The concurrency
// gate in Client caps how many requests are in flight; this stops them arriving in lockstep.
func (d *device) stagger(ctx context.Context) bool {
	if d.maxStagger <= 0 {
		return true
	}
	t := time.NewTimer(rand.N(d.maxStagger))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// deviceTask returns the Task that keeps one device monitored: it creates a subscription, puts
// the device's points on it as monitored items, then pumps notifications until the subscription
// dies, and tells task.Run how to treat the outcome.
//
// It never returns a nil error, since task.Run reads that as the work being finished, and
// keeping a device monitored is never finished until ctx ends.
func (d *device) deviceTask() task.Task {
	// permanent holds the node ids the server has said can never be monitored, so later
	// attempts stop asking about them.
	//
	// Deliberately unlocked. This goroutine writes it during setup, hands it to the straggler
	// goroutine for the life of the pump, and reads it again only after joining that goroutine
	// in runSubscription. It also survives a resubscribe on purpose, which matches what a
	// point being given up on has always meant: nothing short of a config reload asks again.
	permanent := make(map[string]bool)
	// consecutive failed attempts, for the logging ladder. Reset once the device is monitored.
	attempt := 0
	return func(ctx context.Context) (task.Next, error) {
		attempt++
		nodeIds := d.wantedNodes(permanent)
		if len(nodeIds) == 0 {
			// either the device configures no variables, or every one of them has been given
			// up on and the fault naming them is already raised. Either way there is nothing
			// to ask the server for, so a subscription would carry no items.
			return task.StopNow, errNoWorkablePoints
		}

		sub, err := d.client.NewSubscription(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// shutting down: not a fault, and no sense backing off before stopping
				return task.StopNow, err
			}
			// no per-node status to classify - the subscription is what failed, not any item -
			// so this is always retried, and every point is reported as waiting on it
			d.logSubscribeFailure(attempt, "failed to create the device subscription, will keep trying", err)
			d.failAllPoints(permanent, err)
			return task.Normal, err
		}

		results := sub.Monitor(ctx, nodeIds...)
		if ctx.Err() != nil {
			// shutting down, so whatever failed here failed because we stopped asking.
			// Reporting it would raise a fault on a device that is going away, and the
			// subscription is an orphan either way.
			sub.Cancel(ctx)
			return task.StopNow, ctx.Err()
		}
		out := d.applyResults(results, permanent)
		if out.ok == 0 {
			// nothing is delivering, so the subscription is an orphan: left behind it would
			// fold into the client-wide publish timeout and gopcua would spawn a permanently
			// blocked notify goroutine for it on every transport error
			sub.Cancel(ctx)
			if len(out.retry) == 0 {
				// the server refused every point outright, so the next attempt would find
				// nothing to ask about
				return task.StopNow, out.err
			}
			d.logSubscribeFailure(attempt, "failed to monitor any of this device's points, will keep trying", out.err)
			return task.Normal, out.err
		}
		// a partial success is not an orphan: the points that worked are delivering, and the
		// rest rejoin them from retryStragglers without disturbing what works
		if attempt > 1 {
			d.logger.Info("monitoring device points", zap.String("device", d.conf.Name),
				zap.Int("points", out.ok), zap.Int("attempts", attempt))
		}
		attempt = 0

		start := time.Now()
		err = d.runSubscription(ctx, sub, out.retry, permanent)
		// unconditionally, however the pump ended. gopcua otherwise leaves the subscription in
		// the client, folded into its publish timeout, with a blocked notify goroutine spawned
		// for it on every transport error - one leak per resubscribe.
		sub.Cancel(ctx)
		if ctx.Err() != nil {
			return task.StopNow, err
		}
		d.failAllPoints(permanent, err)
		if lived := time.Since(start); lived >= resubscribeGrace {
			// it demonstrably works, so start the backoff over rather than carry a ramp that
			// describes some older problem
			d.logger.Warn("subscription to device ended, resubscribing", zap.String("device", d.conf.Name),
				zap.Duration("lived", lived), zap.Error(err))
			return task.ResetBackoff, err
		}
		// died almost at once, so let the ramp build, else a flapping server is hammered at
		// the initial delay forever
		return task.Normal, err
	}
}

// wantedNodes is the device's configured nodes, less the ones we have given up on, deduped.
//
// Deduped here rather than in ParseConfig because a config listing the same node twice loads
// today and has to keep loading. It used to mean two subscriptions to the same node, which was
// merely wasteful; on one subscription it would mean two monitored items delivering the same
// value under two handles.
func (d *device) wantedNodes(permanent map[string]bool) []*ua.NodeID {
	var nodeIds []*ua.NodeID
	seen := make(map[string]bool, len(d.conf.Variables))
	for _, v := range d.conf.Variables {
		nodeId := v.ParsedNodeId
		if nodeId == nil {
			// ParseConfig fills this in for every variable, so this is a Device assembled in
			// code. Skipping beats a nil dereference in the pump.
			continue
		}
		key := nodeId.String()
		if seen[key] || permanent[key] {
			continue
		}
		seen[key] = true
		nodeIds = append(nodeIds, nodeId)
	}
	return nodeIds
}

// attemptOutcome is what one Monitor call did to the device's points, split by what happens
// to them next.
type attemptOutcome struct {
	ok    int          // how many nodes are now monitored
	retry []*ua.NodeID // nodes that failed for a reason that may go away
	err   error        // every failure joined, nil when there were none
}

// applyResults records the outcome of one Monitor call against the device's point health, and
// reports what to do next.
//
// Failures are split by the status code the server gave that item, which is why Monitor wraps
// each one per node: a device asking about 50 points can get a mix of a node id that will
// never exist and a node that was momentarily unreadable, and they need opposite treatment.
//
// The whole batch is committed to the fault check in one go. Called per point it would publish
// a growing version of the same fault for every point in the batch, each one visible over the
// devices API.
func (d *device) applyResults(results []monitorResult, permanent map[string]bool) attemptOutcome {
	var out attemptOutcome
	var ok []string
	failing, refused := make(map[string]error), make(map[string]error)
	var errs []error
	for _, res := range results {
		nodeId := res.NodeId.String()
		switch {
		case res.Err == nil:
			out.ok++
			ok = append(ok, nodeId)
		case subscribeErrIsPermanent(res.Err):
			permanent[nodeId] = true
			refused[nodeId] = res.Err
			errs = append(errs, res.Err)
			// once per point, and it is never asked about again, so there is nothing here to
			// thin out the way logSubscribeFailure has to
			d.logger.Error("stopped retrying point, the server says this subscription can never work",
				zap.Stringer("point", res.NodeId), zap.Error(res.Err))
		default:
			failing[nodeId] = res.Err
			out.retry = append(out.retry, res.NodeId)
			errs = append(errs, res.Err)
		}
	}
	d.points.applyBatch(ok, failing, refused)
	out.err = errors.Join(errs...)
	return out
}

// failAllPoints reports every point still worth monitoring as failing, for a subscription that
// has died or never opened and so has taken all of them with it.
func (d *device) failAllPoints(permanent map[string]bool, err error) {
	nodeIds := d.wantedNodes(permanent)
	failing := make(map[string]error, len(nodeIds))
	for _, nodeId := range nodeIds {
		failing[nodeId.String()] = err
	}
	d.points.applyBatch(nil, failing, nil)
}

// runSubscription pumps the subscription until it ends, retrying its stragglers alongside.
//
// The straggler goroutine runs on a ctx cancelled the moment the pump stops, and is joined
// before returning. That join is what lets permanent be used without a lock, and what
// guarantees nothing can call Monitor on a subscription the caller is about to cancel.
func (d *device) runSubscription(ctx context.Context, sub pointSubscription, stragglers []*ua.NodeID, permanent map[string]bool) error {
	retryCtx, stopRetries := context.WithCancel(ctx)
	defer stopRetries()

	var retries sync.WaitGroup
	if len(stragglers) > 0 {
		retries.Add(1)
		go func() {
			defer retries.Done()
			d.retryStragglers(retryCtx, sub, stragglers, permanent)
		}()
	}

	err := d.pumpEvents(ctx, sub)
	stopRetries()
	retries.Wait()
	return err
}

// retryStragglers monitors nodes that failed transiently onto the live subscription, so that a
// point which missed its turn joins its siblings without disturbing them.
//
// Its own clock, because it describes a different failure from the device's. The device's
// backoff is about a subscription that cannot be established at all, and retrying one point on
// that cycle would tear down every point that is delivering. It terminates once every
// straggler is monitored or given up on, so a device whose points have settled leaves no
// goroutine of its own behind.
func (d *device) retryStragglers(ctx context.Context, sub pointSubscription, nodeIds []*ua.NodeID, permanent map[string]bool) {
	delay := subscribeRetryInitial
	attempt := 0
	for len(nodeIds) > 0 {
		// wait before the first retry as well as the later ones: these nodes were asked about
		// a moment ago, and nothing has changed since
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
		delay = nextBackoff(delay, subscribeRetryMax)
		attempt++

		results := sub.Monitor(ctx, nodeIds...)
		if ctx.Err() != nil {
			// the pump has stopped, so these results describe a subscription nobody is reading
			// and the device task is about to report every point as failing anyway
			return
		}
		out := d.applyResults(results, permanent)
		if out.ok > 0 {
			d.logger.Info("monitored straggling device points", zap.String("device", d.conf.Name),
				zap.Int("points", out.ok), zap.Int("attempt", attempt))
		}
		if len(out.retry) > 0 {
			d.logSubscribeFailure(attempt, "failed to monitor some of this device's points, will keep trying", out.err)
		}
		nodeIds = out.retry
	}
}

// nextBackoff is the delay to wait after delay, ramping by half up to max.
// Matches task.WithBackoff's ramp, so the two retry loops in this file behave alike.
func nextBackoff(delay, max time.Duration) time.Duration {
	delay += delay / 2
	if delay > max {
		delay = max
	}
	return delay
}

// pumpEvents dispatches notifications from sub to the trait handlers until the subscription
// ends or ctx does. It always reports an error, for the reason on deviceTask.
func (d *device) pumpEvents(ctx context.Context, sub pointSubscription) error {
	notifications := sub.Notifications()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-notifications:
			if !open {
				// gopcua does not close the channel today, so this is belt and braces
				return errSubscriptionClosed
			}
			if event == nil {
				continue
			}
			if err := subscriptionEnded(event); err != nil {
				return err
			}
			d.handleEvent(ctx, event, sub.Node)
		}
	}
}

// subscriptionEnded reports the error a notification carries when it says the subscription
// itself has stopped, or nil when it is an ordinary notification to dispatch.
//
// gopcua never closes the notification channel and repairs a reconnect behind our back, so a
// subscription that worked and then died does not arrive as a closed channel or a read error.
// It arrives as a StatusChangeNotification carrying a Bad status, which handleEvent used to
// discard as an unhandled event, leaving the device silently dead.
func subscriptionEnded(event *opcua.PublishNotificationData) error {
	if event.Error != nil {
		// leave a transport error to handleEvent: gopcua's own reconnect is already repairing
		// it and the subscription survives, so resubscribing here would fight it
		return nil
	}
	change, ok := event.Value.(*ua.StatusChangeNotification)
	if !ok || !statusIsBad(change.Status) {
		return nil
	}
	return fmt.Errorf("subscription ended: %w", change.Status)
}

// logSubscribeFailure logs a failed subscribe attempt, thinning the logging out as attempts
// mount. The message comes from the caller so that the device's own retries and its straggler
// retries thin out independently of each other.
//
// A controller with a few dozen devices against a server that is down would otherwise repeat
// the same line for every device on every backoff period, forever, drowning out whatever else
// is going on. Mirrors the ladder the BACnet driver uses for offline devices.
func (d *device) logSubscribeFailure(attempt int, msg string, err error) {
	switch {
	case attempt <= 2:
		d.logger.Warn(msg, zap.String("device", d.conf.Name), zap.Int("attempt", attempt), zap.Error(err))
	case attempt == 3:
		d.logger.Warn(msg+", reducing logging", zap.String("device", d.conf.Name),
			zap.Int("attempt", attempt), zap.Error(err))
	case attempt%10 == 0:
		d.logger.Debug(msg, zap.String("device", d.conf.Name), zap.Int("attempt", attempt), zap.Error(err))
	}
}

// logUnknownHandle reports a notification that cannot be attributed to a node, thinning the
// logging out as they mount: a server reporting an item we do not know about would otherwise
// repeat the same line every publishing cycle for as long as the config lives.
func (d *device) logUnknownHandle(handle uint32) {
	d.unknownHandles++
	switch {
	case d.unknownHandles <= 2:
		d.logger.Warn("dropping a notification for an unknown monitored item",
			zap.String("device", d.conf.Name), zap.Uint32("handle", handle))
	case d.unknownHandles == 3:
		d.logger.Warn("dropping a notification for an unknown monitored item, reducing logging",
			zap.String("device", d.conf.Name), zap.Uint32("handle", handle))
	case d.unknownHandles%100 == 0:
		d.logger.Debug("still dropping notifications for unknown monitored items",
			zap.String("device", d.conf.Name), zap.Uint32("handle", handle),
			zap.Int("count", d.unknownHandles))
	}
}

// handleEvent processes OPC UA subscription events and routes them to trait handlers.
// Every monitored item on the subscription reports under the client handle we chose for it, so
// lookup is what says which node a value belongs to.
// Values with Bad status codes are logged as warnings and not passed to trait handlers.
func (d *device) handleEvent(ctx context.Context, event *opcua.PublishNotificationData, lookup nodeLookup) {
	if event.Error != nil {
		d.faultCheck.UpdateReliability(ctx, healthpb.ReliabilityFromErr(event.Error))
		service.UpdateSystemCheck(d.systemCheck, event.Error)
		d.logger.Warn("OPC UA server connection error", zap.String("device", d.conf.Name), zap.Error(event.Error))
		return
	}

	switch x := event.Value.(type) {
	case *ua.DataChangeNotification:
		for _, item := range x.MonitoredItems {
			// before the nil-value check: an item we cannot name is not this device's problem
			// whatever it carries
			node, ok := lookup(item.ClientHandle)
			if !ok {
				// dropped rather than faulted. Nothing on the wire says which node this came
				// from, so there is no point to name, and the device's own points are
				// unaffected by the server reporting an item we do not know about.
				d.logUnknownHandle(item.ClientHandle)
				continue
			}
			if item.Value == nil ||
				item.Value.Value == nil {
				continue
			}

			d.handleStatusValue(ctx, node, item.Value.Status, item.Value.Value.Value())
		}

	case *ua.StatusChangeNotification:
		// a Bad status here means the subscription has ended, which pumpEvents intercepts
		// before anything reaches us. Anything else is the server telling us something about
		// a subscription that is still running.
		d.logger.Debug("subscription status changed",
			zap.String("device", d.conf.Name), zap.String("code", x.Status.Error()))

	default:
		// deliberately no *ua.EventNotificationList case. Every item we create asks for the
		// Value attribute with no EventFilter, so a conformant server cannot send us events;
		// and the branch that used to be here pushed every field of an event at the node
		// without mapping the SelectClauses, which is worse than being logged as unhandled.
		d.logger.Warn("unhandled event", zap.String("device", d.conf.Name), zap.Any("event", event.Value))
	}
}

// handleStatusValue dispatches value to the trait handlers unless status is Bad,
// updating the device health to match the status severity.
// Only the severity bits of the status are significant: a Good code carrying info bits,
// notably the Overflow bit a busy subscription queue sets, still delivers a usable value.
func (d *device) handleStatusValue(ctx context.Context, node *ua.NodeID, status ua.StatusCode, value any) {
	switch {
	case statusIsBad(status):
		setPointReadNotOk(ctx, node.String(), status, d.faultCheck)
		d.logger.Warn("error monitoring node", zap.Stringer("node", node), zap.String("code", status.Error()))
		return
	case statusIsUncertain(status):
		setPointReadUncertain(ctx, node.String(), status, d.faultCheck)
		d.logger.Debug("uncertain value for node", zap.Stringer("node", node), zap.String("code", status.Error()))
	default: // Good, whatever info bits it carries
		d.faultCheck.UpdateReliability(ctx, healthpb.ReliabilityFromErr(nil))
		service.UpdateSystemCheck(d.systemCheck, nil)
	}
	d.handleTraitEvent(ctx, node, value)
}

// handleTraitEvent dispatches an OPC UA value change to all configured trait handlers.
// Each trait handler is responsible for checking if the node ID matches its configuration.
func (d *device) handleTraitEvent(ctx context.Context, node *ua.NodeID, value any) {
	for _, handler := range d.eventHandlers {
		handler.handleEvent(ctx, node, value)
	}
}

// nodeIdsAreEqual compares a string node ID with a ua.NodeID for equality.
// Returns true if n is non-nil and its string representation matches nodeId.
func nodeIdsAreEqual(nodeId string, n *ua.NodeID) bool {
	return n != nil && nodeId == n.String()
}
