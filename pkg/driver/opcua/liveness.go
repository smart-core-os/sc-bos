package opcua

import (
	"fmt"
	"sync"

	"github.com/gopcua/opcua/ua"
)

// pointLiveness tracks, for each node a trait reads, whether the subscription vouches for the
// last value it delivered.
//
// OPC UA notifies on change alone, so a steady point's last notification can be hours old and
// still be the server's current value; equally, a point whose value was lost keeps its last
// notification for ever. The subscription vouches for a node once it has delivered a usable
// value for it (Good or Uncertain, as handleStatusValue passes both to the traits), until a Bad
// status, the connection dropping or the subscription ending says otherwise, and only while the
// connection is up. A trait serves a vouched value as current and reads anything else live.
//
// Keyed by the configured node id text, which is what nodeIdsAreEqual matches a notification
// against. Every method ignores a node it was not built with.
//
// Written by the device's pump goroutine and gopcua's connection goroutine, read by every Get.
type pointLiveness struct {
	// applyMu orders the trait's cache updates, and is held across apply so a live read cannot
	// overwrite a newer notification. Separate from mu because apply calls resource.Value.Set,
	// which waits on every Pull listener: holding mu across it would let one slow Pull client
	// stall gopcua's connection goroutine in connectionSuspect, and every Get in vouched.
	// Taken before mu, never while holding it.
	applyMu sync.Mutex
	// mu guards points, and is only ever held briefly.
	mu     sync.Mutex
	points map[string]*pointState
	// connected reports whether the connection is up right now; nil means always. Needed on
	// top of suspect because a value already in flight when the connection drops can land
	// after the drop and clear suspect, and gopcua reports a long outage only once.
	connected func() bool
}

type pointState struct {
	// delivered is whether the current subscription has delivered a usable value. Cleared
	// when the subscription ends, since a new one has to prove itself again.
	delivered bool
	// bad is whether the last notification was Bad, or a value the trait could not use.
	bad bool
	// suspect is whether the connection has dropped or reported a transport error since the
	// last usable value. gopcua transfers the subscription after a blip without resending
	// initial values, so nothing would otherwise tell us an idle point's value survived it.
	suspect bool
	// gen counts usable notifications, so a live read can tell one landed while it was out.
	gen uint64
}

// newPointLiveness tracks nodeIds, vouching for them only while connected reports true. A nil
// connected is always up.
func newPointLiveness(connected func() bool, nodeIds ...string) *pointLiveness {
	l := &pointLiveness{points: make(map[string]*pointState, len(nodeIds)), connected: connected}
	for _, nodeId := range nodeIds {
		l.points[nodeId] = &pointState{}
	}
	return l
}

// vouched reports whether the subscription stands behind node's cached value.
func (l *pointLiveness) vouched(node string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.points[node]
	return ok && p.delivered && !p.bad && !p.suspect && (l.connected == nil || l.connected())
}

// readerConnected is reader's connection state for newPointLiveness, nil when there is no reader.
func readerConnected(reader pointReader) func() bool {
	if reader == nil {
		return nil
	}
	return reader.Connected
}

// good records a usable notification for node and calls apply, which updates the trait's cache,
// in order with any live read's update.
func (l *pointLiveness) good(node string, apply func()) {
	l.applyMu.Lock()
	defer l.applyMu.Unlock()
	l.mu.Lock()
	p, ok := l.points[node]
	if ok {
		p.delivered, p.bad, p.suspect = true, false, false
		p.gen++
	}
	l.mu.Unlock()
	if ok {
		apply()
	}
}

// unusable records that node's last notification was Bad, or carried a value the trait could
// not use. Either way the cached value is no longer the server's.
func (l *pointLiveness) unusable(node string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p, ok := l.points[node]; ok {
		p.bad = true
	}
}

// beginRead returns the generation to hand to readGood once a live read of node is back.
func (l *pointLiveness) beginRead(node string) uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p, ok := l.points[node]; ok {
		return p.gen
	}
	return 0
}

// readGood records a usable live read of node and calls apply to update the cache, unless a
// notification landed while the read was out: that is at least as fresh, so it keeps the cache.
//
// The read restores the subscription's word only when the subscription has delivered before.
// That is the dropped-connection case, where the read proves the value survived. A node that
// has never delivered, or is Bad, stays unvouched, so a monitored item that is broken is never
// vouched for on the strength of one read.
func (l *pointLiveness) readGood(node string, gen uint64, apply func()) {
	// gen only moves in good, under applyMu, so it cannot change between the check and apply
	l.applyMu.Lock()
	defer l.applyMu.Unlock()
	l.mu.Lock()
	p, ok := l.points[node]
	fresh := false
	if ok {
		if p.delivered {
			p.suspect = false
		}
		fresh = p.gen == gen
	}
	l.mu.Unlock()
	if fresh {
		apply()
	}
}

// connectionSuspect records a dropped connection or transport error, which every node shares.
func (l *pointLiveness) connectionSuspect() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.points {
		p.suspect = true
	}
}

// subscriptionLost records the subscription carrying every node ending.
func (l *pointLiveness) subscriptionLost() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.points {
		p.delivered = false
	}
}

// parseLiveNode parses a configured node id for reading it live, or nil when it does not parse.
// A failure is not fatal: config validation never parsed trait node ids, so a config with one
// that does not parse has always loaded. The trait just can't read that node live.
func parseLiveNode(nodeId string) *ua.NodeID {
	node, err := ua.ParseNodeID(nodeId)
	if err != nil {
		return nil
	}
	return node
}

// liveValue is the usable value of a live read, or an error saying why there is none.
// Uncertain is usable, matching what the subscription hands the traits.
func liveValue(dv *ua.DataValue) (any, error) {
	switch {
	case dv == nil:
		return nil, fmt.Errorf("is unavailable: %v", ua.StatusBad)
	case statusIsBad(dv.Status):
		return nil, fmt.Errorf("is unavailable: %v", dv.Status)
	case dv.Value == nil:
		return nil, fmt.Errorf("has no value")
	}
	return dv.Value.Value(), nil
}
