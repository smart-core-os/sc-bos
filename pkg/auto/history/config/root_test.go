package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSource_MaxIntervalDuration(t *testing.T) {
	tests := map[string]struct {
		raw  string
		want time.Duration
	}{
		"absent": {raw: `{}`, want: 0},
		"set":    {raw: `{"maxInterval":"1h"}`, want: time.Hour},
		"zero":   {raw: `{"maxInterval":"0s"}`, want: 0},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var src Source
			if err := json.Unmarshal([]byte(tt.raw), &src); err != nil {
				t.Fatal(err)
			}
			if got := src.MaxIntervalDuration(); got != tt.want {
				t.Errorf("MaxIntervalDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
