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

// harness runs handleMessages inside a synctest bubble, feeding it messages and
// advancing the fake clock.
type harness struct {
	t       *testing.T
	changes chan *udmipb.PullExportMessagesResponse
	pub     *testPublisher
	hb      *heartbeat
	done    chan error
	cancel  context.CancelFunc
}

func newHarness(t *testing.T, interval time.Duration) *harness {
	t.Helper()
	h := &harness{
		t:       t,
		changes: make(chan *udmipb.PullExportMessagesResponse),
		pub:     &testPublisher{},
		hb:      newHeartbeat(interval, zap.NewNop()),
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
		// nil collector: these tests are about the heartbeat, not the points list export.
		done <- handleMessages(ctx, "test-device", changes, h.pub, nil, h.hb)
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

// pointset builds an enveloped pointset event payload, as the drivers do.
func pointset(t *testing.T, value float64) string {
	t.Helper()
	b, err := json.Marshal(PointsetEvent{
		Timestamp: time.Now().UTC(),
		Version:   PointsetVersion,
		Points:    PointsEvent{"ZnTemp": PointValue{PresentValue: value}},
	})
	if err != nil {
		t.Fatalf("marshal pointset: %v", err)
	}
	return string(b)
}

func TestHandleMessages_RepublishesWhenQuiet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		payload := pointset(t, 21.5)

		h.send(eventTopic, payload)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != payload {
			t.Errorf("live publish altered the payload:\n got %s\nwant %s", pubs[0].payload, payload)
		}

		// Nothing yet just before the deadline.
		h.advance(testInterval - time.Minute)
		h.assertTopics()

		// Republished once the interval has elapsed.
		h.advance(2 * time.Minute)
		pubs = h.assertTopics(eventTopic)
		assertSamePoints(t, payload, pubs[0].payload)
	})
}

func TestHandleMessages_RepublishRefreshesTimestamp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		h.advance(testInterval)
		pubs := h.assertTopics(eventTopic)

		var got PointsetEvent
		if err := json.Unmarshal([]byte(pubs[0].payload), &got); err != nil {
			t.Fatalf("unmarshal heartbeat: %v", err)
		}
		if !got.Timestamp.Equal(time.Now()) {
			t.Errorf("heartbeat timestamp is %v, want now (%v)", got.Timestamp, time.Now())
		}
		if got.Version != PointsetVersion {
			t.Errorf("heartbeat version is %q, want %q", got.Version, PointsetVersion)
		}
	})
}

func TestHandleMessages_ActivityResetsTheDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		// A real change 3h in restarts the countdown.
		h.advance(3 * time.Hour)
		h.send(eventTopic, pointset(t, 22))
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
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		h.advance(12 * time.Hour)
		h.assertTopics(eventTopic, eventTopic, eventTopic)
	})
}

func TestHandleMessages_DeadlinesArePerTopic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const busyTopic = "client/site-01/HVAC/AHU-1/events/pointset"
		h := newHarness(t, testInterval)

		h.send(eventTopic, pointset(t, 21.5))
		h.send(busyTopic, pointset(t, 1))
		h.assertTopics(eventTopic, busyTopic)

		// The busy topic keeps changing; the quiet one must still get a heartbeat.
		for range 5 {
			h.advance(time.Hour)
			h.send(busyTopic, pointset(t, 2))
		}
		pubs := h.pub.take()
		var quiet int
		for _, p := range pubs {
			if p.topic == eventTopic {
				quiet++
			}
		}
		if quiet != 1 {
			t.Errorf("quiet topic published %d times over 5h, want 1 heartbeat", quiet)
		}
	})
}

func TestHandleMessages_RepublishesTheLatestPayload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(t, 21.5))
		latest := pointset(t, 30)
		h.send(eventTopic, latest)
		h.assertTopics(eventTopic, eventTopic)

		h.advance(testInterval)
		pubs := h.assertTopics(eventTopic)
		assertSamePoints(t, latest, pubs[0].payload)
	})
}

func TestHandleMessages_IgnoresNonPointsetTopics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		// State is published as normal but must neither be cached nor push the
		// deadline out - the bacnet driver re-announces it on every reconnect.
		h.advance(3 * time.Hour)
		h.send(stateTopic, `{"timestamp":"2026-01-01T00:00:00Z"}`)
		h.assertTopics(stateTopic)

		h.advance(time.Hour + time.Minute)
		h.assertTopics(eventTopic)
	})
}

// The bacnet driver's default topic suffix is the legacy one, whose payload is a
// bare points map with no envelope to restamp.
func TestHandleMessages_LegacyTopicRepublishedVerbatim(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		payload := `{"ZnTemp":{"present_value":21.5}}`

		h.send(legacyTopic, payload)
		h.assertTopics(legacyTopic)

		h.advance(testInterval)
		pubs := h.assertTopics(legacyTopic)
		if pubs[0].payload != payload {
			t.Errorf("heartbeat payload is %s, want it unchanged: %s", pubs[0].payload, payload)
		}
	})
}

func TestHandleMessages_DisabledByZeroInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, 0)
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		h.advance(8 * time.Hour)
		h.assertTopics()
	})
}

func TestHandleMessages_NoHeartbeatBeforeFirstMessage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.advance(8 * time.Hour)
		h.assertTopics()
	})
}

func TestHandleMessages_HeartbeatPublishErrorIsNotFatal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		h.send(eventTopic, pointset(t, 21.5))
		h.assertTopics(eventTopic)

		h.pub.failWith(errors.New("broker unavailable"))
		h.advance(testInterval)
		h.assertTopics() // the failed heartbeat recorded nothing

		select {
		case err := <-h.done:
			t.Fatalf("handleMessages returned %v, want it still running", err)
		default:
		}

		// The handler is still live: messages flow, and the next heartbeat fires.
		h.pub.failWith(nil)
		h.advance(testInterval)
		h.assertTopics(eventTopic)
	})
}

func TestHandleMessages_LivePublishErrorReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, testInterval)
		wantErr := errors.New("broker unavailable")
		h.pub.failWith(wantErr)

		h.send(eventTopic, pointset(t, 21.5))
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
		h.send(eventTopic, pointset(t, 21.5))
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

// assertSamePoints checks that a heartbeat carries the same points as the message
// it replays, ignoring the refreshed timestamp.
func assertSamePoints(t *testing.T, want, got string) {
	t.Helper()
	var wantEvent, gotEvent PointsetEvent
	if err := json.Unmarshal([]byte(want), &wantEvent); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal([]byte(got), &gotEvent); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if !wantEvent.Points.Equal(gotEvent.Points) {
		t.Errorf("heartbeat points are %v, want %v", gotEvent.Points, wantEvent.Points)
	}
}

func TestRestamp(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	nowJSON, err := json.Marshal(now)
	if err != nil {
		t.Fatalf("marshal now: %v", err)
	}

	tests := map[string]struct {
		payload string
		want    string
	}{
		"envelope is restamped, siblings preserved": {
			payload: `{"timestamp":"2020-01-01T00:00:00Z","version":"1.5.2","partial_update":true,"points":{"ZnTemp":{"present_value":21.5}}}`,
			want:    `{"partial_update":true,"points":{"ZnTemp":{"present_value":21.5}},"timestamp":` + string(nowJSON) + `,"version":"1.5.2"}`,
		},
		"legacy bare points map is untouched": {
			payload: `{"ZnTemp":{"present_value":21.5}}`,
			want:    `{"ZnTemp":{"present_value":21.5}}`,
		},
		// A bare points map may hold a point actually called "timestamp"; requiring
		// a sibling "points" key is what stops us corrupting it.
		"bare points map with a timestamp point is untouched": {
			payload: `{"timestamp":{"present_value":"2020-01-01T00:00:00Z"}}`,
			want:    `{"timestamp":{"present_value":"2020-01-01T00:00:00Z"}}`,
		},
		"envelope without a timestamp is untouched": {
			payload: `{"version":"1.5.2","points":{"ZnTemp":{"present_value":21.5}}}`,
			want:    `{"version":"1.5.2","points":{"ZnTemp":{"present_value":21.5}}}`,
		},
		"malformed json is untouched": {
			payload: `not json`,
			want:    `not json`,
		},
		"json array is untouched": {
			payload: `[1,2,3]`,
			want:    `[1,2,3]`,
		},
		"empty payload is untouched": {
			payload: ``,
			want:    ``,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := restamp(tt.payload, now); got != tt.want {
				t.Errorf("restamp(%q):\n got %s\nwant %s", tt.payload, got, tt.want)
			}
		})
	}
}

// restamp must produce the same timestamp encoding the drivers do, so consumers
// can't tell a heartbeat from a live sample by its formatting.
func TestRestamp_MatchesDriverEncoding(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 123456789, time.UTC)
	live, err := json.Marshal(PointsetEvent{
		Timestamp: now,
		Version:   PointsetVersion,
		Points:    PointsEvent{"ZnTemp": PointValue{PresentValue: 21.5}},
	})
	if err != nil {
		t.Fatalf("marshal live event: %v", err)
	}

	stale := `{"timestamp":"2020-01-01T00:00:00Z","version":"` + PointsetVersion +
		`","points":{"ZnTemp":{"present_value":21.5}}}`
	var got, want PointsetEvent
	if err := json.Unmarshal([]byte(restamp(stale, now)), &got); err != nil {
		t.Fatalf("unmarshal restamped: %v", err)
	}
	if err := json.Unmarshal(live, &want); err != nil {
		t.Fatalf("unmarshal live: %v", err)
	}
	if !got.Timestamp.Equal(want.Timestamp) {
		t.Errorf("restamped timestamp is %v, want %v", got.Timestamp, want.Timestamp)
	}
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
