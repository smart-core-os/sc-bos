package healthpb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestFaultCheck_AddOrUpdateFault(t *testing.T) {
	tests := map[string]struct {
		initial []*healthpb.HealthCheck_Error
		new     *healthpb.HealthCheck_Error
		want    []*healthpb.HealthCheck_Error
	}{
		"nil initial": {
			initial: nil,
			new:     newFault("", "", "summary1", "desc1"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
		},
		"nil new": {
			initial: []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     nil,
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
		},
		"add new to end": {
			initial: []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2")},
		},
		"add new to start": {
			initial: []*healthpb.HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			new:     newFault("", "", "summary1", "desc1"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2")},
		},
		"add new in middle": {
			initial: []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary3", "desc3")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2"), newFault("", "", "summary3", "desc3")},
		},
		"replace existing by summary": {
			initial: []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     newFault("", "", "summary1", "desc2"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc2")},
		},
		"replace existing by system/code": {
			initial: []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("sys1", "code1", "summary2", "desc2"),
			want:    []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary2", "desc2")},
		},
		"add new with different system/code": {
			initial: []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("sys2", "code2", "summary1", "desc2"), // same summary
			want:    []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1"), newFault("sys2", "code2", "summary1", "desc2")},
		},
		"replace existing with system/code, add new by summary": {
			initial: []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*healthpb.HealthCheck_Error{newFault("", "", "summary2", "desc2"), newFault("sys1", "code1", "summary1", "desc1")},
		},
		"multiple initial, replace one": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			new: newFault("sys1", "code1", "summary2", "desc2-updated"),
			want: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2-updated"),
				newFault("", "", "summary3", "desc3"),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.AddOrUpdateFault(tt.new)
			if diff := cmp.Diff(tt.want, fc.check.GetFaults().GetCurrentFaults(), protocmp.Transform()); diff != "" {
				t.Errorf("AddOrUpdateFault() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFaultCheck_SetFault(t *testing.T) {
	tests := map[string]struct {
		initial       []*healthpb.HealthCheck_Error
		set           *healthpb.HealthCheck_Error
		wantFaults    []*healthpb.HealthCheck_Error
		wantNormality healthpb.HealthCheck_Normality
	}{
		"set nil clears all faults": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
			},
			set:           nil,
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"set nil on empty does nothing": {
			initial:       nil,
			set:           nil,
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"set fault on empty list": {
			initial:       nil,
			set:           newFault("", "", "summary1", "desc1"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"set fault replaces single existing fault": {
			initial:       []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			set:           newFault("", "", "summary2", "desc2"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"set fault replaces multiple existing faults": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			set:           newFault("sys2", "code2", "newSummary", "newDesc"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("sys2", "code2", "newSummary", "newDesc")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"set fault with system/code": {
			initial:       nil,
			set:           newFault("sys1", "code1", "summary1", "desc1"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"set fault without system/code": {
			initial:       []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			set:           newFault("", "", "summary2", "desc2"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			normality := healthpb.HealthCheck_NORMAL
			if len(tt.initial) > 0 {
				normality = healthpb.HealthCheck_ABNORMAL
			}
			c := &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
				Normality: normality,
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.SetFault(tt.set)

			if diff := cmp.Diff(tt.wantFaults, fc.check.GetFaults().GetCurrentFaults(), protocmp.Transform()); diff != "" {
				t.Errorf("SetFault() faults mismatch (-want +got):\n%s", diff)
			}
			if got := fc.check.GetNormality(); got != tt.wantNormality {
				t.Errorf("SetFault() normality = %v, want %v", got, tt.wantNormality)
			}
			if got := fc.check.GetReliability().GetState(); got != healthpb.HealthCheck_Reliability_RELIABLE {
				t.Errorf("SetFault() reliability = %v, want %v", got, healthpb.HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func TestFaultCheck_ClearFaults(t *testing.T) {
	tests := map[string]struct {
		initial       []*healthpb.HealthCheck_Error
		wantFaults    []*healthpb.HealthCheck_Error
		wantNormality healthpb.HealthCheck_Normality
	}{
		"clear empty": {
			initial:       nil,
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"clear one fault": {
			initial:       []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"clear multiple faults": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
				Normality: healthpb.HealthCheck_ABNORMAL, // start as abnormal
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.ClearFaults()

			if diff := cmp.Diff(tt.wantFaults, fc.check.GetFaults().GetCurrentFaults(), protocmp.Transform()); diff != "" {
				t.Errorf("ClearFaults() faults mismatch (-want +got):\n%s", diff)
			}
			if got := fc.check.GetNormality(); got != tt.wantNormality {
				t.Errorf("ClearFaults() normality = %v, want %v", got, tt.wantNormality)
			}
			if got := fc.check.GetReliability().GetState(); got != healthpb.HealthCheck_Reliability_RELIABLE {
				t.Errorf("ClearFaults() reliability = %v, want %v", got, healthpb.HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func TestFaultCheck_RemoveFault(t *testing.T) {
	tests := map[string]struct {
		initial              []*healthpb.HealthCheck_Error
		remove               *healthpb.HealthCheck_Error
		wantFaults           []*healthpb.HealthCheck_Error
		wantNormality        healthpb.HealthCheck_Normality
		skipReliabilityCheck bool // when true, don't check reliability (e.g., when nil is passed)
	}{
		"remove nil does nothing": {
			initial:              []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:               nil,
			wantFaults:           []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality:        healthpb.HealthCheck_ABNORMAL,
			skipReliabilityCheck: true, // RemoveFault returns early for nil, so no write happens
		},
		"remove from empty": {
			initial:       nil,
			remove:        newFault("", "", "summary1", "desc1"),
			wantFaults:    nil,
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"remove only fault by summary": {
			initial:       []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:        newFault("", "", "summary1", "desc2"), // different description
			wantFaults:    []*healthpb.HealthCheck_Error{},
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"remove only fault by system/code": {
			initial:       []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			remove:        newFault("sys1", "code1", "summary2", "desc2"), // different summary/description
			wantFaults:    []*healthpb.HealthCheck_Error{},
			wantNormality: healthpb.HealthCheck_NORMAL,
		},
		"remove non-existent fault by summary": {
			initial:       []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:        newFault("", "", "summary2", "desc2"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove non-existent fault by system/code": {
			initial:       []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			remove:        newFault("sys2", "code2", "summary1", "desc1"),
			wantFaults:    []*healthpb.HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove first of multiple by summary": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary1", "desc1"),
			wantFaults: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove middle of multiple by summary": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary2", "desc2"),
			wantFaults: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove last of multiple by summary": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary3", "desc3"),
			wantFaults: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
			},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove fault by system/code from mixed list": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("sys1", "code1", "summary2", "desc2"),
			wantFaults: []*healthpb.HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
		"remove fault from multiple system/code faults": {
			initial: []*healthpb.HealthCheck_Error{
				newFault("sys1", "code1", "summary1", "desc1"),
				newFault("sys2", "code2", "summary2", "desc2"),
				newFault("sys3", "code3", "summary3", "desc3"),
			},
			remove: newFault("sys2", "code2", "summary2", "desc2"),
			wantFaults: []*healthpb.HealthCheck_Error{
				newFault("sys1", "code1", "summary1", "desc1"),
				newFault("sys3", "code3", "summary3", "desc3"),
			},
			wantNormality: healthpb.HealthCheck_ABNORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			normality := healthpb.HealthCheck_NORMAL
			if len(tt.initial) > 0 {
				normality = healthpb.HealthCheck_ABNORMAL
			}
			c := &healthpb.HealthCheck{
				Check: &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
				Normality: normality,
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.RemoveFault(tt.remove)

			if diff := cmp.Diff(tt.wantFaults, fc.check.GetFaults().GetCurrentFaults(), protocmp.Transform()); diff != "" {
				t.Errorf("RemoveFault() faults mismatch (-want +got):\n%s", diff)
			}
			if got := fc.check.GetNormality(); got != tt.wantNormality {
				t.Errorf("RemoveFault() normality = %v, want %v", got, tt.wantNormality)
			}
			if !tt.skipReliabilityCheck {
				if got := fc.check.GetReliability().GetState(); got != healthpb.HealthCheck_Reliability_RELIABLE {
					t.Errorf("RemoveFault() reliability = %v, want %v", got, healthpb.HealthCheck_Reliability_RELIABLE)
				}
			}
		})
	}
}

func newFault(system, code, summary, desc string) *healthpb.HealthCheck_Error {
	res := &healthpb.HealthCheck_Error{
		SummaryText: summary,
		DetailsText: desc,
	}
	if system != "" || code != "" {
		res.Code = &healthpb.HealthCheck_Error_Code{
			System: system,
			Code:   code,
		}
	}
	return res
}
