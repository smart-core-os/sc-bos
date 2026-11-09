package connecttelemetry

import (
	"strings"
)

// Grouping sends a piece of plant whose driver announces each point as its own device as one device.
// A device named <parent>/<leaf> of a trait in config.Root.GroupChildren, whose parent is an announced
// device, is sent as a point of <parent>: its telemetry as a point keyed by pointKey(leaf), and its
// health checks in the parent's set, their ids prefixed by the leaf. Telemetry and health share
// groupTarget, and each keeps a parent once found until the config changes, so a child's readings
// and checks land on the same Connect ref.

// splitChild splits name at its last '/' into the parent it may group into and its leaf.
// ok is false for a name with no '/', or an empty parent or leaf.
func splitChild(name string) (parent, leaf string, ok bool) {
	i := strings.LastIndexByte(name, '/')
	if i <= 0 || i == len(name)-1 {
		return "", "", false
	}
	return name[:i], name[i+1:], true
}

// groupTarget returns the device name groups into, a child of a trait being grouped.
// It groups into its parent, the name before the last '/', when parentKnown reports the parent
// as an announced device that isn't a zone. ok is false when name is sent as itself.
// The caller has already decided the child itself isn't a zone.
func groupTarget(name string, parentKnown func(string) bool) (target, leaf string, ok bool) {
	parent, leaf, ok := splitChild(name)
	if !ok || !parentKnown(parent) {
		return "", "", false
	}
	return parent, leaf, true
}

// pointKey is the point a child's reading is sent as: its leaf in lower case, each run of
// characters other than letters and digits becoming one '_', trimmed of '_' at either end.
// chw-primary-flow becomes chw_primary_flow. Connect's per-ref mappings turn these keys into
// fields, so they are the same whatever the point naming.
func pointKey(leaf string) string {
	var b strings.Builder
	b.Grow(len(leaf))
	sep := false
	for _, r := range strings.ToLower(leaf) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if sep && b.Len() > 0 {
				b.WriteByte('_')
			}
			sep = false
			b.WriteRune(r)
			continue
		}
		sep = true
	}
	return b.String()
}
