package xovis

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/xovis/config"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

func Test_PullExportMessages(t *testing.T) {

	enter := int32(0)
	leave := int32(0)
	e := resource.NewValue(
		resource.WithInitialValue(
			&enterleavesensorpb.EnterLeaveEvent{
				EnterTotal: &enter, LeaveTotal: &leave,
			},
		), resource.WithNoDuplicates(),
	)
	o := resource.NewValue(
		resource.WithInitialValue(
			&occupancysensorpb.Occupancy{
				PeopleCount: 0, State: occupancysensorpb.Occupancy_OCCUPIED,
			},
		), resource.WithNoDuplicates(),
	)

	req := &udmipb.PullExportMessagesRequest{
		Name: "test",
	}

	enterTotal := int32(459)
	leaveTotal := int32(987)

	tests := []struct {
		name         string
		createClient func() udmipb.UdmiServiceClient
		set          func()
		want         EventPoints
	}{
		{
			name: "occupancy",
			createClient: func() udmipb.UdmiServiceClient {
				server := newUdmiServiceServer(nil, e, o, "prefix", nil, nil)
				return udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))
			},
			set: func() {
				o.Set(
					&occupancysensorpb.Occupancy{
						PeopleCount: 459,
						State:       occupancysensorpb.Occupancy_OCCUPIED,
					},
				)
			},
			want: EventPoints{
				DeviceType:     &EventPoint[string]{PresentValue: DriverName},
				OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
				PeopleCount:    &EventPoint[int32]{PresentValue: 459},
			},
		},
		{
			name: "enterleave",
			createClient: func() udmipb.UdmiServiceClient {
				server := newUdmiServiceServer(nil, e, o, "prefix", nil, nil)
				return udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))
			},
			set: func() {
				e.Set(
					&enterleavesensorpb.EnterLeaveEvent{
						EnterTotal: &enterTotal,
						LeaveTotal: &leaveTotal,
					},
				)
			},
			want: EventPoints{
				DeviceType: &EventPoint[string]{PresentValue: DriverName},
				EnterCount: &EventPoint[int32]{PresentValue: enterTotal},
				LeaveCount: &EventPoint[int32]{PresentValue: leaveTotal},
			},
		},
		{
			name: "enterleave_occupancy_nil",
			createClient: func() udmipb.UdmiServiceClient {
				server := newUdmiServiceServer(nil, e, nil, "prefix", nil, nil)
				return udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))
			},
			set: func() {
				e.Set(
					&enterleavesensorpb.EnterLeaveEvent{
						EnterTotal: &enterTotal,
						LeaveTotal: &leaveTotal,
					},
				)
			},
			want: EventPoints{
				DeviceType: &EventPoint[string]{PresentValue: DriverName},
				EnterCount: &EventPoint[int32]{PresentValue: enterTotal},
				LeaveCount: &EventPoint[int32]{PresentValue: leaveTotal},
			},
		},
		{
			name: "occupancy_enterleave_nil",
			createClient: func() udmipb.UdmiServiceClient {
				server := newUdmiServiceServer(nil, nil, o, "prefix", nil, nil)
				return udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))
			},
			set: func() {
				o.Set(
					&occupancysensorpb.Occupancy{
						PeopleCount: 459,
						State:       occupancysensorpb.Occupancy_OCCUPIED,
					},
				)
			},
			want: EventPoints{
				DeviceType:     &EventPoint[string]{PresentValue: DriverName},
				OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
				PeopleCount:    &EventPoint[int32]{PresentValue: 459},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {

				client := tt.createClient()

				ctx := t.Context()

				messages, _ := client.PullExportMessages(ctx, req)
				tt.set()
				time.Sleep(1 * time.Millisecond)
				tt.set()

				m, err := messages.Recv()

				if err != nil {
					t.Fatal("messages.RecvMsg(&pointSetMessage) is nil")
				}

				// take the response payload which should be a valid PointsetEventMessage
				var pointSetMessage PointsetEventMessage
				err = json.Unmarshal([]byte(m.Message.Payload), &pointSetMessage)

				if err != nil {
					t.Fatal("json.Unmarshal failed")
				}

				if res := cmp.Diff(pointSetMessage.Points, tt.want); res != "" {
					t.Fatal("trait does not match " + res)
				}

			},
		)
	}
}

func Test_GetExportMessage(t *testing.T) {
	readAll := func(e, o *resource.Value) func(context.Context) error {
		return func(context.Context) error {
			if e != nil {
				e.Set(&enterleavesensorpb.EnterLeaveEvent{EnterTotal: new(int32(12)), LeaveTotal: new(int32(7))})
			}
			if o != nil {
				o.Set(&occupancysensorpb.Occupancy{PeopleCount: 5, State: occupancysensorpb.Occupancy_OCCUPIED})
			}
			return nil
		}
	}
	deviceType := &EventPoint[string]{PresentValue: DriverName}
	enterLeave := EventPoints{
		DeviceType: deviceType,
		EnterCount: &EventPoint[int32]{PresentValue: 12},
		LeaveCount: &EventPoint[int32]{PresentValue: 7},
	}
	occupancy := EventPoints{
		DeviceType:     deviceType,
		PeopleCount:    &EventPoint[int32]{PresentValue: 5},
		OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
	}
	both := enterLeave
	both.PeopleCount = occupancy.PeopleCount
	both.OccupancyState = occupancy.OccupancyState

	tests := []struct {
		name                     string
		hasEnterLeave, hasOccupy bool
		refresh                  func(e, o *resource.Value) func(context.Context) error
		wantCode                 codes.Code
		want                     EventPoints
	}{
		{name: "both logics", hasEnterLeave: true, hasOccupy: true, refresh: readAll, want: both},
		{name: "enter leave only", hasEnterLeave: true, refresh: readAll, want: enterLeave},
		{name: "occupancy only", hasOccupy: true, refresh: readAll, want: occupancy},
		{
			name:          "read fails",
			hasEnterLeave: true, hasOccupy: true,
			refresh: func(_, _ *resource.Value) func(context.Context) error {
				return func(context.Context) error { return errors.New("connection refused") }
			},
			wantCode: codes.Unavailable,
		},
		{
			name:          "logic misconfigured",
			hasEnterLeave: true, hasOccupy: true,
			refresh: func(_, _ *resource.Value) func(context.Context) error {
				return func(context.Context) error { return errNotInOutLogic }
			},
			wantCode: codes.FailedPrecondition,
		},
		{name: "no refresh", hasEnterLeave: true, hasOccupy: true, wantCode: codes.Unavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e, o *resource.Value
			if tt.hasEnterLeave {
				e = resource.NewValue(resource.WithInitialValue(&enterleavesensorpb.EnterLeaveEvent{}))
			}
			if tt.hasOccupy {
				o = resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}))
			}
			var refresh func(context.Context) error
			if tt.refresh != nil {
				refresh = tt.refresh(e, o)
			}

			server := newUdmiServiceServer(nil, e, o, "prefix", refresh, nil)
			client := udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))
			msg, err := client.GetExportMessage(t.Context(), &udmipb.GetExportMessageRequest{Name: "test"})
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("GetExportMessage code = %v, want %v (err %v)", code, tt.wantCode, err)
			}
			if tt.wantCode != codes.OK {
				return
			}
			checkFullPointset(t, msg, "prefix/events/pointset", tt.want)
		})
	}
}

// fakeSensor serves the live logic API of a single sensor with an occupancy logic
// (id 1) and an in/out logic (id 2).
type fakeSensor struct {
	balance, fw, bw atomic.Int32
	down            atomic.Bool
}

func (f *fakeSensor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.down.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"code": 503, "message": "unavailable"}`))
		return
	}
	res := LiveLogicResponse{Time: time.Now()}
	switch r.URL.Path {
	case "/api/v5/singlesensor/data/live/logics/1":
		res.Logic = LiveLogicData{ID: 1, Counts: []Count{{ID: 0, Name: "balance", Value: int(f.balance.Load())}}}
	case "/api/v5/singlesensor/data/live/logics/2":
		res.Logic = LiveLogicData{ID: 2, Counts: []Count{
			// non-zero ids, so a failed id lookup can't match
			{ID: 1, Name: "fw", Value: int(f.fw.Load())},
			{ID: 2, Name: "bw", Value: int(f.bw.Load())},
		}}
	default:
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

// TestDriver_udmi checks the UDMI server against a sensor: the export stream polls it
// and publishes what it reads, and GetExportMessage reads it live, so a change shows
// up before the next poll, and a sensor that stops answering gives Unavailable.
func TestDriver_udmi(t *testing.T) {
	sensor := &fakeSensor{}
	sensor.balance.Store(4)
	sensor.fw.Store(10)
	sensor.bw.Store(6)
	server := httptest.NewTLSServer(sensor)
	defer server.Close()

	cfg, err := config.ParseConfig([]byte(`{
		"name": "xovis",
		"host": "` + strings.TrimPrefix(server.URL, "https://") + `",
		"username": "admin",
		"password": "secret",
		"devices": [
			{"name": "sensors/01", "occupancy": {"id": 1}, "enterLeave": {"id": 2}, "udmiTopicPrefix": "site/01"}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	n := node.New("test")
	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(n),
		health:      healthpb.NewRegistry().ForOwner("driver:xovis"),
		httpMux:     http.NewServeMux(),
		logger:      zap.NewNop(),
		pushDataBus: &minibus.Bus[PushData]{},
	}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	if err := d.applyConfig(ctx, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	client := udmipb.NewUdmiServiceClient(n.ClientConn())
	req := &udmipb.PullExportMessagesRequest{Name: "sensors/01"}

	// Nothing but the poll the stream starts feeds the values, so the stream only
	// sends anything because of it. Each logic polls separately, so expect a message
	// for each.
	streamCtx, stopStream := context.WithCancel(ctx)
	stream, err := client.PullExportMessages(streamCtx, req)
	if err != nil {
		t.Fatalf("PullExportMessages: %v", err)
	}
	var streamed EventPoints
	for streamed.PeopleCount == nil || streamed.EnterCount == nil {
		res, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if got := res.GetMessage().GetTopic(); got != "site/01/events/pointset" {
			t.Errorf("stream topic = %q, want %q", got, "site/01/events/pointset")
		}
		var msg PointsetEventMessage
		if err := json.Unmarshal([]byte(res.GetMessage().GetPayload()), &msg); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if msg.Points.PeopleCount != nil {
			streamed.PeopleCount, streamed.OccupancyState = msg.Points.PeopleCount, msg.Points.OccupancyState
		}
		if msg.Points.EnterCount != nil {
			streamed.EnterCount, streamed.LeaveCount = msg.Points.EnterCount, msg.Points.LeaveCount
		}
	}
	stopStream()
	wantStreamed := EventPoints{
		EnterCount:     &EventPoint[int32]{PresentValue: 10},
		LeaveCount:     &EventPoint[int32]{PresentValue: 6},
		PeopleCount:    &EventPoint[int32]{PresentValue: 4},
		OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
	}
	if diff := cmp.Diff(wantStreamed, streamed); diff != "" {
		t.Errorf("streamed points (-want +got):\n%s", diff)
	}

	// The next poll is 30s away, so only a live read sees this.
	sensor.balance.Store(0)
	sensor.fw.Store(11)
	msg, err := client.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{Name: "sensors/01"})
	if err != nil {
		t.Fatalf("GetExportMessage: %v", err)
	}
	checkFullPointset(t, msg, "site/01/events/pointset", EventPoints{
		DeviceType:     &EventPoint[string]{PresentValue: DriverName},
		EnterCount:     &EventPoint[int32]{PresentValue: 11},
		LeaveCount:     &EventPoint[int32]{PresentValue: 6},
		PeopleCount:    &EventPoint[int32]{PresentValue: 0},
		OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_UNOCCUPIED.String()},
	})

	sensor.down.Store(true)
	_, err = client.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{Name: "sensors/01"})
	if code := status.Code(err); code != codes.Unavailable {
		t.Errorf("GetExportMessage with the sensor down: code = %v, want %v (err %v)", code, codes.Unavailable, err)
	}
}

// checkFullPointset checks msg is a full (not partial) pointset on topic carrying want.
func checkFullPointset(t *testing.T, msg *udmipb.MqttMessage, topic string, want EventPoints) {
	t.Helper()
	if msg.GetTopic() != topic {
		t.Errorf("topic = %q, want %q", msg.GetTopic(), topic)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(msg.GetPayload()), &raw); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := raw["partial_update"]; ok {
		t.Errorf("payload has partial_update, want a full pointset: %s", msg.GetPayload())
	}
	var got PointsetEventMessage
	if err := json.Unmarshal([]byte(msg.GetPayload()), &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if diff := cmp.Diff(want, got.Points); diff != "" {
		t.Errorf("points (-want +got):\n%s", diff)
	}
}
