package history

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// An idle meter whose Pull never sends, as BACnet's does, must still be recorded every maxInterval, starting maxInterval after start.
func Test_heartbeat_idlePull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		meter := &idleMeter{}
		hist := startMeterHistory(t, meter, &jsontypes.Duration{Duration: time.Hour})

		time.Sleep(59 * time.Minute)
		synctest.Wait()
		if got := hist.count(t); got != 0 {
			t.Fatalf("records before maxInterval = %d, want 0", got)
		}
		time.Sleep(time.Minute + time.Second)
		synctest.Wait()
		if got := hist.count(t); got != 1 {
			t.Fatalf("records after maxInterval = %d, want 1", got)
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		if got := hist.count(t); got != 2 {
			t.Fatalf("records after 2x maxInterval = %d, want 2", got)
		}
	})
}

// A meter that can't be read records nothing, is retried a bounded number of times, and is recorded again once it can be read.
func Test_heartbeat_unavailable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		meter := &idleMeter{getErr: status.Error(codes.Unavailable, "device offline")}
		hist := startMeterHistory(t, meter, &jsontypes.Duration{Duration: time.Hour})

		// the beat at 1h and its retries at 1h1m, 1h2m and 1h3m
		time.Sleep(time.Hour + 30*time.Minute)
		synctest.Wait()
		if got := hist.count(t); got != 0 {
			t.Fatalf("records while unavailable = %d, want 0", got)
		}
		if got, want := meter.getCalls(), 1+maxHeartbeatRetries; got != want {
			t.Fatalf("Get calls = %d, want %d", got, want)
		}

		meter.setGetErr(nil)
		// the next beat is maxInterval after the last retry, at 2h3m
		time.Sleep(32 * time.Minute)
		synctest.Wait()
		if got := hist.count(t); got != 0 {
			t.Fatalf("records before the next beat = %d, want 0", got)
		}
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		if got := hist.count(t); got != 1 {
			t.Fatalf("records once readable = %d, want 1", got)
		}
	})
}

// A source that can't be read on demand is not asked again.
func Test_heartbeat_unimplemented(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		meter := &idleMeter{getErr: status.Error(codes.Unimplemented, "no get")}
		startMeterHistory(t, meter, &jsontypes.Duration{Duration: time.Hour})

		time.Sleep(10 * time.Hour)
		synctest.Wait()
		if got := meter.getCalls(); got != 1 {
			t.Fatalf("Get calls = %d, want 1", got)
		}
	})
}

// Without maxInterval an idle meter is never read and nothing is recorded, as before.
func Test_heartbeat_absent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		meter := &idleMeter{}
		hist := startMeterHistory(t, meter, nil)

		time.Sleep(10 * time.Hour)
		synctest.Wait()
		if got := hist.count(t); got != 0 {
			t.Fatalf("records = %d, want 0", got)
		}
		if got := meter.getCalls(); got != 0 {
			t.Fatalf("Get calls = %d, want 0", got)
		}
	})
}

// A reading that fails to store must be recorded the next time it is read, even though it hasn't changed.
func Test_storePayloads_failedWriteRecordsNextReading(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := config.Source{PollingSchedule: jsontypes.MustParseExtendedSchedule("* * * * * *")}
		store := runDedupedCollection(t, src, &failingStore{Store: memstore.New(), failCalls: 1})

		time.Sleep(5*time.Second + 500*time.Millisecond) // polls at 1s (fails to store), 2s, 3s, 4s and 5s
		synctest.Wait()
		if got := storeLen(t, store); got != 1 {
			t.Fatalf("records = %d, want 1", got)
		}
	})
}

// A heartbeat whose reading fails to store is retried after heartbeatRetryDelay rather than a whole maxInterval.
func Test_heartbeat_failedWriteRetried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := config.Source{MaxInterval: &jsontypes.Duration{Duration: time.Hour}}
		store := runDedupedCollection(t, src, &failingStore{Store: memstore.New(), failCalls: 1})

		time.Sleep(time.Hour + heartbeatRetryDelay + time.Second)
		synctest.Wait()
		if got := storeLen(t, store); got != 1 {
			t.Fatalf("records = %d, want 1", got)
		}
	})
}

// A heartbeat whose poll records nothing, leaving the source overdue, must wait a whole maxInterval rather than spin.
func Test_collectChanges_heartbeatOverdue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pullFn := func(ctx context.Context, _ chan<- []byte) error {
			<-ctx.Done()
			return ctx.Err()
		}
		var calls int
		pollFn := func(ctx context.Context, changes chan<- []byte) error {
			calls++
			return sendPayload(ctx, changes, calls)
		}
		src := config.Source{MaxInterval: &jsontypes.Duration{Duration: 10 * time.Second}}
		got := runCollectChanges(t, src, pullFn, pollFn)
		if len(got) < 5 || len(got) > 6 {
			t.Fatalf("heartbeats in a minute = %d, want one every 10s", len(got))
		}
	})
}

type meterHistory struct {
	client meterpb.MeterHistoryClient
}

func (h meterHistory) count(t *testing.T) int32 {
	t.Helper()
	resp, err := h.client.ListMeterReadingHistory(t.Context(), &meterpb.ListMeterReadingHistoryRequest{Name: "meter", PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	return resp.GetTotalSize()
}

// startMeterHistory starts a history auto recording meter into memory, with the given maxInterval.
func startMeterHistory(t *testing.T, meter meterpb.MeterApiServer, maxInterval *jsontypes.Duration) meterHistory {
	t.Helper()
	n := node.New("test")
	n.Logger = zap.NewNop()
	n.Announce("meter",
		node.HasTrait(meterpb.TraitName),
		node.HasServer(meterpb.RegisterMeterApiServer, meter),
	)
	a := &automation{
		clients:   n,
		announcer: node.NewReplaceAnnouncer(n),
		logger:    zap.NewNop(),
	}
	src := &config.Source{Name: "meter", Trait: meterpb.TraitName, MaxInterval: maxInterval}
	if err := a.routeConfig(t.Context(), config.Root{Source: src, Storage: &config.Storage{Type: "memory"}}); err != nil {
		t.Fatal(err)
	}
	return meterHistory{client: meterpb.NewMeterHistoryClient(n.ClientConn())}
}

// runDedupedCollection collects an unchanging reading from src into store, deduped as the collectors do,
// with a pull that never sends.
func runDedupedCollection(t *testing.T, src config.Source, store history.Store) history.Store {
	t.Helper()
	rec := newRecency(src.MaxIntervalDuration())
	d := newDeduper[*meterpb.MeterReading](nil, rec)
	pullFn := func(ctx context.Context, _ chan<- []byte) error {
		<-ctx.Done()
		return ctx.Err()
	}
	pollFn := func(ctx context.Context, changes chan<- []byte) error {
		reading := &meterpb.MeterReading{Usage: 1}
		if !d.Changed(ctx, reading) {
			return nil
		}
		payload, err := proto.Marshal(reading)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- payload:
			return nil
		}
	}
	payloads := make(chan []byte)
	go storePayloads(t.Context(), store, payloads, rec, time.Second, zap.NewNop())
	go collectChanges(t.Context(), src, pullFn, pollFn, payloads, rec, zap.NewNop())
	return store
}

func storeLen(t *testing.T, store history.Store) int {
	t.Helper()
	n, err := store.Len(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// idleMeter has a Pull that never sends and a Get that answers a steady reading, or getErr.
type idleMeter struct {
	meterpb.UnimplementedMeterApiServer
	mu     sync.Mutex
	calls  int
	getErr error
}

func (m *idleMeter) GetMeterReading(context.Context, *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &meterpb.MeterReading{Usage: 100}, nil
}

func (m *idleMeter) PullMeterReadings(_ *meterpb.PullMeterReadingsRequest, stream grpc.ServerStreamingServer[meterpb.PullMeterReadingsResponse]) error {
	<-stream.Context().Done()
	return stream.Context().Err()
}

func (m *idleMeter) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *idleMeter) setGetErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr = err
}

// failingStore fails the first failCalls calls to Append.
type failingStore struct {
	history.Store
	calls     int
	failCalls int
}

func (s *failingStore) Append(ctx context.Context, payload []byte) (history.Record, error) {
	s.calls++
	if s.calls <= s.failCalls {
		return history.Record{}, errors.New("store unavailable")
	}
	return s.Store.Append(ctx, payload)
}

func Test_automation_routeConfig_negativeMaxInterval(t *testing.T) {
	a := &automation{logger: zap.NewNop()}
	src := &config.Source{Name: "meter", Trait: meterpb.TraitName, MaxInterval: &jsontypes.Duration{Duration: -time.Hour}}
	err := a.routeConfig(t.Context(), config.Root{Source: src, Storage: &config.Storage{Type: "memory"}})
	if err == nil {
		t.Fatal("routeConfig accepted a negative maxInterval")
	}
}
