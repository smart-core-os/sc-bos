package opcua

import (
	"context"
	"errors"
	"fmt"
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

// Test_subscribeErrIsPermanent checks which subscribe failures we give up on. The list is a
// deny-list, so the case that matters most is that anything unrecognised is retried: getting
// that wrong abandons a working point for the lifetime of the config.
func Test_subscribeErrIsPermanent(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "not a status code", err: errors.New("connection refused"), want: false},
		{name: "sentinel", err: errUnexpectedResults, want: false},
		{name: "subscription closed", err: errSubscriptionClosed, want: false},
		{name: "context cancelled", err: context.Canceled, want: false},
		{name: "context deadline exceeded", err: context.DeadlineExceeded, want: false},

		// permanent: nothing about waiting makes any of these work
		{name: "StatusBadNodeIDUnknown", err: ua.StatusBadNodeIDUnknown, want: true},
		{name: "StatusBadNodeIDInvalid", err: ua.StatusBadNodeIDInvalid, want: true},
		{name: "StatusBadAttributeIDInvalid", err: ua.StatusBadAttributeIDInvalid, want: true},
		{name: "StatusBadUserAccessDenied", err: ua.StatusBadUserAccessDenied, want: true},
		{name: "StatusBadNotReadable", err: ua.StatusBadNotReadable, want: true},

		// the reported fault: our own RequestTimeout expiring against a slow server
		{name: "StatusBadTimeout", err: ua.StatusBadTimeout, want: false},
		// capacity, which clears as load drops
		{name: "StatusBadTooManyMonitoredItems", err: ua.StatusBadTooManyMonitoredItems, want: false},
		{name: "StatusBadTooManySubscriptions", err: ua.StatusBadTooManySubscriptions, want: false},
		{name: "StatusBadResourceUnavailable", err: ua.StatusBadResourceUnavailable, want: false},
		{name: "StatusBadOutOfMemory", err: ua.StatusBadOutOfMemory, want: false},
		// gopcua's AutoReconnect is already repairing these underneath us
		{name: "StatusBadSessionIDInvalid", err: ua.StatusBadSessionIDInvalid, want: false},
		{name: "StatusBadConnectionClosed", err: ua.StatusBadConnectionClosed, want: false},
		{name: "StatusBadServerNotConnected", err: ua.StatusBadServerNotConnected, want: false},
		// routine on a DA-wrapper server that has not polled its devices yet
		{name: "StatusBadNoCommunication", err: ua.StatusBadNoCommunication, want: false},
		{name: "StatusBadWaitingForInitialData", err: ua.StatusBadWaitingForInitialData, want: false},

		// a server may attach info bits to the code it sends, so the comparison has to mask
		{name: "permanent with info bits", err: ua.StatusCode(uint32(ua.StatusBadNodeIDUnknown) | 0x480), want: true},
		{name: "retryable with info bits", err: ua.StatusCode(uint32(ua.StatusBadTimeout) | 0x480), want: false},

		// Client.Subscribe wraps the code with %w rather than rendering it to a string, which
		// is what keeps the classification possible at all
		{name: "wrapped permanent", err: fmt.Errorf("monitor ns=2;s=Tag1: %w", ua.StatusBadNotReadable), want: true},
		{name: "doubly wrapped permanent", err: fmt.Errorf("outer: %w", fmt.Errorf("monitor ns=2;s=Tag1: %w", ua.StatusBadNodeIDUnknown)), want: true},
		{name: "wrapped retryable", err: fmt.Errorf("monitor ns=2;s=Tag1: %w", ua.StatusBadTimeout), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subscribeErrIsPermanent(tt.err); got != tt.want {
				t.Errorf("subscribeErrIsPermanent(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
