//go:build unix && !aix

package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/sys/unix"
)

// While the installer holds the lock, another flock on the same file fails; once released, it succeeds.
func TestFreeBSDRC_Lock(t *testing.T) {
	i, root := newTestRCInstaller(t)
	_, unlock, err := i.openLocked(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	other, err := os.Open(filepath.Join(root, lockName))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	tryLock := func() error { return unix.Flock(int(other.Fd()), unix.LOCK_EX|unix.LOCK_NB) }

	if err := tryLock(); !errors.Is(err, unix.EWOULDBLOCK) {
		t.Errorf("flock while held: err = %v, want EWOULDBLOCK", err)
	}
	unlock()
	if err := tryLock(); err != nil {
		t.Errorf("flock after release: err = %v, want nil", err)
	}
}

// holdLock takes the installer's lock as another process would, and returns the file holding it.
func holdLock(t *testing.T, root string) *os.File {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	return f
}

// ApplyLocal waits for a held lock until ctx is done, then fails without changing anything.
func TestFreeBSDRC_ApplyLocal_LockTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		i, root := newTestRCInstaller(t)
		holdLock(t, root)
		artefact := writeArtefact(t, releaseEntries("v1"))

		const timeout = 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		start := time.Now()
		_, err := i.ApplyLocal(ctx, artefact, true)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ApplyLocal() error = %v, want DeadlineExceeded", err)
		}
		if waited := time.Since(start); waited < timeout {
			t.Errorf("ApplyLocal() gave up after %v, want %v", waited, timeout)
		}
		assertEntries(t, root, ".lock")
	})
}

// ApplyLocal waits for a held lock and installs once it is released.
func TestFreeBSDRC_ApplyLocal_LockReleased(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		i, root := newTestRCInstaller(t)
		held := holdLock(t, root)
		artefact := writeArtefact(t, releaseEntries("v1"))

		const holdFor = 5 * time.Second
		go func() {
			time.Sleep(holdFor)
			_ = held.Close()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		start := time.Now()
		if _, err := i.ApplyLocal(ctx, artefact, false); err != nil {
			t.Fatalf("ApplyLocal() error = %v", err)
		}
		if waited := time.Since(start); waited < holdFor {
			t.Errorf("ApplyLocal() installed after %v, want it to wait for the lock", waited)
		}
		assertLink(t, root, currentLink, "v1")
	})
}
