package opcua

import (
	"slices"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Test_warnIfRevised checks which revisions reach an operator. A server is allowed to revise
// what we asked for, but only a queue shallower than requested is a problem: that is the one
// that overflows and flags every value with the Overflow info bit. A deeper queue than
// requested is a server applying its own minimum, and warning about it would put a line per
// monitored item in the log on every config load of a server behaving perfectly well.
func Test_warnIfRevised(t *testing.T) {
	const requestedQueue = 10
	const requestedSampling = 250 * time.Millisecond

	tests := []struct {
		name          string
		res           *ua.MonitoredItemCreateResult
		wantWarnings  []string
		wantDebugMsgs []string
	}{
		{
			name: "honoured as requested",
			res:  &ua.MonitoredItemCreateResult{RevisedSamplingInterval: 250, RevisedQueueSize: requestedQueue},
		},
		{
			name:         "queue revised down",
			res:          &ua.MonitoredItemCreateResult{RevisedSamplingInterval: 250, RevisedQueueSize: 4},
			wantWarnings: []string{"server revised the queue size down"},
		},
		{
			name:          "queue revised up",
			res:           &ua.MonitoredItemCreateResult{RevisedSamplingInterval: 250, RevisedQueueSize: 16},
			wantDebugMsgs: []string{"server revised the queue size up"},
		},
		{
			name:         "sampling interval revised either way",
			res:          &ua.MonitoredItemCreateResult{RevisedSamplingInterval: 1000, RevisedQueueSize: requestedQueue},
			wantWarnings: []string{"server revised the sampling interval"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			c := &Client{
				logger:           zap.New(core),
				samplingInterval: requestedSampling,
				queueSize:        requestedQueue,
			}

			c.warnIfRevised(ua.NewStringNodeID(2, "Tag1"), tt.res)

			if got := messages(logs.FilterLevelExact(zapcore.WarnLevel)); !slices.Equal(got, tt.wantWarnings) {
				t.Errorf("warnings = %v, want %v", got, tt.wantWarnings)
			}
			if got := messages(logs.FilterLevelExact(zapcore.DebugLevel)); !slices.Equal(got, tt.wantDebugMsgs) {
				t.Errorf("debug messages = %v, want %v", got, tt.wantDebugMsgs)
			}
		})
	}
}

func messages(logs *observer.ObservedLogs) []string {
	var msgs []string
	for _, e := range logs.All() {
		msgs = append(msgs, e.Message)
	}
	return msgs
}
