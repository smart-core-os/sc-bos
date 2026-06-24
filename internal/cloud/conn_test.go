package cloud

import (
	"context"
	"testing"
	"time"
)

// TestConn_WaitConnected blocks until the first successful check-in flips the connection to Connected,
// then returns immediately on subsequent calls.
func TestConn_WaitConnected(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn := newConnEnv(t, env)

	if got := conn.State().Connectivity; got == Connected {
		t.Fatalf("connection should not be Connected before any check-in, got %v", got)
	}

	// WaitConnected blocks until a successful check-in; run it concurrently and then trigger one.
	errc := make(chan error, 1)
	go func() { errc <- conn.WaitConnected(ctx) }()

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}

	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("WaitConnected = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitConnected did not return after a successful check-in")
	}

	// Already connected: returns immediately.
	if err := conn.WaitConnected(ctx); err != nil {
		t.Fatalf("WaitConnected (already connected) = %v", err)
	}
}

// TestConn_WaitConnected_DoesNotBlockCheckIns is a regression test: WaitConnected must drop its bus
// subscription when it returns, so a later check-in's broadcast is not left blocking on an undrained
// listener. It runs WaitConnected over a long-lived ctx, then checks a second Update still returns.
func TestConn_WaitConnected_DoesNotBlockCheckIns(t *testing.T) {
	env := setupClientEnv(t)
	conn := newConnEnv(t, env)

	// Mimic pkg/app/controller.go wiring waitForCheckIn = c.Cloud.WaitConnected with the long-lived
	// app ctx: it must not depend on that ctx being cancelled to release its subscription.
	appCtx := context.Background()

	errc := make(chan error, 1)
	go func() { errc <- conn.WaitConnected(appCtx) }()

	// First check-in flips to Connected and unblocks WaitConnected.
	if _, err := conn.Update(appCtx); err != nil {
		t.Fatalf("first Update: %v", err)
	}
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("WaitConnected = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitConnected did not return after the first successful check-in")
	}

	// Second check-in: recordCheckIn broadcasts the new state. If WaitConnected leaked its listener,
	// the broadcast blocks forever delivering to the undrained channel and Update never returns.
	done := make(chan error, 1)
	go func() {
		_, err := conn.Update(appCtx)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second Update: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second Update blocked: WaitConnected leaked an undrained bus listener, blocking recordCheckIn's broadcast")
	}
}

// TestConn_WaitConnected_Cancelled returns ctx.Err() when cancelled before a successful check-in.
func TestConn_WaitConnected_Cancelled(t *testing.T) {
	env := setupClientEnv(t)
	conn := newConnEnv(t, env)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := conn.WaitConnected(ctx); err == nil {
		t.Fatal("WaitConnected should return ctx.Err() when cancelled before connecting")
	}
}
