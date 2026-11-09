package airquality

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
)

func TestMergeAirQuality_VOC(t *testing.T) {
	tests := []struct {
		name string
		in   []*airqualitysensorpb.AirQuality
		want *airqualitysensorpb.AirQuality
	}{
		{
			name: "density averaged",
			in: []*airqualitysensorpb.AirQuality{
				{VolatileOrganicCompoundsDensity: new(float32(400))},
				{VolatileOrganicCompoundsDensity: new(float32(600))},
			},
			want: &airqualitysensorpb.AirQuality{VolatileOrganicCompoundsDensity: new(float32(500))},
		},
		{
			// A ppm sensor and a mass sensor in one zone must not be averaged into one number.
			name: "ppm and density kept apart",
			in: []*airqualitysensorpb.AirQuality{
				{VolatileOrganicCompounds: new(float32(0.5))},
				{VolatileOrganicCompoundsDensity: new(float32(2650))},
				{VolatileOrganicCompounds: new(float32(1.5)), VolatileOrganicCompoundsDensity: new(float32(1350))},
			},
			want: &airqualitysensorpb.AirQuality{
				VolatileOrganicCompounds:        new(float32(1)),
				VolatileOrganicCompoundsDensity: new(float32(2000)),
			},
		},
		{
			name: "ppm only leaves density unset",
			in: []*airqualitysensorpb.AirQuality{
				{VolatileOrganicCompounds: new(float32(1))},
				{VolatileOrganicCompounds: new(float32(3))},
			},
			want: &airqualitysensorpb.AirQuality{VolatileOrganicCompounds: new(float32(2))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeAirQuality(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeAirQuality (-want +got):\n%s", diff)
			}
		})
	}
}
