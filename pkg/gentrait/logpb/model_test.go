package logpb

import (
	"sync"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// msg is a convenience constructor for test messages.
func msg(text string, level logpb.LogLevel_Level) *logpb.LogMessage {
	return &logpb.LogMessage{Message: text, Level: level}
}

// ---- ring buffer ----------------------------------------------------------------

func TestAppendMessage_TailMessages_basic(t *testing.T) {
	m := NewModel(5)
	for i, text := range []string{"a", "b", "c"} {
		m.AppendMessage(msg(text, logpb.LogLevel_INFO))
		tail := m.TailMessages(i + 1)
		if len(tail) != i+1 {
			t.Fatalf("after %d appends: TailMessages(%d) returned %d items", i+1, i+1, len(tail))
		}
		if tail[len(tail)-1].Message != text {
			t.Errorf("last message = %q, want %q", tail[len(tail)-1].Message, text)
		}
	}
}

func TestAppendMessage_TailMessages_wraparound(t *testing.T) {
	const cap = 3
	m := NewModel(cap)
	// Fill beyond capacity: a, b, c, d, e → only c, d, e should remain.
	for _, text := range []string{"a", "b", "c", "d", "e"} {
		m.AppendMessage(msg(text, logpb.LogLevel_INFO))
	}
	tail := m.TailMessages(cap)
	if len(tail) != cap {
		t.Fatalf("TailMessages(%d) = %d items, want %d", cap, len(tail), cap)
	}
	want := []string{"c", "d", "e"}
	for i, w := range want {
		if tail[i].Message != w {
			t.Errorf("tail[%d] = %q, want %q", i, tail[i].Message, w)
		}
	}
}

func TestTailMessages_askMoreThanBuffered(t *testing.T) {
	m := NewModel(10)
	m.AppendMessage(msg("only", logpb.LogLevel_INFO))
	tail := m.TailMessages(100)
	if len(tail) != 1 {
		t.Fatalf("TailMessages(100) with 1 message: got %d, want 1", len(tail))
	}
}

func TestTailMessages_zero(t *testing.T) {
	m := NewModel(5)
	m.AppendMessage(msg("x", logpb.LogLevel_INFO))
	if tail := m.TailMessages(0); tail != nil {
		t.Errorf("TailMessages(0) = %v, want nil", tail)
	}
}

func TestTailMessages_chronologicalOrder(t *testing.T) {
	// Verify order is preserved after wraparound.
	m := NewModel(4)
	texts := []string{"a", "b", "c", "d", "e", "f"}
	for _, text := range texts {
		m.AppendMessage(msg(text, logpb.LogLevel_INFO))
	}
	// Ring contains: c, d, e, f
	tail := m.TailMessages(4)
	want := []string{"c", "d", "e", "f"}
	for i, w := range want {
		if tail[i].Message != w {
			t.Errorf("tail[%d] = %q, want %q", i, tail[i].Message, w)
		}
	}
}

// ---- subscribers ----------------------------------------------------------------

func TestSubscribe_receivesMessages(t *testing.T) {
	m := NewModel(10)
	ch, cancel := m.Subscribe()
	defer cancel()

	m.AppendMessage(msg("hello", logpb.LogLevel_INFO))

	batch := <-ch
	if len(batch) != 1 || batch[0].Message != "hello" {
		t.Errorf("received %v, want [{hello}]", batch)
	}
}

func TestSubscribe_cancelRemovesSubscriber(t *testing.T) {
	m := NewModel(10)
	ch, cancel := m.Subscribe()
	cancel()

	// Channel should be closed after cancel.
	_, ok := <-ch
	if ok {
		t.Error("channel still open after cancel")
	}

	// Subsequent Append should not panic (subscriber removed).
	m.AppendMessage(msg("after cancel", logpb.LogLevel_INFO))
}

func TestSubscribe_slowSubscriberDropped(t *testing.T) {
	m := NewModel(10)
	ch, cancel := m.Subscribe()
	defer cancel()

	// Flood the subscriber channel (capacity 32) without reading.
	for i := range 40 {
		m.AppendMessage(msg("flood", logpb.LogLevel_Level(i%4+1)))
	}
	// Should not block; some messages will be dropped silently.
	// Drain whatever arrived.
	drained := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			drained++
		default:
			if drained > 32 {
				t.Errorf("drained %d > channel capacity 32; messages should have been dropped", drained)
			}
			return
		}
	}
}

func TestSubscribe_multipleSubscribers(t *testing.T) {
	m := NewModel(10)
	ch1, cancel1 := m.Subscribe()
	ch2, cancel2 := m.Subscribe()
	defer cancel1()
	defer cancel2()

	m.AppendMessage(msg("broadcast", logpb.LogLevel_INFO))

	b1 := <-ch1
	b2 := <-ch2
	if b1[0].Message != "broadcast" || b2[0].Message != "broadcast" {
		t.Errorf("got %q / %q, want broadcast/broadcast", b1[0].Message, b2[0].Message)
	}
}

// ---- concurrent safety ----------------------------------------------------------

func TestAppendMessage_concurrent(t *testing.T) {
	m := NewModel(50)
	const goroutines = 20
	const msgsEach = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range msgsEach {
				m.AppendMessage(msg("concurrent", logpb.LogLevel_INFO))
			}
		}()
	}
	wg.Wait()
	tail := m.TailMessages(50)
	if len(tail) != 50 {
		t.Errorf("TailMessages(50) = %d, want 50", len(tail))
	}
}
