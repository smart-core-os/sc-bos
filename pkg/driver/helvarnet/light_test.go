package helvarnet

import (
	"testing"
	"time"
)

func Test_parseGetCompletionTimeResponse(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("failed to load Europe/London: %v", err)
	}

	tests := []struct {
		name    string
		r       string
		loc     *time.Location
		want    time.Time
		wantErr bool
	}{
		{
			name: "no timezone, epoch taken as-is",
			r:    "?V:1,C:170,@10.106.4.40=1754495355#",
			loc:  nil,
			want: time.Unix(1754495355, 0),
		},
		{
			name: "UTC timezone, epoch taken as-is",
			r:    "?V:1,C:170,@10.106.4.40=1754495355#",
			loc:  time.UTC,
			want: time.Unix(1754495355, 0),
		},
		{
			// device clock is set to BST (UTC+1), reported wall-clock is 2025-08-06 15:49:15,
			// the true instant is an hour earlier than the raw epoch suggests
			name: "Europe/London during BST shifts back an hour",
			r:    "?V:1,C:170,@10.106.4.40=1754495355#",
			loc:  london,
			want: time.Unix(1754495355, 0).Add(-time.Hour),
		},
		{
			// during GMT Europe/London matches UTC, no shift
			name: "Europe/London during GMT is unchanged",
			r:    "?V:1,C:170,@10.106.4.40=1736500000#",
			loc:  london,
			want: time.Unix(1736500000, 0),
		},
		{
			name:    "zero time means test never run",
			r:       "?V:1,C:170,@10.106.4.40=0#",
			loc:     nil,
			wantErr: true,
		},
		{
			name:    "invalid response",
			r:       "?V:1,C:170,@10.106.4.40",
			loc:     nil,
			wantErr: true,
		},
		{
			name:    "non-numeric time",
			r:       "?V:1,C:170,@10.106.4.40=abc#",
			loc:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGetCompletionTimeResponse(tt.r, tt.loc)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGetCompletionTimeResponse(%q) expected error, got %v", tt.r, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGetCompletionTimeResponse(%q) unexpected error: %v", tt.r, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseGetCompletionTimeResponse(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}
