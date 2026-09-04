package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

// Test_statusSeverity checks the severity classification of a status code ignores the
// sub-code and info bits. 0x480 in particular is Good with the Overflow info bit set,
// which the driver used to treat as a read failure.
func Test_statusSeverity(t *testing.T) {
	tests := []struct {
		name                 string
		status               ua.StatusCode
		good, uncertain, bad bool
	}{
		{name: "StatusOK", status: ua.StatusOK, good: true},
		{name: "Good with overflow info bit", status: ua.StatusCode(0x480), good: true},
		{name: "StatusGoodCallAgain", status: ua.StatusGoodCallAgain, good: true},
		{name: "StatusUncertainSimulatedValue", status: ua.StatusUncertainSimulatedValue, uncertain: true},
		{name: "Uncertain with info bits", status: ua.StatusCode(0x40990480), uncertain: true},
		{name: "StatusBadNodeIDUnknown", status: ua.StatusCode(0x80340000), bad: true},
		{name: "Bad with info bits", status: ua.StatusCode(0x80340480), bad: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusIsGood(tt.status); got != tt.good {
				t.Errorf("statusIsGood(0x%X) = %v, want %v", uint32(tt.status), got, tt.good)
			}
			if got := statusIsUncertain(tt.status); got != tt.uncertain {
				t.Errorf("statusIsUncertain(0x%X) = %v, want %v", uint32(tt.status), got, tt.uncertain)
			}
			if got := statusIsBad(tt.status); got != tt.bad {
				t.Errorf("statusIsBad(0x%X) = %v, want %v", uint32(tt.status), got, tt.bad)
			}
		})
	}
}
