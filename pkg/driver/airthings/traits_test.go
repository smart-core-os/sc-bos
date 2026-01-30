package airthings

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	typespb "github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/local"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/trait/energystoragepb"
)

func TestSampleToAirQuality(t *testing.T) {
	tests := []struct {
		name  string
		input api.DeviceSampleResponseEnriched
		want  *traits.AirQuality
	}{
		{
			name: "all indoor fields",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					AirExchangeRate: newNullableFloat64(1.5),
					Co2:             newNullableFloat64(800.0),
					Pm1:             newNullableControlSignal(10.5),
					Pm25:            newNullableControlSignal(25.3),
					Pm10:            newNullableControlSignal(45.7),
					Pressure:        newNullableControlSignal(1013.25),
					VirusRisk:       newNullableControlSignal(0.3),
					Voc:             newNullableControlSignal(500.0), // ppb
				},
			},
			want: &traits.AirQuality{
				AirChangePerHour:         ptrFloat32(1.5),
				CarbonDioxideLevel:       ptrFloat32(800.0),
				ParticulateMatter_1:      ptrFloat32(10.5),
				ParticulateMatter_25:     ptrFloat32(25.3),
				ParticulateMatter_10:     ptrFloat32(45.7),
				AirPressure:              ptrFloat32(1013.25),
				InfectionRisk:            ptrFloat32(0.3),
				VolatileOrganicCompounds: ptrFloat32(0.5), // converted to ppm
			},
		},
		{
			name: "outdoor overrides indoor",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Pm1:             newNullableControlSignal(10.0),
					Pm25:            newNullableControlSignal(20.0),
					Pm10:            newNullableControlSignal(40.0),
					Pressure:        newNullableControlSignal(1000.0),
					OutdoorPm1:      newNullableControlSignal(15.0),
					OutdoorPm25:     newNullableControlSignal(30.0),
					OutdoorPm10:     newNullableControlSignal(50.0),
					OutdoorPressure: newNullableControlSignal(1010.0),
				},
			},
			want: &traits.AirQuality{
				ParticulateMatter_1:  ptrFloat32(15.0), // outdoor wins
				ParticulateMatter_25: ptrFloat32(30.0),
				ParticulateMatter_10: ptrFloat32(50.0),
				AirPressure:          ptrFloat32(1010.0),
			},
		},
		{
			name: "outdoor only",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					OutdoorPm1:      newNullableControlSignal(12.0),
					OutdoorPm25:     newNullableControlSignal(28.0),
					OutdoorPm10:     newNullableControlSignal(48.0),
					OutdoorPressure: newNullableControlSignal(1015.0),
				},
			},
			want: &traits.AirQuality{
				ParticulateMatter_1:  ptrFloat32(12.0),
				ParticulateMatter_25: ptrFloat32(28.0),
				ParticulateMatter_10: ptrFloat32(48.0),
				AirPressure:          ptrFloat32(1015.0),
			},
		},
		{
			name: "empty data",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{},
			},
			want: &traits.AirQuality{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampleToAirQuality(tt.input)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("sampleToAirQuality() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSampleToAirTemperature(t *testing.T) {
	tests := []struct {
		name  string
		input api.DeviceSampleResponseEnriched
		want  *traits.AirTemperature
	}{
		{
			name: "indoor only",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Temp:     newNullableControlSignal(22.5),
					Humidity: newNullableControlSignal(45.0),
				},
			},
			want: &traits.AirTemperature{
				AmbientTemperature: &typespb.Temperature{ValueCelsius: 22.5},
				AmbientHumidity:    ptrFloat32(45.0),
			},
		},
		{
			name: "outdoor overrides indoor",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Temp:            newNullableControlSignal(20.0),
					Humidity:        newNullableControlSignal(50.0),
					OutdoorTemp:     newNullableControlSignal(15.0),
					OutdoorHumidity: newNullableControlSignal(60.0),
				},
			},
			want: &traits.AirTemperature{
				AmbientTemperature: &typespb.Temperature{ValueCelsius: 15.0},
				AmbientHumidity:    ptrFloat32(60.0),
			},
		},
		{
			name: "outdoor only",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					OutdoorTemp:     newNullableControlSignal(10.0),
					OutdoorHumidity: newNullableControlSignal(70.0),
				},
			},
			want: &traits.AirTemperature{
				AmbientTemperature: &typespb.Temperature{ValueCelsius: 10.0},
				AmbientHumidity:    ptrFloat32(70.0),
			},
		},
		{
			name: "empty data",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{},
			},
			want: &traits.AirTemperature{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampleToAirTemperature(tt.input)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("sampleToAirTemperature() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSampleToEnergyLevel(t *testing.T) {
	tests := []struct {
		name  string
		input api.DeviceSampleResponseEnriched
		want  *traits.EnergyLevel
	}{
		{
			name: "battery present",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Battery: newNullableFloat32(75.5),
				},
			},
			want: &traits.EnergyLevel{
				Quantity: &traits.EnergyLevel_Quantity{
					Percentage: 75.5,
				},
			},
		},
		{
			name: "battery full",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Battery: newNullableFloat32(100.0),
				},
			},
			want: &traits.EnergyLevel{
				Quantity: &traits.EnergyLevel_Quantity{
					Percentage: 100.0,
				},
			},
		},
		{
			name: "battery low",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{
					Battery: newNullableFloat32(5.0),
				},
			},
			want: &traits.EnergyLevel{
				Quantity: &traits.EnergyLevel_Quantity{
					Percentage: 5.0,
				},
			},
		},
		{
			name: "no battery",
			input: api.DeviceSampleResponseEnriched{
				Data: api.SingleSampleDataEnriched{},
			},
			want: &traits.EnergyLevel{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampleToEnergyLevel(tt.input)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("sampleToEnergyLevel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFloat64PtoFloat32P(t *testing.T) {
	tests := []struct {
		name  string
		input *float64
		want  *float32
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "zero",
			input: ptrFloat64(0.0),
			want:  ptrFloat32(0.0),
		},
		{
			name:  "positive",
			input: ptrFloat64(123.456),
			want:  ptrFloat32(123.456),
		},
		{
			name:  "negative",
			input: ptrFloat64(-99.9),
			want:  ptrFloat32(-99.9),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float64PtoFloat32P(tt.input)
			if (got == nil) != (tt.want == nil) {
				t.Errorf("float64PtoFloat32P() nil mismatch, got %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("float64PtoFloat32P() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestPullSampleAirQuality(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	loc := local.NewLocation()
	model := airqualitysensorpb.NewModel()

	// Create a test driver instance
	d := &Driver{}

	dev := config.Device{
		ID:   "test-device",
		Name: "Test Device",
	}

	// Update location with sample data
	sample := api.DeviceSampleResponseEnriched{
		Segment: api.SegmentSimpleResponse{
			Id: "test-device",
		},
		Data: api.SingleSampleDataEnriched{
			Co2: newNullableFloat64(600.0),
		},
	}
	loc.UpdateLatestSamples(api.GetLocationSamplesResponseEnriched{
		Devices: []api.DeviceSampleResponseEnriched{sample},
	})

	// Start pulling in background
	done := make(chan struct{})
	go func() {
		d.pullSampleAirQuality(ctx, dev, loc, model)
		close(done)
	}()

	// Give it time to process initial sample
	time.Sleep(100 * time.Millisecond)

	// Send another update to verify continuous processing
	sample2 := api.DeviceSampleResponseEnriched{
		Segment: api.SegmentSimpleResponse{
			Id: "test-device",
		},
		Data: api.SingleSampleDataEnriched{
			Co2: newNullableFloat64(700.0),
		},
	}
	loc.UpdateLatestSamples(api.GetLocationSamplesResponseEnriched{
		Devices: []api.DeviceSampleResponseEnriched{sample2},
	})

	// Give it time to process second sample
	time.Sleep(100 * time.Millisecond)

	// Cancel context and verify goroutine exits cleanly
	cancel()
	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(time.Second):
		t.Error("pullSampleAirQuality did not exit after context cancellation")
	}
}

func TestPullSampleAirTemperature(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	loc := local.NewLocation()
	model := airtemperaturepb.NewModel()

	d := &Driver{}

	dev := config.Device{
		ID:   "test-device",
		Name: "Test Device",
	}

	// Update location with sample data
	sample := api.DeviceSampleResponseEnriched{
		Segment: api.SegmentSimpleResponse{
			Id: "test-device",
		},
		Data: api.SingleSampleDataEnriched{
			Temp:     newNullableControlSignal(21.5),
			Humidity: newNullableControlSignal(55.0),
		},
	}
	loc.UpdateLatestSamples(api.GetLocationSamplesResponseEnriched{
		Devices: []api.DeviceSampleResponseEnriched{sample},
	})

	// Start pulling in background
	done := make(chan struct{})
	go func() {
		d.pullSampleAirTemperature(ctx, dev, loc, model)
		close(done)
	}()

	// Give it time to process sample
	time.Sleep(100 * time.Millisecond)

	// Cancel context and verify goroutine exits cleanly
	cancel()
	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(time.Second):
		t.Error("pullSampleAirTemperature did not exit after context cancellation")
	}
}

func TestPullSampleEnergyLevel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	loc := local.NewLocation()
	model := energystoragepb.NewModel()

	d := &Driver{}

	dev := config.Device{
		ID:   "test-device",
		Name: "Test Device",
	}

	// Update location with sample data
	sample := api.DeviceSampleResponseEnriched{
		Segment: api.SegmentSimpleResponse{
			Id: "test-device",
		},
		Data: api.SingleSampleDataEnriched{
			Battery: newNullableFloat32(85.0),
		},
	}
	loc.UpdateLatestSamples(api.GetLocationSamplesResponseEnriched{
		Devices: []api.DeviceSampleResponseEnriched{sample},
	})

	// Start pulling in background
	done := make(chan struct{})
	go func() {
		d.pullSampleEnergyLevel(ctx, dev, loc, model)
		close(done)
	}()

	// Give it time to process sample
	time.Sleep(100 * time.Millisecond)

	// Cancel context and verify goroutine exits cleanly
	cancel()
	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(time.Second):
		t.Error("pullSampleEnergyLevel did not exit after context cancellation")
	}
}

func TestAnnounceDevice(t *testing.T) {
	tests := []struct {
		name    string
		traits  []string
		wantErr bool
	}{
		{
			name:    "air quality sensor",
			traits:  []string{string(trait.AirQualitySensor)},
			wantErr: false,
		},
		{
			name:    "air temperature",
			traits:  []string{string(trait.AirTemperature)},
			wantErr: false,
		},
		{
			name:    "energy storage",
			traits:  []string{string(trait.EnergyStorage)},
			wantErr: false,
		},
		{
			name:    "multiple traits",
			traits:  []string{string(trait.AirQualitySensor), string(trait.AirTemperature), string(trait.EnergyStorage)},
			wantErr: false,
		},
		{
			name:    "unsupported trait",
			traits:  []string{"unsupported.Trait"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			d := &Driver{}
			announcer := &testAnnouncer{}
			loc := local.NewLocation()

			dev := config.Device{
				ID:     "test-device",
				Name:   "Test Device",
				Traits: tt.traits,
			}

			err := d.announceDevice(ctx, announcer, dev, loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("announceDevice() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(announcer.announcements) != len(tt.traits) {
				t.Errorf("Expected %d announcements, got %d", len(tt.traits), len(announcer.announcements))
			}
		})
	}
}

func TestROAirTemperatureServer_UpdateAirTemperature(t *testing.T) {
	server := roAirTemperatureServer{}
	_, err := server.UpdateAirTemperature(context.Background(), &traits.UpdateAirTemperatureRequest{})
	if err == nil {
		t.Error("Expected error for read-only operation, got nil")
	}
}

func TestROEnergyStorageServer_Charge(t *testing.T) {
	server := roEnergyStorageServer{}
	_, err := server.Charge(context.Background(), &traits.ChargeRequest{})
	if err == nil {
		t.Error("Expected error for read-only operation, got nil")
	}
}

// Helper functions

func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrFloat32(v float32) *float32 {
	return &v
}

func newNullableFloat64(v float64) api.NullableFloat64 {
	return *api.NewNullableFloat64(&v)
}

func newNullableFloat32(v float32) api.NullableFloat32 {
	return *api.NewNullableFloat32(&v)
}

func newNullableControlSignal(v float64) api.NullableSingleSampleDataEnrichedControlSignal {
	return *api.NewNullableSingleSampleDataEnrichedControlSignal(&api.SingleSampleDataEnrichedControlSignal{
		Float64: &v,
	})
}

// testAnnouncer is a mock announcer for testing
type testAnnouncer struct {
	announcements []struct {
		name     string
		features []node.Feature
	}
}

func (a *testAnnouncer) Announce(name string, features ...node.Feature) node.Undo {
	a.announcements = append(a.announcements, struct {
		name     string
		features []node.Feature
	}{name: name, features: features})
	return func() {}
}
