package hikcentral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// Test_marshalUDMIPayload checks points are named by their json tags, as the MQTT payload names them, and that
// fields tagged json:"-" aren't published (SCB-1463).
func Test_marshalUDMIPayload(t *testing.T) {
	tests := []struct {
		name string
		msg  *CameraState
		want string
	}{
		{
			name: "empty omitempty fields left out",
			msg: &CameraState{
				CamState: true,
			},
			want: `{"camFlt":{"present_value":false},"camState":{"present_value":true}}`,
		},
		{
			name: "every point",
			msg: &CameraState{
				CamState: true,
				CamFlt:   true,
				CamAim:   &PTZ{Pan: "1"},
				CamOcc:   "7",
				CamVideo: "rtsp://cam",
			},
			want: `{` +
				`"camAim":{"present_value":{"pan":"1"}},` +
				`"camFlt":{"present_value":true},` +
				`"camOcc":{"present_value":"7"},` +
				`"camState":{"present_value":true},` +
				`"camVideo":{"present_value":"rtsp://cam"}` +
				`}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalUDMIPayload(tt.msg)
			if err != nil {
				t.Fatalf("marshalUDMIPayload() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("marshalUDMIPayload() got = %s, want %s", got, tt.want)
			}
		})
	}
}

// Test_udmiPoints_tags checks points follow encoding/json's reading of the json tag.
func Test_udmiPoints_tags(t *testing.T) {
	type state struct {
		Untagged string
		Named    string `json:"named"`
		Skipped  string `json:"-"`
		OptsOnly string `json:",omitempty"`
	}
	got := udmiPoints(state{Untagged: "a", Named: "b", Skipped: "c"}, nil)
	want := map[string]udmiPoint{
		"Untagged": {PresentValue: "a"},
		"named":    {PresentValue: "b"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("udmiPoints (-want +got):\n%s", diff)
	}
}

// TestCamera_GetExportMessage checks each point stays current for twice the interval
// of the poll that owns it, however recently the other polls succeeded.
func TestCamera_GetExportMessage(t *testing.T) {
	d := func(d time.Duration) *jsontypes.Duration { return &jsontypes.Duration{Duration: d} }
	polls := &config.Settings{InfoPoll: d(5 * time.Minute), OccupancyPoll: d(time.Minute), EventsPoll: d(30 * time.Second), StreamPoll: d(time.Minute)}
	start := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	now := start
	cam := NewCamera(nil, zap.NewNop(), &config.Camera{Name: "cam/01", Topic: "site/cam01"}, nil, polls, nil)
	cam.Now = func() time.Time { return now }
	client := udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, cam))
	check := func(when string, want map[string]udmiPoint) {
		t.Helper()
		msg, err := client.GetExportMessage(t.Context(), &udmipb.GetExportMessageRequest{Name: "cam/01"})
		if want == nil {
			if status.Code(err) != codes.Unavailable {
				t.Errorf("%s: code = %v, want %v (err %v)", when, status.Code(err), codes.Unavailable, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("%s: %v", when, err)
		}
		if got, want := msg.GetTopic(), "site/cam01/event/pointset/points"; got != want {
			t.Errorf("%s: topic = %q, want %q", when, got, want)
		}
		wantJSON, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		if got := msg.GetPayload(); got != string(wantJSON) {
			t.Errorf("%s: payload = %s, want %s", when, got, wantJSON)
		}
	}

	check("before any read", nil)

	cam.updateActive(t.Context(), true)
	cam.updateCount(t.Context(), "7")
	cam.updateFault(t.Context(), false)
	now = start.Add(time.Minute) // at the events poll's limit is still current
	check("once each poll has read", map[string]udmiPoint{
		"camState": {PresentValue: true},
		"camOcc":   {PresentValue: "7"},
		"camFlt":   {PresentValue: false},
	})

	// only the events poll keeps succeeding
	now = start.Add(2 * time.Minute)
	cam.updateFault(t.Context(), true)
	now = now.Add(time.Second)
	check("once occupancy is stale", map[string]udmiPoint{
		"camState": {PresentValue: true},
		"camFlt":   {PresentValue: true},
	})
	now = start.Add(10 * time.Minute)
	cam.updateFault(t.Context(), true)
	now = now.Add(time.Second)
	check("once info is stale", map[string]udmiPoint{
		"camFlt": {PresentValue: true},
	})

	// then that stops too
	now = now.Add(time.Minute)
	check("once every poll is stale", nil)
}

// TestCamera_GetExportMessage_slowWalk checks a point stays current for as long as the poll takes to come back
// round to its camera, when walking every camera takes longer than the poll's interval (SCB-1473).
func TestCamera_GetExportMessage_slowWalk(t *testing.T) {
	start := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	polls := &config.Settings{InfoPoll: &jsontypes.Duration{Duration: time.Minute}}
	newCam := func(cycles *pollCycles, now *time.Time) *Camera {
		cam := NewCamera(nil, zap.NewNop(), &config.Camera{Name: "cam/01", Topic: "site/cam01"}, nil, polls, cycles)
		cam.Now = func() time.Time { return *now }
		return cam
	}
	current := func(cam *Camera) bool {
		t.Helper()
		_, err := cam.GetExportMessage(t.Context(), &udmipb.GetExportMessageRequest{Name: "cam/01"})
		switch status.Code(err) {
		case codes.OK:
			return true
		case codes.Unavailable:
			return false
		}
		t.Fatalf("GetExportMessage: %v", err)
		return false
	}

	t.Run("after a slow walk", func(t *testing.T) {
		cycles := &pollCycles{}
		cycles.begin(infoPoll, start.Add(-5*time.Minute))
		cycles.end(infoPoll, start) // a walk takes 5 minutes, not the 1 minute interval
		now := start
		cam := newCam(cycles, &now)
		cam.updateActive(t.Context(), true)

		now = start.Add(9 * time.Minute)
		if !current(cam) {
			t.Error("stale before the walk could have come back round to the camera twice")
		}
		now = start.Add(10*time.Minute + time.Second)
		if current(cam) {
			t.Error("still current after two walks without a successful poll")
		}
	})

	t.Run("during the first walk", func(t *testing.T) {
		cycles := &pollCycles{}
		cycles.begin(infoPoll, start)
		now := start
		cam := newCam(cycles, &now)
		cam.updateActive(t.Context(), true) // the first camera in a walk that is still going

		now = start.Add(3 * time.Minute)
		if !current(cam) {
			t.Error("stale while the first walk is still visiting the other cameras")
		}
	})

	t.Run("a walk that drags doesn't hide its failures", func(t *testing.T) {
		cycles := &pollCycles{}
		cycles.begin(infoPoll, start.Add(-time.Minute))
		cycles.end(infoPoll, start) // walks normally take the interval
		now := start
		cam := newCam(cycles, &now)
		cam.updateActive(t.Context(), true)

		// the next walk fails on this camera and crawls on through timeouts
		cycles.begin(infoPoll, start.Add(time.Minute))
		now = start.Add(time.Minute)
		cam.pollFailed(infoPoll)
		now = start.Add(30 * time.Minute)
		if current(cam) {
			t.Error("still current during the slow walk that failed on it")
		}
		// finishing that walk mustn't bring the point back
		cycles.end(infoPoll, start.Add(31*time.Minute))
		now = start.Add(31 * time.Minute)
		if current(cam) {
			t.Error("current again once the slow walk that failed on it finished")
		}
		// a successful read brings back the allowance for the walk
		cam.updateActive(t.Context(), true)
		now = now.Add(time.Hour)
		if !current(cam) {
			t.Error("stale within two walks of a successful read")
		}
	})

	t.Run("a stalled walk doesn't keep the cameras it hasn't reached current", func(t *testing.T) {
		cycles := &pollCycles{}
		cycles.begin(infoPoll, start.Add(-time.Minute))
		cycles.end(infoPoll, start)
		now := start
		cam := newCam(cycles, &now)
		cam.updateActive(t.Context(), true)

		// the next walk stalls on the cameras before this one
		cycles.begin(infoPoll, start.Add(time.Minute))
		now = start.Add(30 * time.Minute)
		if current(cam) {
			t.Error("still current because the walk that hasn't reached it is stalled")
		}
	})

	t.Run("one failed poll on a large site is tolerated", func(t *testing.T) {
		cycles := &pollCycles{}
		cycles.begin(infoPoll, start.Add(-10*time.Minute))
		cycles.end(infoPoll, start) // a walk takes 10 minutes
		now := start
		cam := newCam(cycles, &now)
		cam.updateActive(t.Context(), true)

		now = start.Add(10 * time.Minute)
		cam.pollFailed(infoPoll)
		now = start.Add(15 * time.Minute)
		if !current(cam) {
			t.Error("stale after a single failed poll, inside two walks of the read")
		}
		now = start.Add(20*time.Minute + time.Second)
		if current(cam) {
			t.Error("still current two walks after the read with no successful poll")
		}
	})

	t.Run("one failed poll is tolerated", func(t *testing.T) {
		now := start
		cam := newCam(nil, &now)
		cam.updateActive(t.Context(), true)
		now = start.Add(time.Minute)
		cam.pollFailed(infoPoll)
		now = start.Add(90 * time.Second)
		if !current(cam) {
			t.Error("stale after a single failed poll, inside twice the interval")
		}
		now = start.Add(2*time.Minute + time.Second)
		if current(cam) {
			t.Error("still current after twice the interval without a successful poll")
		}
	})
}

// TestWalkCameras checks a walk visits every camera and records how long it took.
func TestWalkCameras(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	cameras := make([]*Camera, 3)
	for i := range cameras {
		cameras[i] = &Camera{}
	}
	var visited []*Camera
	cycles := &pollCycles{}
	walkCameras(t.Context(), cycles, occupancyPoll, cameras, func(c *Camera, _ context.Context) {
		visited = append(visited, c)
		now = now.Add(time.Minute)
	}, func() time.Time { return now })

	if len(visited) != len(cameras) {
		t.Errorf("visited %d cameras, want %d", len(visited), len(cameras))
	}
	if got := cycles.length(occupancyPoll, now.Add(time.Hour)); got != 3*time.Minute {
		t.Errorf("walk length = %v, want %v", got, 3*time.Minute)
	}
	if got := cycles.length(infoPoll, now); got != 0 {
		t.Errorf("a poll that hasn't walked has length %v, want 0", got)
	}
}

// TestCamera_pollFailures checks each poll records a failed read, which is what stops a slow walk being allowed for.
func TestCamera_pollFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	fc, err := healthpb.NewRegistry().ForOwner("test").NewFaultCheck("cam/01", proto.Clone(deviceHealthCheck).(*healthpb.HealthCheck))
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	defer fc.Dispose()
	client := newClient(&config.API{Address: srv.URL, Timeout: &jsontypes.Duration{Duration: 5 * time.Second}}, nil)
	cam := NewCamera(client, zap.NewNop(), &config.Camera{Name: "cam/01", IndexCode: "1"}, fc, &config.Settings{}, nil)

	for _, tt := range []struct {
		p    poll
		read func(*Camera, context.Context)
	}{
		{infoPoll, (*Camera).getInfo},
		{occupancyPoll, (*Camera).getOcc},
		{eventsPoll, (*Camera).getEvents},
		{streamPoll, (*Camera).getStream},
	} {
		tt.read(cam, t.Context())
		cam.lock.Lock()
		failed := cam.failed[tt.p]
		cam.lock.Unlock()
		if !failed {
			t.Errorf("poll %d: a failed read wasn't recorded", tt.p)
		}
	}
}
