package logpb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// msg is a convenience constructor for a LogMessage with the given text.
func msg(text string) *logpb.LogMessage {
	return &logpb.LogMessage{Message: text}
}

// messages returns the Message field of each entry in msgs.
func messages(msgs []*logpb.LogMessage) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = m.Message
	}
	return out
}

// ---- Ring buffer: append & tail -------------------------------------------------

func TestTailMessages_empty(t *testing.T) {
	m := NewModel(4)
	if got := m.TailMessages(10); len(got) != 0 {
		t.Errorf("empty model: got %d messages, want 0", len(got))
	}
}

func TestTailMessages_partiallyFilled(t *testing.T) {
	m := NewModel(8)
	m.AppendMessage(msg("a"))
	m.AppendMessage(msg("b"))
	m.AppendMessage(msg("c"))

	got := messages(m.TailMessages(10))
	want := []string{"a", "b", "c"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTailMessages_exactCapacity(t *testing.T) {
	m := NewModel(3)
	m.AppendMessage(msg("x"))
	m.AppendMessage(msg("y"))
	m.AppendMessage(msg("z"))

	got := messages(m.TailMessages(3))
	want := []string{"x", "y", "z"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTailMessages_wrapAround(t *testing.T) {
	m := NewModel(3)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		m.AppendMessage(msg(s))
	}

	got := messages(m.TailMessages(10))
	want := []string{"c", "d", "e"}
	if !sliceEq(got, want) {
		t.Errorf("wrap-around: got %v, want %v", got, want)
	}
}

func TestTailMessages_nLessThanBuffered(t *testing.T) {
	m := NewModel(8)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		m.AppendMessage(msg(s))
	}

	got := messages(m.TailMessages(3))
	want := []string{"c", "d", "e"}
	if !sliceEq(got, want) {
		t.Errorf("tail(3): got %v, want %v", got, want)
	}
}

func TestTailMessages_zero(t *testing.T) {
	m := NewModel(8)
	m.AppendMessage(msg("a"))
	if got := m.TailMessages(0); got != nil {
		t.Errorf("tail(0): got %v, want nil", got)
	}
}

func TestTailMessages_chronologicalOrder(t *testing.T) {
	// After wrap-around, messages must still come out oldest-first.
	m := NewModel(4)
	seq := []string{"1", "2", "3", "4", "5", "6"}
	for _, s := range seq {
		m.AppendMessage(msg(s))
	}
	// Buffer holds "3","4","5","6"; tail(4) must preserve that order.
	got := messages(m.TailMessages(4))
	want := []string{"3", "4", "5", "6"}
	if !sliceEq(got, want) {
		t.Errorf("order after wrap: got %v, want %v", got, want)
	}
}

// ---- Subscribe / cancel lifecycle -----------------------------------------------

func TestSubscribe_receivesMessage(t *testing.T) {
	m := NewModel(8)
	ch, cancel := m.Subscribe()
	defer cancel()

	m.AppendMessage(msg("hello"))

	select {
	case batch := <-ch:
		if len(batch) != 1 || batch[0].Message != "hello" {
			t.Errorf("got batch %v, want [hello]", messages(batch))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestSubscribe_cancelClosesChannel(t *testing.T) {
	m := NewModel(8)
	ch, cancel := m.Subscribe()

	cancel()

	// Channel must be closed; receive should return the zero value immediately.
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel was not closed after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed within timeout")
	}
}

func TestSubscribe_noDeliveryAfterCancel(t *testing.T) {
	m := NewModel(8)
	ch, cancel := m.Subscribe()
	cancel()

	// Drain the (now-closed) channel so we can assert nothing new arrives.
	for range ch {
	}

	// AppendMessage must not panic and must not send to the removed subscriber.
	m.AppendMessage(msg("after-cancel"))
	// No assertion needed beyond no panic; the channel is closed and drained.
}

func TestSubscribe_multipleSubscribers(t *testing.T) {
	m := NewModel(8)
	ch1, cancel1 := m.Subscribe()
	ch2, cancel2 := m.Subscribe()
	defer cancel1()
	defer cancel2()

	m.AppendMessage(msg("broadcast"))

	for _, ch := range []<-chan []*logpb.LogMessage{ch1, ch2} {
		select {
		case batch := <-ch:
			if len(batch) == 0 || batch[0].Message != "broadcast" {
				t.Errorf("got %v, want [broadcast]", messages(batch))
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for subscriber")
		}
	}
}

func TestSubscribe_cancelOneLeaveOther(t *testing.T) {
	m := NewModel(8)
	ch1, cancel1 := m.Subscribe()
	ch2, cancel2 := m.Subscribe()
	defer cancel2()

	cancel1()
	for range ch1 {
	} // drain closed channel

	m.AppendMessage(msg("only-ch2"))

	select {
	case batch := <-ch2:
		if len(batch) == 0 || batch[0].Message != "only-ch2" {
			t.Errorf("ch2 got %v, want [only-ch2]", messages(batch))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for ch2")
	}
}

// ---- Slow subscriber ------------------------------------------------------------

// TestSubscribe_slowSubscriberDoesNotBlock verifies that AppendMessage uses a
// non-blocking send: once a subscriber's channel is full, further messages are
// dropped rather than stalling the caller.
//
//goland:noinspection ALL
func TestSubscribe_slowSubscriberDoesNotBlock(t *testing.T) {
	m := NewModel(64)
	_, cancelSlow := m.Subscribe()
	defer cancelSlow()

	// Fill the slow subscriber's channel to its capacity (32).
	for i := range 32 {
		m.AppendMessage(msg(string(rune('A' + i%26))))
	}

	// One more message would deadlock if AppendMessage blocked.
	done := make(chan struct{})
	go func() {
		m.AppendMessage(msg("overflow"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("AppendMessage blocked on a full subscriber channel")
	}
}

// TestSubscribe_fastSubscriberReceivesAll verifies that a subscriber whose channel
// is not full receives every message even when another subscriber is slow/full.
//
//goland:noinspection ALL
func TestSubscribe_fastSubscriberReceivesAll(t *testing.T) {
	m := NewModel(64)

	_, cancelSlow := m.Subscribe()
	defer cancelSlow()
	for i := range 32 {
		m.AppendMessage(msg(string(rune('A' + i%26))))
	}

	chFast, cancelFast := m.Subscribe()
	defer cancelFast()

	const n = 10
	for i := range n {
		m.AppendMessage(msg(string(rune('a' + i))))
	}

	received := 0
	deadline := time.After(time.Second)
	for received < n {
		select {
		case batch := <-chFast:
			received += len(batch)
		case <-deadline:
			t.Fatalf("fast subscriber received %d/%d messages", received, n)
		}
	}
}

// ---- helpers --------------------------------------------------------------------

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
