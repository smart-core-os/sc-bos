package devices

import (
	"strings"
)

// A nameNode is a node in a tree where each node represents a segment of a path.
type nameNode struct {
	// leaf nodes have a name associated with them
	name *name

	children mapping[string, *nameNode]
}

func newTreeFromPaths(paths ...string) *nameNode {
	root := &nameNode{}
	for _, p := range paths {
		root.addName(parseName(p))
	}
	return root
}

// addName adds the given name to the tree.
func (n *nameNode) addName(s *name) {
	n.addSegments(*s, s)
}

// addSegments adds the given segments to the tree, associating the name s with the leaf node.
func (n *nameNode) addSegments(segs []string, s *name) {
	if len(segs) == 0 {
		n.name = s
		return
	}
	seg := segs[0]
	n.addChild(seg).addSegments(segs[1:], s)
}

// addChild returns the child with the given key, creating it if necessary.
func (n *nameNode) addChild(key string) *nameNode {
	if c := n.findChild(key); c != nil {
		return c
	}
	c := &nameNode{}
	n.children.add(key, c)
	return c
}

// findChild returns the child node with the given key, or nil if there is no such child.
func (n *nameNode) findChild(key string) *nameNode {
	child, _ := n.children.find(key)
	return child
}

// matchDescendant returns a leaf node that is an ancestor of the given descendent path d.
// The self return value will be true if d exactly matches a leaf node in the tree.
func (n *nameNode) matchDescendant(d string) (_ *nameNode, self bool) {
	if n == nil {
		return nil, false
	}
	if d == "" {
		if n.name == nil {
			return nil, false
		}
		return n, true
	}
	if n.name != nil {
		return n, false
	}
	seg, rest := firstSegment(d)
	return n.findChild(seg).matchDescendant(rest)
}

// firstSegment returns the first segment of path, before the first '/'.
// If p does not contain a '/', the entire p is returned as the segment, and rest is empty.
func firstSegment(p string) (seg, rest string) {
	seg, rest, _ = strings.Cut(p, "/")
	return seg, rest
}

// name represents a parsed name.
type name []string

func parseName(s string) *name {
	segs := name(strings.Split(s, "/"))
	return &segs
}

func (n *name) String() string {
	return strings.Join(*n, "/")
}

// mapping is copied from the http package to represent a k-v mapping, with optimisations for few entries.
type mapping[K comparable, V any] struct {
	s []entry[K, V] // for few mappings
	m map[K]V       // for many mappings
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// taken from http.ServerMux impl based on benchmarks
const maxSlice = 8

func (h *mapping[K, V]) add(k K, v V) {
	if h.m == nil && len(h.s) < maxSlice {
		h.s = append(h.s, entry[K, V]{k, v})
	} else {
		if h.m == nil {
			h.m = map[K]V{}
			for _, e := range h.s {
				h.m[e.key] = e.value
			}
			h.s = nil
		}
		h.m[k] = v
	}
}

// find returns the value corresponding to the given key.
// The second return value is false if there is no value
// with that key.
func (h *mapping[K, V]) find(k K) (v V, found bool) {
	if h == nil {
		return v, false
	}
	if h.m != nil {
		v, found = h.m[k]
		return v, found
	}
	for _, e := range h.s {
		if e.key == k {
			return e.value, true
		}
	}
	return v, false
}
