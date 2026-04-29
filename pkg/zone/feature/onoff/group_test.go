package onoff

import (
	"slices"
	"testing"
	"testing/synctest"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
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

// TestGroup_PullOnOff_ConflictingStates verifies that PullOnOff closes the stream with
// an error when member devices have conflicting states (some ON, some OFF), rather than
// silently delivering a nil OnOff value.
func TestGroup_PullOnOff_ConflictingStates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		// Build individual device backends routed by name.
		n := node.New("test")
		n.Announce("A", node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(onoffpb.NewModel()))))
		n.Announce("B", node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(onoffpb.NewModel()))))
		impl := onoffpb.NewOnOffApiClient(n.ClientConn())

		// Set conflicting states: A=ON, B=OFF.
		if _, err := impl.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "A", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_ON}}); err != nil {
			t.Fatalf("UpdateOnOff A: %v", err)
		}
		if _, err := impl.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: "B", OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}}); err != nil {
			t.Fatalf("UpdateOnOff B: %v", err)
		}

		// Wrap the zone Group directly without a real gRPC server.
		g := &Group{client: impl, names: []string{"A", "B"}}
		client := onoffpb.NewOnOffApiClient(wrap.ServerToClient(onoffpb.OnOffApi_ServiceDesc, g))

		stream, err := client.PullOnOff(ctx, &onoffpb.PullOnOffRequest{Name: "test"})
		if err != nil {
			t.Fatalf("PullOnOff: %v", err)
		}

		// Read messages from the stream in a goroutine.
		type recvResult struct {
			msg *onoffpb.PullOnOffResponse
			err error
		}
		results := make(chan recvResult, 10)
		go func() {
			for {
				msg, err := stream.Recv()
				results <- recvResult{msg, err}
				if err != nil {
					return
				}
			}
		}()

		synctest.Wait()

		// Drain all results; expect at least one error.
		var gotErr bool
		for {
			select {
			case r := <-results:
				if r.err != nil {
					gotErr = true
				} else {
					for _, ch := range r.msg.Changes {
						if ch.OnOff == nil {
							t.Fatal("stream delivered nil OnOff for conflicting states; expected stream error")
						}
					}
				}
			default:
				if !gotErr {
					t.Fatal("expected stream error on conflicting states, but got none")
				}
				return
			}
		}
	})
}
