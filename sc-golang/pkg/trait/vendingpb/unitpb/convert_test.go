package unitpb

import (
	"testing"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestConvert(t *testing.T) {
	type args struct {
		v    float64
		from traits.Consumable_Unit
		to   traits.Consumable_Unit
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{"zero", args{0, traits.Consumable_CUBIC_METER, traits.Consumable_LITER}, 0, false},
		// just enough testing to make sure we're converting the correct way around :)
		{"volume", args{10, traits.Consumable_CUBIC_METER, traits.Consumable_LITER}, 10_000, false},
		{"volume-inv", args{10, traits.Consumable_LITER, traits.Consumable_CUBIC_METER}, 0.01, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert(tt.args.v, tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Convert() got = %v, want %v", got, tt.want)
			}
		})
	}
}
