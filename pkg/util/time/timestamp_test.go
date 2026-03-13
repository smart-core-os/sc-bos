package time

import (
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCompareAscending(t *testing.T) {
	type args struct {
		t1 *timestamppb.Timestamp
		t2 *timestamppb.Timestamp
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "1 == 1", args: args{ts(1), ts(1)}, want: 0},
		{name: "1 is before 2", args: args{ts(1), ts(2)}, want: -1},
		{name: "2 is after 1", args: args{ts(2), ts(1)}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareAscending(tt.args.t1, tt.args.t2); got != tt.want {
				t.Errorf("CompareAscending() = %v, want %v", got, tt.want)
			}
		})
	}
}
