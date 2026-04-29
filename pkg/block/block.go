// Package block provides a way to compare and apply changes to nested data structures.
//
// The data structures are compared in logical blocks defined by a schema of Block objects.
// Each Patch returned from Diff will replace the value of one such Block. When the Patch is applied, the value of that
// block will be replaced, but other blocks (including blocks that are children of the replaced block) are left unchanged.
// Arrays can be split into a block per element, as long as the elements are struct-like and have a key which can be
// used to identify them.
//
// The package can be used to apply configuration changes from multiple sources to a single configuration object,
// in a way that reduces the chances of conflict or incorrect merging.
package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// Diff finds the changes required to transform a into b, using the provided blocks to define logical sections of
// the data.
// Diffing will be performed on the JSON representation of the data, so the data must be JSON-serializable.
// The returned patches will be non-conflicting, meaning that they can be applied in any order and will produce
// the same result. The order of patches is not guaranteed to be consistent between runs - if a deterministic order
// is required, you can sort the patches by Path.
// Returns an error when a or b fail to marshal to JSON.
func Diff(a, b any, blocks []Block) ([]Patch, error) {
	// convert a and b from any Go type to the default JSON representation: map[string]any, []any, and primitive types
	var err error
	a, err = convert[any](a)
	if err != nil {
		return nil, err
	}
	b, err = convert[any](b)
	if err != nil {
		return nil, err
	}

	patches := diff(a, b, Block{Blocks: blocks})
	return patches, nil
}

// performs a round-trip conversion to and from JSON
func convert[T any](in any) (T, error) {
	var out T
	serialised, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(serialised, &out)
	return out, err
}

// a and b must be map[string]any, []any, or primitive types nested in any arrangement.
// returns patches that transform a into b
func diff(a, b any, block Block) []Patch {
	if equal(a, b) {
		return nil
	}

	switch a := a.(type) {
	case map[string]any:
		bMap, ok := b.(map[string]any)
		if !ok {
			return []Patch{{Value: b}}
		}
		return diffMap(a, bMap, block.Blocks)

	case []any:
		bSlice, ok := b.([]any)
		if !ok {
			return []Patch{{Value: b}}
		}
		return diffArray(a, bSlice, block)

	default:
		// we know that a and b are not equal and cannot be split further
		return []Patch{{Value: b}}
	}
}

func diffMap(a, b map[string]any, blocks []Block) []Patch {
	// alternate representation of block which is better for recursive descent
	tree := buildBlockTree(blocks)
	// a and b might have sub-blocks - need to split them out and compare separately from a and b's own fields
	a, aBlockValues := splitMap(a, tree)
	b, bBlockValues := splitMap(b, tree)

	patches := diffBlockValues(aBlockValues, bBlockValues)
	if !equal(a, b) {
		patches = append(patches, Patch{Value: b})
	}

	return patches
}

func diffBlockValues(a, b []blockValue) []Patch {
	// by sorting the blocks, we can step through them in order (linear time)
	// to find pages with matching paths
	compareBlockValues := func(a, b blockValue) int {
		return slices.CompareFunc(a.Path, b.Path, strings.Compare)
	}
	slices.SortFunc(a, compareBlockValues)
	slices.SortFunc(b, compareBlockValues)

	type change struct {
		Path    []string
		Deleted bool
		A, B    blockValue
	}
	var changes []change

	for len(a) > 0 && len(b) > 0 {
		c := compareBlockValues(a[0], b[0])
		if c < 0 {
			changes = append(changes, change{Path: a[0].Path, A: a[0], Deleted: true})
			a = a[1:]
		} else if c > 0 {
			changes = append(changes, change{Path: b[0].Path, B: b[0]})
			b = b[1:]
		} else {
			if !equal(a[0].Value, b[0].Value) {
				changes = append(changes, change{Path: a[0].Path, A: a[0], B: b[0]})
			}
			a = a[1:]
			b = b[1:]
		}
	}
	for _, p := range a {
		changes = append(changes, change{Path: p.Path, A: p, Deleted: true})
	}
	for _, p := range b {
		changes = append(changes, change{Path: p.Path, B: p})
	}

	var patches []Patch
	for _, c := range changes {
		if c.Deleted {
			patches = append(patches, Patch{Path: fieldPathSegments(c.Path), Deleted: true})
			continue
		}

		subpatches := diff(c.A.Value, c.B.Value, c.B.Block)
		prefixPatches(subpatches, fieldPathSegments(c.Path))
		patches = append(patches, subpatches...)
	}

	return patches
}

func prefixPatches(patches []Patch, prefix Path) {
	for i := range patches {
		var prefixed []PathSegment
		prefixed = append(prefixed, prefix...)
		prefixed = append(prefixed, patches[i].Path...)
		patches[i].Path = prefixed
	}
}

func fieldPathSegments(path []string) Path {
	var segs []PathSegment
	for _, p := range path {
		segs = append(segs, PathSegment{Field: p})
	}
	return segs
}

// A blockTree is like a []Block, but instead block location being identified by a list of field names to traverse,
// it is transformed into a tree.
// A leaf node in this tree is equivalent to a Block from the original []Block
//
// e.g. the following []Block:
//   - Block A at path ["foo", "bar"]
//   - Block B at path ["foo", "baz"]
//   - Block C at path ["qux"]
//
// would be transformed into the following blockTree:
// - foo:
//   - bar - Block A
//   - baz - Block B
//
// - qux - Block C
type blockTree map[string]blockTreeEntry

// blockTreeEntry contains either a Block or a blockTree, not both
type blockTreeEntry struct {
	Block  Block
	Fields blockTree
}

func (e blockTreeEntry) IsLeaf() bool {
	return len(e.Fields) == 0
}

func buildBlockTree(blocks []Block) blockTree {
	tree := make(blockTree)
	for _, block := range blocks {
		node := tree
		path := block.Path
		for len(path) > 0 {
			key := path[0]
			if len(path) == 1 {
				node[key] = blockTreeEntry{Block: block}
			} else {
				if _, ok := node[key]; !ok {
					node[key] = blockTreeEntry{Fields: make(blockTree)}
				}
				node = node[key].Fields
			}
			path = path[1:]
		}
	}
	return tree
}

// splits out parts of m that belong to separate sub-blocks
// returns a copy of m with those parts removed, and a blockValue for each block that was separated
// In the returned map, each removed block is replaced with Ignore{}
// tree should be a blockTree representing the []Block that applies to m
//
// e.g. if m is:
//
//	{
//	  "foo": {
//	    "bar": 42,
//	    "baz": "hello"
//	  },
//	  "qux": true
//	}
//
// and tree is:
//
//	foo -> bar = Block A
//	qux        = Block B
//
// then this will be split into:
//
//	{
//	  "foo": {
//	    "bar": Ignore{},
//	    "baz": "hello"
//	  },
//	  "qux": Ignore{}
//	}
//
// and the following blockValues:
//
//   - Path: ["foo", "bar"], Value: 42, Block: Block A
//   - Path: ["qux"], Value: true, Block: Block B
func splitMap(m map[string]any, tree blockTree) (map[string]any, []blockValue) {
	if len(tree) == 0 {
		return m, nil
	}
	if m == nil {
		m = make(map[string]any)
	} else {
		m = maps.Clone(m)
	}

	var pages []blockValue
	for k, v := range m {
		subfields, ok := tree[k]
		if !ok {
			// don't need to delete this field or any of its children
			continue
		}
		if subfields.IsLeaf() {
			// leaf of the blockTree, need to delete this key
			delete(m, k)
			pages = append(pages, blockValue{Path: []string{k}, Value: v, Block: subfields.Block})
		} else if submap, ok := v.(map[string]any); ok {
			submap, subpages := splitMap(submap, subfields.Fields)
			m[k] = submap
			for _, p := range subpages {
				p.Path = append([]string{k}, p.Path...)
				pages = append(pages, p)
			}
		}
		// if v is not a map, no fields need to be deleted from it
	}
	markIgnored(m, tree)
	return m, pages
}

// places an Ignore{} value at every position in m corresponding to a leaf in tree
func markIgnored(m map[string]any, tree blockTree) {
	if m == nil {
		panic("cannot mark fields in nil map")
	}
	for k, entry := range tree {
		if entry.IsLeaf() {
			m[k] = Ignore{}
		} else if submap, ok := m[k].(map[string]any); ok {
			markIgnored(submap, entry.Fields)
		}
	}
}

// represents a pair of a block (at a certain position in the data) and the value it contains
type blockValue struct {
	Path  []string // hierarchy of object field names to reach this block
	Block Block    // the Block that applies to Value
	Value any
}

func diffArray(a, b []any, block Block) []Patch {
	if block.Key == "" {
		// if the array doesn't have a key, we can only compare the whole thing
		if equal(a, b) {
			return nil
		} else {
			return []Patch{{Value: b}}
		}
	}

	type entry struct {
		A, B         map[string]any
		AType, BType string
		Blocks       []Block
	}
	entries := make(map[any]entry) // we allow any comparable type as array keys
	for _, v := range a {
		key, ok := extractArrayEntryKey(v, block.Key)
		if !ok {
			continue
		}
		// if there is no TypeKey, we'll use "" to indicate that
		entryType, ok := extractArrayEntryType(v, block.TypeKey)
		if !ok {
			entryType = ""
		}
		entries[key] = entry{
			A:     v.(map[string]any),
			AType: entryType,
		}
	}
	for _, v := range b {
		key, ok := extractArrayEntryKey(v, block.Key)
		if !ok {
			continue
		}
		ty, ok := extractArrayEntryType(v, block.TypeKey)
		if !ok {
			ty = ""
		}
		e := entries[key]
		e.B = v.(map[string]any)
		e.BType = ty
		entries[key] = e
	}

	var patches []Patch
	for k, e := range entries {
		if e.B == nil {
			// array element was deleted
			patches = append(patches, Patch{
				Path:    []PathSegment{{ArrayKey: block.Key, ArrayElem: k}},
				Deleted: true,
			})
		} else if e.AType != e.BType {
			// array element changed types and therefore block list, not directly comparable, replace the entire element
			// (this also covers the case where a new element was added)
			patches = append(patches, Patch{
				Path:  []PathSegment{{ArrayKey: block.Key, ArrayElem: k}},
				Value: e.B,
			})
		} else {
			blocks, ok := block.BlocksByType[e.BType]
			if !ok {
				// fall back to the global blocks
				// this case is used when SplitByKey isn't used
				blocks = block.Blocks
			}

			subpatches := diff(e.A, e.B, Block{Blocks: blocks})
			prefixPatches(subpatches, []PathSegment{{ArrayKey: block.Key, ArrayElem: k}})
			patches = append(patches, subpatches...)
		}
	}

	return patches
}

// extracts the logical key from an object which is used to identify it within an array
// if a key is returned, it is guaranteed to be comparable
func extractArrayEntryKey(v any, key string) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	keyValue, ok := m[key]
	if !ok || !reflect.ValueOf(keyValue).Comparable() {
		return nil, false
	}
	return keyValue, true
}

func extractArrayEntryType(v any, typeKey string) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	entryType, ok := m[typeKey]
	if !ok {
		return "", false
	}
	entryTypeStr, ok := entryType.(string)
	return entryTypeStr, ok
}

// deep equality check for supporting the following nested types:
// - map[string]any
// - []any
// - comparable types
func equal(a, b any) bool {
	switch a := a.(type) {
	case map[string]any:
		b, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if !equalKeys(a, b) {
			return false
		}
		for k, v := range a {
			if !equal(v, b[k]) {
				return false
			}
		}
		return true

	case []any:
		b, ok := b.([]any)
		if !ok {
			return false
		}
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !equal(a[i], b[i]) {
				return false
			}
		}
		return true

	default:
		return a == b
	}
}

// returns true if a and b have the same key set
func equalKeys(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// ApplyPatches applies a slice of Patch to a data structure.
// The type must be both JSON-serializable and JSON-deserializable.
// A patched copy of the data structure will be returned. The original will not be modified.
func ApplyPatches[T any](data T, patches []Patch) (T, error) {
	workingCopy, err := convert[any](data)
	if err != nil {
		return data, err
	}
	for _, patch := range patches {
		var err error
		workingCopy, err = applyPatch(workingCopy, patch)
		if err != nil {
			return data, err
		}
	}
	return convert[T](workingCopy)
}

// The patch will be performed in-place if possible, and the modified data structure will be returned.
// The data structure must be a JSON-like structure of map[string]any, []any and primitive types.
func applyPatch(data any, patch Patch) (any, error) {
	if len(patch.Path) == 0 {
		if patch.Deleted {
			return nil, errors.New("cannot delete root")
		} else {
			return patchValue(data, patch.Value)
		}
	}

	segment := patch.Path[0]
	if segment.IsField() {
		m, ok := data.(map[string]any)
		if !ok {
			return data, fmt.Errorf("cannot patch %T with field", data)
		}
		if len(patch.Path) == 1 && patch.Deleted {
			delete(m, segment.Field)
			return m, nil
		}

		fieldValue, ok := m[segment.Field]
		if !ok {
			fieldValue = emptyValue(patch.Path[1:])
		}
		patched, err := applyPatch(fieldValue, Patch{
			Path:    patch.Path[1:],
			Value:   patch.Value,
			Deleted: patch.Deleted,
		})
		if err != nil {
			return data, err
		}
		m[segment.Field] = patched
		return m, nil
	} else if segment.IsArrayElem() {
		a, ok := data.([]any)
		if !ok {
			return data, fmt.Errorf("cannot patch %T with array element", data)
		}

		i := slices.IndexFunc(a, func(a any) bool {
			m, ok := a.(map[string]any)
			if !ok {
				return false
			}
			v, ok := m[segment.ArrayKey]
			if !ok {
				return false
			}
			return reflect.ValueOf(v).Comparable() && v == segment.ArrayElem
		})
		if patch.Deleted && len(patch.Path) == 1 {
			if i >= 0 {
				a = slices.Delete(a, i, i+1)
			}
			return a, nil
		}
		var existing map[string]any
		if i >= 0 {
			// only a map[string]any can pass the slices.IndexFunc check
			existing = a[i].(map[string]any)
		} else {
			// create a placeholder map with the right key to hold the change
			existing = map[string]any{segment.ArrayKey: segment.ArrayElem}
		}

		patched, err := applyPatch(existing, Patch{
			Path:    patch.Path[1:],
			Value:   patch.Value,
			Deleted: patch.Deleted,
		})
		if err != nil {
			return data, err
		}
		if i >= 0 {
			a[i] = patched
		} else {
			a = append(a, patched)
		}
		return a, nil

	} else {
		return data, ErrInvalidPathSegment
	}
}

func patchValue(dst any, value any) (any, error) {
	if dst == nil {
		return value, nil
	}
	if _, ok := value.(Ignore); ok {
		// points represent the limit of the patch
		// nothing to apply
		return dst, nil
	}

	// replace the top-level value
	switch dst := dst.(type) {
	case map[string]any:
		if patch, ok := value.(map[string]any); ok {
			return patchMap(dst, patch)
		} else {
			return dst, fmt.Errorf("cannot patch map with %T", value)
		}

	case []any:
		if patch, ok := value.([]any); ok {
			dst = dst[:0]
			dst = append(dst, patch...)
			return dst, nil
		} else {
			return dst, fmt.Errorf("cannot patch array with %T", value)
		}

	default:
		return value, nil
	}
}

func patchMap(m map[string]any, patch map[string]any) (map[string]any, error) {
	patched := make(map[string]any)
	for k, v := range patch {
		if _, ok := v.(Ignore); ok {
			// we need to preserve the original value, if it exists
			if original, ok := m[k]; ok {
				patched[k] = original
			}
		} else {
			patchedField, err := patchValue(m[k], v)
			if err != nil {
				return m, err
			}
			patched[k] = patchedField
		}
	}

	return patched, nil
}

// creates a new empty value that can be indexed by the given path
func emptyValue(segs []PathSegment) any {
	if len(segs) == 0 {
		return nil
	}

	seg := segs[0]
	if seg.IsField() {
		return map[string]any{}
	} else if seg.IsArrayElem() {
		return []any{}
	} else {
		panic("invalid path segment")
	}
}

type Patch struct {
	// How to reach the section to be patched from the root of the data structure
	Path Path `json:"path"`
	// The new value to replace the section with
	// If an Ignore{} value is present, the data in this position will be left unchanged
	Value any `json:"value,omitempty"`
	// If true, the section will be deleted
	// e.g. if the section is a field, the field will be removed
	//      if the section is an array element, the element will be removed
	Deleted bool `json:"deleted,omitempty"`
}

func (p *Patch) UnmarshalJSON(data []byte) error {
	type patchAlias Patch
	err := json.Unmarshal(data, (*patchAlias)(p))
	if err != nil {
		return err
	}
	// make sure JSON maps of the form {"$block": "ignore"} are converted to Ignore{} values
	// to match the behavior of Ignore.MarshalJSON
	convertIgnores(p.Value)
	return nil
}

// Ignore is a special value that can be used in a Patch value
// to indicate that the data in this position should be left unchanged.
type Ignore struct{}

func (i *Ignore) MarshalJSON() ([]byte, error) {
	return json.Marshal(ignoreSentinal)
}

var ignoreSentinal = map[string]any{"$block": "ignore"}

// modifies data in-place to swap any values of the form map[string]any{"$block": "ignore"} with Ignore{}
func convertIgnores(data any) any {
	switch data := data.(type) {
	case map[string]any:
		if maps.Equal(data, ignoreSentinal) {
			return Ignore{}
		}
		for k, v := range data {
			data[k] = convertIgnores(v)
		}

	case []any:
		for i, v := range data {
			data[i] = convertIgnores(v)
		}
	default:
	}
	return data
}

// Block represents a logical section of a data structure.
// When using Diff, a Block will be compared/replaced in its entirety, except for any child blocks.
type Block struct {
	// The names of fields traversed from the parent block to reach this block (or from the root if there is no
	// parent block).
	Path []string `json:"path"`
	// The name of a field in array elements whose value identifies the array element.
	// When diffing and patching, array elements are located by the value of this field, not the position in the array.
	Key string `json:"key,omitempty"`
	// Where Path points to an array (Key is set), the sub-blocks that apply to each element of the array.
	// Otherwise, the sub-blocks of the object at Path.
	Blocks []Block `json:"blocks,omitempty"`
	// The name of a field in array elements that determines which blocks (from BlocksByType) to use for that element
	// Optional - if absent, all array elements will use the Blocks field.
	// If an array element lacks the TypeKey, or the value of TypeKey does not match any key in BlocksByType, the Blocks field will be used.
	TypeKey      string             `json:"typeKey,omitempty"`
	BlocksByType map[string][]Block `json:"blocksByType,omitempty"`
}

var ErrInvalidPathSegment = errors.New("invalid path segment")
