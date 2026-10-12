//go:build unix && !aix

package install

import (
	"context"
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// lockRetryInterval is how often lock tries again while another process holds the lock.
const lockRetryInterval = 100 * time.Millisecond

// lock takes an exclusive flock(2) on f, waiting until it is free or ctx is done. Closing f releases it.
func lock(ctx context.Context, f *os.File) error {
	ticker := time.NewTicker(lockRetryInterval)
	defer ticker.Stop()
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, unix.EINTR):
			continue
		case !errors.Is(err, unix.EWOULDBLOCK):
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
