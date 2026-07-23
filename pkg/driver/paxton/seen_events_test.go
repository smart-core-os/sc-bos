package paxton

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSeenEvents_NewEventIsAccepted(t *testing.T) {
	s := newSeenEvents()
	assert.True(t, s.markSeen("1"))
}

func TestSeenEvents_DuplicateIsRejected(t *testing.T) {
	s := newSeenEvents()
	s.markSeen("1")
	assert.False(t, s.markSeen("1"))
}

func TestSeenEvents_DifferentIDsAreIndependent(t *testing.T) {
	s := newSeenEvents()
	assert.True(t, s.markSeen("1"))
	assert.True(t, s.markSeen("2"))
	assert.False(t, s.markSeen("1"))
	assert.False(t, s.markSeen("2"))
}

func TestSeenEvents_CleanupRemovesExpiredEntries(t *testing.T) {
	s := newSeenEvents()
	s.markSeen("1")

	// Back-date the entry so it appears old.
	s.mu.Lock()
	s.entries["1"] = time.Now().Add(-10 * time.Minute)
	s.mu.Unlock()

	s.cleanup(5 * time.Minute)

	// Key "1" should be gone and accepted again.
	assert.True(t, s.markSeen("1"))
}

func TestSeenEvents_CleanupKeepsRecentEntries(t *testing.T) {
	s := newSeenEvents()
	s.markSeen("1")

	s.cleanup(5 * time.Minute)

	// Key "1" was just added, should still be rejected.
	assert.False(t, s.markSeen("1"))
}

func TestSeenEvents_CleanupOnlyRemovesExpired(t *testing.T) {
	s := newSeenEvents()
	s.markSeen("1") // recent
	s.markSeen("2") // will be back-dated

	s.mu.Lock()
	s.entries["2"] = time.Now().Add(-10 * time.Minute)
	s.mu.Unlock()

	s.cleanup(5 * time.Minute)

	assert.False(t, s.markSeen("1")) // recent entry survives
	assert.True(t, s.markSeen("2"))  // expired entry was removed
}

func TestSeenEvents_ConcurrentMarkSeen(t *testing.T) {
	s := newSeenEvents()
	const n = 100

	var wg sync.WaitGroup
	accepted := make([]bool, n)

	for i := range n {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			accepted[id] = s.markSeen("42") // all goroutines try the same key
		}(i)
	}
	wg.Wait()

	trueCount := 0
	for _, v := range accepted {
		if v {
			trueCount++
		}
	}
	assert.Equal(t, 1, trueCount, "exactly one goroutine should win the race")
}
