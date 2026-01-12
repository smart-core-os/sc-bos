package xovis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func Test_PullExportMessages(t *testing.T) {

	enter := int32(0)
	leave := int32(0)
	e := resource.NewValue(resource.WithInitialValue(&traits.EnterLeaveEvent{EnterTotal: &enter, LeaveTotal: &leave}), resource.WithNoDuplicates())
	o := resource.NewValue(resource.WithInitialValue(&traits.Occupancy{PeopleCount: 0, State: traits.Occupancy_OCCUPIED}), resource.WithNoDuplicates())

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
				server := newUdmiServiceServer(nil, e, o, "prefix")
				return udmipb.WrapService(server)
			},
			set: func() {
				o.Set(&traits.Occupancy{
					PeopleCount: 459,
					State:       traits.Occupancy_OCCUPIED,
				})
			},
			want: EventPoints{
				DeviceType:     &EventPoint[string]{PresentValue: DriverName},
				OccupancyState: &EventPoint[string]{PresentValue: traits.Occupancy_OCCUPIED.String()},
				PeopleCount:    &EventPoint[int32]{PresentValue: 459},
			},
		},
		{
			name: "enterleave",
			createClient: func() udmipb.UdmiServiceClient {
				server := newUdmiServiceServer(nil, e, o, "prefix")
				return udmipb.WrapService(server)
			},
			set: func() {
				e.Set(&traits.EnterLeaveEvent{
					EnterTotal: &enterTotal,
					LeaveTotal: &leaveTotal,
				})
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
				server := newUdmiServiceServer(nil, e, nil, "prefix")
				return udmipb.WrapService(server)
			},
			set: func() {
				e.Set(&traits.EnterLeaveEvent{
					EnterTotal: &enterTotal,
					LeaveTotal: &leaveTotal,
				})
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
				server := newUdmiServiceServer(nil, nil, o, "prefix")
				return udmipb.WrapService(server)
			},
			set: func() {
				o.Set(&traits.Occupancy{
					PeopleCount: 459,
					State:       traits.Occupancy_OCCUPIED,
				})
			},
			want: EventPoints{
				DeviceType:     &EventPoint[string]{PresentValue: DriverName},
				OccupancyState: &EventPoint[string]{PresentValue: traits.Occupancy_OCCUPIED.String()},
				PeopleCount:    &EventPoint[int32]{PresentValue: 459},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client := tt.createClient()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			messages, err := client.PullExportMessages(ctx, req)
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

		})
	}
}
