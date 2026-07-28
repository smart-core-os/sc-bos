package udmi

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// TestExportRegistry_Add covers a node running two udmi automations — the two-brokers case.
// Both automations' points must appear under the node's name, and the API must stop being
// served only once both have gone.
func TestExportRegistry_Add(t *testing.T) {
	const nodeName = "van/uk/brum/ugs"
	n := node.New(nodeName)
	client := udmipb.NewUdmiExportApiClient(n.ClientConn())
	list := func(t *testing.T) []*udmipb.ExportedMessage {
		t.Helper()
		res, err := client.ListExportedPoints(t.Context(), &udmipb.ListExportedPointsRequest{Name: nodeName})
		if err != nil {
			t.Fatalf("ListExportedPoints: %v", err)
		}
		return res.Messages
	}

	r := &exportRegistry{}

	c1 := newExportCollector(func() time.Time { return time.Unix(1000, 0) })
	c1.Record("dev1", "site/dev1/event/pointset/points", `{"points":{"a":{"present_value":1}}}`)
	remove1 := r.Add(n, c1)

	if got := list(t); len(got) != 1 || got[0].SourceName != "dev1" {
		t.Fatalf("after first automation: %+v, want dev1 only", got)
	}

	// a second automation on the same node shares the announcement rather than colliding
	c2 := newExportCollector(func() time.Time { return time.Unix(2000, 0) })
	c2.Record("dev2", "site/dev2/event/pointset/points", `{"points":{"b":{"present_value":2}}}`)
	remove2 := r.Add(n, c2)

	got := list(t)
	if len(got) != 2 {
		t.Fatalf("after second automation: %d messages, want 2", len(got))
	}
	if got[0].SourceName != "dev1" || got[1].SourceName != "dev2" {
		t.Errorf("sources = %q, %q; want dev1, dev2", got[0].SourceName, got[1].SourceName)
	}

	// removing one automation leaves the other serving
	remove1()
	if got := list(t); len(got) != 1 || got[0].SourceName != "dev2" {
		t.Fatalf("after removing the first automation: %+v, want dev2 only", got)
	}

	// removing the last one takes the announcement with it
	remove2()
	_, err := client.ListExportedPoints(t.Context(), &udmipb.ListExportedPointsRequest{Name: nodeName})
	if code := status.Code(err); code != codes.NotFound && code != codes.Unimplemented {
		t.Fatalf("ListExportedPoints after last removal: err = %v (code %s), want NotFound or Unimplemented", err, code)
	}
}

// TestExportRegistry_Readd covers a reconfigure: the automation re-registers the same
// collector, and the announcement must come back.
func TestExportRegistry_Readd(t *testing.T) {
	const nodeName = "van/uk/brum/ugs"
	n := node.New(nodeName)
	client := udmipb.NewUdmiExportApiClient(n.ClientConn())
	r := &exportRegistry{}

	c := newExportCollector(func() time.Time { return time.Unix(1000, 0) })
	r.Add(n, c)()

	remove := r.Add(n, c)
	defer remove()
	c.Record("dev1", "site/dev1/event/pointset/points", `{"points":{"a":{"present_value":1}}}`)

	res, err := client.ListExportedPoints(t.Context(), &udmipb.ListExportedPointsRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("ListExportedPoints after re-add: %v", err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(res.Messages))
	}
}
