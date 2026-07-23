package udmi

import (
	"context"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

func TestExportCollector_Record(t *testing.T) {
	now := time.Unix(1000, 0)
	c := newExportCollector(func() time.Time { return now })

	const eventTopic = "site/dev1/event/pointset/points"
	const stateTopic = "site/dev1/state"

	c.Record("dev1", eventTopic, `{"points":{"a":{"present_value":1}}}`)
	now = now.Add(time.Second)
	// a second message on the same topic keeps one record but updates payload/count/lastSeen
	c.Record("dev1", eventTopic, `{"points":{"a":{"present_value":2}}}`)
	c.Record("dev1", stateTopic, `{"system":{}}`)

	snap := c.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 records (one per topic), got %d", len(snap))
	}

	// Snapshot is ordered by topic: eventTopic ("...event...") sorts before stateTopic ("...state").
	got := snap[0]
	if got.topic != eventTopic {
		t.Fatalf("expected first record topic %q, got %q", eventTopic, got.topic)
	}
	if got.sourceName != "dev1" {
		t.Errorf("source name = %q, want dev1", got.sourceName)
	}
	if got.count != 2 {
		t.Errorf("count = %d, want 2", got.count)
	}
	if got.payload != `{"points":{"a":{"present_value":2}}}` {
		t.Errorf("payload = %q, want the latest payload", got.payload)
	}
	if !got.firstSeen.Equal(time.Unix(1000, 0)) {
		t.Errorf("firstSeen = %v, want the first record time", got.firstSeen)
	}
	if !got.lastSeen.Equal(time.Unix(1001, 0)) {
		t.Errorf("lastSeen = %v, want the latest record time", got.lastSeen)
	}
}

func TestHandleMessages_RecordsToCollector(t *testing.T) {
	c := newExportCollector(func() time.Time { return time.Unix(3000, 0) })

	changes := make(chan *udmipb.PullExportMessagesResponse, 3)
	changes <- &udmipb.PullExportMessagesResponse{Message: &udmipb.MqttMessage{
		Topic:   "site/dev/event/pointset/points",
		Payload: `{"points":{"a":{"present_value":1}}}`,
	}}
	changes <- &udmipb.PullExportMessagesResponse{Message: nil} // nil messages are skipped, not recorded
	changes <- &udmipb.PullExportMessagesResponse{Message: &udmipb.MqttMessage{
		Topic:   "site/dev/state",
		Payload: `{"system":{}}`,
	}}
	close(changes)

	var published int
	pub := PublisherFunc(func(context.Context, string, any) error {
		published++
		return nil
	})

	if err := handleMessages(context.Background(), "dev", changes, pub, c); err != nil {
		t.Fatalf("handleMessages: %v", err)
	}
	if published != 2 {
		t.Errorf("published = %d, want 2 (nil message skipped)", published)
	}
	snap := c.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("collector captured %d topics, want 2", len(snap))
	}
	for _, rec := range snap {
		if rec.sourceName != "dev" {
			t.Errorf("record %q has source %q, want dev", rec.topic, rec.sourceName)
		}
	}
}

func TestMessageType(t *testing.T) {
	cases := map[string]string{
		"site/dev1/event/pointset/points": "event",
		"site/dev1/events/pointset":       "event",
		"site/dev1/state":                 "state",
		"site/dev1/metadata.json":         "metadata",
		"site/dev1/something":             "other",
		// segments that merely contain the keywords as substrings must not be misclassified
		"site/event-log-1/state":     "state",
		"metadata-panel/dev1/config": "other",
	}
	for topic, want := range cases {
		if got := messageType(topic); got != want {
			t.Errorf("messageType(%q) = %q, want %q", topic, got, want)
		}
	}
}

func TestExportServer_ListExportedPoints(t *testing.T) {
	now := time.Unix(2000, 0)
	c := newExportCollector(func() time.Time { return now })
	c.Record("dev1", "site/dev1/state", `{"system":{}}`)

	srv := &exportServer{collector: c}
	resp, err := srv.ListExportedPoints(context.Background(), &udmipb.ListExportedPointsRequest{Name: "udmi"})
	if err != nil {
		t.Fatalf("ListExportedPoints: %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	msg := resp.Messages[0]
	if msg.SourceName != "dev1" || msg.Topic != "site/dev1/state" || msg.MessageType != "state" {
		t.Errorf("unexpected message: %+v", msg)
	}
	if msg.Count != 1 {
		t.Errorf("count = %d, want 1", msg.Count)
	}
	if msg.FirstSeen.AsTime().Unix() != 2000 || msg.LastSeen.AsTime().Unix() != 2000 {
		t.Errorf("unexpected timestamps: first=%v last=%v", msg.FirstSeen.AsTime(), msg.LastSeen.AsTime())
	}
}
