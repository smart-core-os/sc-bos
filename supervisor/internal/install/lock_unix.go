//go:build unix && !aix

package install

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// lock takes an exclusive flock(2) on f, waiting until it is free. Closing f releases it.
func lock(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
