package connecttelemetry

import (
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// healthGroups folds the checks of grouped child devices into their parents' sets, so a child's
// checks land on the same Connect ref as its telemetry, see groupTarget. Each source, a name with
// checks, is published under a target: its parent if it groups, else itself. A target's set is its
// own checks followed by each child's, their ids prefixed with the child's leaf, so the same check
// on several children, such as one healthbounds automation's, stays unique within the set.
//
// What it knows of each device is kept once seen, as healthPublisher.zones is. Devices go and come
// back whenever a driver is reconfigured, a little before or after their checks do, and forgetting
// a parent or child meanwhile would move every child's checks onto its own ref and back.
type healthGroups struct {
	traits []string // the traits whose children group

	mu sync.Mutex
	// settled is set once the publisher's hold after start is over. Until then a source whose
	// device, or whose parent if it is a child, hasn't been seen is held rather than published as
	// itself, since the device stream may not have caught up with the checks yet. Sending it as
	// itself and then moving it would send its ref an empty set on every start.
	settled  bool
	seen     map[string]bool                    // names announced as devices
	child    map[string]bool                    // names announced with a grouped trait
	zone     map[string]bool                    // names announced as zones
	checks   map[string][]*healthpb.HealthCheck // each source's checks, present until it is removed
	target   map[string]string                  // each placed source's target
	members  map[string]map[string]bool         // each target's sources
	byParent map[string]map[string]bool         // each present source, by the parent it may group into

	// Each grouped target waiting healthGroupSettle after a change of its sources' checks, see
	// publishTargetSoonLocked.
	pending map[string]*pendingFlush
}

// pendingFlush is a target's wait before its set is published. Its timer's callback publishes only
// while this is still the target's pending flush, so a timer that fires as the target is published
// some other way doesn't cut short the wait that follows.
type pendingFlush struct {
	timer *time.Timer
}

// healthGroupSettle is how long a grouped target's set waits after a change of its sources' checks
// before it is published, so children changing together, as a healthbounds automation's checks on every
// point of one device do each time it reads them, send their parent's set once rather than once each.
// Each send is the parent's whole set, so without it a device whose points change in one pass sends
// that set several times within a fraction of a second. Each target waits from its own first change,
// so another device's changes don't cut its wait short.
const healthGroupSettle = 500 * time.Millisecond

func newHealthGroups(traits []string) *healthGroups {
	return &healthGroups{
		traits:   traits,
		seen:     make(map[string]bool),
		child:    make(map[string]bool),
		zone:     make(map[string]bool),
		checks:   make(map[string][]*healthpb.HealthCheck),
		target:   make(map[string]string),
		members:  make(map[string]map[string]bool),
		byParent: make(map[string]map[string]bool),
		pending:  make(map[string]*pendingFlush),
	}
}

// groupTraits folds the checks of the children of traits into their parents' sets, see healthGroups.
// Call it before run. With no traits nothing is folded.
func (p *healthPublisher) groupTraits(traits []string) {
	if len(traits) == 0 {
		p.groups = nil
		return
	}
	p.groups = newHealthGroups(traits)
}

// groupedChecks records name's checks, nil with removed when it has none, and publishes the sets that changed.
func (p *healthPublisher) groupedChecks(name string, checks []*healthpb.HealthCheck, removed bool) {
	g := p.groups
	g.mu.Lock()
	defer g.mu.Unlock()
	parent, _, hasParent := splitChild(name)
	if removed {
		delete(g.checks, name)
		if hasParent {
			deleteMember(g.byParent, parent, name)
		}
	} else {
		g.checks[name] = checks
		if hasParent {
			addMember(g.byParent, parent, name)
		}
	}
	p.placeLocked(name, true)
}

// groupedDevice records what is known of the device name, announced with traits, and moves the
// sources whose target that changes.
func (p *healthPublisher) groupedDevice(name string, traits []string, zone bool) {
	g := p.groups
	g.mu.Lock()
	defer g.mu.Unlock()
	child := slices.ContainsFunc(traits, func(t string) bool { return slices.Contains(g.traits, t) })
	if g.seen[name] && (g.child[name] || !child) && (g.zone[name] || !zone) {
		return // nothing new
	}
	g.seen[name] = true
	if child {
		g.child[name] = true
	}
	if zone {
		g.zone[name] = true
	}
	p.placeLocked(name, false)
	for src := range g.byParent[name] {
		p.placeLocked(src, false)
	}
}

// settleGroups publishes every source held until the publisher's hold after start is over.
func (p *healthPublisher) settleGroups() {
	g := p.groups
	g.mu.Lock()
	defer g.mu.Unlock()
	g.settled = true
	for name := range g.checks {
		if _, placed := g.target[name]; !placed {
			p.placeLocked(name, false)
		}
	}
}

// targetLocked returns the target of the source name, false while it is held, see healthGroups.settled.
func (g *healthGroups) targetLocked(name string) (string, bool) {
	parent, _, hasParent := splitChild(name)
	if !g.settled && (!g.seen[name] || (g.child[name] && hasParent && !g.seen[parent])) {
		return "", false
	}
	if g.child[name] && !g.zone[name] && hasParent && g.seen[parent] && !g.zone[parent] {
		return parent, true
	}
	return name, true
}

// placeLocked brings the source name's target up to date, publishing the sets that change.
// changed is whether name's own checks have changed, so its target's set needs publishing even
// if the target is the same.
func (p *healthPublisher) placeLocked(name string, changed bool) {
	g := p.groups
	_, present := g.checks[name]
	target, ok := g.targetLocked(name)
	if !present {
		target, ok = "", false
	}
	old, placed := g.target[name]
	if placed && (!ok || old != target) {
		delete(g.target, name)
		deleteMember(g.members, old, name)
		p.publishTargetLocked(old)
	}
	if !ok {
		return
	}
	switch {
	case !placed || old != target:
		g.target[name] = target
		addMember(g.members, target, name)
		p.publishTargetLocked(target)
	case changed:
		p.publishTargetSoonLocked(target)
	}
}

// publishTargetSoonLocked publishes target's set after a change to one of its sources' checks. A target
// with children waits healthGroupSettle from its first change, so its set is built once from what its
// sources hold then; anything else is published at once.
func (p *healthPublisher) publishTargetSoonLocked(target string) {
	g := p.groups
	if !hasChildLocked(g, target) {
		p.publishTargetLocked(target)
		return
	}
	if g.pending[target] != nil {
		return // already waiting
	}
	pf := &pendingFlush{}
	g.pending[target] = pf
	pf.timer = time.AfterFunc(healthGroupSettle, func() { p.flushTarget(target, pf) })
}

// hasChildLocked reports whether any source other than target itself is published under it.
func hasChildLocked(g *healthGroups, target string) bool {
	for src := range g.members[target] {
		if src != target {
			return true
		}
	}
	return false
}

// flushTarget publishes target at the end of its wait pf, unless it has been published since.
func (p *healthPublisher) flushTarget(target string, pf *pendingFlush) {
	g := p.groups
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending[target] != pf {
		return
	}
	p.publishTargetLocked(target)
}

// stopGroups stops every pending wait, for when the publisher stops.
// The sets they would have published differ from those acknowledged, so they go after the next start.
func (p *healthPublisher) stopGroups() {
	g := p.groups
	g.mu.Lock()
	defer g.mu.Unlock()
	for target, pf := range g.pending {
		pf.timer.Stop()
		delete(g.pending, target)
	}
}

// publishTargetLocked publishes target's set from its sources' checks, or its removal if it has none.
func (p *healthPublisher) publishTargetLocked(target string) {
	g := p.groups
	if pf := g.pending[target]; pf != nil { // built from what its sources hold now, so nothing is left to flush
		pf.timer.Stop()
		delete(g.pending, target)
	}
	sources := g.members[target]
	if len(sources) == 0 {
		p.removeResource(target)
		return
	}
	var checks []*healthpb.HealthCheck
	if sources[target] {
		checks = g.checks[target]
		if len(sources) == 1 {
			p.setChecks(target, checks) // as it would be without grouping
			return
		}
		checks = slices.Clone(checks)
	}
	children := make([]string, 0, len(sources))
	for src := range sources {
		if src != target {
			children = append(children, src)
		}
	}
	slices.Sort(children)
	for _, src := range children {
		_, leaf, _ := splitChild(src)
		for _, c := range g.checks[src] {
			checks = append(checks, childCheck(leaf, c))
		}
	}
	p.setChecks(target, checks)
}

// childCheck is c, a check of the child leaf, as it goes in its parent's set:
// its id is prefixed with "<leaf>/" and its display name with "<leaf>: ".
func childCheck(leaf string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
	c = proto.Clone(c).(*healthpb.HealthCheck)
	name := c.GetDisplayName()
	if name == "" {
		name = c.GetId()
	}
	c.Id = leaf + "/" + c.GetId()
	c.DisplayName = leaf + ": " + name
	return c
}

func addMember(m map[string]map[string]bool, key, member string) {
	if m[key] == nil {
		m[key] = make(map[string]bool)
	}
	m[key][member] = true
}

func deleteMember(m map[string]map[string]bool, key, member string) {
	delete(m[key], member)
	if len(m[key]) == 0 {
		delete(m, key)
	}
}
