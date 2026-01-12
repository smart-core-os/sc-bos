package hpd

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func Test_PullExportMessages(t *testing.T) {

	co2 := float32(0)
	voc := float32(0)
	humidity := float32(0)
	aq := resource.NewValue(resource.WithInitialValue(&traits.AirQuality{CarbonDioxideLevel: &co2, VolatileOrganicCompounds: &voc}), resource.WithNoDuplicates())
	o := resource.NewValue(resource.WithInitialValue(&traits.Occupancy{PeopleCount: 0, State: traits.Occupancy_OCCUPIED}), resource.WithNoDuplicates())
	temp := resource.NewValue(resource.WithInitialValue(&traits.AirTemperature{AmbientTemperature: &types.Temperature{ValueCelsius: 0}, AmbientHumidity: &humidity}), resource.WithNoDuplicates())

	server := newUdmiServiceServer(nil, aq, o, temp, "prefix")
	client := udmipb.WrapService(server)

	req := &udmipb.PullExportMessagesRequest{
		Name: "test",
	}

	tests := []struct {
		name string
		set  func()
		want EventPoints
	}{
		{
			name: "occupancy",
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
			name: "temp humidity",
			set: func() {
				humidity := float32(98.7)
				temp.Set(&traits.AirTemperature{
					Mode:               0,
					TemperatureGoal:    nil,
					AmbientTemperature: &types.Temperature{ValueCelsius: 765.4},
					AmbientHumidity:    &humidity,
					DewPoint:           nil,
				})
			},
			want: EventPoints{
				DeviceType:  &EventPoint[string]{PresentValue: DriverName},
				Humidity:    &EventPoint[float32]{PresentValue: 98.7},
				Temperature: &EventPoint[float64]{PresentValue: 765.4},
			},
		},
		{
			name: "air quality",
			set: func() {
				co2 := float32(123.4)
				voc := float32(345.6)
				aq.Set(&traits.AirQuality{
					CarbonDioxideLevel:       &co2,
					VolatileOrganicCompounds: &voc,
				})
			},
			want: EventPoints{
				DeviceType: &EventPoint[string]{PresentValue: DriverName},
				Co2Level:   &EventPoint[float32]{PresentValue: 123.4},
				VocLevel:   &EventPoint[float32]{PresentValue: 345.6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
