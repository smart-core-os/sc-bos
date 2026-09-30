package hikcentral

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

func Test_marshalUDMIPayload(t *testing.T) {
	msg := &CameraState{
		CamState:     true,
		CamFlt:       false,
		CamAim:       nil,
		CamOcc:       "",
		CamVideo:     "",
		CamStateTime: time.Date(2023, 6, 15, 17, 32, 0, 0, time.UTC),
		CamFltTime:   time.Time{},
	}
	want := `{` +
		`"CamFlt":{"present_value":false},` +
		`"CamFltTime":{"present_value":"0001-01-01T00:00:00Z"},` +
		`"CamState":{"present_value":true},` +
		`"CamStateTime":{"present_value":"2023-06-15T17:32:00Z"}` +
		`}`
	got, err := marshalUDMIPayload(msg)
	if err != nil {
		t.Fatalf("marshalUDMIPayload() error = %v", err)
	}
	if string(got) != want {
		t.Errorf("marshalUDMIPayload() got = %s, want %v", got, want)
	}
}

// TestCamera_GetExportMessage checks each point stays current for twice the interval
// of the poll that owns it, however recently the other polls succeeded.
func TestCamera_GetExportMessage(t *testing.T) {
	d := func(d time.Duration) *jsontypes.Duration { return &jsontypes.Duration{Duration: d} }
	polls := &config.Settings{InfoPoll: d(5 * time.Minute), OccupancyPoll: d(time.Minute), EventsPoll: d(30 * time.Second), StreamPoll: d(time.Minute)}
	start := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	now := start
	cam := NewCamera(nil, zap.NewNop(), &config.Camera{Name: "cam/01", Topic: "site/cam01"}, nil, polls)
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
	// the keys are as PullExportMessages has always published them
	check("once each poll has read", map[string]udmiPoint{
		"CamState":     {PresentValue: true},
		"CamStateTime": {PresentValue: start},
		"camOcc":       {PresentValue: "7"},
		"CamFlt":       {PresentValue: false},
		"CamFltTime":   {PresentValue: start},
	})

	// only the events poll keeps succeeding
	now = start.Add(2 * time.Minute)
	cam.updateFault(t.Context(), true)
	faultAt := now
	now = now.Add(time.Second)
	check("once occupancy is stale", map[string]udmiPoint{
		"CamState":     {PresentValue: true},
		"CamStateTime": {PresentValue: start},
		"CamFlt":       {PresentValue: true},
		"CamFltTime":   {PresentValue: faultAt},
	})
	now = start.Add(10 * time.Minute)
	cam.updateFault(t.Context(), true)
	faultAt = now
	now = now.Add(time.Second)
	check("once info is stale", map[string]udmiPoint{
		"CamFlt":     {PresentValue: true},
		"CamFltTime": {PresentValue: faultAt},
	})

	// then that stops too
	now = now.Add(time.Minute)
	check("once every poll is stale", nil)
}
