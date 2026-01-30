package pgxalerts

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
)

func Test_fieldMaskIncludesPath(t *testing.T) {
	tests := []struct {
		name string
		m    string
		p    string
		want bool
	}{
		{"nil", "", "", true},
		{"nil prop", "", "prop", true},
		{"has", "prop", "prop", true},
		{"includes", "bar,prop,foo", "prop", true},
		{"not includes", "bar,prop,foo", "baz", false},
		{"parent", "parent.child", "parent", true},
		{"parent.child", "parent.child", "parent.child", true},
		{"match parent", "parent", "parent.child", true},
		{"invert parent", "parent.child", "child", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m *fieldmaskpb.FieldMask
			if tt.m != "" {
				m = &fieldmaskpb.FieldMask{Paths: strings.Split(tt.m, ",")}
			}
			if got := fieldMaskIncludesPath(m, tt.p); got != tt.want {
				t.Errorf("fieldMaskIncludesPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertChangeForQuery(t *testing.T) {
	tests := []struct {
		name   string
		q      *alertpb.Alert_Query
		change *alertpb.PullAlertsResponse_Change
		want   *alertpb.PullAlertsResponse_Change
	}{
		{
			"nil query",
			nil,
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_ADD, NewValue: &alertpb.Alert{Id: "01", Description: "Add alert"}},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_ADD, NewValue: &alertpb.Alert{Id: "01", Description: "Add alert"}},
		},
		{
			"convert to add",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_UPDATE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "2"},
				NewValue: &alertpb.Alert{Id: "01", Floor: "1"},
			},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_ADD, NewValue: &alertpb.Alert{Id: "01", Floor: "1"}},
		},
		{
			"convert to remove",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_UPDATE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "1"},
				NewValue: &alertpb.Alert{Id: "01", Floor: "2"},
			},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_REMOVE, OldValue: &alertpb.Alert{Id: "01", Floor: "1"}},
		},
		{
			"update still applies",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_UPDATE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "1", Zone: "Z1"},
				NewValue: &alertpb.Alert{Id: "01", Floor: "1", Zone: "Z2"},
			},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_UPDATE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "1", Zone: "Z1"},
				NewValue: &alertpb.Alert{Id: "01", Floor: "1", Zone: "Z2"},
			},
		},
		{
			"add doesn't match",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_ADD,
				NewValue: &alertpb.Alert{Id: "01", Floor: "2"},
			},
			nil,
		},
		{
			"delete doesn't match",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_REMOVE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "2"},
			},
			nil,
		},
		{
			"update doesn't match",
			&alertpb.Alert_Query{Floor: "1"},
			&alertpb.PullAlertsResponse_Change{Type: types.ChangeType_UPDATE,
				OldValue: &alertpb.Alert{Id: "01", Floor: "2"},
				NewValue: &alertpb.Alert{Id: "01", Floor: "3"},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertChangeForQuery(tt.q, tt.change)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("convertChangeForQuery() (-want,+got)\n%s", diff)
			}
		})
	}
}

func Test_alertMatchesQuery(t *testing.T) {
	ack := true
	noAck := false
	tests := []struct {
		name string
		q    *alertpb.Alert_Query
		a    *alertpb.Alert
		want bool
	}{
		{"nil query", nil, &alertpb.Alert{Id: "any"}, true},
		{"empty query", &alertpb.Alert_Query{}, &alertpb.Alert{Id: "any"}, true},
		{"floor yes", &alertpb.Alert_Query{Floor: "1"}, &alertpb.Alert{Floor: "1"}, true},
		{"floor no", &alertpb.Alert_Query{Floor: "1"}, &alertpb.Alert{Floor: "2"}, false},
		{"floor absent", &alertpb.Alert_Query{Floor: "1"}, &alertpb.Alert{}, false},
		{"zone yes", &alertpb.Alert_Query{Zone: "1"}, &alertpb.Alert{Zone: "1"}, true},
		{"zone no", &alertpb.Alert_Query{Zone: "1"}, &alertpb.Alert{Zone: "2"}, false},
		{"zone absent", &alertpb.Alert_Query{Zone: "1"}, &alertpb.Alert{}, false},
		{"source yes", &alertpb.Alert_Query{Source: "1"}, &alertpb.Alert{Source: "1"}, true},
		{"source no", &alertpb.Alert_Query{Source: "1"}, &alertpb.Alert{Source: "2"}, false},
		{"source absent", &alertpb.Alert_Query{Source: "1"}, &alertpb.Alert{}, false},
		{"acknowledged yes", &alertpb.Alert_Query{Acknowledged: &ack}, &alertpb.Alert{Acknowledgement: &alertpb.Alert_Acknowledgement{}}, true},
		{"acknowledged no", &alertpb.Alert_Query{Acknowledged: &ack}, &alertpb.Alert{}, false},
		{"not acknowledged yes", &alertpb.Alert_Query{Acknowledged: &noAck}, &alertpb.Alert{}, true},
		{"not acknowledged no", &alertpb.Alert_Query{Acknowledged: &noAck}, &alertpb.Alert{Acknowledgement: &alertpb.Alert_Acknowledgement{}}, false},
		{"create_time before", &alertpb.Alert_Query{CreatedNotBefore: timestamppb.New(time.Unix(100, 1))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 0))}, false},
		{"create_time start", &alertpb.Alert_Query{CreatedNotBefore: timestamppb.New(time.Unix(100, 0))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 0))}, true},
		{"create_time after", &alertpb.Alert_Query{CreatedNotBefore: timestamppb.New(time.Unix(100, 0))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 1))}, true},
		{"create_time early", &alertpb.Alert_Query{CreatedNotAfter: timestamppb.New(time.Unix(100, 1))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 0))}, true},
		{"create_time end", &alertpb.Alert_Query{CreatedNotAfter: timestamppb.New(time.Unix(100, 0))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 0))}, true},
		{"create_time late", &alertpb.Alert_Query{CreatedNotAfter: timestamppb.New(time.Unix(100, 0))}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(100, 1))}, false},
		{"create_time within", &alertpb.Alert_Query{
			CreatedNotBefore: timestamppb.New(time.Unix(100, 0)),
			CreatedNotAfter:  timestamppb.New(time.Unix(200, 0)),
		}, &alertpb.Alert{CreateTime: timestamppb.New(time.Unix(150, 0))}, true},
		{"severity low", &alertpb.Alert_Query{SeverityNotBelow: 2, SeverityNotAbove: 5}, &alertpb.Alert{Severity: alertpb.Alert_Severity(1)}, false},
		{"severity bottom", &alertpb.Alert_Query{SeverityNotBelow: 2, SeverityNotAbove: 5}, &alertpb.Alert{Severity: alertpb.Alert_Severity(2)}, true},
		{"severity within", &alertpb.Alert_Query{SeverityNotBelow: 2, SeverityNotAbove: 5}, &alertpb.Alert{Severity: alertpb.Alert_Severity(4)}, true},
		{"severity top", &alertpb.Alert_Query{SeverityNotBelow: 2, SeverityNotAbove: 5}, &alertpb.Alert{Severity: alertpb.Alert_Severity(5)}, true},
		{"severity high", &alertpb.Alert_Query{SeverityNotBelow: 2, SeverityNotAbove: 5}, &alertpb.Alert{Severity: alertpb.Alert_Severity(6)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := alertMatchesQuery(tt.q, tt.a); got != tt.want {
				t.Errorf("alertMatchesQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
