package healthpb

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFaultCheck_AddOrUpdateFault(t *testing.T) {
	tests := map[string]struct {
		initial []*HealthCheck_Error
		new     *HealthCheck_Error
		want    []*HealthCheck_Error
	}{
		"nil initial": {
			initial: nil,
			new:     newFault("", "", "summary1", "desc1"),
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
		},
		"nil new": {
			initial: []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     nil,
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
		},
		"add new to end": {
			initial: []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2")},
		},
		"add new to start": {
			initial: []*HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			new:     newFault("", "", "summary1", "desc1"),
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2")},
		},
		"add new in middle": {
			initial: []*HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary3", "desc3")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1"), newFault("", "", "summary2", "desc2"), newFault("", "", "summary3", "desc3")},
		},
		"replace existing by summary": {
			initial: []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			new:     newFault("", "", "summary1", "desc2"),
			want:    []*HealthCheck_Error{newFault("", "", "summary1", "desc2")},
		},
		"replace existing by system/code": {
			initial: []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("sys1", "code1", "summary2", "desc2"),
			want:    []*HealthCheck_Error{newFault("sys1", "code1", "summary2", "desc2")},
		},
		"add new with different system/code": {
			initial: []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("sys2", "code2", "summary1", "desc2"), // same summary
			want:    []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1"), newFault("sys2", "code2", "summary1", "desc2")},
		},
		"replace existing with system/code, add new by summary": {
			initial: []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			new:     newFault("", "", "summary2", "desc2"),
			want:    []*HealthCheck_Error{newFault("", "", "summary2", "desc2"), newFault("sys1", "code1", "summary1", "desc1")},
		},
		"multiple initial, replace one": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			new: newFault("sys1", "code1", "summary2", "desc2-updated"),
			want: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2-updated"),
				newFault("", "", "summary3", "desc3"),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
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
		initial       []*HealthCheck_Error
		set           *HealthCheck_Error
		wantFaults    []*HealthCheck_Error
		wantNormality HealthCheck_Normality
	}{
		"set nil clears all faults": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
			},
			set:           nil,
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
		"set nil on empty does nothing": {
			initial:       nil,
			set:           nil,
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
		"set fault on empty list": {
			initial:       nil,
			set:           newFault("", "", "summary1", "desc1"),
			wantFaults:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"set fault replaces single existing fault": {
			initial:       []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			set:           newFault("", "", "summary2", "desc2"),
			wantFaults:    []*HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"set fault replaces multiple existing faults": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			set:           newFault("sys2", "code2", "newSummary", "newDesc"),
			wantFaults:    []*HealthCheck_Error{newFault("sys2", "code2", "newSummary", "newDesc")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"set fault with system/code": {
			initial:       nil,
			set:           newFault("sys1", "code1", "summary1", "desc1"),
			wantFaults:    []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"set fault without system/code": {
			initial:       []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			set:           newFault("", "", "summary2", "desc2"),
			wantFaults:    []*HealthCheck_Error{newFault("", "", "summary2", "desc2")},
			wantNormality: HealthCheck_ABNORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			normality := HealthCheck_NORMAL
			if len(tt.initial) > 0 {
				normality = HealthCheck_ABNORMAL
			}
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
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
			if got := fc.check.GetReliability().GetState(); got != HealthCheck_Reliability_RELIABLE {
				t.Errorf("SetFault() reliability = %v, want %v", got, HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func TestFaultCheck_ClearFaults(t *testing.T) {
	tests := map[string]struct {
		initial       []*HealthCheck_Error
		wantFaults    []*HealthCheck_Error
		wantNormality HealthCheck_Normality
	}{
		"clear empty": {
			initial:       nil,
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
		"clear one fault": {
			initial:       []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
		"clear multiple faults": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
				Normality: HealthCheck_ABNORMAL, // start as abnormal
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
			if got := fc.check.GetReliability().GetState(); got != HealthCheck_Reliability_RELIABLE {
				t.Errorf("ClearFaults() reliability = %v, want %v", got, HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func TestFaultCheck_RemoveFault(t *testing.T) {
	tests := map[string]struct {
		initial              []*HealthCheck_Error
		remove               *HealthCheck_Error
		wantFaults           []*HealthCheck_Error
		wantNormality        HealthCheck_Normality
		skipReliabilityCheck bool // when true, don't check reliability (e.g., when nil is passed)
	}{
		"remove nil does nothing": {
			initial:              []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:               nil,
			wantFaults:           []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality:        HealthCheck_ABNORMAL,
			skipReliabilityCheck: true, // RemoveFault returns early for nil, so no write happens
		},
		"remove from empty": {
			initial:       nil,
			remove:        newFault("", "", "summary1", "desc1"),
			wantFaults:    nil,
			wantNormality: HealthCheck_NORMAL,
		},
		"remove only fault by summary": {
			initial:       []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:        newFault("", "", "summary1", "desc2"), // different description
			wantFaults:    []*HealthCheck_Error{},
			wantNormality: HealthCheck_NORMAL,
		},
		"remove only fault by system/code": {
			initial:       []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			remove:        newFault("sys1", "code1", "summary2", "desc2"), // different summary/description
			wantFaults:    []*HealthCheck_Error{},
			wantNormality: HealthCheck_NORMAL,
		},
		"remove non-existent fault by summary": {
			initial:       []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			remove:        newFault("", "", "summary2", "desc2"),
			wantFaults:    []*HealthCheck_Error{newFault("", "", "summary1", "desc1")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove non-existent fault by system/code": {
			initial:       []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			remove:        newFault("sys2", "code2", "summary1", "desc1"),
			wantFaults:    []*HealthCheck_Error{newFault("sys1", "code1", "summary1", "desc1")},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove first of multiple by summary": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary1", "desc1"),
			wantFaults: []*HealthCheck_Error{
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove middle of multiple by summary": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary2", "desc2"),
			wantFaults: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove last of multiple by summary": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("", "", "summary3", "desc3"),
			wantFaults: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary2", "desc2"),
			},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove fault by system/code from mixed list": {
			initial: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("sys1", "code1", "summary2", "desc2"),
				newFault("", "", "summary3", "desc3"),
			},
			remove: newFault("sys1", "code1", "summary2", "desc2"),
			wantFaults: []*HealthCheck_Error{
				newFault("", "", "summary1", "desc1"),
				newFault("", "", "summary3", "desc3"),
			},
			wantNormality: HealthCheck_ABNORMAL,
		},
		"remove fault from multiple system/code faults": {
			initial: []*HealthCheck_Error{
				newFault("sys1", "code1", "summary1", "desc1"),
				newFault("sys2", "code2", "summary2", "desc2"),
				newFault("sys3", "code3", "summary3", "desc3"),
			},
			remove: newFault("sys2", "code2", "summary2", "desc2"),
			wantFaults: []*HealthCheck_Error{
				newFault("sys1", "code1", "summary1", "desc1"),
				newFault("sys3", "code3", "summary3", "desc3"),
			},
			wantNormality: HealthCheck_ABNORMAL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			normality := HealthCheck_NORMAL
			if len(tt.initial) > 0 {
				normality = HealthCheck_ABNORMAL
			}
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
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
				if got := fc.check.GetReliability().GetState(); got != HealthCheck_Reliability_RELIABLE {
					t.Errorf("RemoveFault() reliability = %v, want %v", got, HealthCheck_Reliability_RELIABLE)
				}
			}
		})
	}
}

func TestFaultCheck_MarkFailed(t *testing.T) {
	tests := map[string]struct {
		err         error
		wantSummary string
	}{
		"uses error message as summary": {
			err:         errors.New("connection timeout"),
			wantSummary: "connection timeout",
		},
		"replaces existing fault": {
			err:         errors.New("new error"),
			wantSummary: "new error",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
					CurrentFaults: []*HealthCheck_Error{newFault("", "", "old error", "")},
				}},
				Normality: HealthCheck_ABNORMAL,
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.MarkFailed(tt.err)
			faults := fc.check.GetFaults().GetCurrentFaults()
			if len(faults) != 1 {
				t.Fatalf("MarkFailed() fault count = %d, want 1", len(faults))
			}
			if got := faults[0].GetSummaryText(); got != tt.wantSummary {
				t.Errorf("MarkFailed() summary = %q, want %q", got, tt.wantSummary)
			}
			if got := fc.check.GetNormality(); got != HealthCheck_ABNORMAL {
				t.Errorf("MarkFailed() normality = %v, want %v", got, HealthCheck_ABNORMAL)
			}
			if got := fc.check.GetReliability().GetState(); got != HealthCheck_Reliability_RELIABLE {
				t.Errorf("MarkFailed() reliability = %v, want %v", got, HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func TestFaultCheck_MarkRunning(t *testing.T) {
	tests := map[string]struct {
		initial []*HealthCheck_Error
	}{
		"clears faults when unhealthy": {
			initial: []*HealthCheck_Error{newFault("", "", "some error", "")},
		},
		"no-op when already healthy": {
			initial: nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			normality := HealthCheck_NORMAL
			if len(tt.initial) > 0 {
				normality = HealthCheck_ABNORMAL
			}
			c := &HealthCheck{
				Check: &HealthCheck_Faults_{Faults: &HealthCheck_Faults{
					CurrentFaults: tt.initial,
				}},
				Normality: normality,
			}
			fc, err := newFaultCheck(c)
			if err != nil {
				t.Fatalf("newFaultCheck() error = %v", err)
			}
			fc.MarkRunning()
			if faults := fc.check.GetFaults().GetCurrentFaults(); len(faults) != 0 {
				t.Errorf("MarkRunning() faults = %v, want none", faults)
			}
			if got := fc.check.GetNormality(); got != HealthCheck_NORMAL {
				t.Errorf("MarkRunning() normality = %v, want %v", got, HealthCheck_NORMAL)
			}
			if got := fc.check.GetReliability().GetState(); got != HealthCheck_Reliability_RELIABLE {
				t.Errorf("MarkRunning() reliability = %v, want %v", got, HealthCheck_Reliability_RELIABLE)
			}
		})
	}
}

func newFault(system, code, summary, desc string) *HealthCheck_Error {
	res := &HealthCheck_Error{
		SummaryText: summary,
		DetailsText: desc,
	}
	if system != "" || code != "" {
		res.Code = &HealthCheck_Error_Code{
			System: system,
			Code:   code,
		}
	}
	return res
}
