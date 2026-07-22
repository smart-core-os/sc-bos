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

// messageType classifies an MQTT topic for the points list export. Event topics
// carry telemetry (pointset events), state topics carry device status, and metadata
// topics carry the declared device model; anything else is reported as "other".
func messageType(topic string) string {
	switch {
	case isEventTopic(topic):
		return "event"
	case strings.Contains(topic, "/state"):
		return "state"
	case strings.Contains(topic, "metadata"):
		return "metadata"
	default:
		return "other"
	}
}

// exportRecord is the most recent message observed for a single MQTT topic.
type exportRecord struct {
	sourceName string
	topic      string
	payload    string
	firstSeen  time.Time
	lastSeen   time.Time
	count      int64
}

// exportCollector records the distinct messages the udmi automation publishes,
// keyed by MQTT topic, so they can be exported as a points list. It keeps the
// latest payload per topic (uniqueness is by topic) along with first/last-seen
// times and a message count. It is safe for concurrent use.
//
// A fresh collector is created for each config generation so that reconfiguring the
// automation resets the captured points: some drivers declare their points
// statically, so a new config can change which points a device exposes.
type exportCollector struct {
	now func() time.Time

	mu      sync.Mutex
	byTopic map[string]*exportRecord
}

func newExportCollector(now func() time.Time) *exportCollector {
	if now == nil {
		now = time.Now
	}
	return &exportCollector{now: now, byTopic: make(map[string]*exportRecord)}
}

// Record captures a message published for topic by the named source. The latest
// payload is kept, replacing any previous payload for the same topic.
func (c *exportCollector) Record(sourceName, topic, payload string) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	rec, ok := c.byTopic[topic]
	if !ok {
		rec = &exportRecord{topic: topic, firstSeen: now}
		c.byTopic[topic] = rec
	}
	rec.sourceName = sourceName
	rec.payload = payload
	rec.lastSeen = now
	rec.count++
}

// Snapshot returns a copy of the collected records, ordered by topic.
func (c *exportCollector) Snapshot() []*exportRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*exportRecord, 0, len(c.byTopic))
	for _, rec := range c.byTopic {
		clone := *rec
		out = append(out, &clone)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].topic < out[j].topic })
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
