package time

import (
	"testing"

	"github.com/smart-core-os/sc-api/go/types/time"
)

func TestPeriodsConnected(t *testing.T) {
	type args struct {
		p1 *time.Period
		p2 *time.Period
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "both nil", args: args{nil, nil}, want: false},
		{name: "one nil", args: args{AllTime(), nil}, want: false},
		// disjoint periods
		{name: "[1,2) [3,4)", args: args{between(1, 2), between(3, 4)}, want: false},
		{name: "[-,2) [3,4)", args: args{before(2), between(3, 4)}, want: false},
		{name: "[1,2) [3,-)", args: args{between(1, 2), onOrAfter(3)}, want: false},
		{name: "[-,2) [3,-)", args: args{before(2), onOrAfter(3)}, want: false},
		// adjacent periods
		{name: "[1,2) [2,3)", args: args{between(1, 2), between(2, 3)}, want: true},
		{name: "[1,2) [2,-)", args: args{between(1, 2), onOrAfter(2)}, want: true},
		{name: "[-,2) [2,3)", args: args{before(2), between(2, 3)}, want: true},
		{name: "[-,2) [2,-)", args: args{before(2), onOrAfter(2)}, want: true},
		// enclosed periods
		{name: "[1, [2,3), 4)", args: args{between(1, 4), between(2, 3)}, want: true},
		{name: "[1, [1,3), 4)", args: args{between(1, 4), between(1, 3)}, want: true},
		{name: "[1, [2,4), 4)", args: args{between(1, 4), between(2, 4)}, want: true},
		{name: "[1, [1,4), 4)", args: args{between(1, 4), between(1, 4)}, want: true},
		{name: "[1, [2,3), -)", args: args{onOrAfter(1), between(2, 3)}, want: true},
		{name: "[1, [1,3), -)", args: args{onOrAfter(1), between(1, 3)}, want: true},
		{name: "[-, [2,3), 4)", args: args{before(4), between(2, 3)}, want: true},
		{name: "[-, [2,4), 4)", args: args{before(4), between(2, 4)}, want: true},
		// overlapping periods
		{name: "[1,3) [2,4)", args: args{between(1, 3), between(2, 4)}, want: true},
		{name: "[1,3) [2,-)", args: args{between(1, 3), onOrAfter(2)}, want: true},
		{name: "[-,3) [2,4)", args: args{before(3), between(2, 4)}, want: true},
		{name: "[-,3) [2,-)", args: args{before(3), onOrAfter(2)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeriodsConnected(tt.args.p1, tt.args.p2); got != tt.want {
				t.Errorf("PeriodsConnected() = %v, want %v", got, tt.want)
			}
		})
		if tt.args.p1 != tt.args.p2 {
			// argument order shouldn't matter
			t.Run("inv:"+tt.name, func(t *testing.T) {
				if got := PeriodsConnected(tt.args.p2, tt.args.p1); got != tt.want {
					t.Errorf("PeriodsConnected() = %v, want %v", got, tt.want)
				}
			})
		}
	}
}

func TestPeriodsIntersect(t *testing.T) {
	type args struct {
		p1 *time.Period
		p2 *time.Period
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "both nil", args: args{nil, nil}, want: false},
		{name: "one nil", args: args{AllTime(), nil}, want: false},
		// disjoint periods
		{name: "[1,2) [3,4)", args: args{between(1, 2), between(3, 4)}, want: false},
		{name: "[-,2) [3,4)", args: args{before(2), between(3, 4)}, want: false},
		{name: "[1,2) [3,-)", args: args{between(1, 2), onOrAfter(3)}, want: false},
		{name: "[-,2) [3,-)", args: args{before(2), onOrAfter(3)}, want: false},
		// adjacent periods
		{name: "[1,2) [2,3)", args: args{between(1, 2), between(2, 3)}, want: false},
		{name: "[1,2) [2,-)", args: args{between(1, 2), onOrAfter(2)}, want: false},
		{name: "[-,2) [2,3)", args: args{before(2), between(2, 3)}, want: false},
		{name: "[-,2) [2,-)", args: args{before(2), onOrAfter(2)}, want: false},
		// enclosed periods
		{name: "[1, [2,3), 4)", args: args{between(1, 4), between(2, 3)}, want: true},
		{name: "[1, [1,3), 4)", args: args{between(1, 4), between(1, 3)}, want: true},
		{name: "[1, [2,4), 4)", args: args{between(1, 4), between(2, 4)}, want: true},
		{name: "[1, [1,4), 4)", args: args{between(1, 4), between(1, 4)}, want: true},
		{name: "[1, [2,3), -)", args: args{onOrAfter(1), between(2, 3)}, want: true},
		{name: "[1, [1,3), -)", args: args{onOrAfter(1), between(1, 3)}, want: true},
		{name: "[-, [2,3), 4)", args: args{before(4), between(2, 3)}, want: true},
		{name: "[-, [2,4), 4)", args: args{before(4), between(2, 4)}, want: true},
		// overlapping periods
		{name: "[1,3) [2,4)", args: args{between(1, 3), between(2, 4)}, want: true},
		{name: "[1,3) [2,-)", args: args{between(1, 3), onOrAfter(2)}, want: true},
		{name: "[-,3) [2,4)", args: args{before(3), between(2, 4)}, want: true},
		{name: "[-,3) [2,-)", args: args{before(3), onOrAfter(2)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeriodsIntersect(tt.args.p1, tt.args.p2); got != tt.want {
				t.Errorf("PeriodsConnected() = %v, want %v", got, tt.want)
			}
		})
		if tt.args.p1 != tt.args.p2 {
			// argument order shouldn't matter
			t.Run("inv:"+tt.name, func(t *testing.T) {
				if got := PeriodsIntersect(tt.args.p2, tt.args.p1); got != tt.want {
					t.Errorf("PeriodsConnected() = %v, want %v", got, tt.want)
				}
			})
		}
	}
}
