package healthpb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestReliabilityFromErr(t *testing.T) {
	tests := map[string]struct {
		err       error
		wantState HealthCheck_Reliability_State
		wantError bool // whether LastError should be set
		wantCode  *HealthCheck_Error_Code
	}{
		"nil error": {
			err:       nil,
			wantState: HealthCheck_Reliability_RELIABLE,
			wantError: false,
		},
		"context.Canceled": {
			err:       context.Canceled,
			wantState: HealthCheck_Reliability_RELIABLE,
			wantError: false,
		},
		"wrapped context.Canceled": {
			err:       errors.Join(context.Canceled, errors.New("wrapped")),
			wantState: HealthCheck_Reliability_RELIABLE,
			wantError: false,
		},
		"context.DeadlineExceeded": {
			err:       context.DeadlineExceeded,
			wantState: HealthCheck_Reliability_NO_RESPONSE,
			wantError: false,
		},
		"wrapped context.DeadlineExceeded": {
			err:       errors.Join(context.DeadlineExceeded, errors.New("timeout")),
			wantState: HealthCheck_Reliability_NO_RESPONSE,
			wantError: false,
		},
		"gRPC NotFound": {
			err:       status.Error(codes.NotFound, "not found"),
			wantState: HealthCheck_Reliability_NOT_FOUND,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "NotFound",
			},
		},
		"gRPC PermissionDenied": {
			err:       status.Error(codes.PermissionDenied, "permission denied"),
			wantState: HealthCheck_Reliability_PERMISSION_DENIED,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "PermissionDenied",
			},
		},
		"gRPC Unauthenticated": {
			err:       status.Error(codes.Unauthenticated, "unauthenticated"),
			wantState: HealthCheck_Reliability_PERMISSION_DENIED,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "Unauthenticated",
			},
		},
		"gRPC DeadlineExceeded": {
			err:       status.Error(codes.DeadlineExceeded, "deadline exceeded"),
			wantState: HealthCheck_Reliability_NO_RESPONSE,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "DeadlineExceeded",
			},
		},
		"gRPC Unavailable": {
			err:       status.Error(codes.Unavailable, "unavailable"),
			wantState: HealthCheck_Reliability_BAD_RESPONSE,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "Unavailable",
			},
		},
		"gRPC Internal": {
			err:       status.Error(codes.Internal, "internal error"),
			wantState: HealthCheck_Reliability_BAD_RESPONSE,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "Internal",
			},
		},
		"gRPC Unknown": {
			err:       status.Error(codes.Unknown, "unknown error"),
			wantState: HealthCheck_Reliability_BAD_RESPONSE,
			wantError: true,
			wantCode: &HealthCheck_Error_Code{
				System: "gRPC",
				Code:   "Unknown",
			},
		},
		"generic error": {
			err:       errors.New("something went wrong"),
			wantState: HealthCheck_Reliability_BAD_RESPONSE,
			wantError: true,
		},
		"custom error": {
			err:       &customError{msg: "custom failure"},
			wantState: HealthCheck_Reliability_BAD_RESPONSE,
			wantError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := ReliabilityFromErr(tt.err)

			if got == nil {
				t.Fatal("ReliabilityFromErr() returned nil")
			}

			if got.State != tt.wantState {
				t.Errorf("ReliabilityFromErr() state = %v, want %v", got.State, tt.wantState)
			}

			if tt.wantError {
				if got.LastError == nil {
					t.Errorf("ReliabilityFromErr() LastError = nil, want error")
				} else {
					// Verify error message is populated
					if got.LastError.SummaryText == "" {
						t.Errorf("ReliabilityFromErr() LastError.SummaryText is empty")
					}
				}
			} else {
				if got.LastError != nil {
					t.Errorf("ReliabilityFromErr() LastError = %v, want nil", got.LastError)
				}
			}

			if tt.wantCode != nil {
				if got.LastError == nil {
					t.Errorf("ReliabilityFromErr() LastError = nil, want error with code")
				} else if diff := cmp.Diff(tt.wantCode, got.LastError.Code, protocmp.Transform()); diff != "" {
					t.Errorf("ReliabilityFromErr() LastError.Code mismatch (-want +got):\n%s", diff)
				}
			}

			// Verify that Cause and Effects are not set (these should be set by UpdateReliability)
			if got.Cause != nil {
				t.Errorf("ReliabilityFromErr() Cause = %v, want nil", got.Cause)
			}
			if got.Effects != nil {
				t.Errorf("ReliabilityFromErr() Effects = %v, want nil", got.Effects)
			}
		})
	}
}

func TestErrorToProto(t *testing.T) {
	tests := map[string]struct {
		err         error
		wantNil     bool
		wantSummary string
	}{
		"nil error": {
			err:     nil,
			wantNil: true,
		},
		"simple error": {
			err:         errors.New("test error"),
			wantSummary: "test error",
		},
		"formatted error": {
			err:         errors.New("failed to connect: timeout"),
			wantSummary: "failed to connect: timeout",
		},
		"gRPC status error": {
			err:         status.Error(codes.NotFound, "resource not found"),
			wantSummary: "rpc error: code = NotFound desc = resource not found",
		},
		"custom error": {
			err:         &customError{msg: "custom failure"},
			wantSummary: "custom failure",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := ErrorToProto(tt.err)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ErrorToProto() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("ErrorToProto() = nil, want non-nil")
			}

			if got.SummaryText != tt.wantSummary {
				t.Errorf("ErrorToProto() SummaryText = %q, want %q", got.SummaryText, tt.wantSummary)
			}
		})
	}
}

// customError is a test helper for custom error types
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// TestCheckBase_concurrent drives one check from several goroutines, as drivers do when
// separate polls share a device's check. Run with -race.
func TestCheckBase_concurrent(t *testing.T) {
	var (
		lastMu   sync.Mutex
		last     *HealthCheck
		disposed atomic.Bool
	)
	reg := NewRegistry(WithOnCheckUpdate(func(_ string, c *HealthCheck) {
		if disposed.Load() {
			t.Error("commit after Dispose returned")
		}
		lastMu.Lock()
		last = c
		lastMu.Unlock()
	}))
	fc, err := reg.ForOwner("owner").NewFaultCheck("dev", &HealthCheck{Id: "comms"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const n = 100
	updaters := []func(i int){
		func(i int) {
			if i%2 == 0 {
				fc.UpdateReliability(ctx, &HealthCheck_Reliability{State: HealthCheck_Reliability_RELIABLE})
			} else {
				fc.UpdateReliability(ctx, &HealthCheck_Reliability{State: HealthCheck_Reliability_NO_RESPONSE})
			}
		},
		func(i int) { fc.SetFault(&HealthCheck_Error{SummaryText: fmt.Sprint("set ", i)}) },
		func(i int) { fc.AddOrUpdateFault(&HealthCheck_Error{SummaryText: fmt.Sprint("add ", i%3)}) },
		func(i int) { fc.UpdateMetadata(ctx, &HealthCheck{DisplayName: fmt.Sprint("name ", i)}) },
		func(int) { reg.GetCheck("dev", "owner:comms").GetReliability().GetState() },
	}
	runAll := func() *sync.WaitGroup {
		var wg sync.WaitGroup
		for _, f := range updaters {
			wg.Go(func() {
				for i := range n {
					f(i)
				}
			})
		}
		return &wg
	}

	runAll().Wait()
	// commits reach the registry in order, so the last one seen is the current state
	lastMu.Lock()
	gotLast := last
	lastMu.Unlock()
	if got := reg.GetCheck("dev", "owner:comms"); got != gotLast {
		t.Errorf("last commit isn't the current state:\ncurrent %v\nlast    %v", got, gotLast)
	}

	wg := runAll()
	fc.Dispose()
	disposed.Store(true)
	wg.Wait()
	if got := reg.GetCheck("dev", "owner:comms"); got != nil {
		t.Errorf("check still registered after Dispose: %v", got)
	}
}
