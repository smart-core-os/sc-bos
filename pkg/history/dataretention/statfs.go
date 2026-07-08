//go:build !windows

package dataretention

import "syscall"

// diskCapacity returns, for the filesystem containing dataDir:
//   - capacity:  total size of the filesystem
//   - otherUsed: bytes used by data other than this store (total used minus dbUsedBytes),
//     clamped to zero if disk accounting would produce a negative value
//   - available: bytes still writable by this store, excluding blocks reserved for root
//     (so it may be less than capacity - dbUsedBytes - otherUsed)
//
// Returns ok=false if the stats cannot be determined.
func diskCapacity(dataDir string, dbUsedBytes uint64) (capacity uint64, otherUsed uint64, available uint64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &st); err != nil || st.Bsize <= 0 || st.Blocks == 0 {
		return
	}
	bsize := uint64(st.Bsize)
	capacity = st.Blocks * bsize
	diskUsed := (st.Blocks - st.Bfree) * bsize
	if diskUsed > dbUsedBytes {
		otherUsed = diskUsed - dbUsedBytes
	}
	available = uint64(st.Bavail) * bsize
	ok = true
	return
}
