package udmi

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// cadence bounds how often the auto publishes pointset events: minSend is the
// floor between two publishes on the same topic, heartbeat the ceiling on how
// long a whole source may stay silent. Either is disabled by a value of zero or
// less.
type cadence struct {
	minSend   time.Duration
	heartbeat time.Duration
}

// tasksForSource returns an array of tasks to run for each UdmiService source/name
// all of these need to be run for the implementation to work.
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub, collector *exportCollector, pace cadence) []task.Task {
	var tasks []task.Task

	// Built out here, not inside the task below, so the deadlines and held payloads
	// survive the task being retried after a publish error: hb's countdown resumes
	// rather than restarting, and th keeps a payload it is holding. The source
	// dedupes, so anything we forget it will never send again.
	hb := newHeartbeat(pace.heartbeat, logger)
	th := newThrottle(pace.minSend)

	tasks = append(tasks, func(ctx context.Context) (task.Next, error) {
		logger.Debug("subscribing")
		topicChanges := make(chan *udmipb.PullControlTopicsResponse)
		grp, ctx := errgroup.WithContext(ctx)
		grp.Go(func() error {
			defer close(topicChanges)
			return pullTopics(ctx, name, logger, client, topicChanges)
		})
		grp.Go(func() error {
			return handleTopicChanges(ctx, name, logger, client, topicChanges, pubsub.Subscriber)
		})
		err := grp.Wait() // this waits for all go routines to finish, so we are safe to then close the channel
		return task.Normal, err
	})
	tasks = append(tasks, func(ctx context.Context) (task.Next, error) {
		messageChanges := make(chan *udmipb.PullExportMessagesResponse)
		grp, ctx := errgroup.WithContext(ctx)
		grp.Go(func() error {
			defer close(messageChanges)
			return pullMessages(ctx, name, logger, client, messageChanges)
		})
		grp.Go(func() error {
			return handleMessages(ctx, name, client, messageChanges, pubsub.Publisher, collector, hb, th)
		})
		err := grp.Wait() // this waits for all go routines to finish, so we are safe to then close the channel
		return task.Normal, err
	})

	return tasks
}

// pullTopics calls pull for control topics (with default backoff/delay) and sends each message on the given channel
func pullTopics(ctx context.Context, name string, logger *zap.Logger, client udmipb.UdmiServiceClient, changes chan<- *udmipb.PullControlTopicsResponse) error {
	puller := &udmiControlTopicsPuller{
		client: client,
		name:   name,
	}
	err := pull.Changes[*udmipb.PullControlTopicsResponse](ctx, puller, changes, pull.WithLogger(logger))
	if status.Code(err) == codes.Unimplemented {
		return nil
	}
	return err
}

// handleTopicChanges will wait for topic messages on the channel, and for each topic an MQTT subscription is created (via
// Subscriber). Messages received for each of those subscriptions is then passed onto the UdmiService using OnMessage.
func handleTopicChanges(ctx context.Context, name string, logger *zap.Logger, client udmipb.UdmiServiceClient, changes <-chan *udmipb.PullControlTopicsResponse, subscriber Subscriber) error {
	subscribeTopic := func(ctx context.Context, topic string) error {
		return subscriber.Subscribe(ctx, topic, func(_ mqtt.Client, message mqtt.Message) {
			payload := string(message.Payload())
			logger.Debug("received MQTT message", zap.String("topic", topic), zap.String("payload", payload))
			_, err := client.OnMessage(ctx, &udmipb.OnMessageRequest{
				Name: name,
				Message: &udmipb.MqttMessage{
					Topic:   message.Topic(),
					Payload: payload,
				},
			})
			if err != nil {
				logger.Warn("unable to call OnMessage", zap.Error(err))
			}
		})
	}

	current := func() {}
	defer func() {
		current()
	}()
	for change := range changes {
		current() // cancel previous subscriptions
		ctx, cancel := context.WithCancel(ctx)
		current = cancel
		// todo: work out topic changes, rather than just restart all
		for _, topic := range change.Topics {
			err := subscribeTopic(ctx, topic)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// pullMessages calls pull for export messages (with default backoff/delay) and sends each message on the given channel
func pullMessages(ctx context.Context, name string, logger *zap.Logger, client udmipb.UdmiServiceClient, changes chan<- *udmipb.PullExportMessagesResponse) error {
	puller := &udmiExportMessagePuller{
		client: client,
		name:   name,
	}
	err := pull.Changes[*udmipb.PullExportMessagesResponse](ctx, puller, changes, pull.WithLogger(logger))
	if status.Code(err) == codes.Unimplemented {
		return nil
	}
	return err
}

// handleMessages waits for messages on the given channel and sends them to the publisher
// ultimately these end up getting sent as MQTT messages. Each message is also offered to
// the collector (when non-nil), which keeps the pointset events for the points list export.
// Between messages it runs the timers that pace publishing: once the source has been
// quiet for longer than the heartbeat interval hb has it asked, via GetExportMessage,
// for a current message to publish, and th releases a payload it held back to keep a
// chatty topic inside its minimum send interval.
func handleMessages(ctx context.Context, name string, client exportMessageGetter, changes <-chan *udmipb.PullExportMessagesResponse, publisher Publisher, collector *exportCollector, hb *heartbeat, th *throttle) error {
	var beat, release deadlineTimer
	defer beat.stop()
	defer release.stop()
	armTimers := func() {
		now := time.Now()
		beat.arm(hb.wait(now))
		release.arm(th.wait(now))
	}
	// Arm before the first receive: on a task retry hb and th already hold deadlines
	// from the previous run, and they must keep running rather than restart.
	armTimers()

	// publish sends a sample the source handed us — straight through, after being held,
	// or in answer to a heartbeat — and records it against everything that tracks what
	// the broker has actually seen, so none of them count a failed write as a publish.
	// Every arm below publishes through here; only their error handling differs.
	publish := func(topic, payload string, now time.Time) error {
		if err := publisher.Publish(ctx, topic, payload); err != nil {
			return err
		}
		// Record only after a successful publish so the export reflects what was actually
		// sent to the broker rather than what we tried to send.
		if collector != nil {
			collector.Record(name, topic, payload)
		}
		hb.record(topic, now)
		th.sent(topic, now)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case change, ok := <-changes:
			if !ok {
				return nil
			}
			if change.Message == nil {
				continue
			}
			now := time.Now()
			topic, payload := change.Message.Topic, change.Message.Payload
			// A held payload is published by the release arm below once the topic's
			// interval expires, so nothing is recorded for it here: it hasn't been sent.
			if !th.hold(topic, payload, now) {
				if err := publish(topic, payload, now); err != nil {
					return err
				}
			}
		case now := <-beat.c:
			if !hb.due(now) {
				break
			}
			// Ask the source for a message rather than replaying one of our own: what
			// comes back is collected and stamped by the driver, so a heartbeat asserts
			// nothing about the device that the device didn't just say.
			msg, err := client.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{Name: name})
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				switch status.Code(err) {
				case codes.Unimplemented:
					// The source can't collect on demand, so stop asking it every interval.
					// pullTopics and pullMessages treat Unimplemented the same way.
					hb.logger.Debug("source does not implement GetExportMessage, heartbeat disabled")
					hb.disable()
				case codes.Unavailable:
					// The source has nothing current to say. Publishing nothing is the point:
					// a dead device produces silence.
					hb.logger.Debug("no heartbeat message available", zap.Error(err))
				default:
					// Unlike a live message, a missed heartbeat isn't worth tearing down a
					// working subscription for; the next one will retry.
					hb.logger.Warn("unable to collect heartbeat message", zap.Error(err))
				}
				break
			}
			// We asked for telemetry; a state or metadata re-announce doesn't answer the
			// question a heartbeat asks, so don't pass it off as one.
			if msg == nil || !isPointsetEventTopic(msg.Topic) {
				break
			}
			// The reply is a live sample like any other, so it goes through the floor:
			// a heartbeat can't breach minSendInterval, and arriving while a payload is
			// held it supersedes it, being the fresher of the two.
			if !th.hold(msg.Topic, msg.Payload, now) {
				if err := publish(msg.Topic, msg.Payload, now); err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					// Unlike a live message, a missed heartbeat isn't worth tearing down a
					// working subscription for; the next one will retry.
					hb.logger.Warn("unable to publish heartbeat",
						zap.String("topic", msg.Topic), zap.Error(err))
				}
			}
		case now := <-release.c:
			// A released payload is a live sample the source has already handed us, so a
			// failed publish is treated exactly as one arriving on changes would be: tear
			// the task down and retry. th.sent is only called on success, so the payload
			// stays held for the next run rather than being lost here.
			for _, msg := range th.due(now) {
				if err := publish(msg.topic, msg.payload, now); err != nil {
					return err
				}
			}
		}
		armTimers()
	}
}

// deadlineTimer is a timer whose select arm disappears when there's nothing to
// wait for. A nil channel is never ready to receive, so leaving c nil disables
// the arm rather than needing a flag at every use.
type deadlineTimer struct {
	timer *time.Timer
	c     <-chan time.Time
}

// arm sets the timer to fire in d, or disarms it when ok is false. The arguments
// are the results of a heartbeat.wait / throttle.wait call.
func (t *deadlineTimer) arm(d time.Duration, ok bool) {
	if !ok {
		t.stop()
		return
	}
	if t.timer == nil {
		t.timer = time.NewTimer(d)
		t.c = t.timer.C
		return
	}
	t.timer.Reset(d) // go1.23+ timers never deliver a stale tick, so no drain needed
}

func (t *deadlineTimer) stop() {
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
		t.c = nil
	}
}
