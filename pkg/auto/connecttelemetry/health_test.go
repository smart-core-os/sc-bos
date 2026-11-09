package connecttelemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

const testManifestInterval = 15 * time.Minute

// fakeSink records what the health publisher sends.
type fakeSink struct {
	mu                sync.Mutex
	upCh              chan struct{} // closed while connected
	connUpCh          chan struct{} // closed, and replaced, each time the connection comes up
	failNext          int           // set publishes to fail as unacknowledged
	failNextManifests int           // manifest publishes to fail as unacknowledged
	hold              chan struct{} // when set, publishes wait for it to close before being acknowledged
	msgs              []sentMsg
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

func (f *fakeSink) publishAcked(ctx context.Context, topic string, payload []byte) error {
	f.mu.Lock()
	hold := f.hold
	f.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return ctx.Err() // as a publish cut short waiting for its PUBACK
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext > 0 && topic == healthTopic("tlm") {
		f.failNext--
		return errors.New("no PUBACK")
	}
	if f.failNextManifests > 0 && topic == healthManifestTopic("tlm") {
		f.failNextManifests--
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

// setHold makes publishes wait until release is called.
func (f *fakeSink) setHold() (release func()) {
	hold := make(chan struct{})
	f.mu.Lock()
	f.hold = hold
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.hold = nil
		f.mu.Unlock()
		close(hold)
	}
}

func (f *fakeSink) setFailNext(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext = n
}

func (f *fakeSink) setFailNextManifests(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNextManifests = n
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
	t                *testing.T
	sink             *fakeSink
	checks           *devicespb.Collection
	node             *node.Node
	manifestInterval time.Duration
	includeZones     bool
	groupTraits      []string       // as config.Root.GroupChildren
	store            healthAckStore // shared by every run, like the node's database
	start            time.Time
}

// newHealthHarness must be called inside a synctest bubble.
// Checks set before calling run are present when the publisher starts.
func newHealthHarness(t *testing.T) *healthHarness {
	return &healthHarness{
		t:                t,
		sink:             newFakeSink(),
		checks:           devicespb.NewCollection(),
		node:             node.New("test-node"),
		manifestInterval: testManifestInterval,
		store:            newMemAckStore(),
	}
}

// run starts a publisher, which is stopped by calling stop or when the test ends.
// Checks persist across runs, as they do across a restart.
func (h *healthHarness) run(maxRate float64) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	p := newHealthPublisher(h.sink, h.checks, h.node, h.store, "tlm", h.manifestInterval, h.includeZones, maxRate, zaptest.NewLogger(h.t))
	p.groupTraits(h.groupTraits)
	done := make(chan error, 1)
	go func() { done <- p.run(ctx) }()
	h.start = time.Now()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			if err := <-done; err != nil {
				h.t.Errorf("run() error = %v", err)
			}
		})
	}
	h.t.Cleanup(stop)
	return stop
}

// memAckStore is a healthAckStore in memory.
type memAckStore struct {
	mu        sync.Mutex
	acks      map[string]healthAck
	failSaves int // saves to fail
}

func newMemAckStore(acks ...healthAck) *memAckStore {
	s := &memAckStore{acks: make(map[string]healthAck)}
	for _, a := range acks {
		s.acks[a.Name] = a
	}
	return s
}

func (s *memAckStore) load() (map[string]healthAck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.acks), nil
}

func (s *memAckStore) save(put []healthAck, del []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSaves > 0 {
		s.failSaves--
		return errors.New("disk full")
	}
	for _, a := range put {
		s.acks[a.Name] = a
	}
	for _, name := range del {
		delete(s.acks, name)
	}
	return nil
}

func (s *memAckStore) get(name string) (healthAck, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.acks[name]
	return a, ok
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

		h.sleepUntil(healthSettleQuiet - time.Second)
		assert.Empty(t, h.sink.sets(""), "nothing sent before the start delay")

		h.sleepUntil(healthSettleQuiet + 10*time.Second)
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
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("dev1"), 1)

		// a value moving within the same state is not a change
		h.setChecks("dev1", boundsCheck("temp", 30, healthpb.HealthCheck_NORMAL))
		h.sleepUntil(healthSettleQuiet + time.Minute)
		require.Len(t, h.sink.sets("dev1"), 1)

		h.setChecks("dev1", boundsCheck("temp", 130, healthpb.HealthCheck_HIGH))
		h.sleepUntil(healthSettleQuiet + 2*time.Minute)
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

		h.sleepUntil(healthSettleQuiet)
		assert.Empty(t, h.sink.sets("dev1"))
		h.sleepUntil(healthSettleQuiet + time.Second)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 1, "retried after a second")
		assert.Equal(t, h.start.Add(healthSettleQuiet+time.Second), sets[0].at)

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

		h.sleepUntil(healthSettleQuiet + time.Minute)
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
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(""), 2)

		h.removeChecks("dev1")
		h.sleepUntil(healthSettleQuiet + 2*time.Second)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 2)
		assert.Empty(t, sets[1].set.Checks, "an empty set removes every check")
		assert.NotNil(t, sets[1].set.Checks)

		h.sleepUntil(healthFirstManifestDelay)
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

		h.sleepUntil(healthFirstManifestDelay - time.Second)
		assert.Empty(t, h.sink.manifests(), "the first manifest waits for checks to be created")
		h.sleepUntil(healthFirstManifestDelay)
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		assert.Equal(t, 1, manifests[0].manifest.Version)
		assert.Equal(t, h.start.Add(healthFirstManifestDelay).UTC(), manifests[0].manifest.Timestamp)
		assert.Equal(t, []manifestEntry{
			{NameHash: nameHash("dev1"), Hash: checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{faultCheck("c", healthpb.HealthCheck_NORMAL)})},
		}, manifests[0].manifest.Resources)

		// but not a whole interval, so a quick restart doesn't leave Connect 30 minutes without one
		require.Less(t, healthFirstManifestDelay, testManifestInterval)
		h.sleepUntil(healthFirstManifestDelay + testManifestInterval - time.Second)
		assert.Len(t, h.sink.manifests(), 1)
		h.sleepUntil(healthFirstManifestDelay + testManifestInterval)
		assert.Len(t, h.sink.manifests(), 2)
	})
}

func TestHealthPublisher_manifestShortInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.manifestInterval = 2 * time.Minute
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)

		h.sleepUntil(h.manifestInterval - time.Second)
		assert.Empty(t, h.sink.manifests())
		h.sleepUntil(h.manifestInterval)
		assert.Len(t, h.sink.manifests(), 1, "the first waits no longer than the interval")
		h.sleepUntil(2 * h.manifestInterval)
		assert.Len(t, h.sink.manifests(), 2)
	})
}

func TestHealthPublisher_manifestRetriesFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.sink.setFailNextManifests(2) // fails at +0s and +1s, next retry +3s
		h.run(10)

		h.sleepUntil(healthFirstManifestDelay)
		assert.Empty(t, h.sink.manifests())
		h.sleepUntil(healthFirstManifestDelay + time.Second)
		assert.Empty(t, h.sink.manifests())
		h.sleepUntil(healthFirstManifestDelay + 3*time.Second)
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1, "retried, not left until the next interval")
		assert.Equal(t, h.start.Add(healthFirstManifestDelay+3*time.Second).UTC(), manifests[0].manifest.Timestamp, "rebuilt for each try")
	})
}

func TestHealthPublisher_dailyResend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const name = "pier-point/hvac/ctrl-3"
		slot := dailySlot(name)
		require.Greater(t, slot, time.Hour, "test assumes the slot is well after startup")

		h := newHealthHarness(t)
		// manifests share the rate limit, so keep them clear of the slot to know when the resends go
		h.manifestInterval = 48 * time.Hour
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
		h.includeZones = true
		h.node.Announce("svc1", node.HasDeviceType(metadatapb.Metadata_SERVICE))
		h.node.Announce("auto1", node.HasDeviceType(metadatapb.Metadata_AUTOMATION))
		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("auto1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("test-node", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)

		kindOf := func(name string) string {
			sets := h.sink.sets(name)
			require.NotEmpty(t, sets, name)
			return sets[len(sets)-1].set.Subject.Kind
		}
		assert.Equal(t, subjectKindService, kindOf("svc1"))
		assert.Equal(t, subjectKindService, kindOf("auto1"))
		assert.Equal(t, subjectKindService, kindOf("zone1"))
		assert.Equal(t, subjectKindNode, kindOf("test-node"))
		assert.Equal(t, subjectKindDevice, kindOf("dev1"))

		// a change of kind is a change of set
		undo := h.node.Announce("dev1", node.HasDeviceType(metadatapb.Metadata_GATEWAY))
		h.sleepUntil(healthSettleQuiet + 2*time.Second)
		require.Len(t, h.sink.sets("dev1"), 2)
		assert.Equal(t, subjectKindNode, kindOf("dev1"))

		undo()
		h.sleepUntil(healthSettleQuiet + 3*time.Second)
		require.Len(t, h.sink.sets("dev1"), 3)
		assert.Equal(t, subjectKindDevice, kindOf("dev1"))
	})
}

// manifestNameHashes returns the name hashes listed in the last manifest sent.
func (h *healthHarness) manifestNameHashes() []string {
	h.t.Helper()
	manifests := h.sink.manifests()
	require.NotEmpty(h.t, manifests)
	var res []string
	for _, e := range manifests[len(manifests)-1].manifest.Resources {
		res = append(res, e.NameHash)
	}
	return res
}

func TestHealthPublisher_zonesExcluded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.node.Announce("svc1", node.HasDeviceType(metadatapb.Metadata_SERVICE))
		h.node.Announce("auto1", node.HasDeviceType(metadatapb.Metadata_AUTOMATION))
		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		for _, name := range []string{"svc1", "auto1", "zone1", "test-node", "dev1"} {
			h.setChecks(name, faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		}
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Minute)

		assert.Empty(t, h.sink.sets("zone1"), "zones send no health by default")
		for _, name := range []string{"svc1", "auto1", "test-node", "dev1"} {
			assert.Len(t, h.sink.sets(name), 1, name)
		}

		h.sleepUntil(healthFirstManifestDelay)
		assert.ElementsMatch(t, []string{nameHash("svc1"), nameHash("auto1"), nameHash("test-node"), nameHash("dev1")},
			h.manifestNameHashes(), "a zone is left out of the manifest, so Connect disposes of any checks it holds")
	})
}

func TestHealthPublisher_excludedZoneForgetsSavedAck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		store := newMemAckStore()
		h.store = store
		h.includeZones = true
		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		stop := h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("zone1"), 1)
		stop()
		_, saved := store.get("zone1")
		require.True(t, saved)

		// zones turned off: Connect disposes of the zone's checks, as the manifest leaves it out
		h.includeZones = false
		h.sink = newFakeSink()
		stop = h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		stop()
		_, saved = store.get("zone1")
		assert.False(t, saved, "Connect no longer holds the set")

		// and back on, so the set must go again
		h.includeZones = true
		h.sink = newFakeSink()
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		assert.Len(t, h.sink.sets("zone1"), 1)
	})
}

func TestHealthPublisher_zoneTypeArrivesLate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// checks and kinds arrive on separate streams, so a zone's set can go before its type is known
		h := newHealthHarness(t)
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("zone1"), 1)
		assert.Equal(t, subjectKindDevice, h.sink.sets("zone1")[0].set.Subject.Kind)

		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		synctest.Wait()
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		h.sleepUntil(healthFirstManifestDelay)
		assert.Len(t, h.sink.sets("zone1"), 1, "no set once it is known to be a zone")
		assert.NotContains(t, h.manifestNameHashes(), nameHash("zone1"), "dropped from the manifest")

		h.sleepUntil(25 * time.Hour)
		assert.Len(t, h.sink.sets("zone1"), 1, "not resent daily")
	})
}

func TestHealthPublisher_zoneRemoval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		undo := h.node.Announce("zone2", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("zone2", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Empty(t, h.sink.sets(""))

		h.removeChecks("zone1")
		h.sleepUntil(healthSettleQuiet + time.Minute)
		assert.Empty(t, h.sink.sets(""), "no empty set for a zone that was never sent")

		// the zone service stops, on a config change say, while an automation keeps its checks on the zone
		undo()
		h.setChecks("zone2", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		h.sleepUntil(healthFirstManifestDelay)
		assert.Empty(t, h.sink.sets(""), "still a zone without its type")
		assert.Empty(t, h.manifestNameHashes())

		// and when it starts again
		h.node.Announce("zone2", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.sleepUntil(healthFirstManifestDelay + time.Minute)
		assert.Empty(t, h.sink.sets(""))
	})
}

func TestHealthPublisher_zonesIncluded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.includeZones = true
		h.node.Announce("zone1", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.setChecks("zone1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)

		sets := h.sink.sets("zone1")
		require.Len(t, sets, 1)
		assert.Equal(t, subjectKindService, sets[0].set.Subject.Kind)
		h.sleepUntil(healthFirstManifestDelay)
		assert.Equal(t, []string{nameHash("zone1")}, h.manifestNameHashes())
	})
}

func TestHealthPublisher_removalWithKind(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// like a driver's system check, the SERVICE type goes with the check
		h := newHealthHarness(t)
		undo := h.node.Announce("svc1", node.HasDeviceType(metadatapb.Metadata_SERVICE))
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("svc1"), 1)

		// the kind changes while the empty set is waiting for its PUBACK
		release := h.sink.setHold()
		h.removeChecks("svc1")
		synctest.Wait()
		undo()
		synctest.Wait()
		release()
		h.sleepUntil(healthSettleQuiet + time.Minute)
		sets := h.sink.sets("svc1")
		require.Len(t, sets, 2, "one empty set, not resent for the change of kind")
		assert.Empty(t, sets[1].set.Checks)

		// the kind is current if the resource comes back
		h.setChecks("svc1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.sleepUntil(healthSettleQuiet + 2*time.Minute)
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
		h.sleepUntil(healthSettleQuiet + time.Minute)
		assert.Empty(t, h.sink.sets("dev1"))
		assert.Len(t, h.sink.sets("dev2"), 1, "other sets still go")

		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		h.sleepUntil(healthSettleQuiet + 2*time.Minute)
		assert.Len(t, h.sink.sets("dev1"), 1, "sent once it changes to fit")
	})
}

func TestHealthPublisher_manifestListsSentHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		abnormal := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		huge := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		huge.Description = string(make([]byte, healthSkipSetBytes))
		hashOf := func(c *healthpb.HealthCheck) string {
			return checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{c})
		}
		h.setChecks("dev1", normal)
		h.setChecks("dev2", normal)
		h.setChecks("dev4", normal)
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(""), 3)

		h.setChecks("dev4", huge)
		h.sleepUntil(healthSettleQuiet + 2*time.Second)
		require.Len(t, h.sink.sets("dev4"), 1, "skipped as too large")

		// When the manifest is built dev1's change is waiting for its PUBACK, already stamped,
		// while dev2's change and dev3 are queued behind it, not yet stamped.
		release := h.sink.setHold()
		h.setChecks("dev1", abnormal)
		synctest.Wait()
		h.setChecks("dev2", abnormal)
		h.setChecks("dev3", normal)
		h.sleepUntil(healthFirstManifestDelay)
		release()
		synctest.Wait()

		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		dev1Sets := h.sink.sets("dev1")
		require.Len(t, dev1Sets, 2)
		require.True(t, dev1Sets[1].set.Timestamp.Before(manifests[0].manifest.Timestamp), "dev1's set is older than the manifest")
		assert.ElementsMatch(t, []manifestEntry{
			{NameHash: nameHash("dev1"), Hash: hashOf(abnormal)}, // what Connect will hold once the set lands
			{NameHash: nameHash("dev2"), Hash: hashOf(normal)},   // Connect won't judge the newer set against it
			{NameHash: nameHash("dev3"), Hash: hashOf(normal)},   // never acknowledged
			{NameHash: nameHash("dev4"), Hash: hashOf(huge)},     // skipped
		}, manifests[0].manifest.Resources)

		h.sleepUntil(healthFirstManifestDelay + testManifestInterval)
		manifests = h.sink.manifests()
		require.Len(t, manifests, 2)
		assert.ElementsMatch(t, []manifestEntry{
			{NameHash: nameHash("dev1"), Hash: hashOf(abnormal)},
			{NameHash: nameHash("dev2"), Hash: hashOf(abnormal)},
			{NameHash: nameHash("dev3"), Hash: hashOf(normal)},
			{NameHash: nameHash("dev4"), Hash: hashOf(huge)},
		}, manifests[1].manifest.Resources, "once acknowledged")
	})
}

func TestHealthPublisher_failedSendIsResent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		abnormal := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		hashOf := func(c *healthpb.HealthCheck) string {
			return checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{c})
		}
		h.setChecks("dev1", normal)
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("dev1"), 1)

		// the change may have reached the broker even though it wasn't acknowledged
		h.sink.setFailNext(1000)
		h.setChecks("dev1", abnormal)
		h.sleepUntil(healthFirstManifestDelay)
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		assert.Equal(t, []manifestEntry{{NameHash: nameHash("dev1"), Hash: hashOf(abnormal)}}, manifests[0].manifest.Resources,
			"the broker's set is unknown, so the manifest lists the current one")

		// so changing back to the set last acknowledged is still sent
		h.sink.setFailNext(0)
		h.setChecks("dev1", normal)
		h.sleepUntil(healthFirstManifestDelay + time.Second)
		sets := h.sink.sets("dev1")
		require.Len(t, sets, 2)
		assert.Equal(t, hashOf(normal), sets[1].set.Hash)
	})
}

func TestHealthPublisher_backoffIsCapped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newHealthPublisher(newFakeSink(), nil, nil, nopAckStore{}, "tlm", testManifestInterval, false, 10, zaptest.NewLogger(t))
		defer p.stopRetries()
		p.setChecks("dev1", []*healthpb.HealthCheck{faultCheck("c", healthpb.HealthCheck_NORMAL)})
		queued := func() bool {
			p.mu.Lock()
			defer p.mu.Unlock()
			return p.queued["dev1"]
		}
		_, err := p.dequeue(context.Background())
		require.NoError(t, err)

		// enough failures for the backoff's shift to overflow
		p.resources["dev1"].failures = 39
		p.sent("dev1", p.resources["dev1"].hash, errors.New("no PUBACK"))
		time.Sleep(healthMaxRetry - time.Millisecond)
		synctest.Wait()
		assert.False(t, queued(), "not retried before the longest backoff")
		time.Sleep(time.Millisecond)
		synctest.Wait()
		assert.True(t, queued(), "retried after the longest backoff")
	})
}

func TestHealthPublisher_setTimestampsIncrease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newHealthPublisher(newFakeSink(), nil, nil, nopAckStore{}, "tlm", testManifestInterval, false, 10, zaptest.NewLogger(t))
		p.setChecks("dev1", []*healthpb.HealthCheck{faultCheck("c", healthpb.HealthCheck_NORMAL)})
		stamp := func() time.Time {
			t.Helper()
			payload, _, ok := p.buildSet("dev1")
			require.True(t, ok)
			var msg healthSetMessage
			require.NoError(t, json.Unmarshal(payload, &msg))
			return msg.Timestamp
		}

		// time doesn't pass in the bubble, so these are built in the same instant
		first := stamp()
		assert.Equal(t, time.Now().UTC().Truncate(time.Millisecond), first)
		assert.Equal(t, first.Add(time.Millisecond), stamp())

		// the clock stepping back doesn't make sets older
		p.resources["dev1"].lastStamp = time.Now().Add(time.Hour).Truncate(time.Millisecond)
		assert.Equal(t, time.Now().Add(time.Hour+time.Millisecond).Truncate(time.Millisecond).UTC(), stamp())
	})
}

func hashOfCheck(c *healthpb.HealthCheck) string {
	return checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{c})
}

func TestHealthPublisher_restartSendsOnlyChangedSets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		abnormal := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		h.setChecks("dev1", normal)
		h.setChecks("dev2", normal)
		stop := h.run(10)
		// stopped before the first periodic flush, so saved as the automation stops
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(""), 2)
		stop()

		// dev2 changes while the node is down
		time.Sleep(time.Minute)
		h.setChecks("dev2", abnormal)
		h.sink = newFakeSink()
		stop = h.run(10)
		h.sleepUntil(healthFirstManifestDelay)
		sets := h.sink.sets("")
		require.Len(t, sets, 1, "only the set that changed")
		assert.Equal(t, "dev2", sets[0].set.Resource)
		assert.Equal(t, hashOfCheck(abnormal), sets[0].set.Hash)
		assert.Equal(t, h.start.Add(healthSettleQuiet), sets[0].at, "once the checks have settled")
		manifests := h.sink.manifests()
		require.Len(t, manifests, 1)
		assert.ElementsMatch(t, []manifestEntry{
			{NameHash: nameHash("dev1"), Hash: hashOfCheck(normal)}, // confirms the set Connect already holds
			{NameHash: nameHash("dev2"), Hash: hashOfCheck(abnormal)},
		}, manifests[0].manifest.Resources)

		// and nothing at all when nothing changed
		stop()
		time.Sleep(time.Minute)
		h.sink = newFakeSink()
		h.run(10)
		h.sleepUntil(healthFirstManifestDelay)
		assert.Empty(t, h.sink.sets(""))
		require.Len(t, h.sink.manifests(), 1)
		assert.Equal(t, manifests[0].manifest.Resources, h.sink.manifests()[0].manifest.Resources)
	})
}

func TestHealthPublisher_sendInFlightAtStopIsResent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		abnormal := faultCheck("c", healthpb.HealthCheck_ABNORMAL)
		store := newMemAckStore(healthAck{Name: "dev1", Hash: hashOfCheck(normal), AckedAt: time.Now()})
		h.store = store
		h.setChecks("dev1", abnormal)
		release := h.sink.setHold()
		stop := h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		// the abnormal set is waiting for its PUBACK when the publisher stops, so it may yet land
		stop()
		release()
		_, saved := store.get("dev1")
		assert.False(t, saved, "the saved hash no longer says what Connect holds")

		// back to normal while down, which must still be sent
		h.setChecks("dev1", normal)
		h.sink = newFakeSink()
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		sets := h.sink.sets("")
		require.Len(t, sets, 1)
		assert.Equal(t, hashOfCheck(normal), sets[0].set.Hash)
	})
}

func TestHealthPublisher_restoredAckDueForResendIsSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		now := time.Now()
		require.NotZero(t, dailySlot("fresh"), "test assumes the slot of fresh is not the start")
		h.store = newMemAckStore(
			healthAck{Name: "fresh", Hash: hashOfCheck(normal), AckedAt: now.Add(-time.Minute)},
			// acknowledged just before the slot yesterday, so the resend since was missed
			healthAck{Name: "due", Hash: hashOfCheck(normal), AckedAt: now.Add(-24*time.Hour + dailySlot("due") - time.Minute)},
			healthAck{Name: "old", Hash: hashOfCheck(normal), AckedAt: now.Add(-25 * time.Hour)},
			healthAck{Name: "changed", Hash: hashOfCheck(faultCheck("c", healthpb.HealthCheck_ABNORMAL)), AckedAt: now.Add(-time.Minute)},
		)
		for _, name := range []string{"fresh", "due", "old", "changed"} {
			h.setChecks(name, normal)
		}
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		var names []string
		for _, s := range h.sink.sets("") {
			names = append(names, s.set.Resource)
		}
		assert.ElementsMatch(t, []string{"due", "old", "changed"}, names)
	})
}

func TestHealthPublisher_savedAcks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		store := newMemAckStore()
		h.store = store
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		h.setChecks("dev1", normal)
		h.setChecks("dev2", normal)
		h.run(10)
		h.sleepUntil(time.Minute + time.Second)
		ack, ok := store.get("dev1")
		require.True(t, ok, "saved every minute")
		assert.Equal(t, hashOfCheck(normal), ack.Hash)
		assert.Equal(t, h.start.Add(healthSettleQuiet), ack.AckedAt)

		// a removed resource is forgotten once its empty set is acknowledged
		h.removeChecks("dev2")
		synctest.Wait()
		// the broker may hold either set after a failed send
		h.sink.setFailNext(1000)
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_ABNORMAL))
		h.sleepUntil(2*time.Minute + time.Second)
		_, ok = store.get("dev2")
		assert.False(t, ok, "removed resource")
		_, ok = store.get("dev1")
		assert.False(t, ok, "failed send")
	})
}

func TestHealthPublisher_failedSaveIsRetried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		store := newMemAckStore()
		store.failSaves = 1
		h.store = store
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(time.Minute + time.Second)
		_, ok := store.get("dev1")
		require.False(t, ok)
		h.sleepUntil(2*time.Minute + time.Second)
		_, ok = store.get("dev1")
		assert.True(t, ok)
	})
}

func TestHealthPublisher_restoredAcksKeptUntilSettled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		normal := faultCheck("c", healthpb.HealthCheck_NORMAL)
		store := newMemAckStore(
			healthAck{Name: "late", Hash: hashOfCheck(normal), AckedAt: time.Now().Add(-time.Minute)},
			healthAck{Name: "gone", Hash: hashOfCheck(normal), AckedAt: time.Now().Add(-time.Minute)},
		)
		h.store = store
		h.setChecks("dev1", normal)
		h.run(10)

		// a resource that comes back late, while checks are still being created, is not resent
		h.sleepUntil(20 * time.Second)
		h.setChecks("dev2", normal)
		h.sleepUntil(40 * time.Second)
		h.setChecks("late", normal)
		// the first save comes while held
		h.sleepUntil(time.Minute + time.Second)
		require.Empty(t, h.sink.sets(""))
		_, ok := store.get("gone")
		assert.True(t, ok, "not deleted while it might still come back")
		h.sleepUntil(2*time.Minute + time.Second)
		names := make(map[string]bool)
		for _, s := range h.sink.sets("") {
			names[s.set.Resource] = true
		}
		assert.Equal(t, map[string]bool{"dev1": true, "dev2": true}, names)
		_, ok = store.get("gone")
		assert.False(t, ok, "deleted once checks have settled")
		_, ok = store.get("late")
		assert.True(t, ok)
	})
}

func TestHealthPublisher_holdWaitsForChecksToSettle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		a := faultCheck("a", healthpb.HealthCheck_NORMAL)
		b := faultCheck("b", healthpb.HealthCheck_NORMAL)
		h.setChecks("dev1", a)
		h.run(10)

		h.sleepUntil(20 * time.Second)
		h.setChecks("dev1", a, b) // a new check id
		h.sleepUntil(40 * time.Second)
		h.setChecks("dev2", a) // a new resource
		h.sleepUntil(60 * time.Second)
		// changes to existing checks are not new checks
		h.setChecks("dev1", boundsCheck("a", 120, healthpb.HealthCheck_HIGH), b)
		h.setChecks("dev2", boundsCheck("a", 20, healthpb.HealthCheck_NORMAL))
		h.sleepUntil(65 * time.Second)
		h.setChecks("dev2", boundsCheck("a", 30, healthpb.HealthCheck_NORMAL))

		h.sleepUntil(70*time.Second - time.Millisecond)
		assert.Empty(t, h.sink.sets(""), "held until 30s after the last new check")
		h.sleepUntil(71 * time.Second)
		dev1 := h.sink.sets("dev1")
		require.Len(t, dev1, 1, "the settled set, once")
		assert.Equal(t, h.start.Add(70*time.Second), dev1[0].at)
		assert.Len(t, dev1[0].set.Checks, 2)
		assert.Len(t, h.sink.sets("dev2"), 1)
	})
}

func TestHealthPublisher_holdIsCapped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.run(10)
		for i := range 6 { // to 100s, so never quiet for 30s before the cap
			h.sleepUntil(time.Duration(i) * 20 * time.Second)
			h.setChecks(fmt.Sprintf("dev%d", i), faultCheck("c", healthpb.HealthCheck_NORMAL))
		}
		h.sleepUntil(healthSettleMax - time.Millisecond)
		assert.Empty(t, h.sink.sets(""))
		h.sleepUntil(healthSettleMax + time.Second)
		sets := h.sink.sets("")
		require.Len(t, sets, 6)
		assert.Equal(t, h.start.Add(healthSettleMax), sets[0].at)
	})
}

func TestHealthPublisher_noStoreResendsAfterRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.store = nopAckStore{}
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		stop := h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets("dev1"), 1)
		stop()

		h.sink = newFakeSink()
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		assert.Len(t, h.sink.sets("dev1"), 1)
	})
}

func TestSlotDue(t *testing.T) {
	day := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	slot := 10 * time.Hour
	tests := []struct {
		name     string
		from, to time.Time
		want     bool
	}{
		{"before", at(8, 0), at(9, 0), false},
		{"in", at(9, 0), at(10, 0), true},
		{"from is exclusive", at(10, 0), at(11, 0), false},
		{"after", at(11, 0), at(12, 0), false},
		{"wraps midnight", at(20, 0), at(34, 0), true},
		{"wraps midnight, not reached", at(20, 0), at(33, 0), false},
		{"a day or more", at(11, 0), at(35, 0), true},
		{"empty", at(10, 0), at(10, 0), false},
		{"backwards", at(11, 0), at(9, 0), false},
		{"in another zone", at(9, 0).In(time.FixedZone("x", 5*3600)), at(10, 0).In(time.FixedZone("y", -3*3600)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slotDue(slot, tt.from, tt.to))
		})
	}
}

// checkIDs returns the ids of the checks in a sent set, with their display names.
func checkIDs(t *testing.T, m sentMsg) map[string]string {
	t.Helper()
	res := make(map[string]string)
	for _, raw := range m.set.Checks {
		var c struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		}
		require.NoError(t, json.Unmarshal(raw, &c))
		res[c.ID] = c.DisplayName
	}
	return res
}

// lastSet returns the checks of the last set sent for resource.
func (h *healthHarness) lastSet(resource string) map[string]string {
	h.t.Helper()
	sets := h.sink.sets(resource)
	require.NotEmpty(h.t, sets, resource)
	return checkIDs(h.t, sets[len(sets)-1])
}

const (
	testPHX       = "floors/01/PHX-S-01-01"
	testPHXFlow   = testPHX + "/chw-primary-flow"
	testPHXReturn = testPHX + "/chw-primary-return"
)

// announcePHX announces a heat exchanger with no Temperature of its own, and two of its points.
func (h *healthHarness) announcePHX() {
	h.node.Announce(testPHX)
	h.node.Announce(testPHXFlow, node.HasTrait(temperaturepb.TraitName))
	h.node.Announce(testPHXReturn, node.HasTrait(temperaturepb.TraitName))
}

// returnRange is the check a healthbounds automation puts on every point it selects, with the same id on each.
func returnRange() *healthpb.HealthCheck {
	return &healthpb.HealthCheck{Id: "healthbounds:chw-return", DisplayName: "CHW Return Temperature Range", Normality: healthpb.HealthCheck_NORMAL}
}

func TestHealthPublisher_groupsChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.announcePHX()
		h.setChecks(testPHX, faultCheck("comms", healthpb.HealthCheck_NORMAL))
		h.setChecks(testPHXFlow, returnRange())
		h.setChecks(testPHXReturn, returnRange())
		h.setChecks("dev1", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)

		assert.Equal(t, map[string]string{
			"comms": "comms",
			"chw-primary-flow/healthbounds:chw-return":   "chw-primary-flow: CHW Return Temperature Range",
			"chw-primary-return/healthbounds:chw-return": "chw-primary-return: CHW Return Temperature Range",
		}, h.lastSet(testPHX), "one set, its ids unique")
		require.Len(t, h.sink.sets(testPHX), 1)
		assert.Empty(t, h.sink.sets(testPHXFlow), "a child has no set of its own")
		assert.Empty(t, h.sink.sets(testPHXReturn))
		assert.Equal(t, map[string]string{"c": "c"}, h.lastSet("dev1"), "other devices are unchanged")

		h.sleepUntil(healthFirstManifestDelay)
		assert.ElementsMatch(t, []string{nameHash(testPHX), nameHash("dev1")}, h.manifestNameHashes())

		// a child's change is a change of its parent's set
		changed := returnRange()
		changed.Normality = healthpb.HealthCheck_HIGH
		h.setChecks(testPHXFlow, changed)
		h.sleepUntil(healthFirstManifestDelay + time.Second)
		require.Len(t, h.sink.sets(testPHX), 2)
		assert.Empty(t, h.sink.sets(testPHXFlow))

		// removing a child takes its checks out of the set
		h.removeChecks(testPHXReturn)
		h.sleepUntil(healthFirstManifestDelay + 2*time.Second)
		assert.Equal(t, map[string]string{
			"comms": "comms",
			"chw-primary-flow/healthbounds:chw-return": "chw-primary-flow: CHW Return Temperature Range",
		}, h.lastSet(testPHX))
		assert.Empty(t, h.sink.sets(testPHXReturn))

		// removing the rest empties it
		h.removeChecks(testPHX)
		h.sleepUntil(healthFirstManifestDelay + 3*time.Second)
		assert.Equal(t, map[string]string{
			"chw-primary-flow/healthbounds:chw-return": "chw-primary-flow: CHW Return Temperature Range",
		}, h.lastSet(testPHX), "the children keep the set when the parent's own checks go")
		h.removeChecks(testPHXFlow)
		h.sleepUntil(healthFirstManifestDelay + 4*time.Second)
		sets := h.sink.sets(testPHX)
		assert.Empty(t, sets[len(sets)-1].set.Checks)
		assert.NotNil(t, sets[len(sets)-1].set.Checks, "an empty set removes every check")
		assert.Empty(t, h.sink.sets(testPHXFlow))
	})
}

// TestHealthPublisher_groupsChildrenAnnouncedLate checks a child whose device the publisher hasn't
// seen yet at start is held until it has, rather than sent as itself and then emptied: otherwise
// every start would send every child's ref an empty set.
func TestHealthPublisher_groupsChildrenAnnouncedLate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.setChecks(testPHXFlow, returnRange())
		h.setChecks(testPHXReturn, returnRange())
		h.run(10)
		h.sleepUntil(5 * time.Second)
		h.announcePHX()
		h.sleepUntil(healthSettleMax + time.Second)

		require.Len(t, h.sink.sets(""), 1)
		assert.Len(t, h.lastSet(testPHX), 2)
	})
}

// TestHealthPublisher_groupedLate checks a child seen only after the hold is first sent as itself,
// then moved into its parent's set, its own ref sent an empty set.
func TestHealthPublisher_groupedLate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.node.Announce(testPHX)
		h.setChecks(testPHX, faultCheck("comms", healthpb.HealthCheck_NORMAL))
		h.setChecks(testPHXFlow, returnRange())
		h.run(10)
		h.sleepUntil(healthSettleMax + time.Second)
		assert.Equal(t, map[string]string{"healthbounds:chw-return": "CHW Return Temperature Range"}, h.lastSet(testPHXFlow),
			"a device never announced goes as itself once the hold is over")
		assert.Equal(t, map[string]string{"comms": "comms"}, h.lastSet(testPHX))

		h.node.Announce(testPHXFlow, node.HasTrait(temperaturepb.TraitName))
		h.sleepUntil(healthSettleMax + 2*time.Second)
		assert.Empty(t, h.lastSet(testPHXFlow), "its own ref is emptied")
		assert.Len(t, h.sink.sets(testPHXFlow), 2)
		assert.Equal(t, map[string]string{
			"comms": "comms",
			"chw-primary-flow/healthbounds:chw-return": "chw-primary-flow: CHW Return Temperature Range",
		}, h.lastSet(testPHX))
	})
}

// TestHealthPublisher_groupsKeepThroughReannounce checks a driver reconfiguring, which removes and
// re-announces its devices, doesn't move children's checks onto their own refs meanwhile.
func TestHealthPublisher_groupsKeepThroughReannounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		undoParent := h.node.Announce(testPHX)
		undoChild := h.node.Announce(testPHXFlow, node.HasTrait(temperaturepb.TraitName))
		h.setChecks(testPHXFlow, returnRange())
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(""), 1)

		undoChild()
		undoParent()
		h.sleepUntil(healthSettleQuiet + 2*time.Second)
		h.node.Announce(testPHX)
		h.node.Announce(testPHXFlow, node.HasTrait(temperaturepb.TraitName))
		h.sleepUntil(healthSettleQuiet + 3*time.Second)
		assert.Len(t, h.sink.sets(""), 1, "nothing to send")
	})
}

func TestHealthPublisher_groupsRespectZones(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.node.Announce("plant", node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.node.Announce("plant/temp", node.HasTrait(temperaturepb.TraitName))
		h.node.Announce("phx")
		h.node.Announce("phx/zone", node.HasTrait(temperaturepb.TraitName), node.HasDeviceType(metadatapb.Metadata_ZONE))
		h.setChecks("plant", faultCheck("c", healthpb.HealthCheck_NORMAL))
		h.setChecks("plant/temp", returnRange())
		h.setChecks("phx/zone", returnRange())
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)

		assert.Empty(t, h.sink.sets("plant"), "zones are left out")
		assert.Equal(t, map[string]string{"healthbounds:chw-return": "CHW Return Temperature Range"}, h.lastSet("plant/temp"),
			"a zone's child isn't grouped into it")
		assert.Empty(t, h.sink.sets("phx/zone"), "a zone isn't grouped, and is left out")
		assert.Empty(t, h.sink.sets("phx"))
	})
}

// TestHealthPublisher_notGrouped checks children go as themselves when nothing is grouped.
func TestHealthPublisher_notGrouped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.announcePHX()
		h.setChecks(testPHXFlow, returnRange())
		h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		assert.Equal(t, map[string]string{"healthbounds:chw-return": "CHW Return Temperature Range"}, h.lastSet(testPHXFlow))
		assert.Empty(t, h.sink.sets(testPHX))
	})
}

// TestHealthPublisher_groupedRestartSendsNothing checks a restart doesn't resend a grouped parent's
// set (SCB-1532): it is rebuilt from its children before the saved acknowledgements are dropped.
func TestHealthPublisher_groupedRestartSendsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.announcePHX()
		h.setChecks(testPHX, faultCheck("comms", healthpb.HealthCheck_NORMAL))
		h.setChecks(testPHXFlow, returnRange())
		h.setChecks(testPHXReturn, returnRange())
		stop := h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(""), 1)
		stop()
		ack, ok := h.store.(*memAckStore).get(testPHX)
		require.True(t, ok, "the parent's set is saved")
		_, ok = h.store.(*memAckStore).get(testPHXFlow)
		require.False(t, ok, "a child has nothing of its own to save")

		h.run(10)
		h.sleepUntil(healthFirstManifestDelay)
		assert.Len(t, h.sink.sets(""), 1, "nothing resent")
		var listed string
		for _, e := range h.sink.manifests()[len(h.sink.manifests())-1].manifest.Resources {
			if e.NameHash == nameHash(testPHX) {
				listed = e.Hash
			}
		}
		assert.Equal(t, ack.Hash, listed, "the manifest lists the saved set")
	})
}

// TestHealthPublisher_groupingChildDropsItsSavedAck checks turning grouping on drops a child's
// saved acknowledgement from before, so its ref leaves the manifest and Connect disposes of its set.
func TestHealthPublisher_groupingChildDropsItsSavedAck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHealthHarness(t)
		h.announcePHX()
		h.setChecks(testPHXFlow, returnRange())
		stop := h.run(10)
		h.sleepUntil(healthSettleQuiet + time.Second)
		require.Len(t, h.sink.sets(testPHXFlow), 1)
		stop()
		_, ok := h.store.(*memAckStore).get(testPHXFlow)
		require.True(t, ok)

		h.groupTraits = []string{string(temperaturepb.TraitName)}
		h.run(10)
		h.sleepUntil(healthFirstManifestDelay)
		assert.Len(t, h.sink.sets(testPHXFlow), 1, "no empty set: the manifest leaving it out is enough")
		require.Len(t, h.sink.sets(testPHX), 1)
		assert.ElementsMatch(t, []string{nameHash(testPHX)}, h.manifestNameHashes())
		_, ok = h.store.(*memAckStore).get(testPHXFlow)
		assert.False(t, ok, "the child's saved acknowledgement is dropped")
	})
}
