package vendingpb

import "testing"

func TestConvertUnit(t *testing.T) {
	type args struct {
		v    float64
		from Consumable_Unit
		to   Consumable_Unit
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{"zero", args{0, Consumable_CUBIC_METER, Consumable_LITER}, 0, false},
		// just enough testing to make sure we're converting the correct way around :)
		{"volume", args{10, Consumable_CUBIC_METER, Consumable_LITER}, 10_000, false},
		{"volume-inv", args{10, Consumable_LITER, Consumable_CUBIC_METER}, 0.01, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertUnit(tt.args.v, tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertUnit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConvertUnit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
