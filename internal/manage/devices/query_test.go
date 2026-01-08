package devices

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/manage/devices/testdata/testproto/querypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func Test_getMessageString(t *testing.T) {
	tests := []struct {
		name string
		path string
		msg  proto.Message
		want []string
	}{
		{"nil msg", "foo.bar", nil, nil},
		{"root string", "name", &traits.Metadata{Name: "foo"}, []string{"foo"}},
		{"root string absent", "name", &traits.Metadata{}, []string{}},
		{"root map", "more.val", &traits.Metadata{More: map[string]string{"val": "foo"}}, []string{"foo"}},
		{"root map nil", "more.val", &traits.Metadata{}, []string{}},
		{"root map absent", "more.val", &traits.Metadata{More: map[string]string{}}, []string{}},
		{"nested string", "id.bacnet", &traits.Metadata{Id: &traits.Metadata_ID{Bacnet: "1234"}}, []string{"1234"}},
		{"nested string absent prop", "id.bacnet", &traits.Metadata{Id: &traits.Metadata_ID{}}, []string{}},
		{"nested string absent message", "id.bacnet", &traits.Metadata{}, []string{}},
		{"nested map", "id.more.foo", &traits.Metadata{Id: &traits.Metadata_ID{More: map[string]string{"foo": "1234"}}}, []string{"1234"}},
		{"trailing .", "id.", &traits.Metadata{Id: &traits.Metadata_ID{More: map[string]string{"foo": "1234"}}}, []string{}},
		{"leading .", ".id", &traits.Metadata{Id: &traits.Metadata_ID{More: map[string]string{"foo": "1234"}}}, []string{}},
		{"property of scalar", "name.foo", &traits.Metadata{Name: "1234"}, []string{}},
		{"match all (one) in array", "traits.name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "foo"}}}, []string{"foo"}},
		{"match all in array", "traits.name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "foo"}, {Name: "bar"}}}, []string{"foo", "bar"}},
		{"match in array with Index", "traits[0].name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "foo"}, {Name: "bar"}}}, []string{"foo"}},
		{"match in array with Index", "traits[1].name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "foo"}, {Name: "bar"}}}, []string{"bar"}},
		{"match in array doesn't exist", "traits[1].name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "foo"}}}, []string{}},
		{"match in array negative", "traits[-1].name", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "bar"}, {Name: "foo"}}}, []string{"foo"}},
		{"match nested in array", "traits.more.units", &traits.Metadata{Traits: []*traits.TraitMetadata{{More: map[string]string{"units": "dogs"}}, {More: map[string]string{"units": "cats"}}}}, []string{"dogs", "cats"}},
		{"match all in array with primitive", "dns", &traits.Metadata_NIC{Dns: []string{"foo", "bar"}}, []string{"foo", "bar"}},
		{"match in array with primitive[0]", "dns[0]", &traits.Metadata_NIC{Dns: []string{"foo", "bar"}}, []string{"foo"}},
		{"match in array with primitive[1]", "dns[1]", &traits.Metadata_NIC{Dns: []string{"foo", "bar"}}, []string{"bar"}},
		{"match in array with primitive[-1]", "dns[-1]", &traits.Metadata_NIC{Dns: []string{"foo", "bar"}}, []string{"bar"}},
		{"match in array with primitive[-2]", "dns[-2]", &traits.Metadata_NIC{Dns: []string{"foo", "bar"}}, []string{"foo"}},
		{"match nested in array with wrong Index", "traits[0].more.units", &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "bar"}, {Name: "foo", More: map[string]string{"units": "cats"}}}}, []string{}},
	}
	cmpStr := func(a, b string) bool { return a < b }
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(getMessageString(tt.path, tt.msg))
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty(), cmpopts.SortSlices(cmpStr)); diff != "" {
				t.Errorf("getMessageString() -want +got:\n%s", diff)
			}
		})
	}
}

func Example_isMessageValueStringFunc() {
	msg := &devicespb.Device{
		Name: "MyDevice",
		Metadata: &traits.Metadata{
			Membership: &traits.Metadata_Membership{
				Subsystem: "Lighting",
			},
		},
	}

	itr := getMessageString("metadata.membership.subsystem", msg)

	found := false

	itr(func(v string) bool {
		if v == "Lighting" {
			found = true
		}
		return found
	})

	fmt.Println(found)
	// Output: true
}

func Test_leaf_toString(t *testing.T) {
	fd := func(field string) protoreflect.FieldDescriptor {
		md := (&querypb.Result{}).ProtoReflect().Descriptor()
		return md.Fields().ByName(protoreflect.Name(field))
	}

	tests := []struct {
		leaf   value
		want   string
		wantOk bool
	}{
		{leaf: value{fd("double_val"), protoreflect.ValueOfFloat64(1.1)}, want: "1.1", wantOk: true},
		{leaf: value{fd("float_val"), protoreflect.ValueOfFloat32(2.1)}, want: "2.1", wantOk: true},
		{leaf: value{fd("int32_val"), protoreflect.ValueOfInt32(-10)}, want: "-10", wantOk: true},
		{leaf: value{fd("int64_val"), protoreflect.ValueOfInt64(-20)}, want: "-20", wantOk: true},
		{leaf: value{fd("uint32_val"), protoreflect.ValueOfUint32(30)}, want: "30", wantOk: true},
		{leaf: value{fd("uint64_val"), protoreflect.ValueOfUint64(40)}, want: "40", wantOk: true},
		{leaf: value{fd("sint32_val"), protoreflect.ValueOfInt32(-50)}, want: "-50", wantOk: true},
		{leaf: value{fd("sint64_val"), protoreflect.ValueOfInt64(-60)}, want: "-60", wantOk: true},
		{leaf: value{fd("fixed32_val"), protoreflect.ValueOfUint32(70)}, want: "70", wantOk: true},
		{leaf: value{fd("fixed64_val"), protoreflect.ValueOfUint64(80)}, want: "80", wantOk: true},
		{leaf: value{fd("sfixed32_val"), protoreflect.ValueOfInt32(-90)}, want: "-90", wantOk: true},
		{leaf: value{fd("sfixed64_val"), protoreflect.ValueOfInt64(-100)}, want: "-100", wantOk: true},
		{leaf: value{fd("bool_val"), protoreflect.ValueOfBool(true)}, want: "true", wantOk: true},
		{leaf: value{fd("enum_val"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_A.Number())}, want: "RESULT_ENUM_A", wantOk: true},
		{leaf: value{fd("duration_val"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 10).ProtoReflect())}, want: "10s", wantOk: true},
		{leaf: value{fd("timestamp_val"), protoreflect.ValueOfMessage(timestamppb.New(time.Date(2025, 8, 10, 12, 5, 40, 111, time.UTC)).ProtoReflect())}, want: "2025-08-10T12:05:40.000000111Z", wantOk: true},
	}
	for _, tt := range tests {
		t.Run(string(tt.leaf.fd.Name()), func(t *testing.T) {
			gotS, gotOk := tt.leaf.toString()
			if gotS != tt.want || gotOk != tt.wantOk {
				t.Errorf("value.toString() = %q, %v; want %q, %v", gotS, gotOk, tt.want, tt.wantOk)
			}
		})
	}
}

func TestRangeValuesOptions_RangeMessage(t *testing.T) {
	// m is the data structure we test against
	m := testData()

	// leafPaths are all leaf paths in m
	leafPaths := testDataLeafPaths()
	leafValues := testDataLeafValues()
	if len(leafValues) != len(leafPaths) {
		t.Fatalf("expected leafValues and leafPaths to have the same length, got %d and %d", len(leafValues), len(leafPaths))
	}

	type testCase struct {
		path     string
		wantVals []value
		wantErr  bool
	}
	tests := []testCase{
		// no path means all leafPaths
		{wantVals: leafValues},
	}

	// newLeafCase returns a testCase matching path against indexes in leafValues.
	newLeafCase := func(path string, wantIndexes ...int) testCase {
		t.Helper()
		if len(wantIndexes) == 0 {
			t.Fatalf("wantIndexes must not be empty for path %q", path)
		}
		wantLeafs := make([]value, len(wantIndexes))
		for i, idx := range wantIndexes {
			if idx < 0 || idx >= len(leafValues) {
				t.Fatalf("index %d out of range for path %q", idx, path)
			}
			wantLeafs[i] = leafValues[idx]
		}
		return testCase{
			path:     path,
			wantVals: wantLeafs,
		}
	}

	// each path should resolve to exactly the wanted value
	for i, path := range leafPaths {
		tests = append(tests, newLeafCase(path, i))
	}

	// repeated values should be optional in the path
	for i, scalarType := range scalarTypes {
		// in leafValues, each scalar type has the non-repeated value followed by two repeated values
		leafIdx := i*3 + 1
		tests = append(tests, newLeafCase(fmt.Sprintf("r_%s", scalarType), leafIdx, leafIdx+1))
	}

	// negative indexes should work for repeated values
	for i, scalarType := range scalarTypes {
		// in leafValues, each scalar type has the non-repeated value followed by two repeated values
		leafIdx := i*3 + 1
		tests = append(tests, newLeafCase(fmt.Sprintf("r_%s[-1]", scalarType), leafIdx+1))
		tests = append(tests, newLeafCase(fmt.Sprintf("r_%s[-2]", scalarType), leafIdx))
	}

	newValueCase := func(path string, fd protoreflect.FieldDescriptor, vals ...any) testCase {
		t.Helper()
		tt := testCase{path: path}
		for _, val := range vals {
			var v protoreflect.Value
			switch val := val.(type) {
			case string:
				v = protoreflect.ValueOfString(val)
			case proto.Message:
				v = protoreflect.ValueOfMessage(val.ProtoReflect())
			default:
				t.Fatalf("invalid value type for %q: %T", path, val)
			}
			tt.wantVals = append(tt.wantVals, value{fd: fd, v: v})
		}
		return tt
	}

	// explicit paths can resolve to any value type
	tests = append(tests, []testCase{
		newValueCase("result", resultFd("result"), m.Result),
		newValueCase("m_string_result", resultMapFd("m_string_result"), m.MStringResult["a"], m.MStringResult["b"], m.MStringResult["c"]),
		newValueCase("m_string_result.a", resultMapFd("m_string_result"), m.MStringResult["a"]),
		newValueCase("m_string_result.a.result", resultFd("result"), m.MStringResult["a"].Result),
		newValueCase("m_string_string", resultMapFd("m_string_string"), m.MStringString["a"], m.MStringString["b"]),
		newValueCase("r_result", resultFd("r_result"), m.RResult[0], m.RResult[1], m.RResult[2]),
		newValueCase("r_result.result", resultFd("result"), m.RResult[0].Result, m.RResult[1].Result),
		newValueCase("r_result[0]", resultFd("r_result"), m.RResult[0]),
		newValueCase("r_result[0].result", resultFd("result"), m.RResult[0].Result),
	}...)

	// tests for when path doesn't match any value, return empty iterator
	for _, path := range []string{
		// value doesn't exist, or is default
		"r_result[2].bool_val",
		"r_result[2].int32_val",
		"r_result[2].string_val",
		"r_result[2].timestamp_val",
		"m_string_range.c.bool_val",
		"m_string_range.c.int32_val",
		"m_string_range.c.string_val",
		"m_string_range.c.timestamp_val",
		"m_string_range.c.result.bool_val",
		"m_string_range.c.result.int32_val",
		"m_string_range.c.result.string_val",
		"m_string_range.c.result.timestamp_val",
		// message fields that don't exist
		"not_found",
		"r_string.not_found",
		"r_string[0].not_found",
		"result.not_found",
		"r_result[0].not_found",
		"m_string_result.a.not_found",
		// fields of non-message types
		"double_val.not_found",
		"string_val.not_found",
		"result.string_val.not_found",
		// repeated field indexes that don't exist
		"r_string[2]",
		"r_string[-3]",
		"r_result[0].r_string[2]",
		"r_result[3].string_val",
		// map keys that don't exist
		"m_int32_string.42",
		"m_string_string.not_found",
		"range.m_string_string.not_found",
		"r_range[0].m_string_string.not_found",
		"m_string_range.a.m_string_string.not_found",
		"m_string_range.not_found.string_val",
		// attempting to access a non-repeating field by index
		"string_val[0]",
		"range[0].string_val",
		"m_string_string[0]",
		"m_string_result[0]",
		// fields of value messages
		"timestamp_val.seconds",
		"r_timestamp[0].seconds",
		"m_string_timestamp.a.seconds",
		// hidden fields of map entries
		"m_int32_string.key",
		"m_int32_string.value",
		// bad map keys
		"m_int32_string.not_int",
		"m_int64_string.not_int",
		"m_uint32_string.not_uint",
		"m_uint64_string.not_uint",
		"m_sint32_string.not_sint",
		"m_sint64_string.not_sint",
		"m_fixed32_string.not_fixed",
		"m_fixed64_string.not_fixed",
		"m_sfixed32_string.not_sfixed",
		"m_sfixed64_string.not_sfixed",
		"m_bool_string.not_bool",
		// attempting to treat maps as repeating fields (non-leaf)
		"m_string_result.result",
	} {
		tests = append(tests, testCase{path: path})
	}

	// invalid paths return an error, most cases are covered by the parsePath tests
	tests = append(tests, testCase{path: ".", wantErr: true})

	for _, tt := range tests {
		name := tt.path
		if name == "" {
			name = "all paths"
		}
		t.Run(name, func(t *testing.T) {
			got, err := rangeValuesOptions{Stable: true}.RangeMessage(tt.path, m)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("rangeMessage() error = %v, wantErr %v. Got %v", err, tt.wantErr, slices.Collect(got))
				}
				return
			}
			var wantFds []protoreflect.FieldDescriptor
			var wantValues []protoreflect.Value
			for _, l := range tt.wantVals {
				wantFds = append(wantFds, l.fd)
				wantValues = append(wantValues, l.v)
			}

			var gotFds []protoreflect.FieldDescriptor
			var gotValues []protoreflect.Value
			for l := range got {
				gotFds = append(gotFds, l.fd)
				gotValues = append(gotValues, l.v)
			}
			if diff := cmp.Diff(wantFds, gotFds, transformReflectFieldDescriptor, protocmp.Transform()); diff != "" {
				t.Errorf("rangeMessage() field descriptors (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(wantValues, gotValues, transformReflectValue, protocmp.Transform()); diff != "" {
				t.Errorf("rangeMessage() values (-want,+got):\n%s", diff)
			}
		})
	}
}

var transformReflectFieldDescriptor = cmp.Transformer("", func(fd protoreflect.FieldDescriptor) any {
	// we only care about the name and number of the field descriptor
	return fmt.Sprintf("%s:%d", fd.FullName(), fd.Number())
})

// copied from the protorange package
var transformReflectValue = cmp.Transformer("", func(v protoreflect.Value) any {
	switch v := v.Interface().(type) {
	case protoreflect.Message:
		return v.Interface()
	case protoreflect.Map:
		ms := map[any]protoreflect.Value{}
		v.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			ms[k.Interface()] = v
			return true
		})
		return ms
	case protoreflect.List:
		ls := make([]protoreflect.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			ls = append(ls, v.Get(i))
		}
		return ls
	default:
		return v
	}
})

func Test_conditionToCmpFunc(t *testing.T) {
	// helper for making StringList values
	sl := func(s ...string) *devicespb.Device_Query_StringList {
		return &devicespb.Device_Query_StringList{Strings: s}
	}

	t.Run("nil", func(t *testing.T) {
		cmpFunc := conditionToCmpFunc(&devicespb.Device_Query_Condition{})
		if cmpFunc == nil {
			t.Errorf("expected nil condition value to return non-nil cmpFunc, got %T", cmpFunc)
		}
		got := cmpFunc(value{})
		if got {
			t.Errorf("expected nil condition value to return false for any value, got true")
		}
	})

	condTestName := func(cond *devicespb.Device_Query_Condition) string {
		name := fmt.Sprintf("%T", cond.Value)
		name = strings.TrimPrefix(name, "*devicespb.Device_Query_Condition_")
		return name
	}

	t.Run("strings", func(t *testing.T) {
		tests := []struct {
			cond               *devicespb.Device_Query_Condition
			positive, negative []string
		}{
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "foo"}},
				positive: []string{"foo"},
				negative: []string{"", "bar", "FOO", "foO", "fooo", "-foo", "10"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringEqualFold{StringEqualFold: "fOo"}},
				positive: []string{"fOo", "FOO", "foo"},
				negative: []string{"", "bar", "fOoO", "-fOo", "10"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}},
				positive: []string{"apple pie", "apple", "pineapple", "apple123"},
				negative: []string{"", "pple", "appl", "Apple"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringContainsFold{StringContainsFold: "aPPle"}},
				positive: []string{"aPPle pie", "apple", "pineappLE", "aPPle123"},
				negative: []string{"", "pple", "appl", "banana"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringIn{StringIn: sl("foo", "BAR")}},
				positive: []string{"foo", "BAR"},
				negative: []string{"", "FOO", "bar", "foO", "fooo", "-foo", "10"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_StringInFold{StringInFold: sl("foo", "BAR")}},
				positive: []string{"foo", "BAR", "FOO", "bar", "foO"},
				negative: []string{"", "fooo", "-foo", "10"},
			},
		}

		strLeaf := func(val string) value {
			md := (&querypb.Result{}).ProtoReflect().Descriptor()
			return value{
				fd: md.Fields().ByName("string_val"),
				v:  protoreflect.ValueOfString(val),
			}
		}

		for _, tt := range tests {
			t.Run(condTestName(tt.cond), func(t *testing.T) {
				cmpFunc := conditionToCmpFunc(tt.cond)
				for _, str := range tt.positive {
					if !cmpFunc(strLeaf(str)) {
						t.Errorf("expected %q to match condition %s", str, tt.cond)
					}
				}
				for _, str := range tt.negative {
					if cmpFunc(strLeaf(str)) {
						t.Errorf("expected %q to not match condition %s", str, tt.cond)
					}
				}
			})
		}
	})

	t.Run("timestamps", func(t *testing.T) {
		now := time.Date(2025, 8, 10, 12, 5, 40, 111, time.UTC)
		var zero time.Time // zero time is used to test empty timestamps
		tests := []struct {
			cond               *devicespb.Device_Query_Condition
			positive, negative []time.Time
		}{
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampEqual{TimestampEqual: timestamppb.New(now)}},
				positive: []time.Time{now},
				negative: []time.Time{zero, now.Add(1), now.Add(-1)},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampLt{TimestampLt: timestamppb.New(now)}},
				positive: []time.Time{now.Add(-1), now.Add(-10 * time.Second)},
				negative: []time.Time{zero, now, now.Add(1), now.Add(10 * time.Second)},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampLte{TimestampLte: timestamppb.New(now)}},
				positive: []time.Time{now, now.Add(-1), now.Add(-10 * time.Second)},
				negative: []time.Time{zero, now.Add(1), now.Add(10 * time.Second)},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampGt{TimestampGt: timestamppb.New(now)}},
				positive: []time.Time{now.Add(1), now.Add(10 * time.Second)},
				negative: []time.Time{zero, now, now.Add(-1), now.Add(-10 * time.Second)},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampGte{TimestampGte: timestamppb.New(now)}},
				positive: []time.Time{now, now.Add(1), now.Add(10 * time.Second)},
				negative: []time.Time{zero, now.Add(-1), now.Add(-10 * time.Second)},
			},
		}

		timestampLeaf := func(val time.Time) value {
			md := (&querypb.Result{}).ProtoReflect().Descriptor()
			return value{
				fd: md.Fields().ByName("timestamp_val"),
				v:  protoreflect.ValueOfMessage(timestamppb.New(val).ProtoReflect()),
			}
		}

		t.Run("not a timestamp", func(t *testing.T) {
			l := value{
				fd: (&querypb.Result{}).ProtoReflect().Descriptor().Fields().ByName("string_val"),
				v:  protoreflect.ValueOfString("not a timestamp"),
			}
			cmpFunc := conditionToCmpFunc(&devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_TimestampEqual{TimestampEqual: timestamppb.New(now)}})
			if cmpFunc(l) {
				t.Errorf("expected condition to not match non-timestamp value, got true")
			}
		})

		for _, tt := range tests {
			t.Run(condTestName(tt.cond), func(t *testing.T) {
				cmpFunc := conditionToCmpFunc(tt.cond)
				for _, ts := range tt.positive {
					if !cmpFunc(timestampLeaf(ts)) {
						t.Errorf("expected %v to match condition %s", ts, tt.cond)
					}
				}
				for _, ts := range tt.negative {
					if cmpFunc(timestampLeaf(ts)) {
						t.Errorf("expected %v to not match condition %s", ts, tt.cond)
					}
				}
			})
		}
	})

	t.Run("names", func(t *testing.T) {
		tests := []struct {
			cond     *devicespb.Device_Query_Condition
			positive []string
			negative []string
		}{
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_NameDescendant{NameDescendant: "a/b"}},
				positive: []string{"a/b/c", "a/b/c/d"},
				negative: []string{"", "a", "a/", "a/b", "a/bc", "a/b/", "x/b/c"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_NameDescendantInc{NameDescendantInc: "a/b"}},
				positive: []string{"a/b", "a/b/c", "a/b/c/d"},
				negative: []string{"", "a", "a/", "a/bc", "a/b/", "x/b/c"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_NameDescendantIn{NameDescendantIn: sl("a/b", "1/2")}},
				positive: []string{"a/b/c", "a/b/c/d", "1/2/3", "1/2/3/4"},
				negative: []string{"", "a", "a/", "a/b", "a/bc", "a/b/", "x/b/c", "1", "1/2"},
			},
			{
				cond:     &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_NameDescendantIncIn{NameDescendantIncIn: sl("a/b", "1/2")}},
				positive: []string{"a/b", "a/b/c", "a/b/c/d", "1/2", "1/2/3", "1/2/3/4"},
				negative: []string{"", "a", "a/", "a/bc", "a/b/", "x/b/c", "1", "2"},
			},
		}

		nameLeaf := func(val string) value {
			md := (&querypb.Result{}).ProtoReflect().Descriptor()
			return value{
				fd: md.Fields().ByName("string_val"),
				v:  protoreflect.ValueOfString(val),
			}
		}

		for _, tt := range tests {
			t.Run(condTestName(tt.cond), func(t *testing.T) {
				cmpFunc := conditionToCmpFunc(tt.cond)
				for _, str := range tt.positive {
					if !cmpFunc(nameLeaf(str)) {
						t.Errorf("expected %q to match condition %s", str, tt.cond)
					}
				}
				for _, str := range tt.negative {
					if cmpFunc(nameLeaf(str)) {
						t.Errorf("expected %q to not match condition %s", str, tt.cond)
					}
				}
			})
		}
	})

	t.Run("presence", func(t *testing.T) {
		mkValue := func(field string, val protoreflect.Value) value {
			md := (&querypb.Result{}).ProtoReflect().Descriptor()
			return value{
				fd: md.Fields().ByName(protoreflect.Name(field)),
				v:  val,
			}
		}
		tests := []struct {
			cond     *devicespb.Device_Query_Condition
			positive []value
			negative []value
		}{
			{
				cond: &devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_Present{Present: &emptypb.Empty{}}},
				positive: []value{
					mkValue("string_val", protoreflect.ValueOfString("a")),
					mkValue("r_string", protoreflect.ValueOfString("b")),
					mkValue("enum_val", protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_A.Number())),
				},
				negative: []value{
					mkValue("string_val", protoreflect.Value{}),
					mkValue("r_string", protoreflect.Value{}),
					mkValue("enum_val", protoreflect.Value{}),
				},
			},
		}
		for _, tt := range tests {
			t.Run(condTestName(tt.cond), func(t *testing.T) {
				cmpFunc := conditionToCmpFunc(tt.cond)
				for _, l := range tt.positive {
					if !cmpFunc(l) {
						t.Errorf("expected %v to match condition %s", l, tt.cond)
					}
				}
				for _, l := range tt.negative {
					if cmpFunc(l) {
						t.Errorf("expected %v to not match condition %s", l, tt.cond)
					}
				}
			})
		}
	})

	t.Run("matches", func(t *testing.T) {
		// Returns a value representing field prop of msg.
		newField := func(prop string, msg *querypb.Result) value {
			fd := resultFd(prop)
			return value{
				fd: fd,
				v:  msg.ProtoReflect().Get(fd),
			}
		}
		// Returns a value representing the r_result field containing msg as the sole element.
		newVal := func(msg *querypb.Result) value {
			return newField("r_result", &querypb.Result{RResult: []*querypb.Result{msg}})
		}

		tests := []struct {
			name     string
			conds    []*devicespb.Device_Query_Condition
			positive []value
			negative []value
		}{
			{
				name:  "no conds",
				conds: nil,
				positive: []value{
					{fd: resultFd("r_result"), v: protoreflect.ValueOfMessage((&querypb.Result{}).ProtoReflect())},
				},
			},
			{
				name:  "match any",
				conds: []*devicespb.Device_Query_Condition{{Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}}},
				positive: []value{
					newVal(&querypb.Result{StringVal: "apple"}),
					newVal(&querypb.Result{RString: []string{"apple"}}),
					newVal(&querypb.Result{Result: &querypb.Result{StringVal: "apple"}}),
					newField("string_val", &querypb.Result{StringVal: "apple"}),
					newField("r_string", &querypb.Result{RString: []string{"apple"}}),
					newField("result", &querypb.Result{Result: &querypb.Result{StringVal: "apple"}}),
				},
				negative: []value{
					newVal(&querypb.Result{}),
					newVal(&querypb.Result{StringVal: "banana"}),
					newVal(&querypb.Result{StringVal: ""}),
					newField("string_val", &querypb.Result{StringVal: "banana"}),
					newField("r_string", &querypb.Result{RString: []string{"banana"}}),
					newField("result", &querypb.Result{Result: &querypb.Result{StringVal: "banana"}}),
				},
			},
			{
				name:  "match field",
				conds: []*devicespb.Device_Query_Condition{{Field: "string_val", Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}}},
				positive: []value{
					newVal(&querypb.Result{StringVal: "apple"}),
				},
				negative: []value{
					newVal(&querypb.Result{}),
					newVal(&querypb.Result{StringVal: "banana"}),
					newVal(&querypb.Result{StringVal: ""}),
					newVal(&querypb.Result{RString: []string{"apple"}}),
					newVal(&querypb.Result{Result: &querypb.Result{StringVal: "apple"}}),
				},
			},
			{
				name: "match multiple any",
				conds: []*devicespb.Device_Query_Condition{
					{Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}},
					{Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "pie"}},
				},
				positive: []value{
					newVal(&querypb.Result{StringVal: "apple pie"}),
					newVal(&querypb.Result{StringVal: "apple", RString: []string{"pie"}}),
					newVal(&querypb.Result{RString: []string{"apple", "pie"}}),
					newVal(&querypb.Result{StringVal: "apple", Result: &querypb.Result{StringVal: "pie"}}),
					newField("string_val", &querypb.Result{StringVal: "apple pie"}),
					newField("r_string", &querypb.Result{RString: []string{"apple", "pie"}}),
					newField("m_string_string", &querypb.Result{MStringString: map[string]string{"kind": "apple pie"}}),
				},
				negative: []value{
					newVal(&querypb.Result{}),
					newVal(&querypb.Result{StringVal: "banana"}),
					newVal(&querypb.Result{StringVal: ""}),
					newVal(&querypb.Result{StringVal: "apple"}),
					newVal(&querypb.Result{StringVal: "pie"}),
					newVal(&querypb.Result{RString: []string{"apple"}}),
					newVal(&querypb.Result{Result: &querypb.Result{StringVal: "apple"}}),
				},
			},
			{
				name: "match multiple fields",
				conds: []*devicespb.Device_Query_Condition{
					{Field: "string_val", Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}},
					{Field: "r_string", Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "pie"}},
				},
				positive: []value{
					newVal(&querypb.Result{StringVal: "apple", RString: []string{"pie"}}),
					newVal(&querypb.Result{StringVal: "apple tart", RString: []string{"pie"}}),
					newVal(&querypb.Result{StringVal: "apple", RString: []string{"mince", "pie"}}),
				},
				negative: []value{
					newVal(&querypb.Result{}),
					newVal(&querypb.Result{StringVal: "banana"}),
					newVal(&querypb.Result{StringVal: ""}),
					newVal(&querypb.Result{StringVal: "apple"}),
					newVal(&querypb.Result{StringVal: "pie"}),
					newVal(&querypb.Result{RString: []string{"apple"}}),
					newVal(&querypb.Result{Result: &querypb.Result{StringVal: "apple"}}),
					newVal(&querypb.Result{StringVal: "apple pie"}),
					newVal(&querypb.Result{RString: []string{"apple", "pie"}}),
					newVal(&querypb.Result{StringVal: "apple", Result: &querypb.Result{StringVal: "pie"}}),
				},
			},
			{
				name: "mixed fields and any",
				conds: []*devicespb.Device_Query_Condition{
					{Field: "string_val", Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "apple"}},
					{Value: &devicespb.Device_Query_Condition_StringContains{StringContains: "pie"}},
				},
				positive: []value{
					newVal(&querypb.Result{StringVal: "apple", RString: []string{"pie"}}),
					newVal(&querypb.Result{StringVal: "apple", MStringString: map[string]string{"kind": "pie"}}),
				},
				negative: []value{
					newVal(&querypb.Result{}),
					newVal(&querypb.Result{RString: []string{"apple", "pie"}}),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmpFunc := conditionToCmpFunc(&devicespb.Device_Query_Condition{Value: &devicespb.Device_Query_Condition_Matches{Matches: &devicespb.Device_Query{Conditions: tt.conds}}})
				for i, v := range tt.positive {
					if !cmpFunc(v) {
						t.Errorf("[%d] expected %+v to match conditions %+v", i, v, tt.conds)
					}
				}
				for i, v := range tt.negative {
					if cmpFunc(v) {
						t.Errorf("[%d] expected %+v to not match conditions %+v", i, v, tt.conds)
					}
				}
			})
		}
	})
}

func Example_healthyDevices() {
	// all checks for this device are normal
	device01 := &devicespb.Device{
		Name: "Device01",
		Metadata: &traits.Metadata{Location: &traits.Metadata_Location{
			Floor: "Ground Floor",
		}},
		HealthChecks: []*healthpb.HealthCheck{
			{Id: "Online", Normality: healthpb.HealthCheck_NORMAL},
			{Id: "Temperature", Normality: healthpb.HealthCheck_NORMAL},
		},
	}
	// some checks for this device are normal, some are abnormal
	device02 := &devicespb.Device{
		Name: "Device02",
		Metadata: &traits.Metadata{Location: &traits.Metadata_Location{
			Floor: "Floor 1",
		}},
		HealthChecks: []*healthpb.HealthCheck{
			{Id: "Online", Normality: healthpb.HealthCheck_NORMAL},
			{Id: "Temperature", Normality: healthpb.HealthCheck_HIGH},
		},
	}
	// all checks for this device are abnormal
	device03 := &devicespb.Device{
		Name: "Device03",
		Metadata: &traits.Metadata{Location: &traits.Metadata_Location{
			Floor: "Floor 2",
		}},
		HealthChecks: []*healthpb.HealthCheck{
			{Id: "Online", Normality: healthpb.HealthCheck_ABNORMAL},
			{Id: "Temperature", Normality: healthpb.HealthCheck_LOW},
		},
	}

	// a device is only normal if ALL health checks are normal
	normalDevicesQuery := &devicespb.Device_Query{
		Conditions: []*devicespb.Device_Query_Condition{
			{
				Field:   "health_checks.normality",
				Matcher: devicespb.Device_Query_Condition_ALL,
				Value:   &devicespb.Device_Query_Condition_StringEqual{StringEqual: "NORMAL"},
			},
		},
	}
	fmt.Println("Healthy devices:")
	for _, device := range []*devicespb.Device{device01, device02, device03} {
		fmt.Printf("  %s matches: %v\n", device.Name, deviceMatchesQuery(normalDevicesQuery, device))
	}

	// a device is abnormal if ANY health check is abnormal
	abnormalDevicesQuery := &devicespb.Device_Query{
		Conditions: []*devicespb.Device_Query_Condition{
			{
				Field:   "health_checks.normality",
				Matcher: devicespb.Device_Query_Condition_ANY,
				Value: &devicespb.Device_Query_Condition_StringIn{StringIn: &devicespb.Device_Query_StringList{Strings: []string{
					"ABNORMAL", "HIGH", "LOW",
				}}},
			},
		},
	}
	fmt.Println("Unhealthy devices:")
	for _, device := range []*devicespb.Device{device01, device02, device03} {
		fmt.Printf("  %s matches: %v\n", device.Name, deviceMatchesQuery(abnormalDevicesQuery, device))
	}

	// you can combine multiple conditions as needed
	abnormalFloor1DevicesQuery := &devicespb.Device_Query{
		Conditions: []*devicespb.Device_Query_Condition{
			{
				Field: "metadata.location.floor",
				Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Floor 1"},
			},
			{
				Field: "health_checks.normality",
				// Matcher defaults to ANY
				Value: &devicespb.Device_Query_Condition_StringIn{StringIn: &devicespb.Device_Query_StringList{Strings: []string{
					"ABNORMAL", "HIGH", "LOW",
				}}},
			},
		},
	}
	fmt.Println("Abnormal devices on Floor 1:")
	for _, device := range []*devicespb.Device{device01, device02, device03} {
		fmt.Printf("  %s matches: %v\n", device.Name, deviceMatchesQuery(abnormalFloor1DevicesQuery, device))
	}

	// Output:
	// Healthy devices:
	//   Device01 matches: true
	//   Device02 matches: false
	//   Device03 matches: false
	// Unhealthy devices:
	//   Device01 matches: false
	//   Device02 matches: true
	//   Device03 matches: true
	// Abnormal devices on Floor 1:
	//   Device01 matches: false
	//   Device02 matches: true
	//   Device03 matches: false
}
