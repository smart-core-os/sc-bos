package paxton

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

const (
	DriverName = "paxton"

	defaultDoorsInterval  = 5 * time.Minute
	defaultEventsInterval = 5 * time.Second
	defaultCardsInterval  = 5 * time.Minute
)

type Driver struct {
	*service.Service[config.Root]
	announcer   *node.ReplaceAnnouncer
	logger      *zap.Logger
	systemCheck service.SystemCheck

	// http client helper
	client *Client

	// doors that are announced by this driver; key is the door ID
	doors sync.Map

	// cardholders are users that are announced by this driver; key is the user ID
	cardholders sync.Map

	// eventsController implements the SecurityEventApiServer
	eventsController *EventsController

	// seen deduplicates events when both polling and SignalR are active
	seen *seenEvents

	// prevDone is closed once the previous generation's background tasks have fully
	// stopped. applyConfig waits on it before mutating shared state so it does not race
	// the previous generation's goroutines. Only accessed from applyConfig, which the
	// service framework calls serially.
	prevDone chan struct{}
}

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)

	d := &Driver{
		logger:      logger,
		announcer:   node.NewReplaceAnnouncer(services.Node),
		systemCheck: services.SystemCheck,
	}

	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser[config.Root](config.ParseConfig),
		service.WithOnStop[config.Root](func() {
			if d.systemCheck != nil {
				d.systemCheck.Dispose()
			}
		}),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logCtx service.RetryContext) {
			logCtx.LogTo("applyConfig", logger)
		})),
	)

	return d
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	// MonoApply has already cancelled the previous generation's context. Wait for its
	// background tasks to fully stop before mutating shared state (d.client, d.seen,
	// d.eventsController, the door/cardholder maps) so the two generations never overlap.
	if d.prevDone != nil {
		select {
		case <-d.prevDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	cli := retryablehttp.NewClient()
	cli.RetryMax = 3
	cli.RetryWaitMax = 10 * time.Second

	if cfg.InsecureSkipVerify {
		cli.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		cli.HTTPClient.Transport = &http.Transport{}
	}

	d.client = NewClient(cli, d.logger.Named("client"), cfg, d.systemCheck)
	d.seen = newSeenEvents()

	announcer := d.announcer.Replace(ctx)

	// Replace has undone the previous generation's announcements, so clear the door and
	// cardholder maps. refreshDoors/refreshCardholders only announce IDs absent from these
	// maps; without clearing, a reconfigure would leave doors and cardholders un-announced.
	d.doors.Clear()
	d.cardholders.Clear()

	if cfg.EnableSecurityEvents {
		d.eventsController = NewEventsController(d)
		announcer.Announce(
			cfg.SecurityEventsName,
			node.HasServer(securityeventpb.RegisterSecurityEventApiServer, securityeventpb.SecurityEventApiServer(d.eventsController)),
			node.HasTrait(securityeventpb.TraitName),
		)
	}

	// doorsReady is closed after the first successful door refresh so that the
	// SignalR goroutine can wait until d.doors is populated before subscribing.
	doorsReady := make(chan struct{})
	// doorsChanged receives a signal (non-blocking) when a later refresh
	// discovers addresses not present at the time of the last SignalR connect,
	// prompting an immediate reconnect to pick up the new per-door subscriptions.
	doorsChanged := make(chan struct{}, 1)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return d.runDoors(ctx, announcer, cfg, doorsReady, doorsChanged) })
	group.Go(func() error { return d.runCardholders(ctx, announcer, cfg) })
	group.Go(func() error { return d.runSeenEventsCleanup(ctx, cfg) })
	if !cfg.DisablePolling {
		group.Go(func() error { return d.runEvents(ctx, cfg) })
	}
	if cfg.EnableSignalR {
		group.Go(func() error { return d.runSignalR(ctx, cfg, doorsReady, doorsChanged) })
	}

	done := make(chan struct{})
	d.prevDone = done
	go func() {
		defer close(done)
		if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			d.logger.Error("background task(s) failed", zap.Error(err))
		}
	}()

	return nil
}

// runDoors periodically refreshes the door list. On the first successful refresh it
// closes doorsReady to unblock runSignalR. On subsequent refreshes it signals
// doorsChanged if the set of door addresses has changed (added or removed) so SignalR
// reconnects with up-to-date per-door subscriptions.
func (d *Driver) runDoors(ctx context.Context, announcer node.Announcer, cfg config.Root, doorsReady chan<- struct{}, doorsChanged chan<- struct{}) error {
	var knownAddresses map[int]struct{}
	doorsReadyClosed := false

	currentAddresses := func() map[int]struct{} {
		addrs := make(map[int]struct{})
		d.doors.Range(func(k, _ any) bool {
			addrs[k.(int)] = struct{}{}
			return true
		})
		return addrs
	}

	refresh := func() {
		if err := d.refreshDoors(ctx, announcer, cfg.DeviceNamePrefix); err != nil {
			d.logger.Error("failed to refresh doors", zap.Error(err))
			return
		}
		addrs := currentAddresses()
		if !doorsReadyClosed {
			// First successful refresh: snapshot addresses and unblock SignalR.
			knownAddresses = addrs
			close(doorsReady)
			doorsReadyClosed = true
			return
		}
		// Signal a reconnect if the address set changed since the last connection.
		if !sameAddressSet(knownAddresses, addrs) {
			knownAddresses = addrs
			select {
			case doorsChanged <- struct{}{}:
			default:
			}
		}
	}

	refresh() // immediate on startup

	throttle := time.NewTicker(cfg.DoorsInterval.Or(defaultDoorsInterval))
	defer throttle.Stop()
	for {
		select {
		case <-throttle.C:
			refresh()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runCardholders periodically refreshes the cardholder list.
func (d *Driver) runCardholders(ctx context.Context, announcer node.Announcer, cfg config.Root) error {
	refresh := func() {
		if err := d.refreshCardholders(ctx, announcer, cfg.CardHolderPrefix); err != nil {
			d.logger.Error("failed to refresh cardholders", zap.Error(err))
		}
	}

	refresh() // immediate on startup

	throttle := time.NewTicker(cfg.CardsInterval.Or(defaultCardsInterval))
	defer throttle.Stop()
	for {
		select {
		case <-throttle.C:
			refresh()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runSeenEventsCleanup periodically evicts expired entries from the deduplication cache.
func (d *Driver) runSeenEventsCleanup(ctx context.Context, cfg config.Root) error {
	ticker := time.NewTicker(cfg.SeenEventsCleanupInterval.Or(seenEventsCleanupInterval))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.seen.cleanup(cfg.SeenEventsMaxAge.Or(seenEventsMaxAge))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// sameAddressSet reports whether a and b contain exactly the same keys.
func sameAddressSet(a, b map[int]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// runEvents polls the REST API for new events on a fixed interval.
func (d *Driver) runEvents(ctx context.Context, cfg config.Root) error {
	// since is the lower bound for the next query. It starts 24h back to backfill recent
	// history, then advances to just before each poll began so consecutive query windows
	// overlap slightly. The seen cache removes the overlapping duplicates; without this,
	// events occurring between a poll and the next tick would fall into a gap and be missed.
	since := time.Now().Add(-24 * time.Hour)

	poll := func() {
		start := time.Now()
		events, err := d.client.GetEvents(ctx, since, false)
		if err != nil {
			d.logger.Error("failed to get events via polling", zap.Error(err))
			return
		}
		d.processEvents(ctx, events, "polling")
		since = start
	}

	poll() // immediate on startup

	throttle := time.NewTicker(cfg.EventsInterval.Or(defaultEventsInterval))
	defer throttle.Stop()
	for {
		select {
		case <-throttle.C:
			poll()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runSignalR maintains a SignalR connection, reconnecting on a drop or when new doors
// are discovered. It waits for doorsReady before the initial connection so that
// per-door subscriptions can be set up with the full door list.
func (d *Driver) runSignalR(ctx context.Context, cfg config.Root, doorsReady <-chan struct{}, doorsChanged <-chan struct{}) error {
	select {
	case <-doorsReady:
	case <-ctx.Done():
		return ctx.Err()
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Collect current door addresses for per-door subscriptions.
		var doorAddresses []int
		d.doors.Range(func(k, _ any) bool {
			doorAddresses = append(doorAddresses, k.(int))
			return true
		})

		// Collect active roll call IDs for per-rollcall subscriptions.
		rollCallIDs, err := d.client.GetRollCallIDs(ctx)
		if err != nil {
			d.logger.Warn("failed to fetch roll call IDs for SignalR", zap.Error(err))
		}

		ch, err := d.client.streamSignalR(ctx, doorAddresses, rollCallIDs)
		if err != nil {
			d.logger.Error("failed to connect to SignalR", zap.Error(err))
			select {
			case <-time.After(10 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		d.logger.Info("signalr connected",
			zap.Int("doors", len(doorAddresses)),
			zap.Int("rollCalls", len(rollCallIDs)),
		)

		newDoorsFound := false
	receiveLoop:
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					break receiveLoop
				}
				switch {
				case msg.liveEvent != nil:
					e, err := msg.liveEvent.toEvent()
					if err != nil {
						// A zero EventTime would poison the dedup fingerprint (so the REST
						// copy of this event wouldn't be recognised as a duplicate) and would
						// stamp the security event and access attempt at the zero time. Drop it.
						d.logger.Warn("failed to parse signalr live event time, dropping event",
							zap.String("eventTime", msg.liveEvent.EventTime),
							zap.Error(err),
						)
						continue
					}
					d.forwardEvents(ctx, []Event{e})
				case msg.doorEvent != nil:
					d.handleSignalRDoorEvent(msg.doorEvent)
				case msg.doorStatusEvent != nil:
					d.handleSignalRDoorStatusEvent(msg.doorStatusEvent)
				case msg.rollCallEvent != nil:
					d.handleSignalRRollCallEvent(msg.rollCallEvent)
				}
			case <-doorsChanged:
				newDoorsFound = true
				break receiveLoop
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if newDoorsFound {
			d.logger.Debug("new doors discovered, reconnecting SignalR for updated subscriptions")
		} else {
			d.logger.Warn("signalr disconnected, reconnecting...")
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// handleSignalRDoorEvent processes a doorEvents message.
func (d *Driver) handleSignalRDoorEvent(e *SignalRDoorEvent) {
	state := "unlocked"
	if e.Locked {
		state = "locked"
	}
	d.logger.Debug("signalr door event",
		zap.Int("doorId", e.DoorID),
		zap.String("state", state),
	)
}

// handleSignalRDoorStatusEvent processes a doorStatusEvents message.
func (d *Driver) handleSignalRDoorStatusEvent(e *SignalRDoorStatusEvent) {
	d.logger.Debug("signalr door status event",
		zap.Int("doorId", e.DoorID),
		zap.Bool("intruderAlarmArmed", e.Status.IntruderAlarmArmed),
		zap.Bool("psuContactClosed", e.Status.PsuContactClosed),
		zap.Bool("tamperContactClosed", e.Status.TamperContactClosed),
		zap.Bool("doorContactClosed", e.Status.DoorContactClosed),
		zap.Bool("alarmTripped", e.Status.AlarmTripped),
		zap.Bool("doorRelayOpen", e.Status.DoorRelayOpen),
	)
}

// forwardEvents deduplicates SignalR liveEvents by content fingerprint (since they carry no
// integer ID) and forwards new ones to door/cardholder state and the security events stream.
// The fingerprint is shared with processEvents so that a REST-polled event registered first
// will suppress the same event arriving later via SignalR, and vice versa.
func (d *Driver) forwardEvents(ctx context.Context, events []Event) {
	var newEvents []Event
	for _, e := range events {
		if !d.seen.markSeen(eventKey(e)) {
			d.logger.Debug("skipping duplicate signalr live event",
				zap.String("description", e.EventDescription),
				zap.Time("time", e.EventTime),
			)
			continue
		}
		newEvents = append(newEvents, e)
	}
	if len(newEvents) == 0 {
		return
	}
	if err := d.linkEvents(ctx, newEvents); err != nil {
		d.logger.Error("failed to link events", zap.Error(err))
	}
	if d.eventsController != nil {
		if err := d.eventsController.processSecurityEvents(ctx, newEvents); err != nil {
			d.logger.Error("failed to process security events", zap.Error(err))
		}
	}
}

// handleSignalRRollCallEvent logs a rollCallEvents message.
func (d *Driver) handleSignalRRollCallEvent(e *SignalRRollCallEvent) {
	fields := []zap.Field{
		zap.Int("rollCallReportId", e.RollCallReportID),
		zap.String("eventType", e.RollCallEventType),
	}
	if e.MusterEventData != nil {
		fields = append(fields,
			zap.Int("userId", e.MusterEventData.UserID),
			zap.String("musteredAt", e.MusterEventData.MusteredAt),
		)
	}
	if e.MarkedSafeRecordEventData != nil {
		fields = append(fields, zap.Int("safeUserId", e.MarkedSafeRecordEventData.UserID))
		if e.MarkedSafeRecordEventData.MarkedSafeBy != nil {
			fields = append(fields, zap.String("markedSafeBy", e.MarkedSafeRecordEventData.MarkedSafeBy.Name))
		}
	}
	d.logger.Debug("signalr roll call event", fields...)
}

// processEvents deduplicates events and forwards new ones to door/cardholder state and the
// security events stream. Each event is registered under both its integer ID and a content
// fingerprint so that a matching SignalR liveEvent (which carries no ID) is also suppressed.
// The source label ("polling" or "signalr") is included in log output.
func (d *Driver) processEvents(ctx context.Context, events []Event, source string) {
	var newEvents []Event
	for _, e := range events {
		idKey := strconv.Itoa(e.ID)
		fpKey := eventKey(e)
		// Register both keys; skip if either was already seen.
		idNew := d.seen.markSeen(idKey)
		fpNew := d.seen.markSeen(fpKey)
		if !idNew || !fpNew {
			d.logger.Debug("skipping duplicate event",
				zap.Int("id", e.ID),
				zap.String("source", source),
			)
			continue
		}
		d.logger.Debug("event received",
			zap.String("source", source),
			zap.Int("id", e.ID),
			zap.String("description", e.EventDescription),
			zap.Time("time", e.EventTime),
		)
		newEvents = append(newEvents, e)
	}

	if len(newEvents) == 0 {
		return
	}

	// GetEvents returns events newest-first (orderBy=eventTime DESC). Apply them in
	// chronological order so the most recent event is set last and becomes the current
	// value exposed by GetLastAccessAttempt and the latest security event.
	slices.Reverse(newEvents)

	if err := d.linkEvents(ctx, newEvents); err != nil {
		d.logger.Error("failed to link events", zap.Error(err))
	}

	if d.eventsController != nil {
		if err := d.eventsController.processSecurityEvents(ctx, newEvents); err != nil {
			d.logger.Error("failed to process security events", zap.Error(err))
		}
	}
}
