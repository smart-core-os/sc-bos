package gallagher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
)

const testZonePrefix = "test/access-control"

// zoneFixture is one access zone the fake Command Centre should serve. zoneCount is the raw
// JSON for that property, so a test can distinguish an absent count from a zero one.
type zoneFixture struct {
	id        string
	name      string
	zoneCount string // e.g. `, "zoneCount": 7`, or "" to omit the property entirely
}

// newFakeCommandCentre serves /access_zones and the per-zone detail endpoints from the given
// fixtures. Tests can swap the fixtures between refreshes to simulate a changing count.
//
// The production newHttpClient only exists to set up mTLS, so a Client can be built directly
// against a plain test server.
func newFakeCommandCentre(t *testing.T, zones *[]zoneFixture) *Client {
	t.Helper()

	mux := http.NewServeMux()
	var baseURL string

	mux.HandleFunc("/access_zones", func(w http.ResponseWriter, _ *http.Request) {
		results := make([]string, 0, len(*zones))
		for _, z := range *zones {
			results = append(results, fmt.Sprintf(`{"id": %q, "name": %q, "href": "%s/access_zones/%s"}`,
				z.id, z.name, baseURL, z.id))
		}
		writeJSON(t, w, fmt.Sprintf(`{"results": [%s]}`, strings.Join(results, ",")))
	})

	mux.HandleFunc("/access_zones/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/access_zones/"):]
		for _, z := range *zones {
			if z.id != id {
				continue
			}
			writeJSON(t, w, fmt.Sprintf(`{"id": %q, "name": %q, "href": "%s/access_zones/%s", "status": "secure"%s}`,
				z.id, z.name, baseURL, z.id, z.zoneCount))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	// unstarted so baseURL is set before any handler can read it
	srv := httptest.NewUnstartedServer(mux)
	baseURL = "http://" + srv.Listener.Addr().String()
	srv.Start()
	t.Cleanup(srv.Close)

	return &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if !json.Valid([]byte(body)) {
		t.Fatalf("fixture is not valid JSON: %s", body)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

// newTestZoneController wires a controller to a fake Command Centre and a real node, so
// assertions go through the announced gRPC API rather than the controller's internals.
func newTestZoneController(t *testing.T, zones *[]zoneFixture) (*AccessZoneController, *node.Node) {
	t.Helper()
	client := newFakeCommandCentre(t, zones)
	cc := newCardholderController(nil, "", zap.NewNop())
	azc := newAccessZoneController(client, cc, zap.NewNop())
	return azc, node.New("test")
}

// getOccupancy reads the zone's occupancy through the node, i.e. only succeeds if the
// OccupancySensor trait was actually announced.
func getOccupancy(t *testing.T, n *node.Node, zoneID string) (*occupancysensorpb.Occupancy, error) {
	t.Helper()
	client := occupancysensorpb.NewOccupancySensorApiClient(n.ClientConn())
	return client.GetOccupancy(context.Background(), &occupancysensorpb.GetOccupancyRequest{
		Name: fmt.Sprintf("%s/access_zones/%s", testZonePrefix, zoneID),
	})
}

// assertNoOccupancy checks the zone has no reachable OccupancySensor. The node reports
// Unimplemented when nothing on it has ever registered the service and NotFound when the
// service exists but this name doesn't serve it; both mean "no occupancy sensor here", and
// which one we get depends on the other devices in the test rather than on the zone.
func assertNoOccupancy(t *testing.T, n *node.Node, zoneID string) {
	t.Helper()
	_, err := getOccupancy(t, n, zoneID)
	switch status.Code(err) {
	case codes.NotFound, codes.Unimplemented:
	default:
		t.Fatalf("GetOccupancy error = %v, want NotFound or Unimplemented", err)
	}
}

func TestAccessZoneOccupancy(t *testing.T) {
	tests := []struct {
		name      string
		zoneCount string
		wantCount int32
		wantState occupancysensorpb.Occupancy_State
	}{
		{
			name:      "counted zone with people",
			zoneCount: `, "zoneCount": 7`,
			wantCount: 7,
			wantState: occupancysensorpb.Occupancy_OCCUPIED,
		},
		{
			name:      "counted zone that is empty",
			zoneCount: `, "zoneCount": 0`,
			wantCount: 0,
			wantState: occupancysensorpb.Occupancy_UNOCCUPIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zones := []zoneFixture{{id: "1", name: "Lobby", zoneCount: tt.zoneCount}}
			azc, n := newTestZoneController(t, &zones)

			if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
				t.Fatalf("refreshAccessZones: %v", err)
			}

			got, err := getOccupancy(t, n, "1")
			if err != nil {
				t.Fatalf("GetOccupancy: %v", err)
			}
			if got.PeopleCount != tt.wantCount {
				t.Errorf("PeopleCount = %d, want %d", got.PeopleCount, tt.wantCount)
			}
			if got.State != tt.wantState {
				t.Errorf("State = %v, want %v", got.State, tt.wantState)
			}
			if got.Confidence != 1 {
				t.Errorf("Confidence = %v, want 1", got.Confidence)
			}
			if got.StateChangeTime == nil {
				t.Error("StateChangeTime is nil, want a timestamp")
			}
		})
	}
}

// A zone without zone counting enabled must not get the trait at all: announcing one would
// leave a sensor reporting zero people forever, which reads as a real, empty zone.
func TestAccessZoneOccupancyNotAnnouncedWithoutZoneCount(t *testing.T) {
	zones := []zoneFixture{{id: "1", name: "Fire Door"}}
	azc, n := newTestZoneController(t, &zones)

	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}

	assertNoOccupancy(t, n, "1")
}

// The trait is announced lazily, so a zone that only starts reporting a count later still
// picks it up on a subsequent refresh.
func TestAccessZoneOccupancyAnnouncedWhenCountAppears(t *testing.T) {
	zones := []zoneFixture{{id: "1", name: "Lobby"}}
	azc, n := newTestZoneController(t, &zones)

	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("first refreshAccessZones: %v", err)
	}
	assertNoOccupancy(t, n, "1")

	zones[0].zoneCount = `, "zoneCount": 4`
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("second refreshAccessZones: %v", err)
	}

	got, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy after count: %v", err)
	}
	if got.PeopleCount != 4 {
		t.Errorf("PeopleCount = %d, want 4", got.PeopleCount)
	}
}

// StateChangeTime should track when the occupied state flipped, not when we last polled.
func TestAccessZoneOccupancyStateChangeTime(t *testing.T) {
	zones := []zoneFixture{{id: "1", name: "Lobby", zoneCount: `, "zoneCount": 7`}}
	azc, n := newTestZoneController(t, &zones)

	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}
	first, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}

	// a different count in the same state must not move the timestamp
	zones[0].zoneCount = `, "zoneCount": 9`
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}
	sameState, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}
	if sameState.PeopleCount != 9 {
		t.Errorf("PeopleCount = %d, want 9", sameState.PeopleCount)
	}
	if !sameState.StateChangeTime.AsTime().Equal(first.StateChangeTime.AsTime()) {
		t.Errorf("StateChangeTime moved while still occupied: %v -> %v",
			first.StateChangeTime.AsTime(), sameState.StateChangeTime.AsTime())
	}

	// emptying the zone is a state change, so the timestamp must move
	zones[0].zoneCount = `, "zoneCount": 0`
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}
	flipped, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}
	if flipped.State != occupancysensorpb.Occupancy_UNOCCUPIED {
		t.Errorf("State = %v, want UNOCCUPIED", flipped.State)
	}
	if !flipped.StateChangeTime.AsTime().After(first.StateChangeTime.AsTime()) {
		t.Errorf("StateChangeTime did not move on state flip: %v -> %v",
			first.StateChangeTime.AsTime(), flipped.StateChangeTime.AsTime())
	}
}

// Zone counting can be turned off in Command Centre on a zone that had it, and the details
// response then simply omits the property. Serving the last count forever is worse than
// serving nothing, because seven people in an empty lobby looks like a live reading rather
// than a stopped one.
//
// Clearing ZoneCount before unmarshalling was only half of this: setOccupancy returns early
// on a nil count, so the model kept its value and the trait stayed announced.
func TestAccessZoneOccupancyWithdrawnWhenCountDisappears(t *testing.T) {
	zones := []zoneFixture{{id: "1", name: "Lobby", zoneCount: `, "zoneCount": 7`}}
	azc, n := newTestZoneController(t, &zones)

	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("first refreshAccessZones: %v", err)
	}
	got, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}
	if got.PeopleCount != 7 {
		t.Fatalf("PeopleCount = %d, want 7", got.PeopleCount)
	}

	zones[0].zoneCount = "" // counting turned off
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("second refreshAccessZones: %v", err)
	}
	assertNoOccupancy(t, n, "1")

	// And it comes back if counting is turned on again, since announceOccupancy is lazy.
	zones[0].zoneCount = `, "zoneCount": 3`
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("third refreshAccessZones: %v", err)
	}
	again, err := getOccupancy(t, n, "1")
	if err != nil {
		t.Fatalf("GetOccupancy after count returned: %v", err)
	}
	if again.PeopleCount != 3 {
		t.Errorf("PeopleCount = %d, want 3", again.PeopleCount)
	}
}

// A zone removed from Gallagher must have all its features unannounced, occupancy included.
func TestAccessZoneOccupancyUnannouncedWhenZoneRemoved(t *testing.T) {
	zones := []zoneFixture{{id: "1", name: "Lobby", zoneCount: `, "zoneCount": 7`}}
	azc, n := newTestZoneController(t, &zones)

	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}
	if _, err := getOccupancy(t, n, "1"); err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}

	zones = nil
	if err := azc.refreshAccessZones(n, testZonePrefix); err != nil {
		t.Fatalf("refreshAccessZones: %v", err)
	}
	assertNoOccupancy(t, n, "1")
}
