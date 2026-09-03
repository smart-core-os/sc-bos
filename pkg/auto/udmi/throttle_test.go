package udmi

import (
	"encoding/json"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// testMinSend is the minimum send interval used throughout, matching the 5m floor
// a chatty BACnet device is configured with in the field.
const testMinSend = 5 * time.Minute

// heartbeatOff disables the heartbeat so a test isolates throttling.
const heartbeatOff = 0

func TestThrottle_FirstMessagePublishesImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		payload := pointset(21.5)

		h.send(eventTopic, payload)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != payload {
			t.Errorf("first publish altered the payload:\n got %s\nwant %s", pubs[0].payload, payload)
		}
	})
}

// The case the setting exists for: a device reporting a change on every 10s poll
// should publish once per interval, carrying the newest value.
func TestThrottle_ChattyDeviceSendsOncePerInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)

		h.send(eventTopic, pointset(21.5)) // the first is always live
		h.assertTopics(eventTopic)

		// 29 further changes, one every 10s, are all held.
		var last string
		for i := range 29 {
			h.advance(10 * time.Second)
			last = pointset(21.5+float64(i+1)/10)
			h.send(eventTopic, last)
			h.assertTopics()
		}

		// At the 5m mark the newest of them goes out, and only it.
		h.advance(10 * time.Second)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != last {
			t.Errorf("released the wrong payload:\n got %s\nwant %s", pubs[0].payload, last)
		}
	})
}

// A held payload is published as observed, not restamped: the reading was taken
// when the source reported it, which may be most of an interval ago.
func TestThrottle_ReleasedPayloadKeepsItsTimestamp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(time.Minute)
		observedAt := time.Now()
		h.send(eventTopic, pointset(22))
		h.assertTopics()

		h.advance(4 * time.Minute)
		pubs := h.assertTopics(eventTopic)
		var got PointsetEvent
		if err := json.Unmarshal([]byte(pubs[0].payload), &got); err != nil {
			t.Fatalf("unmarshal release: %v", err)
		}
		if !got.Timestamp.Equal(observedAt) {
			t.Errorf("released timestamp is %v, want the observation time %v", got.Timestamp, observedAt)
		}
	})
}

// A change arriving after the interval has already elapsed isn't delayed: the
// floor limits how often we publish, it doesn't batch onto a fixed grid.
func TestThrottle_ChangeAfterIntervalIsLive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(testMinSend + time.Second)
		h.send(eventTopic, pointset(22))
		h.assertTopics(eventTopic)

		// ...and that publish starts a fresh interval.
		h.advance(time.Minute)
		h.send(eventTopic, pointset(22.5))
		h.assertTopics()
	})
}

// A release starts a fresh interval, so a device changing constantly settles into
// one publish per interval rather than two in quick succession.
func TestThrottle_ReleaseStartsAFreshInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(time.Minute)
		h.send(eventTopic, pointset(22))
		h.advance(4 * time.Minute)
		h.assertTopics(eventTopic) // released at t=5m

		h.advance(time.Minute)
		h.send(eventTopic, pointset(22.5))
		h.advance(3 * time.Minute)
		h.assertTopics() // t=9m, still inside the interval that began at t=5m

		h.advance(time.Minute)
		h.assertTopics(eventTopic) // t=10m
	})
}

func TestThrottle_IntervalIsPerTopic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const otherTopic = "client/site-01/HVAC/PICV-99999/events/pointset"
		h := newHarnessWith(t, heartbeatOff, testMinSend)

		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		// A different device's first message publishes on arrival, unaffected by
		// the first device's interval.
		h.advance(time.Minute)
		h.send(otherTopic, pointset(18))
		h.assertTopics(otherTopic)

		// Each then releases on its own deadline.
		h.advance(time.Minute)
		h.send(eventTopic, pointset(22))
		h.send(otherTopic, pointset(18.5))
		h.assertTopics()

		h.advance(3 * time.Minute)
		h.assertTopics(eventTopic) // t=5m
		h.advance(time.Minute)
		h.assertTopics(otherTopic) // t=6m
	})
}

func TestThrottle_LegacyTopicIsThrottled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(legacyTopic, pointset(21.5))
		h.assertTopics(legacyTopic)

		h.advance(time.Minute)
		h.send(legacyTopic, pointset(22))
		h.assertTopics()

		h.advance(4 * time.Minute)
		h.assertTopics(legacyTopic)
	})
}

// State and metadata are rare, retained, and describe the device rather than
// sampling it, so they publish on arrival however tight the floor.
func TestThrottle_IgnoresNonPointsetTopics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(stateTopic, `{"system":{"operation":{"operational":true}}}`)
		h.assertTopics(stateTopic)

		h.advance(time.Second)
		h.send(stateTopic, `{"system":{"operation":{"operational":false}}}`)
		h.assertTopics(stateTopic)
	})
}

func TestThrottle_DisabledByZeroInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, 0)
		for range 3 {
			h.send(eventTopic, pointset(21.5))
			h.assertTopics(eventTopic)
			h.advance(time.Second)
		}
	})
}

// The heartbeat's reply is a sample like any other, so it goes through the floor
// too: a heartbeat can't breach minSendInterval. Arriving while a payload is held
// it supersedes it, being the fresher reading of the two. Only reachable when the
// heartbeat interval is shorter than the send floor, which is a misconfiguration,
// but the floor should hold either way.
func TestThrottle_HeartbeatReplyIsThrottled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, time.Minute, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(30 * time.Second)
		h.send(eventTopic, pointset(22))
		h.assertTopics()

		// The heartbeat is due at t=1m, but its reply is held rather than published:
		// the topic published at t=0 and the floor doesn't expire until t=5m.
		beat := pointset(23)
		h.get.answerWith(eventTopic, beat)
		h.advance(time.Minute)
		h.assertTopics()

		// At t=5m the heartbeat's reading goes out, not the change held before it.
		h.advance(4 * time.Minute)
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != beat {
			t.Errorf("published %s, want the heartbeat's reading %s", pubs[0].payload, beat)
		}
	})
}

// With nothing held the heartbeat behaves as it always has.
func TestThrottle_HeartbeatStillFiresWhenNothingHeld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, testInterval, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(testInterval)
		h.assertTopics(eventTopic)
	})
}

// Setting minSendInterval and heartbeatInterval to the same value turns the pair
// into a metronome: the floor bounds a chatty device and the heartbeat covers a
// quiet one, so a topic publishes exactly once per interval either way. This is
// the configuration for a site that wants a fixed sample rate rather than just a
// cap, so it's worth pinning both halves.
func TestThrottle_EqualIntervalsGiveAFixedRate(t *testing.T) {
	t.Run("chatty device is bounded by the floor", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHarnessWith(t, testMinSend, testMinSend)
			h.send(eventTopic, pointset(21.5))
			h.assertTopics(eventTopic)

			// Four intervals of changes every 10s: one publish per interval, no more.
			for i := range 4 {
				for range 30 {
					h.advance(10 * time.Second)
					h.send(eventTopic, pointset(22+float64(i)))
				}
				h.assertTopics(eventTopic)
			}
		})
	})
	t.Run("quiet device is covered by the heartbeat", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHarnessWith(t, testMinSend, testMinSend)
			h.send(eventTopic, pointset(21.5))
			h.assertTopics(eventTopic)

			// Nothing arrives on changes at all, so every publish is a heartbeat's
			// reply — and the floor never suppresses one, since the interval has
			// always just expired when it fires.
			for range 4 {
				h.advance(testMinSend)
				h.assertTopics(eventTopic)
			}
		})
	})
}

// A heartbeat is a publish on the topic, so it starts a fresh send interval
// rather than letting a change arrive on its heels.
func TestThrottle_HeartbeatResetsTheFloor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, 10*time.Minute, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		h.advance(10 * time.Minute)
		h.assertTopics(eventTopic) // heartbeat

		// t=11m is 11 minutes after the last live sample, but only one minute after
		// the heartbeat, so the change waits.
		h.advance(time.Minute)
		h.send(eventTopic, pointset(22))
		h.assertTopics()

		h.advance(4 * time.Minute)
		h.assertTopics(eventTopic) // t=15m
	})
}

// A release is a live sample, so a publish failure tears the task down as one on
// the changes channel would, and the payload stays held for the retry.
func TestThrottle_ReleaseSurvivesPublishError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		held := pointset(22)
		h.advance(time.Minute)
		h.send(eventTopic, held)
		h.assertTopics()

		wantErr := errors.New("broker unavailable")
		h.pub.failWith(wantErr)
		h.advance(4 * time.Minute)
		select {
		case err := <-h.done:
			if !errors.Is(err, wantErr) {
				t.Errorf("handleMessages returned %v, want %v", err, wantErr)
			}
		default:
			t.Fatal("handleMessages still running, want it to return the release error")
		}

		// The task is retried with the same throttle, which still holds the payload.
		h.pub.failWith(nil)
		h.changes = make(chan *udmipb.PullExportMessagesResponse)
		h.start()
		pubs := h.assertTopics(eventTopic)
		if pubs[0].payload != held {
			t.Errorf("published %s, want the held payload %s", pubs[0].payload, held)
		}
	})
}

// The throttle is built by tasksForSource, outside the retried task, so a retry
// must resume the existing interval rather than start a fresh one.
func TestThrottle_IntervalSurvivesRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarnessWith(t, heartbeatOff, testMinSend)
		h.send(eventTopic, pointset(21.5))
		h.assertTopics(eventTopic)

		close(h.changes)
		synctest.Wait()
		h.advance(time.Minute)
		h.changes = make(chan *udmipb.PullExportMessagesResponse)
		h.start()

		// t=1m, so still inside the interval that began before the restart.
		h.send(eventTopic, pointset(22))
		h.assertTopics()
		h.advance(4 * time.Minute)
		h.assertTopics(eventTopic)
	})
}

func TestThrottle_DisabledIsInert(t *testing.T) {
	th := newThrottle(0)
	now := time.Now()
	if th.hold(eventTopic, "payload", now) {
		t.Error("hold reported a hold on a disabled throttle")
	}
	th.sent(eventTopic, now)
	if _, ok := th.wait(now); ok {
		t.Error("wait armed a timer on a disabled throttle")
	}
	if msgs := th.due(now); msgs != nil {
		t.Errorf("due returned %v on a disabled throttle, want nothing", msgs)
	}
}

func TestThrottle_WaitAndDue(t *testing.T) {
	th := newThrottle(testMinSend)
	start := time.Now()

	// Nothing to wait for until something is held.
	if _, ok := th.wait(start); ok {
		t.Error("wait armed a timer before any message")
	}
	th.sent(eventTopic, start)
	if _, ok := th.wait(start); ok {
		t.Error("wait armed a timer with nothing held")
	}

	if !th.hold(eventTopic, "held", start.Add(time.Minute)) {
		t.Fatal("hold published a change one minute into a five minute interval")
	}
	d, ok := th.wait(start.Add(time.Minute))
	if !ok || d != 4*time.Minute {
		t.Errorf("wait is (%v, %v), want (4m, true)", d, ok)
	}
	if msgs := th.due(start.Add(time.Minute)); msgs != nil {
		t.Errorf("due returned %v before the interval expired", msgs)
	}

	// Overdue waits are clamped rather than negative.
	if d, ok := th.wait(start.Add(10 * time.Minute)); !ok || d != 0 {
		t.Errorf("wait is (%v, %v) when overdue, want (0s, true)", d, ok)
	}
	msgs := th.due(start.Add(testMinSend))
	if len(msgs) != 1 || msgs[0].topic != eventTopic || msgs[0].payload != "held" {
		t.Fatalf("due returned %v, want the held payload", msgs)
	}
	// due doesn't consume: only a reported publish does.
	if again := th.due(start.Add(testMinSend)); len(again) != 1 {
		t.Errorf("due returned %v on a second call, want the hold to survive until sent", again)
	}
	th.sent(eventTopic, start.Add(testMinSend))
	if msgs := th.due(start.Add(testMinSend)); msgs != nil {
		t.Errorf("due returned %v after sent, want the hold cleared", msgs)
	}
	if _, ok := th.wait(start.Add(testMinSend)); ok {
		t.Error("wait armed a timer after the hold was released")
	}
}
