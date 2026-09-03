package udmi

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// exportMessageGetter is the slice of udmipb.UdmiServiceClient the heartbeat needs.
type exportMessageGetter interface {
	GetExportMessage(context.Context, *udmipb.GetExportMessageRequest, ...grpc.CallOption) (*udmipb.MqttMessage, error)
}

// heartbeat tracks how long a source has been quiet, so the auto can ask it for
// a current reading when nothing has been published for a while.
//
// Sources only emit on change — the BACnet merge driver gates its send on
// !points.Equal(events), Steinel and Xovis pull with resource.WithUpdatesOnly —
// so a device whose readings never move sends one pointset and then nothing.
// Event topics are published unretained (see config.Root.Retained), so such a
// device simply vanishes from the broker and a consumer cannot tell "unchanged"
// from "dead".
//
// The heartbeat closes that gap without inventing data: on expiry the caller
// asks the source for a message via GetExportMessage, whose contract is to
// "collect data explicitly to return" and to answer Unavailable when it has
// nothing. So every heartbeat carries a reading the source actually took and
// stamped, and a source that cannot produce one stays silent — silence still
// means dead.
//
// Deadlines are tracked per source, not per topic, because GetExportMessage is
// addressed by source name: a source publishing two pointset topics would have
// one Get reset only the returned topic's deadline, leaving the other
// permanently overdue and the timer in a hot loop. A source is heartbeated as a
// unit.
//
// A heartbeat is owned by one source and only ever touched from that source's
// handleMessages goroutine, so it needs no locking. It is deliberately created
// outside the retried task (see tasksForSource) so that a publish error, which
// restarts the task, resumes the countdown rather than restarting it.
type heartbeat struct {
	interval time.Duration // <= 0 disables the heartbeat entirely
	logger   *zap.Logger

	// deadline is when the next heartbeat is due; zero until the source has
	// published its first pointset event.
	deadline time.Time
	// disabled is set when the source answers Unimplemented, so a driver that
	// can't collect a message on demand isn't asked again every interval.
	disabled bool
}

func newHeartbeat(interval time.Duration, logger *zap.Logger) *heartbeat {
	return &heartbeat{
		interval: interval,
		logger:   logger,
	}
}

func (h *heartbeat) enabled() bool {
	return h.interval > 0 && !h.disabled
}

// disable stops this source being heartbeated for the lifetime of the auto,
// used when it tells us it can't produce a message on demand.
func (h *heartbeat) disable() {
	h.disabled = true
}

// record notes that a message was published on topic at now, resetting the
// source's heartbeat deadline.
//
// Only pointset event topics count. State and metadata are published retained,
// so the broker already holds the latest, and — more importantly — the BACnet
// merge driver re-announces both on every stream reconnect, so letting them
// reset the deadline would suppress heartbeats indefinitely on a flapping link.
func (h *heartbeat) record(topic string, now time.Time) {
	if !h.enabled() || !isPointsetEventTopic(topic) {
		return
	}
	h.deadline = now.Add(h.interval)
}

// wait reports how long until the next heartbeat is due, clamped at zero for an
// overdue one. ok is false when there's nothing to wait for — the heartbeat is
// disabled, or no pointset event has been seen yet — and the caller should leave
// its timer unarmed.
//
// The deadline is absolute, so re-entering handleMessages after a task retry
// resumes the existing countdown rather than starting a fresh interval.
func (h *heartbeat) wait(now time.Time) (time.Duration, bool) {
	if !h.enabled() || h.deadline.IsZero() {
		return 0, false
	}
	return max(h.deadline.Sub(now), 0), true
}

// due reports whether the source has been quiet for at least the interval, and
// if so pushes the deadline out by another one. Advancing here rather than after
// a successful publish means a refusing broker, or a source with nothing to
// report, costs one attempt per interval instead of a hot loop.
func (h *heartbeat) due(now time.Time) bool {
	if !h.enabled() || h.deadline.IsZero() {
		return false
	}
	if now.Before(h.deadline) {
		// The timer is only armed off this same deadline, so this means it woke
		// early — benign, the caller re-arms — but worth a trace of.
		h.logger.Debug("heartbeat woke with nothing due")
		return false
	}
	h.deadline = now.Add(h.interval)
	return true
}

// isPointsetEventTopic reports whether topic is a UDMI pointset event topic —
// either the spec form ".../events/pointset" or the legacy form
// ".../event/pointset/points" (see pkg/driver/bacnet/merge/config). It is
// narrower than isEventTopic on purpose: other UDMI event subfolders such as
// events/system and events/discovery are not telemetry samples, so they neither
// reset the deadline nor satisfy a heartbeat.
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
