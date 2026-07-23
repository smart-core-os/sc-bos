package udmi

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
//
// A fresh collector is created for each config generation so that reconfiguring the
// automation resets the captured points: some drivers declare their points
// statically, so a new config can change which points a device exposes.
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

// exportServer implements udmipb.UdmiExportApiServer, exposing an exportCollector's
// snapshot over gRPC. The request name is used only for node routing.
type exportServer struct {
	udmipb.UnimplementedUdmiExportApiServer
	collector *exportCollector
}

func (s *exportServer) ListExportedPoints(_ context.Context, _ *udmipb.ListExportedPointsRequest) (*udmipb.ListExportedPointsResponse, error) {
	if s.collector == nil {
		return nil, status.Error(codes.Unavailable, "no messages collected")
	}
	records := s.collector.Snapshot()
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
