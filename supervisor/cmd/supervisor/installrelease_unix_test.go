//go:build unix && !aix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// While another process holds the releases lock, the command gives up after -lock-timeout.
func TestInstallRelease_LockHeld(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	holdLock(t, h.root)

	_, err := runInstall(t, "-config", h.config, "-lock-timeout", "10ms", writeRelease(t, "v1"))
	if err == nil || !strings.Contains(err.Error(), "an update is in progress") {
		t.Fatalf("error = %v, want an update in progress error", err)
	}
	if _, err := os.Lstat(filepath.Join(h.root, "current")); !os.IsNotExist(err) {
		t.Errorf("current link exists (err %v), want none", err)
	}
}

// If the Supervisor accepts an update while the command waits for the lock, the command still installs,
// and warns that current will change.
func TestInstallRelease_UpdateStartedWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHost(t, "freebsdrc")
		held := holdLock(t, h.root)
		tarball := writeRelease(t, "v1")

		var out, errOut bytes.Buffer
		done := make(chan error)
		go func() { done <- installRelease([]string{"-config", h.config, tarball}, &out, &errOut, zap.NewNop()) }()
		synctest.Wait() // the command is now waiting for the lock

		if err := os.MkdirAll(h.stateDir, 0o700); err != nil {
			t.Fatal(err)
		}
		state := `{"target": {"version": "v2", "url": "https://example.com/v2.tar.gz", "checksum": "sha256:00"}}`
		if err := os.WriteFile(filepath.Join(h.stateDir, "state.json"), []byte(state), 0o600); err != nil {
			t.Fatal(err)
		}
		_ = held.Close()

		if err := <-done; err != nil {
			t.Fatalf("installRelease() error = %v", err)
		}
		if want := "installed v1; current -> v1 (was: none)\n"; out.String() != want {
			t.Errorf("stdout = %q, want %q", out.String(), want)
		}
		if !strings.Contains(errOut.String(), "started an update to v2") {
			t.Errorf("stderr = %q, want a warning about the update to v2", errOut.String())
		}
	})
}

// holdLock takes the releases lock as another process would, and returns the file holding it.
func holdLock(t *testing.T, root string) *os.File {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(root, ".lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	return f
}
