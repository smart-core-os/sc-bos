package connecttelemetry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

const testManifestInterval = 15 * time.Minute

// fakeSink records what the health publisher sends.
type fakeSink struct {
	mu       sync.Mutex
	upCh     chan struct{} // closed while connected
	connUpCh chan struct{} // closed, and replaced, each time the connection comes up
	failNext int           // publishes to fail as unacknowledged
	hold     chan struct{} // when set, publishes wait for it to close before being acknowledged
	msgs     []sentMsg
}

type sentMsg struct {
	at       time.Time
	topic    string
	set      healthSetMessage
	manifest healthManifestMessage
}

func newFakeSink() *fakeSink {
	up := make(chan struct{})
	close(up)
	return &fakeSink{upCh: up, connUpCh: make(chan struct{})}
}

func (f *fakeSink) awaitConnection(ctx context.Context) error {
	f.mu.Lock()
	up := f.upCh
	f.mu.Unlock()
	select {
	case <-up:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeSink) connectionUp() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connUpCh
}

func (f *fakeSink) publishAcked(_ context.Context, topic string, payload []byte) error {
	f.mu.Lock()
	hold := f.hold
	f.mu.Unlock()
	if hold != nil {
		<-hold
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext > 0 {
		f.failNext--
		return errors.New("no PUBACK")
	}
	m := sentMsg{at: time.Now(), topic: topic}
	var err error
	if topic == healthManifestTopic("tlm") {
		err = json.Unmarshal(payload, &m.manifest)
	} else {
		err = json.Unmarshal(payload, &m.set)
	}
	if err != nil {
		panic(err)
	}
	f.msgs = append(f.msgs, m)
	return nil
}

func (f *fakeSink) setConnected(connected bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !connected {
		f.upCh = make(chan struct{})
		return
	}
	close(f.upCh)
	close(f.connUpCh)
	f.connUpCh = make(chan struct{})
}

func (f *fakeSink) setFailNext(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext = n
}

// sets returns the sets sent, optionally only those for resource.
func (f *fakeSink) sets(resource string) []sentMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	var res []sentMsg
	for _, m := range f.msgs {
		if m.topic == healthTopic("tlm") && (resource == "" || m.set.Resource == resource) {
			res = append(res, m)
		}
	}
	return res
}

func (f *fakeSink) manifests() []sentMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	var res []sentMsg
	for _, m := range f.msgs {
		if m.topic == healthManifestTopic("tlm") {
			res = append(res, m)
		}
	}
	return res
}

type healthHarness struct {
	t      *testing.T
	sink   *fakeSink
	checks *devicespb.Collection
	node   *node.Node
	start  time.Time
}

// newHealthHarness must be called inside a synctest bubble.
// Checks set before calling run are present when the publisher starts.
func newHealthHarness(t *testing.T) *healthHarness {
	return &healthHarness{
		t:      t,
		sink:   newFakeSink(),
		checks: devicespb.NewCollection(),
		node:   node.New("test-node"),
	}
}

func (h *healthHarness) run(maxRate float64) {
	ctx, cancel := context.WithCancel(context.Background())
	p := newHealthPublisher(h.sink, h.checks, h.node, "tlm", testManifestInterval, maxRate, zaptest.NewLogger(h.t))
	done := make(chan error, 1)
	go func() { done <- p.run(ctx) }()
	h.start = time.Now()
	h.t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			h.t.Errorf("run() error = %v", err)
		}
	})
}

func (h *healthHarness) setChecks(name string, checks ...*healthpb.HealthCheck) {
	h.t.Helper()
	_, err := h.checks.Update(&devicespb.Device{Name: name, HealthChecks: checks}, resource.WithCreateIfAbsent())
	require.NoError(h.t, err)
}

func (h *healthHarness) removeChecks(name string) {
	h.t.Helper()
	_, err := h.checks.Delete(name)
	require.NoError(h.t, err)
}

// sleepUntil advances the bubble's clock to d after run, then waits for the publisher to settle.
func (h *healthHarness) sleepUntil(d time.Duration) {
	time.Sleep(time.Until(h.start.Add(d)))
	synctest.Wait()
}

func faultCheck(id string, normality healthpb.HealthCheck_Normality) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{Id: id, DisplayName: id, Normality: normality}
}

func TestHealthPublisher_initialSendIsRateLimited(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		for _, name := range []string{"dev1", "dev2", "dev3"} {
			h.setChecks(name, faultCheck("c", healthpb.HealthCheck_NORMAL))
		}
		h.run(1)

		h.sleepUntil(healthStartDelay - time.Second)
		assert.Empty(t, h.sink.sets(""), "nothing sent before the start delay")

		h.sleepUntil(healthStartDelay + 10*time.Second)
		sets := h.sink.sets("")
		var names []string
		for _, s := range sets {
			names = append(names, s.set.Resource)
		}
		assert.ElementsMatch(t, []string{"dev1", "dev2", "dev3"}, names, "each set once")
		require.Len(t, sets, 3)
		for i := 1; i < len(sets); i++ {
			assert.GreaterOrEqual(t, sets[i].at.Sub(sets[i-1].at), time.Second, "no faster than the rate limit")
		}
		for _, s := range sets {
			assert.Equal(t, 1, s.set.Version)
			assert.Equal(t, subjectKindDevice, s.set.Subject.Kind)
			assert.Equal(t, checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{faultCheck("c", healthpb.HealthCheck_NORMAL)}), s.set.Hash)
			assert.Len(t, s.set.Checks, 1)
		}
	})
}

func TestHealthPublisher_sendsOnlyRealChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", boundsCheck("temp", 20, healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthStartDelay + time.Second)
		require.Len(t, h.sink.sets("dev1"), 1)

		// a value moving within the same state is not a change
		h.setChecks("dev1", boundsCheck("temp", 30, healthpb.HealthCheck_NORMAL))
		h.sleepUntil(healthStartDelay + time.Minute)
		require.Len(t, h.sink.sets("dev1"), 1)

		h.setChecks("dev1", boundsCheck("temp", 130, healthpb.HealthCheck_HIGH))
		h.sleepUntil(healthStartDelay + 2*time.Minute)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 2)
		var got map[string]any
		require.NoError(t, json.Unmarshal(sets[1].set.Checks[0], &got))
		assert.Equal(t, "HIGH", got["normality"])
	})
}

func TestHealthPublisher_retriesFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.sink.setFailNext(1)
		h.run(10)

		h.sleepUntil(healthStartDelay)
		assert.Empty(t, h.sink.sets("dev1"))
		h.sleepUntil(healthStartDelay + time.Second)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 1, "retried after a second")
		assert.Equal(t, h.start.Add(healthStartDelay+time.Second), sets[0].at)

		// unacknowledged sends back off, until the connection comes back
		h.sink.setFailNext(4) // fails at +0s, +1s, +3s, +7s, next retry +15s
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		changedAt := time.Since(h.start)
		h.sleepUntil(changedAt + 10*time.Second)
		require.Len(t, h.sink.sets("dev1"), 1)
		h.sink.setConnected(false)
		h.sink.setConnected(true)
		synctest.Wait()
		sets = h.sink.sets("dev1")
		require.Len(t, sets, 2, "retried as soon as the connection came back")
		assert.Equal(t, h.start.Add(changedAt+10*time.Second), sets[1].at)
	})
}

func TestHealthPublisher_waitsForConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.sink.setConnected(false)
		h.run(10)

		h.sleepUntil(healthStartDelay + time.Minute)
		assert.Empty(t, h.sink.sets(""))
		// the set built when the connection comes back is current
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		synctest.Wait()
		h.sink.setConnected(true)
		synctest.Wait()
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 1)
		var got map[string]any
		require.NoError(t, json.Unmarshal(sets[0].set.Checks[0], &got))
		assert.Equal(t, "ABNORMAL", got["normality"])
	})
}

func TestHealthPublisher_removal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("dev2", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthStartDelay + time.Second)
		require.Len(t, h.sink.sets(""), 2)

		h.removeChecks("dev1")
		h.sleepUntil(healthStartDelay + 2*time.Second)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 2)
		assert.Empty(t, sets[1].set.Checks, "an empty set removes every check")
		assert.NotNil(t, sets[1].set.Checks)

		h.sleepUntil(testManifestInterval)
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		assert.Equal(t, []manifestEntry{
			{NameHash: nameHash("dev2"), Hash: h.sink.sets("dev2")[0].set.Hash},
		}, manifests[0].manifest.Resources)
	})
}

func TestHealthPublisher_manifest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("empty") // a resource with no checks isn't in the manifest
		h.run(10)

		h.sleepUntil(testManifestInterval - time.Second)
		assert.Empty(t, h.sink.manifests(), "the first manifest waits a whole interval")
		h.sleepUntil(testManifestInterval)
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		assert.Equal(t, 1, manifests[0].manifest.Version)
		assert.Equal(t, []manifestEntry{
			{NameHash: nameHash("dev1"), Hash: checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{faultCheck("c", healthpb.HealthCheck_NORMAL)})},
		}, manifests[0].manifest.Resources)

		h.sleepUntil(2 * testManifestInterval)
		assert.Len(t, h.sink.manifests(), 2)
	})
}

func TestHealthPublisher_dailyResend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const name = "pier-point/hvac/ctrl-3"
		slot := dailySlot(name)
		require.Greater(t, slot, time.Hour, "test assumes the slot is well after startup")

		h := newHealthHarness(t)
		h.setChecks(name, faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		midnight := h.start.UTC().Truncate(24 * time.Hour)
		require.True(t, h.start.Equal(midnight), "synctest starts at midnight UTC")

		h.sleepUntil(slot - time.Second)
		require.Len(t, h.sink.sets(name), 1)
		h.sleepUntil(slot + time.Second)
		sets := h.sink.sets(name)
		require.Len(t, sets, 2, "resent at its slot")
		assert.Equal(t, sets[0].set.Hash, sets[1].set.Hash)

		h.sleepUntil(24*time.Hour + slot - time.Second)
		require.Len(t, h.sink.sets(name), 2)
		h.sink.setFailNext(1)
		h.sleepUntil(24*time.Hour + slot + 2*time.Second)
		sets = h.sink.sets(name)
		require.Len(t, sets, 3, "and again the next day, retrying a failure")
		assert.Equal(t, h.start.Add(24*time.Hour+slot+time.Second), sets[2].at)
	})
}

func TestHealthPublisher_kinds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.node.Announce("svc1", node.HasDeviceType(metadatapb.Metadata_SERVICE))
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("test-node", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthStartDelay + time.Second)

		kindOf := func(name string) string {
			sets := h.sink.sets(name)
			require.NotEmpty(t, sets, name)
			return sets[len(sets)-1].set.Subject.Kind
		}
		assert.Equal(t, subjectKindService, kindOf("svc1"))
		assert.Equal(t, subjectKindNode, kindOf("test-node"))
		assert.Equal(t, subjectKindDevice, kindOf("dev1"))

		// a change of kind is a change of set
		undo := h.node.Announce("dev1", node.HasDeviceType(metadatapb.Metadata_GATEWAY))
		h.sleepUntil(healthStartDelay + 2*time.Second)
		require.Len(t, h.sink.sets("dev1"), 2)
		assert.Equal(t, subjectKindNode, kindOf("dev1"))

		undo()
		h.sleepUntil(healthStartDelay + 3*time.Second)
		require.Len(t, h.sink.sets("dev1"), 3)
		assert.Equal(t, subjectKindDevice, kindOf("dev1"))
	})
}

func TestHealthPublisher_removalWithKind(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// like a driver's system check, the SERVICE type goes with the check
		h := newHealthHarness(t)
		undo := h.node.Announce("svc1", node.HasDeviceType(metadatapb.Metadata_SERVICE))
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthStartDelay + time.Second)
		require.Len(t, h.sink.sets("svc1"), 1)

		// the kind changes while the empty set is waiting for its PUBACK
		hold := make(chan struct{})
		h.sink.mu.Lock()
		h.sink.hold = hold
		h.sink.mu.Unlock()
		h.removeChecks("svc1")
		synctest.Wait()
		undo()
		synctest.Wait()
		h.sink.mu.Lock()
		h.sink.hold = nil
		h.sink.mu.Unlock()
		close(hold)
		h.sleepUntil(healthStartDelay + time.Minute)
		sets := h.sink.sets("svc1")
		require.Len(t, sets, 2, "one empty set, not resent for the change of kind")
		assert.Empty(t, sets[1].set.Checks)

		// the kind is current if the resource comes back
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.sleepUntil(healthStartDelay + 2*time.Minute)
		sets = h.sink.sets("svc1")
		require.Len(t, sets, 3)
		assert.Equal(t, subjectKindDevice, sets[2].set.Subject.Kind)
	})
}

func TestHealthPublisher_skipsHugeSets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		huge := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		huge.Description = string(make([]byte, healthSkipSetBytes))
		h.setChecks("dev1", huge)
		h.setChecks("dev2", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthStartDelay + time.Minute)
		assert.Empty(t, h.sink.sets("dev1"))
		assert.Len(t, h.sink.sets("dev2"), 1, "other sets still go")

		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		h.sleepUntil(healthStartDelay + 2*time.Minute)
		assert.Len(t, h.sink.sets("dev1"), 1, "sent once it changes to fit")
	})
}
