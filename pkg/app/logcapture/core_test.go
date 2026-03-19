package logcapture

import (
	"sync"
	"testing"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newNopCore returns a zapcore.Core that accepts all levels and discards entries.
func newNopCore() zapcore.Core {
	return zapcore.NewNopCore()
}

// newObserverCore returns a zapcore.Core that records entries for inspection.
func newObserverCore() (*observer.ObservedLogs, zapcore.Core) {
	core, logs := observer.New(zapcore.DebugLevel)
	return logs, core
}

func writeEntry(core zapcore.Core, msg string) {
	entry := zapcore.Entry{Level: zapcore.InfoLevel, Message: msg}
	if ce := core.Check(entry, nil); ce != nil {
		ce.Write()
	}
}

// ---- Add / deregister -----------------------------------------------------------

func TestAdd_receiveEntries(t *testing.T) {
	logs, extra := newObserverCore()
	c := Wrap(newNopCore())
	remove := c.Add(extra)
	defer remove()

	writeEntry(c, "hello")

	if got := logs.Len(); got != 1 {
		t.Errorf("observer got %d entries, want 1", got)
	}
	if msg := logs.All()[0].Message; msg != "hello" {
		t.Errorf("message = %q, want hello", msg)
	}
}

func TestAdd_deregisterStopsDelivery(t *testing.T) {
	logs, extra := newObserverCore()
	c := Wrap(newNopCore())
	remove := c.Add(extra)

	writeEntry(c, "before")
	remove()
	writeEntry(c, "after")

	if got := logs.Len(); got != 1 {
		t.Errorf("observer got %d entries after remove, want 1", got)
	}
}

func TestAdd_multipleExtras(t *testing.T) {
	logs1, extra1 := newObserverCore()
	logs2, extra2 := newObserverCore()
	c := Wrap(newNopCore())
	r1 := c.Add(extra1)
	r2 := c.Add(extra2)
	defer r1()
	defer r2()

	writeEntry(c, "broadcast")

	if logs1.Len() != 1 || logs2.Len() != 1 {
		t.Errorf("logs1=%d, logs2=%d; want 1 each", logs1.Len(), logs2.Len())
	}
}

func TestAdd_removeOneLeaveOther(t *testing.T) {
	logs1, extra1 := newObserverCore()
	logs2, extra2 := newObserverCore()
	c := Wrap(newNopCore())
	remove1 := c.Add(extra1)
	defer c.Add(extra2)()

	writeEntry(c, "first")
	remove1()
	writeEntry(c, "second")

	if logs1.Len() != 1 {
		t.Errorf("extra1 got %d entries, want 1", logs1.Len())
	}
	if logs2.Len() != 2 {
		t.Errorf("extra2 got %d entries, want 2", logs2.Len())
	}
}

// ---- WrapCoreFunc ---------------------------------------------------------------

func TestWrapCoreFunc_setsBase(t *testing.T) {
	c := &Core{}
	base := newNopCore()
	wrapFn := c.WrapCoreFunc()
	result := wrapFn(base)
	if result != c {
		t.Error("WrapCoreFunc returned a different core")
	}
	if c.base != base {
		t.Error("WrapCoreFunc did not set c.base")
	}
}

// ---- childCore (With) -----------------------------------------------------------

func TestWith_extraCoresStillReceive(t *testing.T) {
	// Add extra AFTER calling With — the child should still forward via parent.
	logs, extra := newObserverCore()
	c := Wrap(newNopCore())
	child := c.With([]zapcore.Field{zapcore.Field{Key: "k", Type: zapcore.StringType, String: "v"}})

	remove := c.Add(extra)
	defer remove()

	writeEntry(child, "via child")

	if logs.Len() != 1 {
		t.Errorf("observer got %d entries, want 1", logs.Len())
	}
}

// ---- Enabled gating -------------------------------------------------------------

// TestEnabled_extraBelowBaseLevel verifies that an extra core whose level is lower
// than the base core's level still receives entries — i.e. Enabled returns true
// when any extra accepts the level, even if the base does not.
func TestEnabled_extraBelowBaseLevel(t *testing.T) {
	// Base accepts WARN and above only.
	baseCore, _ := observer.New(zapcore.WarnLevel)
	// Extra accepts DEBUG and above.
	extraLogs, extraCore := newObserverCore()

	c := Wrap(baseCore)
	remove := c.Add(extraCore)
	defer remove()

	// Write a DEBUG entry — base should NOT receive it, extra SHOULD.
	entry := zapcore.Entry{Level: zapcore.DebugLevel, Message: "debug-msg"}
	if ce := c.Check(entry, nil); ce != nil {
		ce.Write()
	}

	if extraLogs.Len() != 1 {
		t.Errorf("extra got %d entries, want 1", extraLogs.Len())
	}
}

// TestEnabled_childExtraBelowBaseLevel is the same check via a childCore (With).
func TestEnabled_childExtraBelowBaseLevel(t *testing.T) {
	baseCore, _ := observer.New(zapcore.WarnLevel)
	extraLogs, extraCore := newObserverCore()

	c := Wrap(baseCore)
	child := c.With(nil)
	remove := c.Add(extraCore)
	defer remove()

	entry := zapcore.Entry{Level: zapcore.DebugLevel, Message: "child-debug"}
	if ce := child.Check(entry, nil); ce != nil {
		ce.Write()
	}

	if extraLogs.Len() != 1 {
		t.Errorf("extra got %d entries via child, want 1", extraLogs.Len())
	}
}

// ---- concurrent safety ----------------------------------------------------------

func TestCore_concurrent(t *testing.T) {
	c := Wrap(newNopCore())
	const goroutines = 10
	const ops = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent writers.
	for range goroutines {
		go func() {
			defer wg.Done()
			for range ops {
				writeEntry(c, "msg")
			}
		}()
	}

	// Concurrent add/remove.
	for range goroutines {
		go func() {
			defer wg.Done()
			for range ops {
				_, extra := newObserverCore()
				remove := c.Add(extra)
				remove()
			}
		}()
	}

	wg.Wait()
}
