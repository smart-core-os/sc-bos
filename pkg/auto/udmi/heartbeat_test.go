package udmi

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

const (
	testInterval = 4 * time.Hour
	eventTopic   = "client/site-01/HVAC/PICV-12345/events/pointset"
	legacyTopic  = "client/site-01/HVAC/PICV-12345/event/pointset/points"
	stateTopic   = "client/site-01/HVAC/PICV-12345/state"
)

// publication is one call to Publisher.Publish.
type publication struct {
	topic   string
	payload string
}

// testPublisher records what was published and can be made to fail.
type testPublisher struct {
	mu   sync.Mutex
	pubs []publication
	err  error
}

func (p *testPublisher) Publish(_ context.Context, topic string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	str, ok := payload.(string)
	if !ok {
		return errors.New("payload was not a string")
	}
	p.pubs = append(p.pubs, publication{topic: topic, payload: str})
	return nil
}

// take returns everything published since the last take.
func (p *testPublisher) take() []publication {
	p.mu.Lock()
	defer p.mu.Unlock()
	pubs := p.pubs
	p.pubs = nil
	return pubs
}

func (p *testPublisher) failWith(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}

// testGetter stands in for the source's UdmiServiceClient, answering
// GetExportMessage with whatever the test has set and counting the calls.
type testGetter struct {
	mu    sync.Mutex
	reply func() (*udmipb.MqttMessage, error)
	calls int
}

func (g *testGetter) GetExportMessage(_ context.Context, _ *udmipb.GetExportMessageRequest, _ ...grpc.CallOption) (*udmipb.MqttMessage, error) {
	g.mu.Lock()
	reply := g.reply
	g.calls++
	g.mu.Unlock()
	return reply()
}

// answerWith makes the source return the given message, as a driver that
// collected a reading on demand would.
func (g *testGetter) answerWith(topic, payload string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reply = func() (*udmipb.MqttMessage, error) {
		return &udmipb.MqttMessage{Topic: topic, Payload: payload}, nil
	}
}

func (g *testGetter) failWith(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reply = func() (*udmipb.MqttMessage, error) { return nil, err }
}

func (g *testGetter) callCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

// harness runs handleMessages inside a synctest bubble, feeding it messages and
// advancing the fake clock.
type harness struct {
	t         *testing.T
	changes   chan *udmipb.PullExportMessagesResponse
	pub       *testPublisher
	get       *testGetter
	collector *exportCollector
	hb        *heartbeat
	done      chan error
	cancel    context.CancelFunc
}

func newHarness(t *testing.T, interval time.Duration) *harness {
	t.Helper()
	get := &testGetter{}
	// By default the source is healthy and collects a fresh reading on demand,
	// stamped at the moment it was asked.
	get.reply = func() (*udmipb.MqttMessage, error) {
		return &udmipb.MqttMessage{Topic: eventTopic, Payload: pointsetAt(time.Now().UTC(), 21.5)}, nil
	}
	h := &harness{
		t:         t,
		changes:   make(chan *udmipb.PullExportMessagesResponse),
		pub:       &testPublisher{},
		get:       get,
		collector: newExportCollector(time.Now),
		hb:        newHeartbeat(interval, zap.NewNop()),
	}
	h.start()
	t.Cleanup(func() {
		h.cancel()
		synctest.Wait()
	})
	return h
}

// start runs handleMessages on its own goroutine, as the auto's task does.
// Each run gets its own done channel so a restart doesn't block on the previous
// run's result.
func (h *harness) start() {
	ctx, cancel := context.WithCancel(h.t.Context())
	h.cancel = cancel
	done := make(chan error, 1)
	h.done = done
	changes := h.changes
	go func() {
		done <- handleMessages(ctx, "test-device", h.get, changes, h.pub, h.collector, h.hb)
	}()
	synctest.Wait()
}

// send delivers a message on the export stream and lets the handler settle.
func (h *harness) send(topic, payload string) {
	h.t.Helper()
	h.changes <- &udmipb.PullExportMessagesResponse{
		Name:    "test-device",
		Message: &udmipb.MqttMessage{Topic: topic, Payload: payload},
	}
	synctest.Wait()
}

// advance moves the fake clock forward and lets the handler react.
func (h *harness) advance(d time.Duration) {
	h.t.Helper()
	time.Sleep(d)
	synctest.Wait()
}

// assertTopics checks the topics published since the last take, in order.
func (h *harness) assertTopics(want ...string) []publication {
	h.t.Helper()
	pubs := h.pub.take()
	if len(pubs) != len(want) {
		h.t.Fatalf("published %d messages, want %d: %v", len(pubs), len(want), pubs)
	}
	for i, w := range want {
		if pubs[i].topic != w {
			h.t.Errorf("publication %d was on %q, want %q", i, pubs[i].topic, w)
		}
	}
	return pubs
}

// pointsetAt builds an enveloped pointset event payload, as the drivers do.
func pointsetAt(ts time.Time, value float64) string {
	b, err := json.Marshal(PointsetEvent{
		Timestamp: ts,
		Version:   PointsetVersion,
		Points:    PointsEvent{"ZnTemp": PointValue{PresentValue: value}},
	})
	if err != nil {
		panic(err) // a fixed literal payload; marshalling it can't fail
	}
	return string(b)
}

// pointset builds a pointset event payload stamped now.
func pointset(value float64) string {
	return pointsetAt(time.Now().UTC(), value)
}

func TestHandleMessages_HeartbeatsWhenQuiet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		payload := pointset(21.5)

		h.send(eventTopic, payload)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != payload {
			t.Errorf("live publish altered the payload:\n got %s\nwant %s", pubs[0].payload, payload)
		}

		// Nothing yet just before the deadline, and the source hasn't been bothered.
		h.advance(testInterval - time.Minute)
		h.assertTopics()
		if got := h.get.callCount(); got != 0 {
			t.Errorf("GetExportMessage called %d times before the deadline, want 0", got)
		}

		// Once the interval has elapsed the source is asked for a current message.
		h.advance(2 * time.Minute)
		h.assertTopics(eventTopic)
		if got := h.get.callCount(); got != 1 {
			t.Errorf("GetExportMessage called %d times, want 1", got)
		}
	})
}

// The heartbeat publishes what the driver gave it, byte for byte: the auto no
// longer rewrites the timestamp, because the driver stamped it when it collected
// the reading.
func TestHandleMessages_PublishesTheDriversMessageVerbatim(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		collected := pointsetAt(time.Now().UTC().Add(-30*time.Second), 22.5)
		h.get.answerWith(eventTopic, collected)

		h.advance(testInterval)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != collected {
			t.Errorf("heartbeat payload is\n %s\nwant the driver's bytes unchanged:\n %s",
				pubs[0].payload, collected)
		}
	})
}

// The legacy bacnet topic carries a bare points map; it is still a pointset event
// and is passed through untouched.
func TestHandleMessages_LegacyTopicHeartbeats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		payload := `{"ZnTemp":{"present_value":21.5}}`
		h.get.answerWith(legacyTopic, payload)

		h.send(legacyTopic, payload)
		h.assertTopics(legacyTopic)

		h.advance(testInterval)
		pubs := h.assertTopics(legacyTopic)
		if pubs[0].payload != payload {
			t.Errorf("heartbeat payload is %s, want it unchanged: %s", pubs[0].payload, payload)
		}
	})
}

func TestHandleMessages_ActivityResetsTheDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		// A real change 3h in restarts the countdown.
		h.advance(3 * time.Hour)
		h.send(eventTopic, pointset(22))
		h.assertTopics(eventTopic)

		// So nothing at t=4h...
		h.advance(time.Hour + time.Minute)
		h.assertTopics()

		// ...but a heartbeat at t=7h.
		h.advance(3 * time.Hour)
		h.assertTopics(eventTopic)
	})
}

func TestHandleMessages_RepeatsWhileQuiet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(12 * time.Hour)
		h.assertTopics(eventTopic, eventTopic, eventTopic)
	})
}

// GetExportMessage is addressed by source, so the deadline is per source rather
// than per topic: any pointset traffic from the source means it is alive, and
// asking it again would only duplicate what it just sent.
func TestHandleMessages_DeadlineIsPerSource(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const otherTopic = "client/site-01/HVAC/AHU-1/events/pointset"
		h := newHarness(t, testInterval)

		h.send(eventTopic, pointset(21.5))
		h.send(otherTopic, pointset(1))
		h.assertTopics(eventTopic, otherTopic)

		// One of the source's topics keeps changing, so the source is never quiet.
		for range 5 {
			h.advance(time.Hour)
			h.send(otherTopic, pointset(2))
		}
		if got := h.get.callCount(); got != 0 {
			t.Errorf("GetExportMessage called %d times while the source was publishing, want 0", got)
		}
		h.pub.take()

		// Once it stops, the heartbeat resumes.
		h.advance(testInterval + time.Minute)
		if got := h.get.callCount(); got != 1 {
			t.Errorf("GetExportMessage called %d times after the source went quiet, want 1", got)
		}
	})
}

func TestHandleMessages_IgnoresNonPointsetTopics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		// State is published as normal but must not push the deadline out - the
		// bacnet driver re-announces it on every reconnect.
		h.advance(3 * time.Hour)
		h.send(stateTopic, `{"timestamp":"2026-01-01T00:00:00Z"}`)
		h.assertTopics(stateTopic)

		h.advance(time.Hour + time.Minute)
		h.assertTopics(eventTopic)
	})
}

// We asked for telemetry. A source that answers with state or metadata hasn't
// given us a reading, so nothing is published.
func TestHandleMessages_NonPointsetReplyIsNotPublished(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.get.answerWith(stateTopic, `{"timestamp":"2026-01-01T00:00:00Z"}`)
		h.advance(testInterval)
		h.assertTopics()
	})
}

// Unavailable is the source saying it has nothing current to report, which is the
// liveness signal: a dead device produces silence, just as it did before the
// heartbeat existed. The deadline still moves on, so we ask once per interval.
func TestHandleMessages_UnavailableSourcePublishesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.get.failWith(status.Error(codes.Unavailable, "device not operational"))
		h.advance(12 * time.Hour)
		h.assertTopics()
		if got := h.get.callCount(); got != 3 {
			t.Errorf("GetExportMessage called %d times over 12h, want 3 (one per interval, no hot loop)", got)
		}

		// The source recovering resumes heartbeats.
		h.get.answerWith(eventTopic, pointset(21.5))
		h.advance(testInterval)
		h.assertTopics(eventTopic)
	})
}

// A source that can't collect on demand is asked once and then left alone, rather
// than being retried every interval for the life of the automation.
func TestHandleMessages_UnimplementedDisablesTheHeartbeat(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.get.failWith(status.Error(codes.Unimplemented, "not implemented"))

		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(24 * time.Hour)
		h.assertTopics()
		if got := h.get.callCount(); got != 1 {
			t.Errorf("GetExportMessage called %d times over 24h, want 1", got)
		}

		// Later traffic doesn't re-arm it either.
		h.send(eventTopic, pointset(22))
		h.assertTopics(eventTopic)
		h.advance(24 * time.Hour)
		h.assertTopics()
		if got := h.get.callCount(); got != 1 {
			t.Errorf("GetExportMessage called %d times after disabling, want 1", got)
		}
	})
}

// A heartbeat message is a real reading, so it counts towards the points list
// export like any other.
func TestHandleMessages_HeartbeatIsCollected(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		collected := pointsetAt(time.Now().UTC(), 30)
		h.get.answerWith(eventTopic, collected)
		h.advance(testInterval)
		h.assertTopics(eventTopic)

		records := h.collector.Snapshot()
		if len(records) != 1 {
			t.Fatalf("collector holds %d records, want 1: %v", len(records), records)
		}
		if records[0].payload != collected {
			t.Errorf("collector holds\n %s\nwant the heartbeat payload\n %s", records[0].payload, collected)
		}
	})
}

func TestHandleMessages_DisabledByZeroInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, 0)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(8 * time.Hour)
		h.assertTopics()
		if got := h.get.callCount(); got != 0 {
			t.Errorf("GetExportMessage called %d times with the heartbeat disabled, want 0", got)
		}
	})
}

func TestHandleMessages_NoHeartbeatBeforeFirstMessage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.advance(8 * time.Hour)
		h.assertTopics()
		if got := h.get.callCount(); got != 0 {
			t.Errorf("GetExportMessage called %d times before any message, want 0", got)
		}
	})
}

func TestHandleMessages_HeartbeatPublishErrorIsNotFatal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.pub.failWith(errors.New("broker unavailable"))
		h.advance(testInterval)
		h.assertTopics()

		select {
		case err := <-h.done:
			t.Fatalf("handleMessages returned %v, want it still running", err)
		default:
		}

		// The handler is still live: the next heartbeat fires.
		h.pub.failWith(nil)
		h.advance(testInterval)
		h.assertTopics(eventTopic)
	})
}

// Any other error from the source is logged and skipped; a missed heartbeat isn't
// worth tearing down a working subscription for.
func TestHandleMessages_HeartbeatGetErrorIsNotFatal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.get.failWith(status.Error(codes.Internal, "boom"))
		h.advance(testInterval)
		h.assertTopics()

		select {
		case err := <-h.done:
			t.Fatalf("handleMessages returned %v, want it still running", err)
		default:
		}

		h.get.answerWith(eventTopic, pointset(21.5))
		h.advance(testInterval)
		h.assertTopics(eventTopic)
	})
}

func TestHandleMessages_LivePublishErrorReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		wantErr := errors.New("broker unavailable")
		h.pub.failWith(wantErr)

		h.send(eventTopic, pointset(21.5))
		select {
		case err := <-h.done:
			if !errors.Is(err, wantErr) {
				t.Errorf("handleMessages returned %v, want %v", err, wantErr)
			}
		default:
			t.Fatal("handleMessages still running, want it to return the publish error")
		}
	})
}

func TestHandleMessages_ClosedChangesReturnsNil(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		close(h.changes)
		synctest.Wait()

		select {
		case err := <-h.done:
			if err != nil {
				t.Errorf("handleMessages returned %v, want nil", err)
			}
		default:
			t.Fatal("handleMessages still running after changes closed")
		}
	})
}

func TestHandleMessages_CancelReturnsCtxErr(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.cancel()
		synctest.Wait()

		select {
		case err := <-h.done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("handleMessages returned %v, want context.Canceled", err)
			}
		default:
			t.Fatal("handleMessages still running after cancel")
		}
	})
}

// The heartbeat is built by tasksForSource, outside the retried task, so a task
// retry must resume the existing countdown rather than restart it.
func TestHandleMessages_DeadlineSurvivesRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		// The task dies and is retried 3h later with the same heartbeat.
		close(h.changes)
		synctest.Wait()
		h.advance(3 * time.Hour)
		h.changes = make(chan *udmipb.PullExportMessagesResponse)
		h.start()

		h.advance(time.Hour + time.Minute)
		h.assertTopics(eventTopic)
	})
}

func TestIsPointsetEventTopic(t *testing.T) {
	tests := map[string]bool{
		// both the spec topic and the legacy one carry pointset telemetry
		"client/site-01/HVAC/PICV-12345/events/pointset":       true,
		"client/site-01/HVAC/PICV-12345/event/pointset/points": true,
		"test/mock/dev-1/event/pointset/points":                true,
		// other event subfolders aren't telemetry samples
		"client/site-01/HVAC/PICV-12345/events/system":    false,
		"client/site-01/HVAC/PICV-12345/events/discovery": false,
		// state and metadata are retained, never heartbeated
		"client/site-01/HVAC/PICV-12345/state":         false,
		"client/site-01/HVAC/PICV-12345/metadata.json": false,
		// a "pointset" segment that isn't under an event folder
		"client/site-01/HVAC/PICV-12345/config/pointset": false,
		"": false,
	}
	for topic, want := range tests {
		if got := isPointsetEventTopic(topic); got != want {
			t.Errorf("isPointsetEventTopic(%q) = %v, want %v", topic, got, want)
		}
	}
}
