package logpb

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

func TestFilterMessages_unspecified_passthrough(t *testing.T) {
	msgs := []*logpb.LogMessage{
		{Level: logpb.LogLevel_DEBUG},
		{Level: logpb.LogLevel_INFO},
		{Level: logpb.LogLevel_WARN},
	}
	out := filterMessages(msgs, logpb.LogLevel_LEVEL_UNSPECIFIED)
	if len(out) != len(msgs) {
		t.Errorf("got %d messages, want %d", len(out), len(msgs))
	}
}

func TestFilterMessages_minLevel(t *testing.T) {
	msgs := []*logpb.LogMessage{
		{Level: logpb.LogLevel_DEBUG, Message: "dbg"},
		{Level: logpb.LogLevel_INFO, Message: "info"},
		{Level: logpb.LogLevel_WARN, Message: "warn"},
		{Level: logpb.LogLevel_ERROR, Message: "err"},
	}

	tests := []struct {
		minLevel logpb.LogLevel_Level
		wantMsgs []string
	}{
		{logpb.LogLevel_DEBUG, []string{"dbg", "info", "warn", "err"}},
		{logpb.LogLevel_INFO, []string{"info", "warn", "err"}},
		{logpb.LogLevel_WARN, []string{"warn", "err"}},
		{logpb.LogLevel_ERROR, []string{"err"}},
	}

	for _, tt := range tests {
		t.Run(tt.minLevel.String(), func(t *testing.T) {
			out := filterMessages(msgs, tt.minLevel)
			if len(out) != len(tt.wantMsgs) {
				t.Fatalf("got %d messages, want %d", len(out), len(tt.wantMsgs))
			}
			for i, w := range tt.wantMsgs {
				if out[i].Message != w {
					t.Errorf("out[%d].Message = %q, want %q", i, out[i].Message, w)
				}
			}
		})
	}
}

func TestFilterMessages_empty(t *testing.T) {
	out := filterMessages(nil, logpb.LogLevel_INFO)
	if len(out) != 0 {
		t.Errorf("filtering nil: got %d messages, want 0", len(out))
	}
}
