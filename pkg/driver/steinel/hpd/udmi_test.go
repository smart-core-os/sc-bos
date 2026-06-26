package hpd

import (
	"encoding/json"
	"testing"
	"testing/synctest"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

func Test_PullExportMessages(t *testing.T) {
	req := &udmipb.PullExportMessagesRequest{
		Name: "test",
	}

	tests := []struct {
		name string
		set  func(aq, o, temp *resource.Value)
		want EventPoints
	}{
		{
			name: "occupancy",
			set: func(_, o, _ *resource.Value) {
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
			name: "temp humidity",
			set: func(_, _, temp *resource.Value) {
				humidity := float32(98.7)
				temp.Set(
					&airtemperaturepb.AirTemperature{
						Mode:               0,
						TemperatureGoal:    nil,
						AmbientTemperature: &typespb.Temperature{ValueCelsius: 765.4},
						AmbientHumidity:    &humidity,
						DewPoint:           nil,
					},
				)
			},
			want: EventPoints{
				DeviceType:  &EventPoint[string]{PresentValue: DriverName},
				Humidity:    &EventPoint[float32]{PresentValue: 98.7},
				Temperature: &EventPoint[float64]{PresentValue: 765.4},
			},
		},
		{
			name: "air quality",
			set: func(aq, _, _ *resource.Value) {
				co2 := float32(123.4)
				voc := float32(345.6)
				aq.Set(
					&airqualitysensorpb.AirQuality{
						CarbonDioxideLevel:       &co2,
						VolatileOrganicCompounds: &voc,
					},
				)
			},
			want: EventPoints{
				DeviceType: &EventPoint[string]{PresentValue: DriverName},
				Co2Level:   &EventPoint[float32]{PresentValue: 123.4},
				VocLevel:   &EventPoint[float32]{PresentValue: 345.6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx := t.Context()

					co2 := float32(0)
					voc := float32(0)
					humidity := float32(0)
					aq := resource.NewValue(
						resource.WithInitialValue(
							&airqualitysensorpb.AirQuality{
								CarbonDioxideLevel: &co2, VolatileOrganicCompounds: &voc,
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
					temp := resource.NewValue(
						resource.WithInitialValue(
							&airtemperaturepb.AirTemperature{
								AmbientTemperature: &typespb.Temperature{ValueCelsius: 0}, AmbientHumidity: &humidity,
							},
						), resource.WithNoDuplicates(),
					)

					server := newUdmiServiceServer(nil, aq, o, temp, "prefix")
					client := udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))

					messages, err := client.PullExportMessages(ctx, req)
					if err != nil {
						t.Fatalf("PullExportMessages: %v", err)
					}

					// Receive on a goroutine so the bubble isn't held on a blocking Recv.
					type recvResult struct {
						msg *udmipb.PullExportMessagesResponse
						err error
					}
					results := make(chan recvResult, 1)
					go func() {
						msg, err := messages.Recv()
						results <- recvResult{msg, err}
					}()

					// Wait for the server handler to register its Pull subscriptions before
					// changing the value. The handler pulls with WithUpdatesOnly, so a change
					// emitted before the subscription exists is lost and Recv blocks forever.
					synctest.Wait()
					tt.set(aq, o, temp)
					// Let the change propagate through the server and back to Recv.
					synctest.Wait()

					var r recvResult
					select {
					case r = <-results:
					default:
						t.Fatal("no message received")
					}
					if r.err != nil {
						t.Fatalf("messages.Recv: %v", r.err)
					}

					// take the response payload which should be a valid PointsetEventMessage
					var pointSetMessage PointsetEventMessage
					if err := json.Unmarshal([]byte(r.msg.Message.Payload), &pointSetMessage); err != nil {
						t.Fatal("json.Unmarshal failed")
					}

					if res := cmp.Diff(pointSetMessage.Points, tt.want); res != "" {
						t.Fatal("trait does not match " + res)
					}
				})
			},
		)
	}
}
