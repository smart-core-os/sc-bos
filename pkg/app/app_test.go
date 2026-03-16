package app

import (
	"context"
	"testing"
)

func TestRestartNowError(t *testing.T) {
	code := RunUntilInterrupt(func(_ context.Context) error {
		return restartNowError{}
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}
