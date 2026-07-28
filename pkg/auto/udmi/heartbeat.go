package udmi

import (
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"
)

// heartbeat remembers the last pointset event published for each of a source's
// topics so the auto can republish it when the source goes quiet.
//
// Sources only emit on change — the BACnet merge driver gates its send on
// !points.Equal(events), Steinel and Xovis pull with resource.WithUpdatesOnly —
// so a device whose readings never move sends one pointset and then nothing.
// Event topics are published unretained (see config.Root.Retained), so such a
// device simply vanishes from the broker and a consumer cannot tell "unchanged"
// from "dead". Republishing the last message on a timer closes that gap.
//
// A heartbeat is owned by one source and only ever touched from that source's
// handleMessages goroutine, so it needs no locking. It is deliberately created
// outside the retried task (see tasksForSource) so that a publish error, which
// restarts the task, doesn't discard a payload the source will never resend.
//
// Note that Steinel and Xovis emit partial pointsets (partial_update: true,
// carrying only the points of the trait that changed). The auto can replay only
// what it last saw, so for those drivers a heartbeat repeats that partial set
// rather than a full snapshot. The payload says as much, so consumers that
// honour partial_update are unaffected.
type heartbeat struct {
	interval time.Duration // <= 0 disables the heartbeat entirely
	logger   *zap.Logger
	last     map[string]heartbeatEntry // pointset event topic -> what was published, and when
}

// heartbeatEntry is the last message published on a topic.
type heartbeatEntry struct {
	payload string
	sentAt  time.Time
}

// heartbeatMessage is a message that is due to be republished.
type heartbeatMessage struct {
	topic   string
	payload string
}

func newHeartbeat(interval time.Duration, logger *zap.Logger) *heartbeat {
	return &heartbeat{
		interval: interval,
		logger:   logger,
		last:     make(map[string]heartbeatEntry),
	}
}

func (h *heartbeat) enabled() bool {
	return h.interval > 0
}

// record notes that msg was published on topic at now, resetting that topic's
// heartbeat deadline.
//
// Only pointset event topics are tracked. State and metadata are published
// retained, so the broker already holds the latest, and — more importantly —
// the BACnet merge driver re-announces both on every stream reconnect, so
// letting them reset the deadline would suppress heartbeats indefinitely on a
// flapping link.
func (h *heartbeat) record(topic, payload string, now time.Time) {
	if !h.enabled() || !isPointsetEventTopic(topic) {
		return
	}
	h.last[topic] = heartbeatEntry{payload: payload, sentAt: now}
}

// wait reports how long until the next heartbeat is due, clamped at zero for an
// overdue one. ok is false when there's nothing to wait for — the heartbeat is
// disabled, or no pointset event has been seen yet — and the caller should leave
// its timer unarmed.
//
// Deadlines are absolute, relative to when each topic last published, so
// re-entering handleMessages after a task retry resumes the existing countdown
// rather than starting a fresh interval.
func (h *heartbeat) wait(now time.Time) (time.Duration, bool) {
	if !h.enabled() || len(h.last) == 0 {
		return 0, false
	}
	var next time.Time
	for _, entry := range h.last {
		if deadline := entry.sentAt.Add(h.interval); next.IsZero() || deadline.Before(next) {
			next = deadline
		}
	}
	return max(next.Sub(now), 0), true
}

// due returns the messages whose topics have been quiet for at least the
// interval, and marks them as published at now. Marking here rather than after a
// successful publish means a broker that is refusing writes costs one attempt
// per interval instead of a hot loop.
func (h *heartbeat) due(now time.Time) []heartbeatMessage {
	if !h.enabled() {
		return nil
	}
	var msgs []heartbeatMessage
	for topic, entry := range h.last {
		if now.Sub(entry.sentAt) < h.interval {
			continue
		}
		msgs = append(msgs, heartbeatMessage{topic: topic, payload: entry.payload})
		entry.sentAt = now
		h.last[topic] = entry
	}
	if len(msgs) == 0 {
		// The timer is only armed off these same deadlines, so an empty result means
		// it woke early — benign, the caller re-arms — but worth a trace of.
		h.logger.Debug("heartbeat woke with nothing due")
	}
	return msgs
}

// restamp returns payload with the UDMI envelope timestamp replaced by now, so a
// republished sample is recorded as observed now rather than being dropped or
// bucketed into the past by ingest.
//
// Only a payload carrying both "timestamp" and "points" is treated as an
// envelope. Requiring "points" means the legacy bare points map — which is a
// map of point name to value, and could legitimately hold a point called
// "timestamp" — is never rewritten; it has no envelope timestamp, so ingest
// stamps it on receipt, which is the behaviour we want anyway. Anything that
// doesn't parse as a JSON object is returned untouched.
func restamp(payload string, now time.Time) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return payload
	}
	if _, ok := fields["timestamp"]; !ok {
		return payload
	}
	if _, ok := fields["points"]; !ok {
		return payload
	}
	// json.RawMessage round-trips the fields we don't know about (version,
	// partial_update, and anything a future driver adds) unchanged.
	ts, err := json.Marshal(now.UTC())
	if err != nil {
		return payload
	}
	fields["timestamp"] = ts
	restamped, err := json.Marshal(fields)
	if err != nil {
		return payload
	}
	return string(restamped)
}

// isPointsetEventTopic reports whether topic is a UDMI pointset event topic —
// either the spec form ".../events/pointset" or the legacy form
// ".../event/pointset/points" (see pkg/driver/bacnet/merge/config). It is
// narrower than isEventTopic on purpose: other UDMI event subfolders such as
// events/system and events/discovery are not telemetry samples and shouldn't be
// replayed.
func isPointsetEventTopic(topic string) bool {
	segments := strings.Split(topic, "/")
	for i, segment := range segments {
		if segment != "event" && segment != "events" {
			continue
		}
		if i+1 < len(segments) && segments[i+1] == "pointset" {
			return true
		}
	}
	return false
}
