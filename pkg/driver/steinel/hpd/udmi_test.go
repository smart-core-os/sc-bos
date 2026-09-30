package hpd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/steinel/hpd/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
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

					server := newUdmiServiceServer(nil, aq, o, temp, "prefix", nil)
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

// Test_PullExportMessages_noAirQuality checks the stream works for an HPD, which has
// no air quality module and so no air quality value.
func Test_PullExportMessages_noAirQuality(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}))
		temp := resource.NewValue(resource.WithInitialValue(&airtemperaturepb.AirTemperature{}))
		server := newUdmiServiceServer(nil, nil, o, temp, "prefix", nil)
		client := udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, server))

		messages, err := client.PullExportMessages(t.Context(), &udmipb.PullExportMessagesRequest{Name: "test"})
		if err != nil {
			t.Fatalf("PullExportMessages: %v", err)
		}
		results := make(chan *udmipb.PullExportMessagesResponse, 1)
		go func() {
			msg, _ := messages.Recv()
			results <- msg
		}()

		synctest.Wait()
		o.Set(&occupancysensorpb.Occupancy{PeopleCount: 1, State: occupancysensorpb.Occupancy_OCCUPIED})
		synctest.Wait()

		var msg *udmipb.PullExportMessagesResponse
		select {
		case msg = <-results:
		default:
			t.Fatal("no message received")
		}
		var got PointsetEventMessage
		if err := json.Unmarshal([]byte(msg.GetMessage().GetPayload()), &got); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		want := EventPoints{
			DeviceType:     &EventPoint[string]{PresentValue: DriverName},
			PeopleCount:    &EventPoint[int32]{PresentValue: 1},
			OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
		}
		if diff := cmp.Diff(want, got.Points); diff != "" {
			t.Errorf("points (-want +got):\n%s", diff)
		}
	})
}

func Test_GetExportMessage(t *testing.T) {
	co2 := float32(412)
	voc := float32(0.3)
	humidity := float32(45.5)
	readAll := func(aq, o, temp *resource.Value) func(context.Context) error {
		return func(context.Context) error {
			if aq != nil {
				aq.Set(&airqualitysensorpb.AirQuality{CarbonDioxideLevel: &co2, VolatileOrganicCompounds: &voc})
			}
			o.Set(&occupancysensorpb.Occupancy{PeopleCount: 3, State: occupancysensorpb.Occupancy_OCCUPIED})
			temp.Set(&airtemperaturepb.AirTemperature{
				AmbientTemperature: &typespb.Temperature{ValueCelsius: 21.5},
				AmbientHumidity:    &humidity,
			})
			return nil
		}
	}
	occupancyAndTemp := EventPoints{
		DeviceType:     &EventPoint[string]{PresentValue: DriverName},
		Temperature:    &EventPoint[float64]{PresentValue: 21.5},
		Humidity:       &EventPoint[float32]{PresentValue: 45.5},
		PeopleCount:    &EventPoint[int32]{PresentValue: 3},
		OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
	}
	withAirQuality := occupancyAndTemp
	withAirQuality.Co2Level = &EventPoint[float32]{PresentValue: 412}
	withAirQuality.VocLevel = &EventPoint[float32]{PresentValue: 0.3}

	tests := []struct {
		name          string
		hasAirQuality bool
		refresh       func(aq, o, temp *resource.Value) func(context.Context) error
		wantCode      codes.Code
		want          EventPoints
	}{
		{
			name:          "multisensor",
			hasAirQuality: true,
			refresh:       readAll,
			want:          withAirQuality,
		},
		{
			name:    "no air quality module",
			refresh: readAll,
			want:    occupancyAndTemp,
		},
		{
			name:          "read fails",
			hasAirQuality: true,
			refresh: func(_, _, _ *resource.Value) func(context.Context) error {
				return func(context.Context) error { return errors.New("connection refused") }
			},
			wantCode: codes.Unavailable,
		},
		{
			name:          "no refresh",
			hasAirQuality: true,
			wantCode:      codes.Unavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var aq *resource.Value
			if tt.hasAirQuality {
				aq = resource.NewValue(resource.WithInitialValue(&airqualitysensorpb.AirQuality{}))
			}
			o := resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}))
			temp := resource.NewValue(resource.WithInitialValue(&airtemperaturepb.AirTemperature{}))
			var refresh func(context.Context) error
			if tt.refresh != nil {
				refresh = tt.refresh(aq, o, temp)
			}

			server := newUdmiServiceServer(nil, aq, o, temp, "prefix", refresh)
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

// TestDriver_GetExportMessage checks that GetExportMessage reads the device rather
// than returning what the last poll cached: a change on the device shows up straight
// away, and once the device stops answering it returns Unavailable, not the old values.
func TestDriver_GetExportMessage(t *testing.T) {
	var (
		body atomic.Value // string
		down atomic.Bool
	)
	body.Store(`{"SensorName": "HPD3", "Temperature": 20.5, "Humidity": 50, "CO2": 400, "VOC": 300}`)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code": 503, "info": "unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(body.Load().(string)))
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "https://")

	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data := `{
		"name": "steinel-hpd",
		"passwordFile": "` + strings.ReplaceAll(passwordFile, `\`, `\\`) + `",
		"pollInterval": "1h",
		"devices": [
			{"name": "sensors/hpd", "ipAddress": "` + host + `", "model": "hpd", "udmiTopicPrefix": "site/hpd"},
			{"name": "sensors/multi", "ipAddress": "` + host + `", "model": "multisensor", "udmiTopicPrefix": "site/multi"}
		]
	}`
	cfg, err := config.ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	n := node.New("test")
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(n),
		health:    healthpb.NewRegistry().ForOwner("driver:steinel-hpd"),
		logger:    zap.NewNop(),
	}
	ctx, stop := context.WithCancel(context.Background())
	if err := d.applyConfig(ctx, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	defer func() {
		stop()
		d.devicesStopped()
	}()
	client := udmipb.NewUdmiServiceClient(n.ClientConn())

	// The next scheduled poll is an hour away, so only a live read sees this.
	body.Store(`{"SensorName": "HPD3", "Temperature": 22.5, "Humidity": 55, "CO2": 450, "VOC": 500, "ZonePeople0": 2}`)

	common := EventPoints{
		DeviceType:     &EventPoint[string]{PresentValue: DriverName},
		Temperature:    &EventPoint[float64]{PresentValue: 22.5},
		Humidity:       &EventPoint[float32]{PresentValue: 55},
		PeopleCount:    &EventPoint[int32]{PresentValue: 2},
		OccupancyState: &EventPoint[string]{PresentValue: occupancysensorpb.Occupancy_OCCUPIED.String()},
	}
	multi := common
	multi.Co2Level = &EventPoint[float32]{PresentValue: 450}
	multi.VocLevel = &EventPoint[float32]{PresentValue: 0.5}

	for name, tc := range map[string]struct {
		topic string
		want  EventPoints
	}{
		"sensors/hpd":   {topic: "site/hpd/events/pointset", want: common},
		"sensors/multi": {topic: "site/multi/events/pointset", want: multi},
	} {
		msg, err := client.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{Name: name})
		if err != nil {
			t.Fatalf("%s: GetExportMessage: %v", name, err)
		}
		checkFullPointset(t, msg, tc.topic, tc.want)
	}

	down.Store(true)
	for _, name := range []string{"sensors/hpd", "sensors/multi"} {
		_, err := client.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{Name: name})
		if code := status.Code(err); code != codes.Unavailable {
			t.Errorf("%s: GetExportMessage with the device down: code = %v, want %v (err %v)", name, code, codes.Unavailable, err)
		}
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

// TestDriver_GetExportMessage_deviceStops checks stopping a device abandons a
// GetExportMessage read in flight, answering Unavailable rather than waiting on the
// device until the request times out, and releases the device's health check.
func TestDriver_GetExportMessage_deviceStops(t *testing.T) {
	var requests atomic.Int32
	exporting := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			// the first poll
			_, _ = w.Write([]byte(`{"SensorName": "HPD3", "Temperature": 20.5}`))
			return
		}
		close(exporting)
		<-r.Context().Done()
	}))
	defer server.Close()

	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.ParseConfig([]byte(`{
		"name": "steinel-hpd",
		"passwordFile": "` + strings.ReplaceAll(passwordFile, `\`, `\\`) + `",
		"pollInterval": "1h",
		"devices": [
			{"name": "sensors/hpd", "ipAddress": "` + strings.TrimPrefix(server.URL, "https://") + `", "model": "hpd", "udmiTopicPrefix": "site/hpd"}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	n := node.New("test")
	polled := make(chan struct{})
	markPolled := sync.OnceFunc(func() { close(polled) })
	registry := healthpb.NewRegistry(healthpb.WithOnCheckUpdate(func(_ string, c *healthpb.HealthCheck) {
		if c.GetReliability().GetState() == healthpb.HealthCheck_Reliability_RELIABLE {
			markPolled()
		}
	}))
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(n),
		health:    registry.ForOwner("driver:steinel-hpd"),
		logger:    zap.NewNop(),
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	if err := d.applyConfig(ctx, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	client := udmipb.NewUdmiServiceClient(n.ClientConn())

	// wait for the first poll, so the export is the read in flight when the device stops
	select {
	case <-polled:
	case <-time.After(5 * time.Second):
		t.Fatal("the first poll didn't report the device reliable")
	}

	exported := make(chan error, 1)
	go func() {
		_, err := client.GetExportMessage(t.Context(), &udmipb.GetExportMessageRequest{Name: "sensors/hpd"})
		exported <- err
	}()
	select {
	case <-exporting:
	case err := <-exported:
		t.Fatalf("GetExportMessage returned before reading the device: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("GetExportMessage didn't read the device")
	}

	stop()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		d.devicesStopped()
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("device didn't stop while a GetExportMessage read was in flight")
	}
	if code := status.Code(<-exported); code != codes.Unavailable {
		t.Errorf("GetExportMessage code = %v, want %v", code, codes.Unavailable)
	}
	if registry.GetCheck("sensors/hpd", healthpb.AbsID("driver:steinel-hpd", "commsCheck")) != nil {
		t.Error("health check still registered after the device stopped")
	}
}
