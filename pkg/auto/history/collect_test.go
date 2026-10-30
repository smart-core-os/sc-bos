package history

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// A scheduled poll whose Get never answers must time out, say so, and not stop the next tick from recording.
func Test_automation_applyConfig_pollTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zapcore.WarnLevel)
		meter := &stuckMeter{stuckCalls: 1}

		n := node.New("test")
		n.Logger = zap.NewNop()
		n.Announce("meter",
			node.HasTrait(meterpb.TraitName),
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meter)),
		)

		a := &automation{
			clients:   n,
			announcer: node.NewReplaceAnnouncer(n),
			logger:    zap.New(core),
		}
		src := &config.Source{
			Name:            "meter",
			Trait:           meterpb.TraitName,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("* * * * * *"),
			Timeout:         &jsontypes.Duration{Duration: 500 * time.Millisecond},
		}
		if err := a.applyConfig(t.Context(), config.Root{Source: src, Storage: &config.Storage{Type: "memory"}}); err != nil {
			t.Fatal(err)
		}

		// the first tick (t=1s) is stuck and times out at 1.5s
		time.Sleep(1750 * time.Millisecond)
		synctest.Wait()
		aborted := logs.FilterMessage("poll aborted").All()
		if len(aborted) != 1 {
			t.Fatalf("want 1 'poll aborted' log, got %d: %v", len(aborted), logs.All())
		}
		if got, want := aborted[0].ContextMap()["source"], src.SourceName(); got != want {
			t.Errorf("'poll aborted' source = %v, want %v", got, want)
		}
		if got, want := fmt.Sprint(aborted[0].ContextMap()["error"]), "poll timed out after 500ms"; !strings.Contains(got, want) {
			t.Errorf("'poll aborted' error = %q, want it to contain %q", got, want)
		}

		// the second tick (t=2s) answers
		time.Sleep(time.Second)
		synctest.Wait()
		hist, err := meterpb.NewMeterHistoryClient(n.ClientConn()).
			ListMeterReadingHistory(t.Context(), &meterpb.ListMeterReadingHistoryRequest{Name: "meter", PageSize: 10})
		if err != nil {
			t.Fatal(err)
		}
		if got := hist.GetTotalSize(); got != 1 {
			t.Fatalf("history records = %d, want 1", got)
		}
	})
}

// When pull is unimplemented, a poll that never answers must be retried rather than end collection.
func Test_collectChanges_fallbackPollTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pullFn := func(context.Context, chan<- []byte) error {
			return status.Error(codes.Unimplemented, "no pull")
		}
		var calls int
		pollFn := func(ctx context.Context, changes chan<- []byte) error {
			calls++
			if calls <= 3 {
				<-ctx.Done()
				return ctx.Err()
			}
			return sendPayload(ctx, changes, calls)
		}

		src := config.Source{Timeout: &jsontypes.Duration{Duration: time.Second}}
		got := runCollectChanges(t, src, pullFn, pollFn)
		if len(got) == 0 {
			t.Fatalf("no payloads after %d polls, collection stalled", calls)
		}
	})
}

// A poll that has fetched a value must deliver it however long storage takes to accept it:
// collectors dedupe before sending, so an abandoned payload is never sent again.
func Test_collectChanges_pollTimeoutSlowStorage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pollFn := func(ctx context.Context, changes chan<- []byte) error {
			return sendPayload(ctx, changes, 1)
		}
		src := config.Source{
			PollingSchedule: jsontypes.MustParseExtendedSchedule("* * * * * *"),
			Timeout:         &jsontypes.Duration{Duration: time.Second},
		}
		payloads := make(chan []byte) // nobody reads until storage "recovers"
		go collectChanges(t.Context(), src, nil, pollFn, payloads, zap.NewNop())

		time.Sleep(5 * time.Second) // first tick at t=1s, well past its 1s timeout
		select {
		case got := <-payloads:
			if string(got) != "1" {
				t.Fatalf("payload = %q, want %q", got, "1")
			}
		default:
			t.Fatal("payload from the first poll was dropped")
		}
	})
}

// A pull that fails with a context error that isn't ours, from a deadline set further down for example,
// must be retried rather than end collection.
func Test_collectChanges_pullDownstreamDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls int
		pullFn := func(ctx context.Context, changes chan<- []byte) error {
			calls++
			if calls == 1 {
				return fmt.Errorf("stream: %w", context.DeadlineExceeded)
			}
			if err := sendPayload(ctx, changes, calls); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}
		pollFn := func(context.Context, chan<- []byte) error {
			t.Error("poll should not be called while pull is implemented")
			return nil
		}

		got := runCollectChanges(t, config.Source{}, pullFn, pollFn)
		if len(got) == 0 {
			t.Fatal("no payloads, collection stopped after the first pull error")
		}
	})
}

// An Append that never returns must time out, be logged, and not stop later payloads being stored.
func Test_storePayloads_timeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zapcore.WarnLevel)
		store := &stuckStore{Store: memstore.New(), stuckCalls: 1}
		payloads := make(chan []byte)

		go storePayloads(t.Context(), store, payloads, time.Second, zap.New(core))

		payloads <- []byte("first")  // stuck, dropped after 1s
		payloads <- []byte("second") // can only be received once the first has timed out
		synctest.Wait()

		n, err := store.Len(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("stored %d records, want 1", n)
		}
		into := make([]history.Record, 1)
		if _, err := store.Read(t.Context(), into); err != nil {
			t.Fatal(err)
		}
		if got := string(into[0].Payload); got != "second" {
			t.Errorf("stored %q, want %q", got, "second")
		}
		if got := logs.FilterMessage("storage failed").Len(); got != 1 {
			t.Errorf("want 1 'storage failed' log, got %d: %v", got, logs.All())
		}
	})
}

// runCollectChanges runs collectChanges for a minute of fake time, checks it didn't return,
// then stops it and returns the payloads it sent.
func runCollectChanges(t *testing.T, src config.Source, pullFn, pollFn getter) [][]byte {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	payloads := make(chan []byte, 100)
	done := make(chan error, 1)
	go func() {
		done <- collectChanges(ctx, src, pullFn, pollFn, payloads, zap.NewNop())
	}()

	time.Sleep(time.Minute)
	synctest.Wait()
	select {
	case err := <-done:
		t.Fatalf("collectChanges returned early: %v", err)
	default:
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("collectChanges after cancel = %v, want nil", err)
	}
	close(payloads)
	var got [][]byte
	for p := range payloads {
		got = append(got, p)
	}
	return got
}

func sendPayload(ctx context.Context, changes chan<- []byte, n int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case changes <- []byte(fmt.Sprint(n)):
		return nil
	}
}

// stuckMeter blocks the first stuckCalls calls to GetMeterReading until the request ctx is done.
type stuckMeter struct {
	meterpb.UnimplementedMeterApiServer
	mu         sync.Mutex
	calls      int
	stuckCalls int
}

func (m *stuckMeter) GetMeterReading(ctx context.Context, _ *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	m.mu.Lock()
	m.calls++
	calls := m.calls
	m.mu.Unlock()
	if calls <= m.stuckCalls {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &meterpb.MeterReading{Usage: float32(calls)}, nil
}

// stuckStore blocks the first stuckCalls calls to Append until ctx is done.
type stuckStore struct {
	history.Store
	calls      int
	stuckCalls int
}

func (s *stuckStore) Append(ctx context.Context, payload []byte) (history.Record, error) {
	s.calls++
	if s.calls <= s.stuckCalls {
		<-ctx.Done()
		return history.Record{}, ctx.Err()
	}
	return s.Store.Append(ctx, payload)
}
