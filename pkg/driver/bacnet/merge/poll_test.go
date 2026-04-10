package merge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func Test_pollUntil(t *testing.T) {
	type B bool
	type args struct {
		ctx     func() context.Context
		timeout time.Duration
		test    func() func(B) bool
	}

	poll := func(_ context.Context) (B, error) {
		return true, nil
	}

	tests := []struct {
		name    string
		args    args
		want    B
		wantErr error
	}{
		{
			name: "no deadline - test succeeds after default deadline",
			args: args{
				ctx: func() context.Context {
					return context.Background()
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					count := 0
					return func(t B) bool {
						count++

						// fail until at least 10 ms has passed
						if count > 2 {
							return true
						}

						return false
					}
				},
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "no deadline - test succeeds before default deadline",
			args: args{
				ctx: func() context.Context {
					return context.Background()
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					return func(t B) bool {
						// succeed straight away
						return true
					}
				},
			},
			wantErr: nil,
			want:    true,
		},
		{
			name: "deadline before timeout - test succeeds after deadline",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
					t.Cleanup(cancel)
					return ctx
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					count := 0
					return func(t B) bool {
						count++
						return count > 2
					}
				},
			},
			wantErr: context.DeadlineExceeded,
			want:    false,
		},
		{
			name: "deadline before timeout - test succeeds before deadline",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
					t.Cleanup(cancel)
					return ctx
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					return func(t B) bool {
						return true
					}
				},
			},
			wantErr: nil,
			want:    true,
		},
		{
			name: "deadline after timeout - test succeeds after deadline",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
					t.Cleanup(cancel)
					return ctx
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					count := 0
					return func(t B) bool {
						count++
						return count > 2
					}
				},
			},
			wantErr: context.DeadlineExceeded,
			want:    false,
		},
		{
			name: "deadline after timeout - test succeeds before deadline",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
					t.Cleanup(cancel)
					return ctx
				},
				timeout: 10 * time.Millisecond,
				test: func() func(t B) bool {
					return func(t B) bool {
						return true
					}
				},
			},
			wantErr: nil,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := pollUntil[B](tt.args.ctx(), tt.args.timeout, poll, tt.args.test())

			if diff := cmp.Diff(tt.want, res); diff != "" {
				t.Errorf("pollUntil(): (-want +got)\n%s", diff)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("pollUntil() error = %v - wantErr %v", err, tt.wantErr)
			}
		})
	}
}
