package job

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func Test_shouldExecuteImmediately(t *testing.T) {
	var tests = []struct {
		name     string
		schedule *jsontypes.Schedule
		now      time.Time
		previous time.Time
		expected bool
	}{
		{
			name:     "now is behind the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:59"),
			previous: makeTime("09:00").Add(-24 * time.Hour),
			expected: false,
		},
		{
			name:     "now is on the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "now is after the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:01"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "now is well behind the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:00"),
			previous: makeTime("09:00").Add(-24 * time.Hour),
			expected: false,
		},
		{
			name:     "now is well after the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("10:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "execution was skipped",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:59"),
			previous: makeTime("09:00").Add(-48 * time.Hour),
			expected: true,
		},
		{
			name:     "execution was skipped twice",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:01"),
			previous: makeTime("09:00").Add(-48 * time.Hour),
			expected: true,
		},
		{
			name:     "execution is at halfway point of interval",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("21:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "execution is past halfway point of interval",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("22:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "nil previous execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: time.Time{},
			expected: true,
		},
		{
			name:     "schedule updated to after last execution time",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: makeTime("08:00"),
			expected: true,
		},
		{
			name:     "schedule updated to before last execution time",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("10:00"),
			previous: makeTime("10:00"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExecuteImmediately(tt.schedule, tt.now, tt.previous)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func makeTime(s string) time.Time {
	t, err := time.Parse("15:04", s)
	if err != nil {
		panic(err)
	}
	t = t.Add(time.Hour * 24 * 365) // increment by one year to avoid zero-time issues
	return t
}

// fakeJob implements Job for testing ExecuteAll in isolation.
// GetNextExecution fires once immediately (buffered channel) then blocks.
// It records the times passed to SetLastAttempt and SetPreviousExecution.
type fakeJob struct {
	doErr          error
	nextExecCh     chan time.Time
	doCalled       chan struct{}
	lastAttemptSet time.Time
	prevExecSet    time.Time
}

func newFakeJob(doErr error) *fakeJob {
	ch := make(chan time.Time, 1)
	ch <- time.Now() // ready to fire immediately once, then blocks
	return &fakeJob{
		doErr:      doErr,
		nextExecCh: ch,
		doCalled:   make(chan struct{}),
	}
}

func (f *fakeJob) GetLogger() *zap.Logger            { return zap.NewNop() }
func (f *fakeJob) GetNextExecution() <-chan time.Time { return f.nextExecCh }
func (f *fakeJob) SetLastAttempt(t time.Time)         { f.lastAttemptSet = t }
func (f *fakeJob) SetPreviousExecution(t time.Time)   { f.prevExecSet = t }
func (f *fakeJob) Do(_ context.Context, _ sender) error {
	select {
	case <-f.doCalled:
	default:
		close(f.doCalled)
	}
	return f.doErr
}

func noopSender(_ context.Context, _ string, _ []byte) error { return nil }

// TestExecuteAll_previousExecution verifies that SetPreviousExecution is only
// advanced on a successful Do, and that SetLastAttempt is always called.
// This guards against the pre-fix behaviour where a failed run left
// PreviousExecution stale, causing GetNextExecution to fire again immediately.
func TestExecuteAll_previousExecution(t *testing.T) {
	t.Run("called on success", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		j := newFakeJob(nil)
		go func() { <-j.doCalled; cancel() }()
		_ = ExecuteAll(ctx, noopSender, j)

		if j.lastAttemptSet.IsZero() {
			t.Error("SetLastAttempt should always be called before Do")
		}
		if j.prevExecSet.IsZero() {
			t.Error("SetPreviousExecution should be called after a successful Do")
		}
	})

	t.Run("not called on failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		j := newFakeJob(errors.New("boom"))
		go func() { <-j.doCalled; cancel() }()
		_ = ExecuteAll(ctx, noopSender, j)

		if j.lastAttemptSet.IsZero() {
			t.Error("SetLastAttempt should always be called before Do")
		}
		if !j.prevExecSet.IsZero() {
			t.Error("SetPreviousExecution should not be called after a failed Do")
		}
	})
}

// TestBaseJob_GetNextExecution_pacesAfterFailure verifies that after a failed
// run (SetLastAttempt recorded, PreviousExecution not advanced), the next call
// to GetNextExecution does not fire immediately. Before the fix, a stale
// PreviousExecution caused shouldExecuteImmediately to return true on every
// loop iteration, creating a tight retry loop.
func TestBaseJob_GetNextExecution_pacesAfterFailure(t *testing.T) {
	db, err := bolthold.Open(filepath.Join(t.TempDir(), "test.db"), 0600, nil)
	if err != nil {
		t.Fatalf("open bolthold: %v", err)
	}
	defer db.Close()

	base := &BaseJob{
		Db:       db,
		AutoName: "auto",
		ScName:   "site",
		Name:     "job",
		Schedule: jsontypes.MustParseSchedule("@every 1m"),
		Logger:   zap.NewNop(),
	}

	// Simulate a failed run: attempt time is recorded but PreviousExecution is not advanced.
	base.SetLastAttempt(time.Now())

	select {
	case <-base.GetNextExecution():
		t.Fatal("GetNextExecution fired immediately after a failed run; tight retry loop detected")
	case <-time.After(50 * time.Millisecond):
		// expected: the schedule interval is respected
	}
}
