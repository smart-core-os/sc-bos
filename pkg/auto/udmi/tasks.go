package udmi

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"
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
// pace bounds how often pointset events are published; see cadence.
// Writes received over MQTT are recorded with auditor, if non-nil, attributed to brokerHost.
func tasksForSource(name string, logger *zap.Logger, client udmipb.UdmiServiceClient, pubsub *PubSub, collector *exportCollector, pace cadence, auditor auto.Auditor, brokerHost string) []task.Task {
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
// ultimately these end up getting sent as MQTT messages. Each message is also offered to
// the collector (when non-nil), which keeps the pointset events for the points list export.
// Between messages it runs the timers that pace publishing: once the source has been
// quiet for longer than the heartbeat interval hb has it asked, via GetExportMessage,
// for a current message to publish, and th releases a payload it held back to keep a
// chatty topic inside its minimum send interval.
func handleMessages(ctx context.Context, name string, client exportMessageGetter, changes <-chan *udmipb.PullExportMessagesResponse, publisher Publisher, collector *exportCollector, hb *heartbeat, th *throttle) error {
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
		// beat and release are nil, and so never ready, until there is something to wait
		// for: beat until the source has published a pointset event, release until th is
		// holding a payload. Recomputing them each iteration is what re-arms them, and
		// since hb's and th's deadlines are absolute a task retry resumes their countdowns
		// rather than restarting them.
		var beat, release <-chan time.Time
		now := time.Now()
		if d, ok := hb.wait(now); ok {
			beat = time.After(d)
		}
		if d, ok := th.wait(now); ok {
			release = time.After(d)
		}

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
			// We asked for telemetry; a state or metadata re-announce does not answer
			// the question a heartbeat asks, so do not pass it off as one. That leaves
			// this beat unanswered, so ask again shortly rather than going quiet for a
			// whole interval: a source is free to answer with its last published
			// message, which may be telemetry again by then.
			if msg == nil || !isPointsetEventTopic(msg.Topic) {
				topic := ""
				if msg != nil {
					topic = msg.Topic
				}
				hb.logger.Debug("heartbeat answered without telemetry",
					zap.String("topic", topic), zap.Bool("retrying", hb.retry(time.Now())))
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
		case now := <-release:
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
	}
}
