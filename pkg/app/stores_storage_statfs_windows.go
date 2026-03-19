//go:build windows

package app

// sqliteDiskCapacity is not supported on Windows; always returns ok=false.
func sqliteDiskCapacity(_ string, _ uint64) (capacity uint64, utilization float32, ok bool) {
	return
}
