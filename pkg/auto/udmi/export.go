package udmi

import (
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// pointsetEventRe matches the pointset-event portion of a UDMI topic, covering both the
// UDMI standard suffix ".../event/pointset/points" and the shorter ".../events/pointset"
// some drivers use. Whole path segments are matched, so a site or device segment that
// merely contains "event" isn't mistaken for one.
var pointsetEventRe = regexp.MustCompile(`/events?/pointset`)

// isPointsetEvent reports whether topic carries a pointset event, the only message type a
// points list is built from. State, metadata and other topics describe the device rather
// than its points.
func isPointsetEvent(topic string) bool {
	return pointsetEventRe.MatchString(topic)
}

// bdnsAssetName returns the BDNS functional asset name from a pointset event topic: the
// single path segment immediately before the "/event(s)/pointset" portion, e.g.
// "AMP-109151" for "JLL/GB-LON-1BG/AV/AMP-109151/events/pointset". Returns an empty string
// when topic isn't a pointset event topic.
func bdnsAssetName(topic string) string {
	loc := pointsetEventRe.FindStringIndex(topic)
	if loc == nil {
		return ""
	}
	segments := strings.Split(topic[:loc[0]], "/")
	return segments[len(segments)-1]
}

// parsePoints returns the sorted point names carried by a UDMI pointset payload. It handles
// both the conformant envelope ({"points": {name: {...}}}) and a bare points map
// ({name: {"present_value": ...}}, as the mock driver emits). Payloads that aren't a
// pointset yield no points.
func parsePoints(payload string) []string {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return nil
	}
	// Prefer the conformant envelope — every key under it is a point.
	if envelope, ok := parsed["points"]; ok {
		var points map[string]json.RawMessage
		if err := json.Unmarshal(envelope, &points); err == nil && points != nil {
			return slices.Sorted(maps.Keys(points))
		}
	}
	// Fall back to a bare points map: keep only the keys whose value looks like a point
	// value, so sibling metadata keys (timestamp, version, ...) aren't taken for points.
	var names []string
	for name, raw := range parsed {
		var value map[string]json.RawMessage
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if _, ok := value["present_value"]; ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// exportRecord is the most recent pointset event observed for a single (source, topic) pair.
type exportRecord struct {
	sourceName string
	topic      string
	payload    string
	// lastSeen orders payloads for the same (source, topic) when merging collectors.
	lastSeen time.Time
}

// recordKey identifies a collected record by the source that produced it and the topic it
// was published to, so two sources publishing the same topic don't overwrite each other.
type recordKey struct {
	source string
	topic  string
}

// exportCollector records the pointset events the udmi automation publishes, keyed by
// source and MQTT topic, so the points they carry can be exported as a points list. It
// keeps the latest payload per (source, topic). It is safe for concurrent use.
type exportCollector struct {
	now func() time.Time

	mu    sync.Mutex
	byKey map[recordKey]*exportRecord
}

func newExportCollector(now func() time.Time) *exportCollector {
	if now == nil {
		now = time.Now
	}
	return &exportCollector{now: now, byKey: make(map[recordKey]*exportRecord)}
}

// Record captures a message published for topic by the named source, ignoring topics that
// don't carry a pointset event. The latest payload is kept, replacing any previous payload
// for the same (source, topic) pair.
func (c *exportCollector) Record(sourceName, topic, payload string) {
	if !isPointsetEvent(topic) {
		return
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	key := recordKey{source: sourceName, topic: topic}
	rec, ok := c.byKey[key]
	if !ok {
		rec = &exportRecord{sourceName: sourceName, topic: topic}
		c.byKey[key] = rec
	}
	rec.payload = payload
	rec.lastSeen = now
}

// Reset discards every collected record.
//
// The automation calls this each time it applies a config so that reconfiguring resets the
// captured points: some drivers declare their points statically, so a new config can
// change which points a device exposes.
func (c *exportCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byKey = make(map[recordKey]*exportRecord)
}

// Snapshot returns a copy of the collected records, ordered by topic then source.
func (c *exportCollector) Snapshot() []*exportRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*exportRecord, 0, len(c.byKey))
	for _, rec := range c.byKey {
		clone := *rec
		out = append(out, &clone)
	}
	slices.SortFunc(out, compareRecords)
	return out
}

// exportServer implements udmipb.UdmiExportApiServer, turning the merged snapshots of one
// or more exportCollectors into points list rows. The request name is used only for node
// routing.
//
// A server has a collector per udmi automation announced against the same node, so its
// snapshot covers every device the node publishes. It is safe for concurrent use.
type exportServer struct {
	udmipb.UnimplementedUdmiExportApiServer

	mu         sync.Mutex
	collectors []*exportCollector
}

func (s *exportServer) addCollector(c *exportCollector) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collectors = append(s.collectors, c)
}

// removeCollector drops the first entry equal to c, reporting how many collectors remain.
func (s *exportServer) removeCollector(c *exportCollector) (remaining int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.collectors {
		if existing == c {
			s.collectors = append(s.collectors[:i], s.collectors[i+1:]...)
			break
		}
	}
	return len(s.collectors)
}

func (s *exportServer) snapshot() []*exportCollector {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.collectors)
}

func (s *exportServer) ListExportedPoints(_ context.Context, _ *udmipb.ListExportedPointsRequest) (*udmipb.ListExportedPointsResponse, error) {
	records := mergeRecords(s.snapshot())
	devices := make([]*udmipb.DevicePoints, 0, len(records))
	for _, rec := range records {
		devices = append(devices, &udmipb.DevicePoints{
			SourceName: rec.sourceName,
			Topic:      rec.topic,
			AssetName:  bdnsAssetName(rec.topic),
			Points:     parsePoints(rec.payload),
		})
	}
	return &udmipb.ListExportedPointsResponse{Devices: devices}, nil
}

// mergeRecords combines the snapshots of collectors into a single set of records ordered by
// topic then source, as exportCollector.Snapshot orders its own.
//
// Two collectors report the same (source, topic) when a node publishes the same device to
// more than one broker. Both carry the same points, so the most recently seen payload wins.
func mergeRecords(collectors []*exportCollector) []*exportRecord {
	if len(collectors) == 1 {
		return collectors[0].Snapshot() // already merged and ordered
	}
	byKey := make(map[recordKey]*exportRecord)
	for _, c := range collectors {
		for _, rec := range c.Snapshot() {
			key := recordKey{source: rec.sourceName, topic: rec.topic}
			existing, ok := byKey[key]
			if !ok {
				byKey[key] = rec // Snapshot already returns copies we own
				continue
			}
			if rec.lastSeen.After(existing.lastSeen) {
				existing.payload = rec.payload
				existing.lastSeen = rec.lastSeen
			}
		}
	}
	return slices.SortedFunc(maps.Values(byKey), compareRecords)
}

func compareRecords(a, b *exportRecord) int {
	return cmp.Or(
		strings.Compare(a.topic, b.topic),
		strings.Compare(a.sourceName, b.sourceName),
	)
}
