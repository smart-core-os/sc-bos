//go:build !windows

package app

import "syscall"

// sqliteDiskCapacity returns the total disk capacity and utilization for the filesystem
// containing dataDir, expressed relative to the current DB size.
// Returns ok=false if the stats cannot be determined.
func sqliteDiskCapacity(dataDir string, dbUsedBytes uint64) (capacity uint64, utilization float32, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &st); err != nil || st.Bsize <= 0 || st.Blocks == 0 {
		return
	}
	capacity = st.Blocks * uint64(st.Bsize)
	utilization = float32(dbUsedBytes) / float32(capacity) * 100
	ok = true
	return
}
