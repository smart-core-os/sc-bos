package opcua

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const (
	// subscribeRetryInitial is the delay before the first retry of a failed subscribe.
	subscribeRetryInitial = 2 * time.Second
	// subscribeRetryMax caps the retry delay. A point behind a server that is down for the
	// night should still come back on its own, without asking for it every couple of seconds
	// all night.
	subscribeRetryMax = 5 * time.Minute
	// subscribeStagger bounds the random delay before a point's first subscribe attempt, so
	// that a config load spreads its opening requests instead of firing them all at once.
	subscribeStagger = 500 * time.Millisecond
	// resubscribeGrace is how long a subscription has to survive before we treat it as having
	// demonstrably worked, and so start the retry backoff over when it later dies.
	resubscribeGrace = time.Minute
)

// subscriber creates OPC UA subscriptions for individual points.
// It is the one method of *Client that a device needs, kept narrow so tests can drive the
// retry behaviour without an OPC UA server.
type subscriber interface {
	Subscribe(ctx context.Context, nodeId *ua.NodeID) (<-chan *opcua.PublishNotificationData, error)
}

// device represents an OPC UA device that subscribes to variable nodes and updates trait implementations.
// It manages subscriptions for a single logical device and routes OPC UA events to the appropriate trait handlers.
// The type is intentionally unexported as it's an internal implementation detail of the driver.
type device struct {
	conf   *config.Device
	logger *zap.Logger
	client subscriber

	eventHandlers []EventHandler

	faultCheck  *healthpb.FaultCheck
	systemCheck service.SystemCheck
	points      *pointHealth

	// maxStagger bounds the random delay before each point's first subscribe attempt.
	// A field so tests can zero it, which makes the retry timings the only thing on the clock.
	maxStagger time.Duration
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

// subscribe subscribes to every configured variable and pumps its notifications to the trait
// handlers, retrying a point that fails for a reason that might go away until it works.
// It blocks until ctx is cancelled.
//
// A point the server says can never be subscribed is given up on and reported as a device
// config fault; every other failure is retried with backoff, because the alternative - what
// this replaced - abandoned a working point for the lifetime of the config over a single
// slow answer.
func (d *device) subscribe(ctx context.Context) error {
	grp, ctx := errgroup.WithContext(ctx)
	for _, point := range d.conf.Variables {
		nodeId := point.ParsedNodeId
		grp.Go(func() error {
			if !d.stagger(ctx) {
				return ctx.Err()
			}
			err := task.Run(ctx, d.pointTask(nodeId),
				task.WithRetry(task.RetryUnlimited),
				task.WithBackoff(subscribeRetryInitial, subscribeRetryMax),
				// no WithTimeout: Runner.Step cancels the attempt ctx as soon as the Task
				// returns, and this Task only returns when the subscription dies, so a
				// per-attempt deadline would sever every live subscription on a timer.
				// Client.Subscribe puts a deadline on the bounded part instead.
				// no WithErrorLogger either, logSubscribeFailure thins the logging out.
			)
			// with RetryUnlimited the only way task.Run gives up while its ctx is still live is
			// a point we decided can never work, so this is that point being abandoned
			if err != nil && ctx.Err() == nil {
				d.logger.Error("stopped retrying point, the server says this subscription can never work",
					zap.Stringer("point", nodeId), zap.Error(err))
			}
			// park rather than return the error. Returning it would cancel the group ctx and
			// tear down every healthy sibling over one bad node id, and would let grp.Wait
			// return, which applyConfig reads as the config being finished and takes as its
			// cue to close the client every other device is still using.
			<-ctx.Done()
			return ctx.Err()
		})
	}
	return grp.Wait()
}

// stagger waits a random part of maxStagger, reporting false if ctx ended first.
//
// Every device subscribes the moment a config loads and every point costs two sequential round
// trips, so without this a controller with a few hundred points opens with a burst the server
// answers by timing requests out. The concurrency gate in Client caps how many are in flight;
// this stops them arriving in lockstep.
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

// pointTask returns the Task that keeps one point subscribed: it subscribes, then pumps
// notifications until the subscription dies, and tells task.Run how to treat the outcome.
//
// It never returns a nil error, since task.Run reads that as the work being finished, and
// keeping a point subscribed is never finished until ctx ends.
func (d *device) pointTask(nodeId *ua.NodeID) task.Task {
	// consecutive failures, for the logging ladder. Reset by a successful subscribe.
	attempt := 0
	return func(ctx context.Context) (task.Next, error) {
		attempt++
		ch, err := d.client.Subscribe(ctx, nodeId)
		if err != nil {
			switch {
			case ctx.Err() != nil:
				// shutting down: not a fault, and no sense backing off before stopping
				return task.StopNow, err
			case subscribeErrIsPermanent(err):
				// the caller logs this one, since task.Run giving up is the same event
				d.points.setPermanent(nodeId.String(), err)
				return task.StopNow, err
			default:
				d.logSubscribeFailure(nodeId, attempt, err)
				d.points.setFailing(nodeId.String(), err)
				return task.Normal, err
			}
		}

		if attempt > 1 {
			d.logger.Info("subscribed to point", zap.Stringer("point", nodeId), zap.Int("attempts", attempt))
		}
		attempt = 0
		d.points.setOk(nodeId.String())

		start := time.Now()
		err = d.pumpEvents(ctx, ch, nodeId)
		if ctx.Err() != nil {
			return task.StopNow, err
		}
		d.points.setFailing(nodeId.String(), err)
		if lived := time.Since(start); lived >= resubscribeGrace {
			// it demonstrably works, so start the backoff over rather than carry a ramp that
			// describes some older problem
			d.logger.Warn("subscription to point ended, resubscribing",
				zap.Stringer("point", nodeId), zap.Duration("lived", lived), zap.Error(err))
			return task.ResetBackoff, err
		}
		// died almost at once, so let the ramp build, else a flapping server is hammered at
		// the initial delay forever
		return task.Normal, err
	}
}

// pumpEvents dispatches notifications from ch to the trait handlers until the subscription
// ends or ctx does. It always reports an error, for the reason on pointTask.
func (d *device) pumpEvents(ctx context.Context, ch <-chan *opcua.PublishNotificationData, nodeId *ua.NodeID) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-ch:
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
			d.handleEvent(ctx, event, nodeId)
		}
	}
}

// subscriptionEnded reports the error a notification carries when it says the subscription
// itself has stopped, or nil when it is an ordinary notification to dispatch.
//
// gopcua never closes the notification channel and repairs a reconnect behind our back, so a
// subscription that worked and then died does not arrive as a closed channel or a read error.
// It arrives as a StatusChangeNotification carrying a Bad status, which handleEvent used to
// discard as an unhandled event, leaving the point silently dead.
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

// logSubscribeFailure logs a failed subscribe, thinning the logging out as attempts mount.
//
// A controller with a few hundred points against a server that is down would otherwise repeat
// the same line for every point on every backoff period, forever, drowning out whatever else
// is going on. Mirrors the ladder the BACnet driver uses for offline devices.
func (d *device) logSubscribeFailure(nodeId *ua.NodeID, attempt int, err error) {
	switch {
	case attempt <= 2:
		d.logger.Warn("failed to subscribe to point, will keep trying",
			zap.Stringer("point", nodeId), zap.Int("attempt", attempt), zap.Error(err))
	case attempt == 3:
		d.logger.Warn("failed to subscribe to point, reducing logging",
			zap.Stringer("point", nodeId), zap.Int("attempt", attempt), zap.Error(err))
	case attempt%10 == 0:
		d.logger.Debug("still failing to subscribe to point",
			zap.Stringer("point", nodeId), zap.Int("attempt", attempt), zap.Error(err))
	}
}

// handleEvent processes OPC UA subscription events and routes them to trait handlers.
// It handles both DataChangeNotification (variable value changes) and EventNotificationList (OPC UA events).
// Values with Bad status codes are logged as warnings and not passed to trait handlers.
func (d *device) handleEvent(ctx context.Context, event *opcua.PublishNotificationData, node *ua.NodeID) {
	if event.Error != nil {
		d.faultCheck.UpdateReliability(ctx, healthpb.ReliabilityFromErr(event.Error))
		service.UpdateSystemCheck(d.systemCheck, event.Error)
		d.logger.Warn("OPC UA server connection error", zap.Stringer("node", node), zap.Error(event.Error))
		return
	}

	switch x := event.Value.(type) {
	case *ua.DataChangeNotification:
		for _, item := range x.MonitoredItems {
			if item.Value == nil ||
				item.Value.Value == nil {
				continue
			}

			d.handleStatusValue(ctx, node, item.Value.Status, item.Value.Value.Value())
		}

	case *ua.EventNotificationList:
		for _, item := range x.Events {
			for _, field := range item.EventFields {
				d.handleStatusValue(ctx, node, field.StatusCode(), field.Value())
			}
		}

	case *ua.StatusChangeNotification:
		// a Bad status here means the subscription has ended, which pumpEvents intercepts
		// before anything reaches us. Anything else is the server telling us something about
		// a subscription that is still running.
		d.logger.Debug("subscription status changed",
			zap.Stringer("node", node), zap.String("code", x.Status.Error()))

	default:
		d.logger.Warn("unhandled event", zap.Any("energyValue", event.Value))
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
