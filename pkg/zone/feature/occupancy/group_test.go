package occupancy

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
)

func Test_mergeOccupancy(t *testing.T) {
	tests := []struct {
		name    string
		args    []*occupancysensorpb.Occupancy
		want    *occupancysensorpb.Occupancy
		wantErr bool
	}{
		{"empty", nil, nil, true},
		{"one", []*occupancysensorpb.Occupancy{{State: occupancysensorpb.Occupancy_OCCUPIED}}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED}, false},
		{"earliestOccupancy", []*occupancysensorpb.Occupancy{
			{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))}, false},
		{"latestUnoccupied", []*occupancysensorpb.Occupancy{
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))}, false},
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
