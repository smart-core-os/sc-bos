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
// hbInterval is how long a pointset event topic may stay quiet before its last
// message is republished; zero or less disables that.
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub, collector *exportCollector, hbInterval time.Duration) []task.Task {
	var tasks []task.Task

	// Built out here, not inside the task below, so the cached payloads and their
	// deadlines survive the task being retried after a publish error. The source
	// dedupes, so anything we forget it will never send again.
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
			return handleMessages(ctx, name, messageChanges, pubsub.Publisher, collector, hb)
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
// Between messages it runs hb's timer, republishing the last pointset event for any
// topic that has been quiet for longer than the heartbeat interval.
func handleMessages(ctx context.Context, name string, changes <-chan *udmipb.PullExportMessagesResponse, publisher Publisher, collector *exportCollector, hb *heartbeat) error {
	// The nil channel is what disables the heartbeat arm of the select: until a
	// pointset event has been seen there's nothing to replay, and beat is never ready.
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
	// Arm before the first receive: on a task retry hb already holds deadlines from
	// the previous run, and they must keep running rather than restart.
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
			hb.record(change.Message.Topic, change.Message.Payload, time.Now())
		case now := <-beat:
			// Heartbeats aren't offered to the collector. hb only ever replays a pointset
			// event topic it has already cached, which is the same class of topic the
			// collector records, so by now it already holds this topic's points; a replay
			// carries the same points and would only restamp them.
			for _, msg := range hb.due(now) {
				err := publisher.Publish(ctx, msg.topic, restamp(msg.payload, now))
				if err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					// Unlike a live message, a missed heartbeat isn't worth tearing down a
					// working subscription (and its cache) for; the next one will retry.
					hb.logger.Warn("unable to publish heartbeat",
						zap.String("topic", msg.topic), zap.Error(err))
				}
			}
		}
		armTimer()
	}
}
