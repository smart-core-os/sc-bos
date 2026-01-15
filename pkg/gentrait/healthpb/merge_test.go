package healthpb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name           string
		src, dst, want *healthpb.HealthCheck
		ignore         string
	}{
		{
			name: "oneof changes",
			src: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			dst: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{
					Faults: &healthpb.HealthCheck_Faults{},
				},
			},
			want: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
		},
		{
			name: "oneof changes reverse",
			src: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{
					Faults: &healthpb.HealthCheck_Faults{},
				},
			},
			dst: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						CurrentValue: intValue(1),
					},
				},
			},
			want: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{
					Faults: &healthpb.HealthCheck_Faults{},
				},
			},
		},
		{
			name: "bounds.normal_values",
			src: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
			dst: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intNormalValues(10, 20),
					},
				},
			},
			want: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intNormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "bounds.abnormal_values",
			src: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
			dst: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intAbnormalValues(10, 20),
					},
				},
			},
			want: &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: intAbnormalValues(1, 2),
					},
				},
			},
		},
		{
			name: "compliance_impact",
			src: &healthpb.HealthCheck{
				ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
					{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
				},
			},
			dst: &healthpb.HealthCheck{
				ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
					{Contribution: healthpb.HealthCheck_ComplianceImpact_RATING},
				},
			},
			want: &healthpb.HealthCheck{
				ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
					{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
				},
			},
		},
		{
			ignore: "consistency with other merge functions",
			name:   "timestamp",
			src: &healthpb.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
			dst: &healthpb.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(10, 100)),
			},
			want: &healthpb.HealthCheck{
				AbnormalTime: timestamppb.New(time.Unix(100, 0)),
			},
		},
		{
			name: "create_time both nil",
			src:  &healthpb.HealthCheck{},
			dst:  &healthpb.HealthCheck{},
			want: &healthpb.HealthCheck{},
		},
		{
			name: "create_time src nil",
			src:  &healthpb.HealthCheck{},
			dst: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time dst nil",
			src: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &healthpb.HealthCheck{},
			want: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src < dst",
			src: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			dst: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			want: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
		},
		{
			name: "create_time src > dst",
			src: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(20, 0)),
			},
			dst: &healthpb.HealthCheck{
				CreateTime: timestamppb.New(time.Unix(10, 0)),
			},
			want: &healthpb.HealthCheck{
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
		dst  []*healthpb.HealthCheck
		src  []*healthpb.HealthCheck
		want []*healthpb.HealthCheck
	}{
		{
			name: "empty dst",
			dst:  nil,
			src: []*healthpb.HealthCheck{
				{Id: "b"},
				{Id: "a"},
			},
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "empty src",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
			src: nil,
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "merge existing",
			dst: []*healthpb.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_RATING},
					},
				},
			},
			src: []*healthpb.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
			want: []*healthpb.HealthCheck{
				{
					Id: "test",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
					},
				},
			},
		},
		{
			name: "add new checks",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
			src: []*healthpb.HealthCheck{
				{Id: "b"},
				{Id: "d"},
			},
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
				{Id: "d"},
			},
		},
		{
			name: "mixed merge and add",
			dst: []*healthpb.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_RATING},
					},
				},
				{Id: "c"},
			},
			src: []*healthpb.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
					},
				},
				{Id: "b"},
			},
			want: []*healthpb.HealthCheck{
				{
					Id: "a",
					ComplianceImpacts: []*healthpb.HealthCheck_ComplianceImpact{
						{Contribution: healthpb.HealthCheck_ComplianceImpact_FAIL},
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
		dst  []*healthpb.HealthCheck
		id   string
		want []*healthpb.HealthCheck
	}{
		{
			name: "empty slice",
			dst:  nil,
			id:   "test",
			want: nil,
		},
		{
			name: "non-existent id",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "d",
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove first element",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "a",
			want: []*healthpb.HealthCheck{
				{Id: "b"},
				{Id: "c"},
			},
		},
		{
			name: "remove middle element",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "b",
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "c"},
			},
		},
		{
			name: "remove last element",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
			id: "c",
			want: []*healthpb.HealthCheck{
				{Id: "a"},
				{Id: "b"},
			},
		},
		{
			name: "remove only element",
			dst: []*healthpb.HealthCheck{
				{Id: "a"},
			},
			id:   "a",
			want: []*healthpb.HealthCheck{},
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

func intValue(value int) *healthpb.HealthCheck_Value {
	return &healthpb.HealthCheck_Value{
		Value: &healthpb.HealthCheck_Value_IntValue{IntValue: int64(value)},
	}
}

func intNormalValues(values ...int) *healthpb.HealthCheck_Bounds_NormalValues {
	valuespb := make([]*healthpb.HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &healthpb.HealthCheck_Bounds_NormalValues{
		NormalValues: &healthpb.HealthCheck_Values{Values: valuespb},
	}
}

func intAbnormalValues(values ...int) *healthpb.HealthCheck_Bounds_AbnormalValues {
	valuespb := make([]*healthpb.HealthCheck_Value, len(values))
	for i, v := range values {
		valuespb[i] = intValue(v)
	}
	return &healthpb.HealthCheck_Bounds_AbnormalValues{
		AbnormalValues: &healthpb.HealthCheck_Values{Values: valuespb},
	}
}
