package service

import (
	"context"
	"errors"
	"testing"
)

type stubSystemCheck struct {
	markRunningCalled int
	markFailedErr     error
}

func (s *stubSystemCheck) Dispose()            {}
func (s *stubSystemCheck) MarkRunning()        { s.markRunningCalled++ }
func (s *stubSystemCheck) MarkFailed(err error) { s.markFailedErr = err }

func TestUpdateSystemCheck_NilCheck(t *testing.T) {
	// Must not panic when check is nil.
	UpdateSystemCheck(nil, errors.New("some error"))
	UpdateSystemCheck(nil, nil)
}

func TestUpdateSystemCheck_ContextErrors(t *testing.T) {
	for _, ctxErr := range []error{context.Canceled, context.DeadlineExceeded} {
		s := &stubSystemCheck{}
		UpdateSystemCheck(s, ctxErr)
		if s.markRunningCalled != 0 || s.markFailedErr != nil {
			t.Errorf("UpdateSystemCheck(%v): expected no calls, got running=%d failed=%v",
				ctxErr, s.markRunningCalled, s.markFailedErr)
		}
	}
}

func TestUpdateSystemCheck_RealError(t *testing.T) {
	s := &stubSystemCheck{}
	want := errors.New("connection refused")
	UpdateSystemCheck(s, want)
	if s.markFailedErr != want {
		t.Errorf("MarkFailed got %v, want %v", s.markFailedErr, want)
	}
	if s.markRunningCalled != 0 {
		t.Errorf("MarkRunning called unexpectedly")
	}
}

func TestUpdateSystemCheck_NilError(t *testing.T) {
	s := &stubSystemCheck{}
	UpdateSystemCheck(s, nil)
	if s.markRunningCalled != 1 {
		t.Errorf("MarkRunning called %d times, want 1", s.markRunningCalled)
	}
	if s.markFailedErr != nil {
		t.Errorf("MarkFailed called unexpectedly with %v", s.markFailedErr)
	}
}
