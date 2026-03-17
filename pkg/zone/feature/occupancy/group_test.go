package occupancy

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func Test_mergeOccupancy(t *testing.T) {
	tests := []struct {
		name    string
		args    []*traits.Occupancy
		want    *traits.Occupancy
		wantErr bool
	}{
		{"empty", nil, nil, true},
		{"one", []*traits.Occupancy{{State: traits.Occupancy_OCCUPIED}}, &traits.Occupancy{State: traits.Occupancy_OCCUPIED}, false},
		{"earliestOccupancy", []*traits.Occupancy{
			{State: traits.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: traits.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &traits.Occupancy{State: traits.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))}, false},
		{"latestUnoccupied", []*traits.Occupancy{
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeOccupancy(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeOccupancy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeOccupancy() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
