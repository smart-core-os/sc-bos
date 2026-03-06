package onoff

import (
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func TestMergeOnOff(t *testing.T) {
	tests := []struct {
		name     string
		input    []*traits.OnOff
		want     traits.OnOff_State
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:    "all on",
			input:   []*traits.OnOff{{State: traits.OnOff_ON}, {State: traits.OnOff_ON}},
			want:    traits.OnOff_ON,
			wantErr: false,
		},
		{
			name:    "all off",
			input:   []*traits.OnOff{{State: traits.OnOff_OFF}, {State: traits.OnOff_OFF}},
			want:    traits.OnOff_OFF,
			wantErr: false,
		},
		{
			name:     "mixed states",
			input:    []*traits.OnOff{{State: traits.OnOff_ON}, {State: traits.OnOff_OFF}},
			want:     traits.OnOff_STATE_UNSPECIFIED,
			wantErr:  true,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "empty input",
			input:    []*traits.OnOff{},
			want:     traits.OnOff_STATE_UNSPECIFIED,
			wantErr:  true,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:    "ignore unspecified",
			input:   []*traits.OnOff{{State: traits.OnOff_ON}, {State: traits.OnOff_STATE_UNSPECIFIED}, {State: traits.OnOff_ON}},
			want:    traits.OnOff_ON,
			wantErr: false,
		},
		{
			name:    "all unspecified",
			input:   []*traits.OnOff{{State: traits.OnOff_STATE_UNSPECIFIED}, {State: traits.OnOff_STATE_UNSPECIFIED}},
			want:    traits.OnOff_STATE_UNSPECIFIED,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := slices.Values(tt.input)
			got, err := mergeOnOff(seq)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeOnOff() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got == nil {
				if tt.want != traits.OnOff_STATE_UNSPECIFIED {
					t.Errorf("mergeOnOff() got = nil, want state %v", tt.want)
				}
			} else if got.State != tt.want {
				t.Errorf("mergeOnOff() got.State = %v, want %v", got.State, tt.want)
			}
			if tt.wantErr && err != nil && tt.wantCode != codes.OK {
				st, ok := status.FromError(err)
				if !ok || st.Code() != tt.wantCode {
					t.Errorf("mergeOnOff() error code = %v, want %v", st.Code(), tt.wantCode)
				}
			}
		})
	}
}
