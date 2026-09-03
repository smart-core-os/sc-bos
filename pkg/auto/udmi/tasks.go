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

// tasksForSource returns an array of tasks to run for each UdmiService source/name
// all of these need to be run for the implementation to work.
// hbInterval is how long a source may stay quiet before it is asked for a
// current message to publish; zero or less disables that.
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub, collector *exportCollector, hbInterval time.Duration) []task.Task {
	var tasks []task.Task

	// Built out here, not inside the task below, so the quiet-since deadline
	// survives the task being retried after a publish error, and a retry resumes
	// the countdown rather than restarting it.
	hb := newHeartbeat(hbInterval, logger)

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
			return handleMessages(ctx, name, client, messageChanges, pubsub.Publisher, collector, hb)
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
// Between messages it runs hb's timer: once the source has been quiet for longer than the
// heartbeat interval it is asked, via GetExportMessage, for a current message to publish.
func handleMessages(ctx context.Context, name string, client exportMessageGetter, changes <-chan *udmipb.PullExportMessagesResponse, publisher Publisher, collector *exportCollector, hb *heartbeat) error {
	// The nil channel is what disables the heartbeat arm of the select: until a
	// pointset event has been seen there's nothing to keep alive, and beat is never ready.
	var timer *time.Timer
	var beat <-chan time.Time
	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			beat = nil
		}
	}
	defer stopTimer()
	armTimer := func() {
		d, ok := hb.wait(time.Now())
		if !ok {
			stopTimer()
			return
		}
		if timer == nil {
			timer = time.NewTimer(d)
			beat = timer.C
			return
		}
		timer.Reset(d) // go1.23+ timers never deliver a stale tick, so no drain needed
	}
	// Arm before the first receive: on a task retry hb already holds a deadline from
	// the previous run, and it must keep running rather than restart.
	armTimer()

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
			err := publisher.Publish(ctx, change.Message.Topic, change.Message.Payload)
			if err != nil {
				return err
			}
			// Record only after a successful publish so the export reflects what was actually
			// sent to the broker rather than what we tried to send.
			if collector != nil {
				collector.Record(name, change.Message.Topic, change.Message.Payload)
			}
			hb.record(change.Message.Topic, time.Now())
		case now := <-beat:
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
			if err := publisher.Publish(ctx, msg.Topic, msg.Payload); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				hb.logger.Warn("unable to publish heartbeat",
					zap.String("topic", msg.Topic), zap.Error(err))
				break
			}
			if collector != nil {
				collector.Record(name, msg.Topic, msg.Payload)
			}
			hb.record(msg.Topic, time.Now())
		}
		armTimer()
	}
}
