//go:build !unix || aix

package install

import (
	"errors"
	"os"
)

// lock always fails: file locking is not implemented for this OS.
func lock(*os.File) error {
	return errors.New("file locking is not supported on this OS")
}
