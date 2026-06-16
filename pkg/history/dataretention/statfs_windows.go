//go:build windows

package dataretention

// diskCapacity is not supported on Windows; always returns ok=false.
func diskCapacity(_ string, _ uint64) (capacity uint64, otherUsed uint64, available uint64, ok bool) {
	return
}
