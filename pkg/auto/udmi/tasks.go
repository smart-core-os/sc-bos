package udmi

import (
	"context"

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
// all of these need to be run for the implementation to work
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub) []task.Task {
	var tasks []task.Task

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
			return handleMessages(ctx, messageChanges, pubsub.Publisher)
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
// ultimately these end up getting sent as MQTT messages
func handleMessages(ctx context.Context, changes <-chan *udmipb.PullExportMessagesResponse, publisher Publisher) error {
	for change := range changes {
		if change.Message == nil {
			continue
		}
		err := publisher.Publish(ctx, change.Message.Topic, change.Message.Payload)
		if err != nil {
			return err
		}
	}
	return nil
}
