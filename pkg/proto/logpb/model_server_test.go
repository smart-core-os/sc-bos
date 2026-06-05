package logpb

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// fakeMessagesStream is a LogApi_PullLogMessagesServer that records sent responses.
type fakeMessagesStream struct {
	grpc.ServerStream
	ctx context.Context

	mu   sync.Mutex
	sent []*PullLogMessagesResponse
}

func (f *fakeMessagesStream) Send(r *PullLogMessagesResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, r)
	return nil
}

func (f *fakeMessagesStream) Context() context.Context { return f.ctx }

// sentMessages returns the Message field of every LogMessage sent so far.
func (f *fakeMessagesStream) sentMessages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, res := range f.sent {
		for _, c := range res.Changes {
			out = append(out, messages(c.Messages)...)
		}
	}
	return out
}

// fieldMsg builds a LogMessage with the given text, level and fields.
func fieldMsg(text string, level Level, fields map[string]string) *LogMessage {
	return &LogMessage{Message: text, Level: level, Fields: fields}
}

// pullReplay runs PullLogMessages with a cancelled-after-replay context and
// returns the messages sent during the initial replay.
func pullReplay(t *testing.T, model *Model, req *PullLogMessagesRequest) []string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeMessagesStream{ctx: ctx}
	srv := NewModelServer(model)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.PullLogMessages(req, stream)
	}()
	// The replay happens synchronously before the live loop; give it a moment
	// then cancel to unblock the live loop.
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("PullLogMessages did not return after context cancel")
	}
	return stream.sentMessages()
}

func TestPullLogMessages_fieldFilterMatch(t *testing.T) {
	model := NewModel(8)
	model.AppendMessage(fieldMsg("ours1", Level_LEVEL_INFO, map[string]string{"service.id": "a", "service.kind": "bacnet"}))
	model.AppendMessage(fieldMsg("theirs", Level_LEVEL_INFO, map[string]string{"service.id": "b", "service.kind": "bacnet"}))
	model.AppendMessage(fieldMsg("nofields", Level_LEVEL_INFO, nil))
	model.AppendMessage(fieldMsg("ours2", Level_LEVEL_INFO, map[string]string{"service.id": "a", "service.kind": "bacnet"}))

	got := pullReplay(t, model, &PullLogMessagesRequest{
		FieldFilter: map[string]string{"service.id": "a", "service.kind": "bacnet"},
	})
	want := []string{"ours1", "ours2"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPullLogMessages_fieldFilterPartialMismatch(t *testing.T) {
	model := NewModel(8)
	model.AppendMessage(fieldMsg("a", Level_LEVEL_INFO, map[string]string{"service.id": "a", "service.kind": "bacnet"}))

	// One key matches, the other doesn't — AND semantics means no match.
	got := pullReplay(t, model, &PullLogMessagesRequest{
		FieldFilter: map[string]string{"service.id": "a", "service.kind": "modbus"},
	})
	if len(got) != 0 {
		t.Errorf("got %v, want no messages", got)
	}
}

func TestPullLogMessages_emptyFilterReturnsAll(t *testing.T) {
	model := NewModel(8)
	model.AppendMessage(fieldMsg("a", Level_LEVEL_INFO, map[string]string{"k": "v"}))
	model.AppendMessage(fieldMsg("b", Level_LEVEL_INFO, nil))

	got := pullReplay(t, model, &PullLogMessagesRequest{})
	want := []string{"a", "b"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPullLogMessages_fieldFilterAndMinLevel(t *testing.T) {
	model := NewModel(8)
	fields := map[string]string{"service.id": "a"}
	model.AppendMessage(fieldMsg("debug", Level_LEVEL_DEBUG, fields))
	model.AppendMessage(fieldMsg("info", Level_LEVEL_INFO, fields))
	model.AppendMessage(fieldMsg("warn-other", Level_LEVEL_WARN, map[string]string{"service.id": "b"}))

	got := pullReplay(t, model, &PullLogMessagesRequest{
		MinLevel:    Level_LEVEL_INFO,
		FieldFilter: map[string]string{"service.id": "a"},
	})
	want := []string{"info"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPullLogMessages_initialCountCountsMatching(t *testing.T) {
	model := NewModel(16)
	fields := map[string]string{"service.id": "a"}
	// Interleave 5 matching with non-matching noise.
	for _, s := range []string{"m1", "m2", "m3", "m4", "m5"} {
		model.AppendMessage(fieldMsg(s, Level_LEVEL_INFO, fields))
		model.AppendMessage(fieldMsg("noise", Level_LEVEL_INFO, nil))
	}

	got := pullReplay(t, model, &PullLogMessagesRequest{
		InitialCount: 3,
		FieldFilter:  map[string]string{"service.id": "a"},
	})
	// Up to initial_count MATCHING messages, newest first, chronological order.
	want := []string{"m3", "m4", "m5"}
	if !sliceEq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPullLogMessages_livePathFiltered(t *testing.T) {
	model := NewModel(8)
	srv := NewModelServer(model)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := &fakeMessagesStream{ctx: ctx}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.PullLogMessages(&PullLogMessagesRequest{
			UpdatesOnly: true,
			FieldFilter: map[string]string{"service.id": "a"},
		}, stream)
	}()

	// Give the server a moment to subscribe before appending.
	time.Sleep(10 * time.Millisecond)
	model.AppendMessage(fieldMsg("ours", Level_LEVEL_INFO, map[string]string{"service.id": "a"}))
	model.AppendMessage(fieldMsg("theirs", Level_LEVEL_INFO, map[string]string{"service.id": "b"}))

	deadline := time.After(time.Second)
	for {
		got := stream.sentMessages()
		if len(got) > 0 {
			if !sliceEq(got, []string{"ours"}) {
				t.Errorf("got %v, want [ours]", got)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for live message")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("PullLogMessages did not return after cancel")
	}

	// "theirs" must never arrive, even after waiting.
	if got := stream.sentMessages(); !sliceEq(got, []string{"ours"}) {
		t.Errorf("after cancel: got %v, want [ours]", got)
	}
}
