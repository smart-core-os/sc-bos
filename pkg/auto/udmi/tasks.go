package udmi

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

// tasksForSource returns an array of tasks to run for each UdmiService source/name
// all of these need to be run for the implementation to work.
// Writes received over MQTT are recorded with auditor, if non-nil, attributed to brokerHost.
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub, auditor auto.Auditor, brokerHost string) []task.Task {
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
			return handleTopicChanges(ctx, name, logger, client, topicChanges, pubsub.Subscriber, auditor, brokerHost)
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
//
// This is the only production caller of OnMessage, so it is where MQTT-originated writes are audited: they never pass
// through the gRPC server interceptors that audit writes arriving over the network. auditor may be nil.
func handleTopicChanges(ctx context.Context, name string, logger *zap.Logger, client udmipb.UdmiServiceClient, changes <-chan *udmipb.PullControlTopicsResponse, subscriber Subscriber, auditor auto.Auditor, brokerHost string) error {
	peer := redactBrokerURL(brokerHost)
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
			if auditor != nil {
				auditor.AuditIngress(onMessageAuditEntry(peer, name, message.Topic(), payload, err))
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

// maxAuditPayload caps how much of an MQTT payload is copied into its audit entry.
const maxAuditPayload = 1024

// onMessageAuditEntry describes an MQTT message that was dispatched to UdmiService.OnMessage for target, with err
// being the result. The broker supplies no identity for the publisher, so none is recorded.
func onMessageAuditEntry(peer, target, topic, payload string, err error) policy.IngressEntry {
	// Audit entries are served over the LogApi, where invalid UTF-8 would fail to marshal.
	auditPayload, truncated := truncatePayload(strings.ToValidUTF8(payload, string(utf8.RuneError)), maxAuditPayload)
	fields := map[string]string{
		// "name" is taken by the token name, so the device is recorded as the target.
		"target":      target,
		"topic":       topic,
		"payload":     auditPayload,
		"payloadSize": strconv.Itoa(len(payload)),
	}
	if truncated {
		fields["payloadTruncated"] = "true"
	}
	return policy.IngressEntry{
		Ingress: "mqtt",
		Peer:    peer,
		Service: udmipb.UdmiService_ServiceDesc.ServiceName,
		Method:  "OnMessage",
		Err:     err,
		Fields:  fields,
	}
}

// truncatePayload returns s cut to at most limit bytes without splitting a UTF-8 rune, and whether it was cut.
func truncatePayload(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// redactBrokerURL masks any password embedded in the broker URL so it isn't written to the audit log.
func redactBrokerURL(host string) string {
	u, err := url.Parse(host)
	if err != nil || u.User == nil {
		return host
	}
	return u.Redacted()
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
