package udmi

import (
	"time"
)

// throttle enforces a minimum interval between publishes on the same pointset
// event topic, so a device whose readings change on every poll doesn't publish on
// every poll.
//
// Sources emit on change — the BACnet merge driver gates its send on
// !points.Equal(events), Steinel and Xovis pull with resource.WithUpdatesOnly —
// and that comparison is bit-exact, so a value that never settles is a change
// every time it is read. At the BACnet driver's default 10s poll period that is
// one publish per device per 10s indefinitely, and because a pointset event
// carries the whole device, one restless point republishes every point on it.
//
// A change arriving inside the interval isn't dropped, it's held. Only the newest
// held payload is kept, and it goes out as soon as the interval expires, so
// consumers see the current value at a bounded rate rather than a decimated
// sample of the changes. Intermediate values are lost by design; that is what a
// rate limit is. Note this is a floor on publishing, not a change-of-value
// deadband: it bounds how often a value may be reported, not how far it must move
// to be worth reporting.
//
// Only pointset event topics are throttled — the same class of topic heartbeat
// tracks, though it does so for the source as a whole rather than per topic.
// State and metadata publish on an operational transition or a stream reconnect
// and are retained, so there is no volume to limit there.
//
// Like heartbeat, a throttle is owned by one source and only ever touched from
// that source's handleMessages goroutine, so it needs no locking, and it is
// created outside the retried task (see tasksForSource) so that a publish error,
// which restarts the task, doesn't discard a held payload the source will never
// resend.
type throttle struct {
	interval time.Duration            // <= 0 disables the throttle entirely
	byTopic  map[string]throttleEntry // pointset event topic -> when it last published, and what waits on it
}

// throttleEntry is a topic's publish floor and the payload waiting on it.
type throttleEntry struct {
	sentAt time.Time // when this topic last published
	// pending is the newest payload held back, meaningful only when held is set.
	// The flag is separate rather than inferred from a non-empty payload so an
	// empty payload can't be mistaken for nothing being held.
	pending string
	held    bool
}

// throttleMessage is a held payload that is now due to be published.
type throttleMessage struct {
	topic   string
	payload string
}

func newThrottle(interval time.Duration) *throttle {
	return &throttle{interval: interval, byTopic: make(map[string]throttleEntry)}
}

func (t *throttle) enabled() bool {
	return t.interval > 0
}

// hold reports whether payload must wait before being published on topic, storing
// it as that topic's pending payload when so. A false return means the caller
// should publish now and tell the throttle it did, via sent.
//
// The first message on a topic always publishes immediately: a consumer shouldn't
// have to wait out an interval for a value that has never been sent.
func (t *throttle) hold(topic, payload string, now time.Time) bool {
	if !t.enabled() || !isPointsetEventTopic(topic) {
		return false
	}
	entry, seen := t.byTopic[topic]
	if !seen || now.Sub(entry.sentAt) >= t.interval {
		return false
	}
	entry.pending, entry.held = payload, true
	t.byTopic[topic] = entry
	return true
}

// sent records that topic published at now, starting a fresh interval and
// clearing any payload that was held for it.
//
// Called only after a successful publish, so a broker that is refusing writes
// leaves the held payload in place for the next attempt rather than losing it.
func (t *throttle) sent(topic string, now time.Time) {
	if !t.enabled() || !isPointsetEventTopic(topic) {
		return
	}
	t.byTopic[topic] = throttleEntry{sentAt: now}
}

// wait reports how long until a held payload is due, clamped at zero for one
// already overdue. ok is false when there's nothing to wait for — the throttle is
// disabled, or no topic is holding — and the caller should leave its timer
// unarmed.
//
// Deadlines are absolute, relative to when each topic last published, so
// re-entering handleMessages after a task retry resumes the existing countdown
// rather than starting a fresh interval.
func (t *throttle) wait(now time.Time) (time.Duration, bool) {
	if !t.enabled() {
		return 0, false
	}
	var next time.Time
	for _, entry := range t.byTopic {
		if !entry.held {
			continue
		}
		if deadline := entry.sentAt.Add(t.interval); next.IsZero() || deadline.Before(next) {
			next = deadline
		}
	}
	if next.IsZero() {
		return 0, false
	}
	return max(next.Sub(now), 0), true
}

// due returns the held payloads whose intervals have expired. Nothing is marked
// here: the caller reports each publish through sent, so a failed publish leaves
// the payload held and due again.
func (t *throttle) due(now time.Time) []throttleMessage {
	if !t.enabled() {
		return nil
	}
	var msgs []throttleMessage
	for topic, entry := range t.byTopic {
		if !entry.held || now.Sub(entry.sentAt) < t.interval {
			continue
		}
		msgs = append(msgs, throttleMessage{topic: topic, payload: entry.pending})
	}
	return msgs
}
