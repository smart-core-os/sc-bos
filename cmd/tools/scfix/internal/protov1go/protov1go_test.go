package protov1go

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestProtov1go(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"enter leave history", "testdata/enter_leave_history.txtar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtest.Run(t, tt.file, run)
		})
	}
}

func TestTryRenameSymbol(t *testing.T) {
	renames := map[string]string{
		"EnterLeaveHistory": "EnterLeaveSensorHistory",
	}

	r := &serviceRewriter{renames: renames}

	tests := []struct {
		input string
		want  string
	}{
		// Server patterns
		{"UnimplementedEnterLeaveHistoryServer", "UnimplementedEnterLeaveSensorHistoryServer"},
		{"UnsafeEnterLeaveHistoryServer", "UnsafeEnterLeaveSensorHistoryServer"},
		{"RegisterEnterLeaveHistoryServer", "RegisterEnterLeaveSensorHistoryServer"},
		{"EnterLeaveHistoryServer", "EnterLeaveSensorHistoryServer"},

		// Client patterns
		{"NewEnterLeaveHistoryClient", "NewEnterLeaveSensorHistoryClient"},
		{"EnterLeaveHistoryClient", "EnterLeaveSensorHistoryClient"},

		// Wrapper patterns
		{"WrapEnterLeaveHistory", "WrapEnterLeaveSensorHistory"},
		{"EnterLeaveHistoryWrapper", "EnterLeaveSensorHistoryWrapper"},

		// Router patterns
		{"NewEnterLeaveHistoryRouter", "NewEnterLeaveSensorHistoryRouter"},
		{"EnterLeaveHistoryRouter", "EnterLeaveSensorHistoryRouter"},
		{"WithEnterLeaveHistoryClientFactory", "WithEnterLeaveSensorHistoryClientFactory"},

		// ServiceDesc pattern
		{"EnterLeaveHistory_ServiceDesc", "EnterLeaveSensorHistory_ServiceDesc"},

		// Unchanged patterns
		{"SomeOtherSymbol", "SomeOtherSymbol"},
		{"MeterApiClient", "MeterApiClient"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := r.tryRenameSymbol(tt.input)
			if got != tt.want {
				t.Errorf("tryRenameSymbol(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
