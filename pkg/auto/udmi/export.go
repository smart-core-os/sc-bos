package udmi

import (
	"context"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// messageType classifies an MQTT topic for the points list export by matching whole path
// segments (not substrings, so a site or device segment that merely contains "event",
// "state" or "metadata" isn't misclassified). Event topics carry telemetry (pointset
// events), state topics carry device status, and metadata topics carry the declared device
// model; anything else is reported as "other".
func messageType(topic string) string {
	for _, seg := range strings.Split(topic, "/") {
		switch {
		case seg == "event" || seg == "events":
			return "event"
		case seg == "state":
			return "state"
		case seg == "metadata" || strings.HasPrefix(seg, "metadata."):
			return "metadata"
		}
	}
	return "other"
}

// exportRecord is the most recent message observed for a single (source, topic) pair.
type exportRecord struct {
	sourceName string
	topic      string
	payload    string
	firstSeen  time.Time
	lastSeen   time.Time
	count      int64
}

// recordKey identifies a collected record by the source that produced it and the topic it
// was published to, so two sources publishing the same topic don't overwrite each other.
type recordKey struct {
	source string
	topic  string
}

// exportCollector records the distinct messages the udmi automation publishes, keyed by
// source and MQTT topic, so they can be exported as a points list. It keeps the latest
// payload per (source, topic) along with first/last-seen times and a message count. It is
// safe for concurrent use.
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

// Record captures a message published for topic by the named source. The latest payload is
// kept, replacing any previous payload for the same (source, topic) pair.
func (c *exportCollector) Record(sourceName, topic, payload string) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	key := recordKey{source: sourceName, topic: topic}
	rec, ok := c.byKey[key]
	if !ok {
		rec = &exportRecord{sourceName: sourceName, topic: topic, firstSeen: now}
		c.byKey[key] = rec
	}
	rec.payload = payload
	rec.lastSeen = now
	rec.count++
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
	sort.Slice(out, func(i, j int) bool {
		if out[i].topic != out[j].topic {
			return out[i].topic < out[j].topic
		}
		return out[i].sourceName < out[j].sourceName
	})
	return out
}

// exportServer implements udmipb.UdmiExportApiServer, exposing the merged snapshots of one
// or more exportCollectors over gRPC. The request name is used only for node routing.
//
// A server has a collector per udmi automation announced against the same node, so its
// snapshot covers everything the node publishes. It is safe for concurrent use.
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
	messages := make([]*udmipb.ExportedMessage, 0, len(records))
	for _, rec := range records {
		messages = append(messages, &udmipb.ExportedMessage{
			SourceName:  rec.sourceName,
			Topic:       rec.topic,
			MessageType: messageType(rec.topic),
			Payload:     rec.payload,
			FirstSeen:   timestamppb.New(rec.firstSeen),
			LastSeen:    timestamppb.New(rec.lastSeen),
			Count:       rec.count,
		})
	}
	return &udmipb.ListExportedPointsResponse{Messages: messages}, nil
}

// mergeRecords combines the snapshots of collectors into a single set of records ordered by
// topic then source, as exportCollector.Snapshot orders its own.
//
// Two collectors report the same (source, topic) when a node publishes the same device to
// more than one broker. ExportedMessage has no broker field, so the brokers can't be told
// apart in the response either way: the records are merged, keeping the most recent payload
// and spanning both collectors' first/last-seen and counts.
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
			if rec.firstSeen.Before(existing.firstSeen) {
				existing.firstSeen = rec.firstSeen
			}
			existing.count += rec.count
		}
	}
	out := slices.Collect(maps.Values(byKey))
	sort.Slice(out, func(i, j int) bool {
		if out[i].topic != out[j].topic {
			return out[i].topic < out[j].topic
		}
		return out[i].sourceName < out[j].sourceName
	})
	return out
}
