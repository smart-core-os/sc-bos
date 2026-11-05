package occupancy

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/occupancy/config"
)

func TestFeature_describeOccupancy(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.Root
		wantDescribed bool
	}{
		{
			name: "enter leave",
			cfg: config.Root{
				Config:                     zone.Config{Name: "zone"},
				EnterLeaveOccupancySensors: []string{"counter"},
			},
			wantDescribed: true,
		},
		{
			name: "enter leave and pir",
			cfg: config.Root{
				Config:                     zone.Config{Name: "zone"},
				OccupancySensors:           []string{"pir"},
				EnterLeaveOccupancySensors: []string{"counter"},
			},
			wantDescribed: false,
		},
		{
			name: "pir only",
			cfg: config.Root{
				Config:           zone.Config{Name: "zone"},
				OccupancySensors: []string{"pir"},
			},
			wantDescribed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, stop := context.WithCancel(t.Context())
			defer stop()
			n := node.New("test")
			f := &feature{
				announcer: node.NewReplaceAnnouncer(n),
				devices:   &zone.Devices{},
				clients:   n,
				logger:    zap.NewNop(),
			}
			if err := f.applyConfig(ctx, tt.cfg); err != nil {
				t.Fatalf("applyConfig: %v", err)
			}

			info := occupancysensorpb.NewOccupancySensorInfoClient(n.ClientConn())
			got, err := info.DescribeOccupancy(ctx, &occupancysensorpb.DescribeOccupancyRequest{Name: "zone"})
			if !tt.wantDescribed {
				if err == nil {
					t.Fatalf("DescribeOccupancy = %v, want an error for an undescribed zone", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("DescribeOccupancy: %v", err)
			}
			if got.GetMaxPeople() <= 0 {
				t.Errorf("max_people = %d, want above 0", got.GetMaxPeople())
			}
			if rs := got.GetResourceSupport(); !rs.GetReadable() || !rs.GetObservable() {
				t.Errorf("resource_support = %v, want readable and observable", rs)
			}
		})
	}
}
