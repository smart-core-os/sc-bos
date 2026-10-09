//go:build unix && !aix

package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// While the installer holds the lock, another flock on the same file fails; once released, it succeeds.
func TestFreeBSDRC_Lock(t *testing.T) {
	i, root := newTestRCInstaller(t)
	_, unlock, err := i.openLocked()
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
