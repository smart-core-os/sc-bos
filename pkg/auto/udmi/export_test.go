package udmi

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

func TestExportCollector_Record(t *testing.T) {
	now := time.Unix(1000, 0)
	c := newExportCollector(func() time.Time { return now })

	const dev1Topic = "site/dev1/event/pointset/points"
	const dev2Topic = "site/dev2/events/pointset"

	c.Record("dev1", dev1Topic, `{"points":{"a":{"present_value":1}}}`)
	now = now.Add(time.Second)
	// a second message on the same topic keeps one record but updates payload/lastSeen
	c.Record("dev1", dev1Topic, `{"points":{"a":{"present_value":2}}}`)
	c.Record("dev2", dev2Topic, `{"points":{"b":{"present_value":3}}}`)
	// non-pointset topics describe the device rather than its points, so aren't collected
	c.Record("dev1", "site/dev1/state", `{"system":{}}`)
	c.Record("dev1", "site/dev1/metadata.json", `{"version":1}`)

	snap := c.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 records (one per pointset topic), got %d", len(snap))
	}

	// Snapshot is ordered by topic, so dev1's topic sorts first.
	got := snap[0]
	if got.topic != dev1Topic {
		t.Fatalf("expected first record topic %q, got %q", dev1Topic, got.topic)
	}
	if got.sourceName != "dev1" {
		t.Errorf("source name = %q, want dev1", got.sourceName)
	}
	if got.payload != `{"points":{"a":{"present_value":2}}}` {
		t.Errorf("payload = %q, want the latest payload", got.payload)
	}
	if !got.lastSeen.Equal(time.Unix(1001, 0)) {
		t.Errorf("lastSeen = %v, want the latest record time", got.lastSeen)
	}
}

func TestExportCollector_Reset(t *testing.T) {
	c := newExportCollector(func() time.Time { return time.Unix(1000, 0) })
	c.Record("dev1", "site/dev1/event/pointset/points", `{"points":{"a":{"present_value":1}}}`)
	c.Reset()

	if snap := c.Snapshot(); len(snap) != 0 {
		t.Fatalf("Reset left %d records, want none", len(snap))
	}
	// the collector must still be usable afterwards
	c.Record("dev1", "site/dev1/event/pointset/points", `{"points":{"b":{"present_value":2}}}`)
	snap := c.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("captured %d records after Reset, want 1", len(snap))
	}
	if got := parsePoints(snap[0].payload); !slices.Equal(got, []string{"b"}) {
		t.Errorf("points = %v, want [b] — Reset should clear the previous payload", got)
	}
}

func TestMergeRecords(t *testing.T) {
	const sharedTopic = "site/dev1/event/pointset/points"

	// c1 and c2 stand in for two udmi automations publishing dev1 to different brokers,
	// while c2 alone publishes dev2.
	c1 := newExportCollector(func() time.Time { return time.Unix(1000, 0) })
	c1.Record("dev1", sharedTopic, `{"points":{"a":{"present_value":1}}}`)

	c2 := newExportCollector(func() time.Time { return time.Unix(2000, 0) })
	c2.Record("dev1", sharedTopic, `{"points":{"a":{"present_value":9}}}`)
	c2.Record("dev2", "site/dev2/event/pointset/points", `{"points":{"b":{"present_value":2}}}`)

	got := mergeRecords([]*exportCollector{c1, c2})
	if len(got) != 2 {
		t.Fatalf("merged to %d records, want 2 (dev1 deduped, dev2 kept)", len(got))
	}
	// ordered by topic, so dev1's topic ("site/dev1/...") sorts first
	dev1, dev2 := got[0], got[1]
	if dev1.sourceName != "dev1" || dev2.sourceName != "dev2" {
		t.Fatalf("unexpected order: %q then %q", dev1.sourceName, dev2.sourceName)
	}
	if dev1.payload != `{"points":{"a":{"present_value":9}}}` {
		t.Errorf("payload = %q, want the most recently seen payload", dev1.payload)
	}
	if !dev1.lastSeen.Equal(time.Unix(2000, 0)) {
		t.Errorf("lastSeen = %v, want the latest across both collectors", dev1.lastSeen)
	}

	// merging must not disturb the collectors it read from
	if snap := c1.Snapshot(); snap[0].payload != `{"points":{"a":{"present_value":1}}}` {
		t.Errorf("c1 payload = %q after merge, want its own", snap[0].payload)
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
	// everything is published, but only the pointset event contributes to the export
	snap := c.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("collector captured %d topics, want 1", len(snap))
	}
	if snap[0].sourceName != "dev" {
		t.Errorf("record %q has source %q, want dev", snap[0].topic, snap[0].sourceName)
	}
}

func TestIsPointsetEvent(t *testing.T) {
	cases := map[string]bool{
		"site/dev1/event/pointset/points": true,
		"site/dev1/events/pointset":       true,
		"site/dev1/state":                 false,
		"site/dev1/metadata.json":         false,
		"site/dev1/something":             false,
		// segments that merely contain the keywords as substrings must not match
		"site/event-log-1/state":            false,
		"site/dev1/eventual/pointset-stuff": false,
	}
	for topic, want := range cases {
		if got := isPointsetEvent(topic); got != want {
			t.Errorf("isPointsetEvent(%q) = %t, want %t", topic, got, want)
		}
	}
}

func TestBdnsAssetName(t *testing.T) {
	cases := map[string]string{
		"JLL/GB-LON-1BG/AV/AMP-109151/events/pointset": "AMP-109151",
		"site/floor/FCU-LN1-01/event/pointset/points":  "FCU-LN1-01",
		// not a pointset event topic
		"site/dev1/state": "",
		// no segment before the pointset portion
		"/events/pointset": "",
	}
	for topic, want := range cases {
		if got := bdnsAssetName(topic); got != want {
			t.Errorf("bdnsAssetName(%q) = %q, want %q", topic, got, want)
		}
	}
}

func TestParsePoints(t *testing.T) {
	cases := map[string]struct {
		payload string
		want    []string
	}{
		"conformant envelope": {
			payload: `{"points":{"b":{"present_value":2},"a":{"present_value":1}},"timestamp":"2026-07-28T00:00:00Z"}`,
			want:    []string{"a", "b"},
		},
		"empty envelope": {
			payload: `{"points":{}}`,
			want:    nil,
		},
		"bare points map ignores sibling metadata": {
			payload: `{"zone_temp":{"present_value":21.5},"timestamp":"2026-07-28T00:00:00Z","version":1}`,
			want:    []string{"zone_temp"},
		},
		"null envelope falls back to the bare map": {
			payload: `{"points":null,"zone_temp":{"present_value":21.5}}`,
			want:    []string{"zone_temp"},
		},
		"non-pointset payload": {
			payload: `{"system":{"make_model":"acme"}}`,
			want:    nil,
		},
		"not an object": {
			payload: `["a","b"]`,
			want:    nil,
		},
		"not json": {
			payload: `nonsense`,
			want:    nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := parsePoints(tc.payload); !slices.Equal(got, tc.want) {
				t.Errorf("parsePoints(%q) = %v, want %v", tc.payload, got, tc.want)
			}
		})
	}
}

func TestExportServer_ListExportedPoints(t *testing.T) {
	c := newExportCollector(func() time.Time { return time.Unix(2000, 0) })
	c.Record("dev1", "JLL/GB-LON-1BG/AV/AMP-109151/events/pointset", `{"points":{"b":{"present_value":2},"a":{"present_value":1}}}`)
	c.Record("dev1", "site/dev1/state", `{"system":{}}`)

	srv := &exportServer{}
	srv.addCollector(c)
	resp, err := srv.ListExportedPoints(context.Background(), &udmipb.ListExportedPointsRequest{Name: "van/uk/brum/ugs"})
	if err != nil {
		t.Fatalf("ListExportedPoints: %v", err)
	}
	if len(resp.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(resp.Devices))
	}
	got := resp.Devices[0]
	if got.SourceName != "dev1" {
		t.Errorf("source name = %q, want dev1", got.SourceName)
	}
	if got.AssetName != "AMP-109151" {
		t.Errorf("asset name = %q, want AMP-109151", got.AssetName)
	}
	if !slices.Equal(got.Points, []string{"a", "b"}) {
		t.Errorf("points = %v, want [a b]", got.Points)
	}
}
