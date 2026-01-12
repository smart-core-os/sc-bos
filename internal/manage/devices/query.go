package devices

import (
	"bytes"
	"fmt"
	"iter"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
)

// deviceMatchesQuery returns true if all the conditions in query match fields of device.
func deviceMatchesQuery(query *devicespb.Device_Query, device *devicespb.Device) bool {
	if query == nil {
		return true
	}
	for _, condition := range query.Conditions {
		if !conditionMatchesMessage(condition, device) {
			return false
		}
	}

	// this means a query with no conditions always returns true
	return true
}

// conditionMatchesMessage returns true if the condition matches values in msg.
// If the condition has a path, only values matching the path are considered,
// otherwise all leafs in msg are considered.
func conditionMatchesMessage(cond *devicespb.Device_Query_Condition, msg proto.Message) bool {
	cmp := conditionToCmpFunc(cond)
	values, err := rangeMessage(cond.Field, msg)
	if err != nil {
		return false
	}
	return conditionMatchesValues(cond, values, cmp)
}

// valueMatchesQuery returns true if all the conditions in query match fields of v.
func valueMatchesQuery(query *devicespb.Device_Query, v value) bool {
	for _, condition := range query.Conditions {
		if !conditionMatchesValue(condition, v) {
			return false
		}
	}
	return true
}

// conditionMatchesValue returns true if the condition matches values in v.
// If the condition has a path, only values matching the path are considered,
// otherwise all leafs in v are considered, including v itself.
func conditionMatchesValue(cond *devicespb.Device_Query_Condition, v value) bool {
	cmp := conditionToCmpFunc(cond)
	values, err := rangeValue(cond.Field, v)
	if err != nil {
		return false
	}
	return conditionMatchesValues(cond, values, cmp)
}

// conditionMatchesValues returns true if values match according to cmp.
// cond.RepeatedMatch determines whether any or all values must match.
func conditionMatchesValues(cond *devicespb.Device_Query_Condition, values iter.Seq[value], cmp func(value) bool) bool {
	switch cond.Matcher {
	case devicespb.Device_Query_Condition_MATCHER_UNSPECIFIED, devicespb.Device_Query_Condition_ANY:
		for v := range values {
			if cmp(v) {
				return true
			}
		}
		return false
	case devicespb.Device_Query_Condition_ALL:
		found := false
		for v := range values {
			found = true
			if !cmp(v) {
				return false
			}
		}
		return found
	default:
		return false // unknown repeated match type
	}
}

// conditionToCmpFunc converts a Device_Query_Condition into a function that checks if value values match the condition.
func conditionToCmpFunc(cond *devicespb.Device_Query_Condition) func(value) bool {
	strCmp := func(f func(string) bool) func(v value) bool {
		return func(v value) bool {
			s, ok := v.toString()
			if !ok {
				return false
			}
			return f(s)
		}
	}
	timestampCmp := func(f func(*timestamppb.Timestamp) bool) func(value) bool {
		return func(v value) bool {
			t, ok := v.toTimestamp()
			if !ok {
				return false
			}
			return f(t)
		}
	}
	descendantCmp := func(f func(string) bool) func(v value) bool {
		return strCmp(func(s string) bool {
			if strings.HasSuffix(s, "/") {
				return false // trailing /'s don't match, they'd result in an empty segment
			}
			return f(s)
		})
	}

	switch c := cond.Value.(type) {
	case *devicespb.Device_Query_Condition_StringEqual:
		return strCmp(func(v string) bool {
			return v == c.StringEqual
		})
	case *devicespb.Device_Query_Condition_StringEqualFold:
		return strCmp(func(v string) bool {
			return strings.EqualFold(v, c.StringEqualFold)
		})
	case *devicespb.Device_Query_Condition_StringContains:
		return strCmp(func(v string) bool {
			return strings.Contains(v, c.StringContains)
		})
	case *devicespb.Device_Query_Condition_StringContainsFold:
		ls := strings.ToLower(c.StringContainsFold)
		return strCmp(func(v string) bool {
			return strings.Contains(strings.ToLower(v), ls)
		})
	case *devicespb.Device_Query_Condition_StringIn:
		set := make(map[string]struct{}, len(c.StringIn.Strings))
		for _, s := range c.StringIn.Strings {
			set[s] = struct{}{}
		}
		return strCmp(func(v string) bool {
			_, ok := set[v]
			return ok
		})
	case *devicespb.Device_Query_Condition_StringInFold:
		set := make(map[string]struct{}, len(c.StringInFold.Strings))
		for _, s := range c.StringInFold.Strings {
			set[strings.ToLower(s)] = struct{}{}
		}
		return strCmp(func(v string) bool {
			_, ok := set[strings.ToLower(v)]
			return ok
		})

	case *devicespb.Device_Query_Condition_TimestampEqual:
		return timestampCmp(func(t *timestamppb.Timestamp) bool {
			return t.AsTime().Equal(c.TimestampEqual.AsTime())
		})
	case *devicespb.Device_Query_Condition_TimestampGt:
		return timestampCmp(func(t *timestamppb.Timestamp) bool {
			return t.AsTime().After(c.TimestampGt.AsTime())
		})
	case *devicespb.Device_Query_Condition_TimestampGte:
		return timestampCmp(func(t *timestamppb.Timestamp) bool {
			return !t.AsTime().Before(c.TimestampGte.AsTime())
		})
	case *devicespb.Device_Query_Condition_TimestampLt:
		return timestampCmp(func(t *timestamppb.Timestamp) bool {
			// zero times shouldn't match
			goT := t.AsTime()
			if goT.IsZero() {
				return false
			}
			return goT.Before(c.TimestampLt.AsTime())
		})
	case *devicespb.Device_Query_Condition_TimestampLte:
		return timestampCmp(func(t *timestamppb.Timestamp) bool {
			// zero times shouldn't match
			goT := t.AsTime()
			if goT.IsZero() {
				return false
			}
			return !goT.After(c.TimestampLte.AsTime())
		})

	case *devicespb.Device_Query_Condition_NameDescendant:
		return descendantCmp(func(v string) bool {
			return strings.HasPrefix(v, c.NameDescendant+"/")
		})
	case *devicespb.Device_Query_Condition_NameDescendantInc:
		return descendantCmp(func(v string) bool {
			return v == c.NameDescendantInc || strings.HasPrefix(v, c.NameDescendantInc+"/")
		})
	case *devicespb.Device_Query_Condition_NameDescendantIn:
		tree := newTreeFromPaths(c.NameDescendantIn.Strings...)
		return descendantCmp(func(v string) bool {
			n, self := tree.matchDescendant(v)
			return n != nil && !self
		})
	case *devicespb.Device_Query_Condition_NameDescendantIncIn:
		tree := newTreeFromPaths(c.NameDescendantIncIn.Strings...)
		return descendantCmp(func(v string) bool {
			n, _ := tree.matchDescendant(v)
			return n != nil
		})

	case *devicespb.Device_Query_Condition_Present:
		return func(v value) bool {
			return v.v.IsValid()
		}

	case *devicespb.Device_Query_Condition_Matches:
		return func(v value) bool {
			return valueMatchesQuery(c.Matches, v)
		}
	}

	return func(v value) bool {
		return false // no condition matches, return false
	}
}

// getMessageString returns an iterator over the string values identified by path from msg.
// See valueString for details of string conversion.
func getMessageString(path string, msg proto.Message) iter.Seq[string] {
	return func(yield func(string) bool) {
		values, err := rangeMessage(path, msg)
		if err != nil {
			return
		}
		for v := range values {
			str, got := v.toString()
			if !got || str == "" {
				continue // not a string or empty
			}
			if !yield(str) {
				return // stop iterating
			}
		}
	}
}

// value is a searchable value in a proto message.
// A searchable value can be matched against a query condition.
type value struct {
	fd protoreflect.FieldDescriptor
	v  protoreflect.Value
}

func (v value) String() string {
	s, ok := v.toString()
	if !ok {
		return "<invalid>"
	}
	return fmt.Sprintf("%s %s %s", v.fd.Cardinality(), v.fd.Kind(), s)
}

// rangeValuesOptions defines options for the rangeMessage function.
// Useful for testing where stable order is important, search matching doesn't need to be stable.
type rangeValuesOptions struct {
	// If true, the iterator will yield values in the same order as a stable protorange.Range.
	Stable bool
}

var emptyValues = func(yield func(value) bool) {}

func (opts rangeValuesOptions) RangeMessage(path string, msg proto.Message) (iter.Seq[value], error) {
	if msg == nil {
		return emptyValues, nil
	}
	var segments []pathSegment
	if path != "" {
		var err error
		segments, err = parsePath(path)
		if err != nil {
			return nil, fmt.Errorf("path: %w", err)
		}
	}
	return opts.rangeMessage(segments, msg.ProtoReflect()), nil
}

// rangeMessage returns an iterator over values of msg matching path.
// If path is empty, it returns all leaf values in msg,
// see isLeaf for details on what a leaf value is.
// An error is returned if the path is invalid.
func rangeMessage(path string, msg proto.Message) (iter.Seq[value], error) {
	return rangeValuesOptions{}.RangeMessage(path, msg)
}

func (opts rangeValuesOptions) RangeValue(path string, v value) (iter.Seq[value], error) {
	var segments []pathSegment
	if path != "" {
		var err error
		segments, err = parsePath(path)
		if err != nil {
			return nil, fmt.Errorf("path: %w", err)
		}
	}
	return opts.rangeValue(segments, v), nil
}

// rangeValue returns an iterator over values of v matching path.
// Lists and Maps are traversed as if via scanList and scanMap.
// If path is empty, it returns all leaf values in v,
// see isLeaf for details on what a leaf value is.
// An error is returned if the path is invalid.
func rangeValue(path string, v value) (iter.Seq[value], error) {
	return rangeValuesOptions{}.RangeValue(path, v)
}

// rangeMessage returns all values identified by path in msg.
// If path is empty, all leaf values in msg are returned,
// see isLeaf for details on what a leaf value is.
// Otherwise, only the values matching the path are returned.
func (opts rangeValuesOptions) rangeMessage(path []pathSegment, msg protoreflect.Message) iter.Seq[value] {
	if len(path) == 0 {
		rangeOpts := protorange.Options{
			Stable: opts.Stable,
		}
		return func(yield func(value) bool) {
			// we never return an error from the RangeMessage, so we can ignore the error return value
			_ = rangeOpts.Range(msg, func(values protopath.Values) error {
				fd, isLeaf := getLeafDescriptor(values)
				if !isLeaf {
					return nil
				}
				if !yield(value{fd, values.Index(-1).Value}) {
					return protorange.Terminate
				}
				return nil
			}, nil)
		}
	}
	return opts.scanMessage(path, msg)
}

// rangeValue returns all values identified by path in v.
// Lists and Maps are traversed as if via scanList and scanMap.
// If v.fd is a list, v.v may refer to the protoreflect.List or one of the list items.
func (opts rangeValuesOptions) rangeValue(path []pathSegment, v value) iter.Seq[value] {
	switch {
	case v.fd.IsMap():
		return opts.scanMap(path, v.fd, v.v.Map())
	case v.fd.IsList():
		// a []string (for example) could have a list fd but a string v, so be careful of this
		switch vt := v.v.Interface().(type) {
		case protoreflect.List:
			return opts.scanList(path, v.fd, vt)
		case protoreflect.Message:
			return opts.rangeMessage(path, vt)
		default:
			// a scalar value
			if len(path) > 0 {
				// can't path into a scalar
				return emptyValues
			}
			return func(yield func(value) bool) {
				yield(v)
			}
		}
	default:
		if v.fd.Message() == nil {
			// a scalar value
			if len(path) > 0 {
				// can't path into a scalar
				return emptyValues
			}
			return func(yield func(value) bool) {
				yield(v)
			}
		}
		return opts.rangeMessage(path, v.v.Message())
	}
}

// scanMessage returns all values identified by path in msg.
// The first entry of path should be a field name in msg, otherwise an empty iterator is returned.
func (opts rangeValuesOptions) scanMessage(path []pathSegment, msg protoreflect.Message) iter.Seq[value] {
	if len(path) == 0 {
		return emptyValues // callers should handle empty paths
	}
	head := path[0]
	if head.IsIndex {
		return emptyValues // path doesn't match the message structure
	}

	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(head.Name))
	if fd == nil {
		return emptyValues // field doesn't exist in the message
	}

	v := msg.Get(fd)
	if v.Equal(fd.Default()) {
		return emptyValues // field exists but has no value
	}

	switch {
	case fd.IsMap():
		return opts.scanMap(path[1:], fd, v.Map())
	case fd.IsList():
		return opts.scanList(path[1:], fd, v.List())
	case fd.Message() != nil:
		vm := v.Message()
		if !vm.IsValid() {
			return emptyValues // field exists but has no value
		}
		if isLeaf(fd) && len(path) > 1 {
			return emptyValues // path has more segments, but the value is a value
		}
		if len(path) == 1 {
			// end of the path, return the value
			return func(yield func(value) bool) {
				yield(value{fd, v})
			}
		}
		return opts.scanMessage(path[1:], vm)
	case isLeaf(fd) && len(path) == 1:
		return func(yield func(value) bool) {
			yield(value{fd, v})
		}
	default:
		return emptyValues // the field value is of an unknown type

	}
}

// scanMap returns all values identified by path in the map m.
// If path is empty, all values in the map are returned.
func (opts rangeValuesOptions) scanMap(path []pathSegment, fd protoreflect.FieldDescriptor, m protoreflect.Map) iter.Seq[value] {
	if len(path) == 0 {
		return func(yield func(value) bool) {
			if !opts.Stable {
				m.Range(func(key protoreflect.MapKey, v protoreflect.Value) bool {
					if !yield(value{fd.MapValue(), v}) {
						return false
					}
					return true
				})
				return
			}

			// need stable order, capture and sort the entries
			type entry struct {
				k protoreflect.MapKey
				v protoreflect.Value
			}
			entries := make([]entry, 0, m.Len())
			m.Range(func(key protoreflect.MapKey, v protoreflect.Value) bool {
				entries = append(entries, entry{key, v})
				return true
			})
			// same logic that exists in the protorange package.
			// sorts false before true, numeric keys in ascending order,
			// and strings in lexicographical ordering according to UTF-8 codepoints.
			sort.Slice(entries, func(xe, ye int) bool {
				x := entries[xe].k.Value()
				y := entries[ye].k.Value()
				switch x.Interface().(type) {
				case bool:
					return !x.Bool() && y.Bool()
				case int32, int64:
					return x.Int() < y.Int()
				case uint32, uint64:
					return x.Uint() < y.Uint()
				case string:
					return x.String() < y.String()
				default:
					panic("invalid map key type")
				}
			})
			for _, e := range entries {
				if !yield(value{fd.MapValue(), e.v}) {
					return
				}
			}
		}
	}

	head := path[0]
	if head.IsIndex {
		return emptyValues // path expects a list, got a map
	}

	k, ok := parseMapKey(head.Name, fd.MapKey())
	if !ok {
		return emptyValues // invalid map key
	}
	v := m.Get(k)
	if !v.IsValid() || v.Equal(fd.MapValue().Default()) {
		return emptyValues // key doesn't exist in the map
	}

	if len(path) == 1 {
		// end of the path
		return func(yield func(value) bool) {
			yield(value{fd.MapValue(), v})
		}
	}

	// more to the path
	if isLeaf(fd.MapValue()) {
		return emptyValues // path has more segments, but the map value is a value
	}
	if md := fd.MapValue().Message(); md != nil {
		return opts.scanMessage(path[1:], v.Message())
	}
	return emptyValues // there's more to the path but the value has no properties
}

// scanList returns all values identified by path in the list l.
// If path is empty, all values in the list are returned.
func (opts rangeValuesOptions) scanList(path []pathSegment, fd protoreflect.FieldDescriptor, l protoreflect.List) iter.Seq[value] {
	if len(path) == 0 {
		return func(yield func(value) bool) {
			for i := 0; i < l.Len(); i++ {
				if !yield(value{fd, l.Get(i)}) {
					return
				}
			}
		}
	}

	head := path[0]
	if head.IsIndex {
		i := head.Index
		if i < 0 {
			// negative index means counting from the end
			i = l.Len() + i
		}
		if i < 0 || i >= l.Len() {
			return emptyValues // index out of range
		}
		v := l.Get(i)
		if isLeaf(fd) && len(path) > 1 {
			return emptyValues // path has more segments, but the list contains values
		}
		if len(path) == 1 {
			// end of the path, return the value
			return func(yield func(value) bool) {
				yield(value{fd, v})
			}
		}

		if fd.Message() == nil {
			return emptyValues // it's not a value, but also not a message. Not sure what it is.
		}

		return opts.scanMessage(path[1:], v.Message())
	}

	// there's more to the path, but this is a list of leafs which can't have more path segments
	if isLeaf(fd) {
		return emptyValues // path has more segments, but the list contains value values
	}

	return func(yield func(value) bool) {
		for i := 0; i < l.Len(); i++ {
			v := l.Get(i)
			for leaf := range opts.scanMessage(path, v.Message()) {
				if !yield(leaf) {
					return
				}
			}
		}
	}
}

// getLeafDescriptor returns the most relevant FieldDescriptor for values, and whether values is a value or not.
func getLeafDescriptor(values protopath.Values) (protoreflect.FieldDescriptor, bool) {
	// getFd returns the most useful FieldDescriptor for values at index i.
	getFd := func(i int) protoreflect.FieldDescriptor {
		step := values.Index(i).Step
		switch step.Kind() {
		case protopath.FieldAccessStep:
			return step.FieldDescriptor()
		case protopath.MapIndexStep:
			return values.Index(i - 1).Step.FieldDescriptor().MapValue()
		case protopath.ListIndexStep:
			return values.Index(i - 1).Step.FieldDescriptor()
		default:
			return nil // unsupported step kind
		}
	}

	fd := getFd(-1)
	if fd == nil {
		return nil, false
	}

	switch values.Index(-1).Step.Kind() {
	case protopath.FieldAccessStep:
		// check the parent isn't a value
		if parentFd := getFd(-2); parentFd != nil && isLeaf(parentFd) {
			return nil, false
		}
		fd := getFd(-1)
		return fd, isLeaf(fd) && !fd.IsList()
	case protopath.MapIndexStep:
		fd := getFd(-1)
		return fd, isLeaf(fd)
	case protopath.ListIndexStep:
		fd := getFd(-1)
		return fd, isLeaf(fd)
	default:
		return nil, false
	}
}

// isLeaf returns true if fd is a value.
// A value is a scalar value, or a well known value type.
func isLeaf(fd protoreflect.FieldDescriptor) bool {
	if fd == nil {
		return false
	}
	if msg := fd.Message(); msg != nil {
		switch msg.FullName() {
		case "google.protobuf.Timestamp",
			"google.protobuf.Duration":
			return true // treat special messages as leafs
		default:
			return false
		}
	}
	return true // all other types are scalars, aka leafs
}

// toString converts the value into a string ready for comparison to another string.
// Unlike l.v.String() this converts enum values to their enum name where available,
// otherwise converts them to a string representation of the enum number.
// Bytes are converted to string.
func (v value) toString() (string, bool) {
	if v.fd == nil {
		return "", false
	}
	switch v.fd.Kind() {
	case protoreflect.BoolKind:
		return strconv.FormatBool(v.v.Bool()), true
	case protoreflect.FloatKind:
		return strconv.FormatFloat(v.v.Float(), 'f', -1, 32), true
	case protoreflect.DoubleKind:
		return strconv.FormatFloat(v.v.Float(), 'f', -1, 64), true
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(v.v.Int(), 10), true
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(v.v.Uint(), 10), true
	case protoreflect.StringKind:
		return v.v.String(), true
	case protoreflect.EnumKind:
		enum := v.v.Enum()
		enumDesc := v.fd.Enum().Values().ByNumber(enum)
		if enumDesc == nil {
			// unknown enum
			return strconv.FormatInt(int64(enum), 10), true
		}
		return string(enumDesc.Name()), true
	case protoreflect.BytesKind:
		return string(v.v.Bytes()), true
	default:
		// for value messages, we rely on the json representation to turn them into strings.
		if msg := v.v.Message(); msg != nil {
			bs, err := protojson.Marshal(msg.Interface())
			if err != nil {
				// if we can't marshal the message, we can't convert it to a string
				return "", false
			}
			bs = bytes.Trim(bs, `"`) // remove quotes around the string
			return string(bs), true
		}
		// unsupported kinds
		return "", false
	}
}

// toTimestamp converts the value into a *timestamppb.Timestamp if it is a valid timestamp.
func (v value) toTimestamp() (*timestamppb.Timestamp, bool) {
	if v.fd == nil {
		return nil, false
	}
	if v.fd.Message() == nil || v.fd.Message().FullName() != "google.protobuf.Timestamp" {
		return nil, false
	}
	tm := v.v.Message()
	if !tm.IsValid() {
		return nil, false
	}
	t, ok := tm.Interface().(*timestamppb.Timestamp)
	if !ok {
		return nil, false
	}

	return t, true
}

// parseMapKey converts keyStr into a protoreflect.MapKey using fd to choose the correct conversion method to use.
func parseMapKey(keyStr string, fd protoreflect.FieldDescriptor) (protoreflect.MapKey, bool) {
	fail := func() (protoreflect.MapKey, bool) {
		return protoreflect.MapKey{}, false
	}
	switch fd.Kind() {
	case protoreflect.BoolKind:
		gt, err := strconv.ParseBool(keyStr)
		if err != nil {
			return fail()
		}
		return protoreflect.ValueOfBool(gt).MapKey(), true
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		gt, err := strconv.ParseInt(keyStr, 10, 32)
		if err != nil {
			return fail()
		}
		return protoreflect.ValueOfInt32(int32(gt)).MapKey(), true
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		gt, err := strconv.ParseInt(keyStr, 10, 64)
		if err != nil {
			return fail()
		}
		return protoreflect.ValueOfInt64(gt).MapKey(), true
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		gt, err := strconv.ParseUint(keyStr, 10, 32)
		if err != nil {
			return fail()
		}
		return protoreflect.ValueOfUint32(uint32(gt)).MapKey(), true
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		gt, err := strconv.ParseUint(keyStr, 10, 64)
		if err != nil {
			return fail()
		}
		return protoreflect.ValueOfUint64(gt).MapKey(), true
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(keyStr).MapKey(), true
	default:
		// unknown map key type!
		return fail()
	}
}
