package time

import (
	"reflect"
	"testing"

	"github.com/smart-core-os/sc-api/go/types/time"
)

func Test_cutPeriod(t *testing.T) {
	type args struct {
		p *time.Period
	}
	tests := []struct {
		name      string
		args      args
		wantLower cut
		wantUpper cut
	}{
		{name: "nil nil", args: args{AllTime()}, wantLower: cutBelowAll(), wantUpper: cutAboveAll()},
		{name: "10 nil", args: args{PeriodOnOrAfter(ts(10))}, wantLower: cutBelow(ts(10)), wantUpper: cutAboveAll()},
		{name: "nil 20", args: args{PeriodBefore(ts(20))}, wantLower: cutBelowAll(), wantUpper: cutBelow(ts(20))},
		{name: "10 20", args: args{PeriodBetween(ts(10), ts(20))}, wantLower: cutBelow(ts(10)), wantUpper: cutBelow(ts(20))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLower, gotUpper := cutPeriod(tt.args.p)
			if !reflect.DeepEqual(gotLower, tt.wantLower) {
				t.Errorf("cutPeriod() gotLower = %v, want %v", gotLower, tt.wantLower)
			}
			if !reflect.DeepEqual(gotUpper, tt.wantUpper) {
				t.Errorf("cutPeriod() gotUpper = %v, want %v", gotUpper, tt.wantUpper)
			}
		})
	}
}
