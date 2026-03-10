package enterleavesensorpb

import (
	"testing"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestModel_ResetTotals(t *testing.T) {
	m := newModel(t)
	m.enter(3)
	m.leave(5)
	m.assertTotals(3, 5)

	m.resetTotals()
	m.assertTotals(0, 0)
}

func newModel(t *testing.T) *modelTester {
	return &modelTester{
		t: t,
		m: NewModel(),
	}
}

type modelTester struct {
	t *testing.T
	m *Model
}

func (mt *modelTester) enter(n int) {
	for i := 0; i < n; i++ {
		err := mt.m.CreateEnterLeaveEvent(&traits.EnterLeaveEvent{
			Direction: traits.EnterLeaveEvent_ENTER,
		})
		if err != nil {
			mt.t.Fatal(err)
		}
	}
}

func (mt *modelTester) leave(n int) {
	for i := 0; i < n; i++ {
		err := mt.m.CreateEnterLeaveEvent(&traits.EnterLeaveEvent{
			Direction: traits.EnterLeaveEvent_LEAVE,
		})
		if err != nil {
			mt.t.Fatal(err)
		}
	}
}

func (mt *modelTester) resetTotals() {
	err := mt.m.ResetTotals()
	if err != nil {
		mt.t.Fatal(err)
	}
}

func (mt *modelTester) assertTotals(enter, leave int32) {
	totals, err := mt.m.GetEnterLeaveEvent()
	if err != nil {
		mt.t.Fatal(err)
	}
	if enter == 0 { // allow nil to also mean 0
		if totals.EnterTotal != nil && *totals.EnterTotal != 0 {
			mt.t.Fatalf("expected enter total to be 0, got %d", *totals.EnterTotal)
		}
	} else {
		if totals.EnterTotal == nil {
			mt.t.Fatalf("expected enter total to be %d, got nil", enter)
		}
		if *totals.EnterTotal != enter {
			mt.t.Fatalf("expected enter total to be %d, got %d", enter, totals.EnterTotal)
		}
	}
	if leave == 0 { // allow nil to also mean 0
		if totals.LeaveTotal != nil && *totals.LeaveTotal != 0 {
			mt.t.Fatalf("expected leave total to be 0, got %d", *totals.LeaveTotal)
		}
	} else {
		if totals.LeaveTotal == nil {
			mt.t.Fatalf("expected leave total to be %d, got nil", leave)
		}
		if *totals.LeaveTotal != leave {
			mt.t.Fatalf("expected leave total to be %d, got %d", leave, totals.LeaveTotal)
		}
	}
}
