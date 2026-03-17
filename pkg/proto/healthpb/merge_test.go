package healthpb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name           string
		src, dst, want *HealthCheck
		ignore         string
	}{
		{
			name: "oneof changes",
			src: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			dst: &HealthCheck{
				Check: &HealthCheck_Faults_{
					Faults: &HealthCheck_Faults{},
				},
			},
			want: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
		},
		{
			name: "oneof changes reverse",
			src: &HealthCheck{
				Check: &HealthCheck_Faults_{
					Faults: &HealthCheck_Faults{},
				},
			},
			dst: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			want: &HealthCheck{
				Check: &HealthCheck_Faults_{
					Faults: &HealthCheck_Faults{},
				},
			},
		},
		{
			name: "bounds.normal_values",
			src: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
			dst: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intNormalValues(10, 20),
					},
				},
			},
			want: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "bounds.abnormal_values",
			src: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
			dst: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intAbnormalValues(10, 20),
					},
				},
			},
			want: &HealthCheck{
				Check: &HealthCheck_Bounds_{
					Bounds: &HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "compliance_impact",
			src: &HealthCheck{
				ComplianceImpacts: []*HealthCheck_ComplianceImpact{
					{Contribution: HealthCheck_ComplianceImpact_FAIL},
				},
			},
			dst: &HealthCheck{
				ComplianceImpacts: []*HealthCheck_ComplianceImpact{
					{Contribution: HealthCheck_ComplianceImpact_RATING},
				},
			},
			want: &HealthCheck{
				ComplianceImpacts: []*HealthCheck_ComplianceImpact{
					{Contribution: HealthCheck_ComplianceImpact_FAIL},
				},
			},
		},
		{
			ignore: "consistency with other merge functions",
			name:   "timestamp",
			src: &HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
			dst: &HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(10, 100)),
			},
			want: &HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
		},
		{
			name: "create_time both nil",
			src:  &HealthCheck{},
			dst:  &HealthCheck{},
			want: &HealthCheck{},
		},
		{
			name: "create_time src nil",
			src:  &HealthCheck{},
			dst: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time dst nil",
			src: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &HealthCheck{},
			want: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src < dst",
			src: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			want: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src > dst",
			src: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			dst: &HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &HealthCheck{
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
		dst  []*HealthCheck
		src  []*HealthCheck
		want []*HealthCheck
	}{
		{
			name: "empty dst",
			dst:  nil,
			src: []*HealthCheck{
				{Id: "b"},
				{Id: "a"},
			},
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "empty src",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
			src: nil,
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "merge existing",
			dst: []*HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_RATING},
					},
				},
			},
			src: []*HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
			want: []*HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
		},
		{
			name: "add new checks",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
			src: []*HealthCheck{
				{Id: "b"},
				{Id: "d"},
			},
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
				{Id: "d"},
			},
		},
		{
			name: "mixed merge and add",
			dst: []*HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_RATING},
					},
				},
				{Id: "c"},
			},
			src: []*HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_FAIL},
					},
				},
				{Id: "b"},
			},
			want: []*HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*HealthCheck_ComplianceImpact{
						{Contribution: HealthCheck_ComplianceImpact_FAIL},
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
		dst  []*HealthCheck
		id   string
		want []*HealthCheck
	}{
		{
			name: "empty slice",
			dst:  nil,
			id:   "test",
			want: nil,
		},
		{
			name: "non-existent id",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "d",
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove first element",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "a",
			want: []*HealthCheck{
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove middle element",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "b",
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
		},
		{
			name: "remove last element",
			dst: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "c",
			want: []*HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "remove only element",
			dst: []*HealthCheck{
				{Id: "a"},
			},
			id:   "a",
			want: []*HealthCheck{},
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

func intValue(value int) *HealthCheck_Value {
	return &HealthCheck_Value{
		Value: &HealthCheck_Value_IntValue{IntValue: int64(value)},
	}
}

func intNormalValues(values ...int) *HealthCheck_Bounds_NormalValues {
	valuespb := make([]*HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &HealthCheck_Bounds_NormalValues{
		NormalValues: &HealthCheck_Values{Values: valuespb},
	}
}

func intAbnormalValues(values ...int) *HealthCheck_Bounds_AbnormalValues {
	valuespb := make([]*HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &HealthCheck_Bounds_AbnormalValues{
		AbnormalValues: &HealthCheck_Values{Values: valuespb},
	}
}
