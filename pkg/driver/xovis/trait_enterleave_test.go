package xovis

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// TestEnterLeaveServer_PullEnterLeaveEvents checks the stream starts from the sensor's
// totals, then counts push data and polls against them, finding the counts by id.
func TestEnterLeaveServer_PullEnterLeaveEvents(t *testing.T) {
	sensor := &fakeSensor{}
	sensor.fw.Store(10)
	sensor.bw.Store(6)
	httpServer := httptest.NewTLSServer(sensor)
	defer httpServer.Close()
	fc, err := healthpb.NewRegistry().ForOwner("driver:xovis").NewFaultCheck("sensors/01", proto.Clone(commsHealthCheck).(*healthpb.HealthCheck))
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	newServer := func(logicID int) (*enterLeaveServer, enterleavesensorpb.EnterLeaveSensorApiClient) {
		e := &enterLeaveServer{
			client:          newInsecureClient(strings.TrimPrefix(httpServer.URL, "https://"), "admin", "secret"),
			logicID:         logicID,
			bus:             &minibus.Bus[PushData]{},
			faultCheck:      fc,
			EnterLeaveTotal: resource.NewValue(resource.WithInitialValue(&enterleavesensorpb.EnterLeaveEvent{})),
		}
		// the test sends polls itself, rather than waiting for them
		e.pollInit.Do(func() {
			e.polls = &minibus.Bus[LiveLogicResponse]{}
			e.poll = task.Poll(func(context.Context) {}, time.Hour)
		})
		return e, enterleavesensorpb.NewEnterLeaveSensorApiClient(wrap.ServerToClient(enterleavesensorpb.EnterLeaveSensorApi_ServiceDesc, e))
	}
	// send retries until the stream is listening on bus
	send := func(ctx context.Context, send func(ctx context.Context) int) {
		t.Helper()
		for send(ctx) == 0 {
			select {
			case <-ctx.Done():
				t.Fatalf("stream never listened: %v", ctx.Err())
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
	recv := func(stream enterleavesensorpb.EnterLeaveSensorApi_PullEnterLeaveEventsClient) []*enterleavesensorpb.EnterLeaveEvent {
		t.Helper()
		res, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		var got []*enterleavesensorpb.EnterLeaveEvent
		for _, c := range res.GetChanges() {
			if c.GetName() != "sensors/01" {
				t.Errorf("change name = %q, want %q", c.GetName(), "sensors/01")
			}
			got = append(got, c.GetEnterLeaveEvent())
		}
		return got
	}
	event := func(dir enterleavesensorpb.EnterLeaveEvent_Direction, enter, leave int32) *enterleavesensorpb.EnterLeaveEvent {
		return &enterleavesensorpb.EnterLeaveEvent{Direction: dir, EnterTotal: &enter, LeaveTotal: &leave}
	}
	check := func(t *testing.T, step string, got []*enterleavesensorpb.EnterLeaveEvent, want ...*enterleavesensorpb.EnterLeaveEvent) {
		t.Helper()
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("%s changes (-want +got):\n%s", step, diff)
		}
	}

	t.Run("in/out logic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		e, client := newServer(2)
		stream, err := client.PullEnterLeaveEvents(ctx, &enterleavesensorpb.PullEnterLeaveEventsRequest{Name: "sensors/01"})
		if err != nil {
			t.Fatalf("PullEnterLeaveEvents: %v", err)
		}
		check(t, "initial", recv(stream), event(enterleavesensorpb.EnterLeaveEvent_DIRECTION_UNSPECIFIED, 10, 6))

		// push data carries the counts since the last push, keyed by id
		send(ctx, func(ctx context.Context) int {
			return e.bus.Send(ctx, PushData{LogicsData: &LogicsPushData{Logics: []Logic{
				{ID: 1, Records: []LogicRecord{{Counts: []Count{{ID: 1, Value: 100}}}}}, // another logic
				{ID: 2, Records: []LogicRecord{{To: time.Now(), Counts: []Count{{ID: 1, Value: 2}, {ID: 2, Value: 1}}}}},
			}}})
		})
		check(t, "push data", recv(stream),
			event(enterleavesensorpb.EnterLeaveEvent_ENTER, 11, 6),
			event(enterleavesensorpb.EnterLeaveEvent_ENTER, 12, 6),
			event(enterleavesensorpb.EnterLeaveEvent_LEAVE, 12, 7),
		)

		// polls carry the sensor's totals
		send(ctx, func(ctx context.Context) int {
			return e.polls.Send(ctx, LiveLogicResponse{Time: time.Now(), Logic: LiveLogicData{ID: 2, Counts: []Count{
				{ID: 1, Name: "fw", Value: 12},
				{ID: 2, Name: "bw", Value: 8},
			}}})
		})
		check(t, "poll", recv(stream), event(enterleavesensorpb.EnterLeaveEvent_LEAVE, 12, 8))
	})

	t.Run("not an in/out logic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		_, client := newServer(1) // the occupancy logic
		stream, err := client.PullEnterLeaveEvents(ctx, &enterleavesensorpb.PullEnterLeaveEventsRequest{Name: "sensors/01"})
		if err == nil {
			_, err = stream.Recv()
		}
		if code := status.Code(err); code != codes.FailedPrecondition {
			t.Errorf("PullEnterLeaveEvents code = %v, want %v (err %v)", code, codes.FailedPrecondition, err)
		}
	})
}
