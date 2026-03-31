package onoff

import (
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
)

func TestMergeOnOff(t *testing.T) {
	tests := []struct {
		name     string
		input    []*onoffpb.OnOff
		want     onoffpb.OnOff_State
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:    "all on",
			input:   []*onoffpb.OnOff{{State: onoffpb.OnOff_ON}, {State: onoffpb.OnOff_ON}},
			want:    onoffpb.OnOff_ON,
			wantErr: false,
		},
		{
			name:    "all off",
			input:   []*onoffpb.OnOff{{State: onoffpb.OnOff_OFF}, {State: onoffpb.OnOff_OFF}},
			want:    onoffpb.OnOff_OFF,
			wantErr: false,
		},
		{
			name:     "mixed states",
			input:    []*onoffpb.OnOff{{State: onoffpb.OnOff_ON}, {State: onoffpb.OnOff_OFF}},
			want:     onoffpb.OnOff_STATE_UNSPECIFIED,
			wantErr:  true,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "empty input",
			input:    []*onoffpb.OnOff{},
			want:     onoffpb.OnOff_STATE_UNSPECIFIED,
			wantErr:  true,
			wantCode: codes.FailedPrecondition,
		},
		{
			name:    "ignore unspecified",
			input:   []*onoffpb.OnOff{{State: onoffpb.OnOff_ON}, {State: onoffpb.OnOff_STATE_UNSPECIFIED}, {State: onoffpb.OnOff_ON}},
			want:    onoffpb.OnOff_ON,
			wantErr: false,
		},
		{
			name:    "all unspecified",
			input:   []*onoffpb.OnOff{{State: onoffpb.OnOff_STATE_UNSPECIFIED}, {State: onoffpb.OnOff_STATE_UNSPECIFIED}},
			want:    onoffpb.OnOff_STATE_UNSPECIFIED,
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
				if tt.want != onoffpb.OnOff_STATE_UNSPECIFIED {
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
