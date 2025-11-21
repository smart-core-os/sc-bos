package healthpb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name           string
		src, dst, want *gen.HealthCheck
		ignore         string
	}{
		{
			name: "oneof changes",
			src: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			dst: &gen.HealthCheck{
				Check: &gen.HealthCheck_Faults_{
					Faults: &gen.HealthCheck_Faults{},
				},
			},
			want: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
		},
		{
			name: "oneof changes reverse",
			src: &gen.HealthCheck{
				Check: &gen.HealthCheck_Faults_{
					Faults: &gen.HealthCheck_Faults{},
				},
			},
			dst: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			want: &gen.HealthCheck{
				Check: &gen.HealthCheck_Faults_{
					Faults: &gen.HealthCheck_Faults{},
				},
			},
		},
		{
			name: "bounds.normal_values",
			src: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
			dst: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intNormalValues(10, 20),
					},
				},
			},
			want: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "bounds.abnormal_values",
			src: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
			dst: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intAbnormalValues(10, 20),
					},
				},
			},
			want: &gen.HealthCheck{
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "compliance_impact",
			src: &gen.HealthCheck{
				ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
					{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
				},
			},
			dst: &gen.HealthCheck{
				ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
					{Contribution: gen.HealthCheck_ComplianceImpact_RATING},
				},
			},
			want: &gen.HealthCheck{
				ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
					{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
				},
			},
		},
		{
			ignore: "consistency with other merge functions",
			name:   "timestamp",
			src: &gen.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
			dst: &gen.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(10, 100)),
			},
			want: &gen.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
		},
		{
			name: "create_time both nil",
			src:  &gen.HealthCheck{},
			dst:  &gen.HealthCheck{},
			want: &gen.HealthCheck{},
		},
		{
			name: "create_time src nil",
			src:  &gen.HealthCheck{},
			dst: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time dst nil",
			src: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &gen.HealthCheck{},
			want: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src < dst",
			src: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			want: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src > dst",
			src: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			dst: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &gen.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ignore != "" {
				t.Skipf("skipping test %q: %s", tt.name, tt.ignore)
			}
			MergeCheck(proto.Merge, tt.dst, tt.src)
			if diff := cmp.Diff(tt.want, tt.dst, protocmp.Transform()); diff != "" {
				t.Errorf("mergeCheck() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestChecks(t *testing.T) {
	tests := []struct {
		name string
		dst  []*gen.HealthCheck
		src  []*gen.HealthCheck
		want []*gen.HealthCheck
	}{
		{
			name: "empty dst",
			dst:  nil,
			src: []*gen.HealthCheck{
				{Id: "b"},
				{Id: "a"},
			},
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "empty src",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
			src: nil,
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "merge existing",
			dst: []*gen.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_RATING},
					},
				},
			},
			src: []*gen.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
			want: []*gen.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
		},
		{
			name: "add new checks",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
			src: []*gen.HealthCheck{
				{Id: "b"},
				{Id: "d"},
			},
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
				{Id: "d"},
			},
		},
		{
			name: "mixed merge and add",
			dst: []*gen.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_RATING},
					},
				},
				{Id: "c"},
			},
			src: []*gen.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
					},
				},
				{Id: "b"},
			},
			want: []*gen.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
						{Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
					},
				},
				{Id: "b"},
				{Id: "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeChecks(proto.Merge, tt.dst, tt.src...)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("Checks() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name string
		dst  []*gen.HealthCheck
		id   string
		want []*gen.HealthCheck
	}{
		{
			name: "empty slice",
			dst:  nil,
			id:   "test",
			want: nil,
		},
		{
			name: "non-existent id",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "d",
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove first element",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "a",
			want: []*gen.HealthCheck{
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove middle element",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "b",
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
		},
		{
			name: "remove last element",
			dst: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "c",
			want: []*gen.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "remove only element",
			dst: []*gen.HealthCheck{
				{Id: "a"},
			},
			id:   "a",
			want: []*gen.HealthCheck{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveCheck(tt.dst, tt.id)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("Remove() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func intValue(value int) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{
		Value: &gen.HealthCheck_Value_IntValue{IntValue: int64(value)},
	}
}

func intNormalValues(values ...int) *gen.HealthCheck_Bounds_NormalValues {
	valuespb := make([]*gen.HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &gen.HealthCheck_Bounds_NormalValues{
		NormalValues: &gen.HealthCheck_Values{Values: valuespb},
	}
}

func intAbnormalValues(values ...int) *gen.HealthCheck_Bounds_AbnormalValues {
	valuespb := make([]*gen.HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &gen.HealthCheck_Bounds_AbnormalValues{
		AbnormalValues: &gen.HealthCheck_Values{Values: valuespb},
	}
}
