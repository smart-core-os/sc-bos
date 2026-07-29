package paxton

import (
	"sync"
	"time"
)

const (
	// seenEventsMaxAge is how long event IDs are retained for deduplication between polling and SignalR.
	seenEventsMaxAge          = 5 * time.Minute
	seenEventsCleanupInterval = time.Minute
)

// seenEvents tracks recently processed event keys to prevent duplicate delivery
// when both REST polling and SignalR are active simultaneously.
// Keys are either a stringified integer event ID (REST) or a content fingerprint (SignalR liveEvents).
type seenEvents struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newSeenEvents() *seenEvents {
	return &seenEvents{entries: make(map[string]time.Time)}
}

// markSeen marks a key as seen and returns true if it is new, false if already seen.
func (s *seenEvents) markSeen(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[key]; ok {
		return false
	}
	s.entries[key] = time.Now()
	return true
}

func (s *seenEvents) cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for key, t := range s.entries {
		if t.Before(cutoff) {
			delete(s.entries, key)
		}
	}
}
