package healthpb

import (
	"errors"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"
)

func newCountingFaultCheck(t *testing.T) (*FaultCheck, *[]*HealthCheck) {
	t.Helper()
	fc, err := newFaultCheck(&HealthCheck{})
	if err != nil {
		t.Fatalf("newFaultCheck() error = %v", err)
	}
	var commits []*HealthCheck
	fc.onCommit = func(c *HealthCheck) {
		commits = append(commits, proto.Clone(c).(*HealthCheck))
	}
	return fc, &commits
}

// MarkFailed used to commit the fault (with RELIABLE) and then the reliability separately (SCB-1452).
func TestFaultCheck_MarkFailed_SingleCommit(t *testing.T) {
	fc, commits := newCountingFaultCheck(t)
	fc.MarkRunning()
	*commits = nil

	failure := errors.New("doors: 400 Bad Request")
	fc.MarkFailed(failure)
	if len(*commits) != 1 {
		t.Fatalf("first MarkFailed made %d commits, want 1", len(*commits))
	}
	first := (*commits)[0]
	if got := first.GetNormality(); got != HealthCheck_ABNORMAL {
		t.Errorf("normality = %v, want ABNORMAL", got)
	}
	if got := first.GetReliability().GetState(); got != HealthCheck_Reliability_BAD_RESPONSE {
		t.Errorf("reliability = %v, want BAD_RESPONSE", got)
	}

	fc.MarkFailed(failure)
	if len(*commits) != 1 {
		t.Fatalf("repeated MarkFailed made %d more commits, want 0", len(*commits)-1)
	}
	now := fc.load()
	if !proto.Equal(now.GetReliability().GetReliableTime(), first.GetReliability().GetReliableTime()) {
		t.Errorf("ReliableTime changed on a repeated MarkFailed: %v -> %v",
			first.GetReliability().GetReliableTime(), now.GetReliability().GetReliableTime())
	}
	if !proto.Equal(now.GetReliability().GetUnreliableTime(), first.GetReliability().GetUnreliableTime()) {
		t.Errorf("UnreliableTime changed on a repeated MarkFailed: %v -> %v",
			first.GetReliability().GetUnreliableTime(), now.GetReliability().GetUnreliableTime())
	}
}

// Every committed state must be a whole MarkFailed or a whole MarkRunning, never half of each.
func TestFaultCheck_MarkFailed_Concurrent(t *testing.T) {
	fc, err := newFaultCheck(&HealthCheck{})
	if err != nil {
		t.Fatalf("newFaultCheck() error = %v", err)
	}
	var bad []*HealthCheck
	consistent := func(c *HealthCheck) bool {
		switch c.GetNormality() {
		case HealthCheck_NORMAL:
			return c.GetReliability().GetState() == HealthCheck_Reliability_RELIABLE
		case HealthCheck_ABNORMAL:
			return c.GetReliability().GetState() == HealthCheck_Reliability_BAD_RESPONSE
		}
		return false
	}
	fc.onCommit = func(c *HealthCheck) { // called with fc.mu held
		if !consistent(c) {
			bad = append(bad, proto.Clone(c).(*HealthCheck))
		}
	}

	failure := errors.New("boom")
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 500 {
				fc.MarkFailed(failure)
			}
		}()
		go func() {
			defer wg.Done()
			for range 500 {
				fc.MarkRunning()
			}
		}()
	}
	wg.Wait()

	if len(bad) > 0 {
		t.Fatalf("%d commits mixed normality and reliability, first: normality=%v reliability=%v",
			len(bad), bad[0].GetNormality(), bad[0].GetReliability().GetState())
	}
	if final := fc.load(); !consistent(final) {
		t.Fatalf("final state normality=%v reliability=%v", final.GetNormality(), final.GetReliability().GetState())
	}
}
