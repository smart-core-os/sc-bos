package devices

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/manage/devices/testdata/testproto/querypb"
)

// testData returns a *Result with all fields populated.
// Nested result fields are populated to a depth of 2: result.result == something, result.result.result == nil.
// The third entry in result collections will be an empty message: r_result[2].* and m_string_result["c"].* are zero.
func testData() *querypb.Result {
	// newResult makes a fully populated Result.
	// Nested result fields are populated to a depth of d.
	var newResult func(d int) *querypb.Result
	newResult = func(d int) *querypb.Result {
		if d == 0 {
			return new(querypb.Result)
		}
		return &querypb.Result{
			DoubleVal:        1.1,
			RDouble:          []float64{1.2, 1.3},
			FloatVal:         2.1,
			RFloat:           []float32{2.2, 2.3},
			Int32Val:         -10,
			RInt32:           []int32{-11, -12},
			Int64Val:         -20,
			RInt64:           []int64{-21, -22},
			Uint32Val:        30,
			RUint32:          []uint32{31, 32},
			Uint64Val:        40,
			RUint64:          []uint64{41, 42},
			Sint32Val:        -50,
			RSint32:          []int32{-51, -52},
			Sint64Val:        -60,
			RSint64:          []int64{-61, -62},
			Fixed32Val:       70,
			RFixed32:         []uint32{71, 72},
			Fixed64Val:       80,
			RFixed64:         []uint64{81, 82},
			Sfixed32Val:      -90,
			RSfixed32:        []int32{-91, -92},
			Sfixed64Val:      -100,
			RSfixed64:        []int64{-101, -102},
			BoolVal:          true,
			RBool:            []bool{false, true},
			StringVal:        "a",
			RString:          []string{"b", "c"},
			BytesVal:         []byte("abc"),
			RBytes:           [][]byte{[]byte("def"), []byte("ghi")},
			EnumVal:          querypb.ResultEnum_RESULT_ENUM_A,
			REnum:            []querypb.ResultEnum{querypb.ResultEnum_RESULT_ENUM_B, querypb.ResultEnum_RESULT_ENUM_C},
			DurationVal:      durationpb.New(time.Second * 10),
			RDuration:        []*durationpb.Duration{durationpb.New(time.Second * 20), durationpb.New(time.Second * 30)},
			TimestampVal:     timestamppb.New(time.Unix(100, 10)),
			RTimestamp:       []*timestamppb.Timestamp{timestamppb.New(time.Unix(200, 20)), timestamppb.New(time.Unix(300, 30))},
			MInt32String:     map[int32]string{-11: "a", -10: "b"},
			MInt64String:     map[int64]string{-21: "a", -20: "b"},
			MUint32String:    map[uint32]string{30: "a", 31: "b"},
			MUint64String:    map[uint64]string{40: "a", 41: "b"},
			MSint32String:    map[int32]string{-51: "a", -50: "b"},
			MSint64String:    map[int64]string{-61: "a", -60: "b"},
			MFixed32String:   map[uint32]string{70: "a", 71: "b"},
			MFixed64String:   map[uint64]string{80: "a", 81: "b"},
			MSfixed32String:  map[int32]string{-91: "a", -90: "b"},
			MSfixed64String:  map[int64]string{-101: "a", -100: "b"},
			MBoolString:      map[bool]string{false: "a", true: "b"},
			MStringString:    map[string]string{"a": "a", "b": "b"},
			MStringEnum:      map[string]querypb.ResultEnum{"a": querypb.ResultEnum_RESULT_ENUM_A, "b": querypb.ResultEnum_RESULT_ENUM_B},
			MStringDuration:  map[string]*durationpb.Duration{"a": durationpb.New(time.Second * 10), "b": durationpb.New(time.Second * 20)},
			MStringTimestamp: map[string]*timestamppb.Timestamp{"a": timestamppb.New(time.Unix(100, 10)), "b": timestamppb.New(time.Unix(200, 20))},
			Result:           newResult(d - 1),
			RResult: []*querypb.Result{
				newResult(d - 1),
				newResult(d - 1),
				new(querypb.Result),
			},
			MStringResult: map[string]*querypb.Result{
				"a": newResult(d - 1),
				"b": newResult(d - 1),
				"c": new(querypb.Result),
			},
		}
	}
	return newResult(3) // Result.Result is filled, Result.Result.Result is empty
}

// testDataLeafPaths returns all leaf paths in the testData Result, including nested messages.
func testDataLeafPaths() []string {
	paths := resultPaths("", nil)
	paths = nestedResultPaths("", paths)
	for _, nestingPath := range nestedMessagePaths {
		paths = nestedResultPaths(nestingPath, paths)
	}
	return paths
}

// testDataLeafValues returns all expected leaf values in the testData Result, including nested messages.
func testDataLeafValues() []value {
	leafs := resultLeafs(nil)
	leafs = nestedResultLeafs(leafs)
	for range nestedMessagePaths {
		leafs = nestedResultLeafs(leafs)
	}
	return leafs
}

// scalarTypes is the list of all scalar types in Result.
var scalarTypes = []string{
	"double",
	"float",
	"int32",
	"int64",
	"uint32",
	"uint64",
	"sint32",
	"sint64",
	"fixed32",
	"fixed64",
	"sfixed32",
	"sfixed64",
	"bool",
	"string",
	"bytes",
	"enum",
	"duration",
	"timestamp",
}

// resultPaths adds to paths all populated paths in the testData Result, returning the new slice.
func resultPaths(prefix string, paths []string) []string {
	mapKeyTypes := [][]string{
		{"int32", "-11", "-10"},
		{"int64", "-21", "-20"},
		{"uint32", "30", "31"},
		{"uint64", "40", "41"},
		{"sint32", "-51", "-50"},
		{"sint64", "-61", "-60"},
		{"fixed32", "70", "71"},
		{"fixed64", "80", "81"},
		{"sfixed32", "-91", "-90"},
		{"sfixed64", "-101", "-100"},
		{"bool", "false", "true"},
		{"string", "a", "b"},
	}

	prefixStart := len(paths)
	for _, t := range scalarTypes {
		paths = append(paths, fmt.Sprintf("%s_val", t))
		paths = append(paths, fmt.Sprintf("r_%s[0]", t), fmt.Sprintf("r_%s[1]", t))
	}
	for _, t := range mapKeyTypes {
		ts := t[0]
		for _, v := range t[1:] {
			paths = append(paths, fmt.Sprintf("m_%s_string.%s", ts, v))
		}
	}
	paths = append(paths, "m_string_enum.a", "m_string_enum.b")
	paths = append(paths, "m_string_duration.a", "m_string_duration.b")
	paths = append(paths, "m_string_timestamp.a", "m_string_timestamp.b")

	// prepend the prefix
	needsPrefix := paths[prefixStart:]
	for i, path := range needsPrefix {
		needsPrefix[i] = prefix + path
	}
	return paths
}

var nestedMessagePaths = []string{
	"result.",
	"r_result[0].",
	"r_result[1].",
	"m_string_result.a.",
	"m_string_result.b.",
}

func nestedResultPaths(prefix string, paths []string) []string {
	for _, p := range nestedMessagePaths {
		paths = resultPaths(prefix+p, paths)
	}
	return paths
}

// resultLeafs adds to leafs all populated leaf values in the testData Result, returning the new slice.
func resultLeafs(leafs []value) []value {
	valFd := resultFd
	mapFd := resultMapFd

	return append(leafs, []value{
		{valFd("double_val"), protoreflect.ValueOfFloat64(1.1)},
		{valFd("r_double"), protoreflect.ValueOfFloat64(1.2)},
		{valFd("r_double"), protoreflect.ValueOfFloat64(1.3)},
		{valFd("float_val"), protoreflect.ValueOfFloat32(2.1)},
		{valFd("r_float"), protoreflect.ValueOfFloat32(2.2)},
		{valFd("r_float"), protoreflect.ValueOfFloat32(2.3)},
		{valFd("int32_val"), protoreflect.ValueOfInt32(-10)},
		{valFd("r_int32"), protoreflect.ValueOfInt32(-11)},
		{valFd("r_int32"), protoreflect.ValueOfInt32(-12)},
		{valFd("int64_val"), protoreflect.ValueOfInt64(-20)},
		{valFd("r_int64"), protoreflect.ValueOfInt64(-21)},
		{valFd("r_int64"), protoreflect.ValueOfInt64(-22)},
		{valFd("uint32_val"), protoreflect.ValueOfUint32(30)},
		{valFd("r_uint32"), protoreflect.ValueOfUint32(31)},
		{valFd("r_uint32"), protoreflect.ValueOfUint32(32)},
		{valFd("uint64_val"), protoreflect.ValueOfUint64(40)},
		{valFd("r_uint64"), protoreflect.ValueOfUint64(41)},
		{valFd("r_uint64"), protoreflect.ValueOfUint64(42)},
		{valFd("sint32_val"), protoreflect.ValueOfInt32(-50)},
		{valFd("r_sint32"), protoreflect.ValueOfInt32(-51)},
		{valFd("r_sint32"), protoreflect.ValueOfInt32(-52)},
		{valFd("sint64_val"), protoreflect.ValueOfInt64(-60)},
		{valFd("r_sint64"), protoreflect.ValueOfInt64(-61)},
		{valFd("r_sint64"), protoreflect.ValueOfInt64(-62)},
		{valFd("fixed32_val"), protoreflect.ValueOfUint32(70)},
		{valFd("r_fixed32"), protoreflect.ValueOfUint32(71)},
		{valFd("r_fixed32"), protoreflect.ValueOfUint32(72)},
		{valFd("fixed64_val"), protoreflect.ValueOfUint64(80)},
		{valFd("r_fixed64"), protoreflect.ValueOfUint64(81)},
		{valFd("r_fixed64"), protoreflect.ValueOfUint64(82)},
		{valFd("sfixed32_val"), protoreflect.ValueOfInt32(-90)},
		{valFd("r_sfixed32"), protoreflect.ValueOfInt32(-91)},
		{valFd("r_sfixed32"), protoreflect.ValueOfInt32(-92)},
		{valFd("sfixed64_val"), protoreflect.ValueOfInt64(-100)},
		{valFd("r_sfixed64"), protoreflect.ValueOfInt64(-101)},
		{valFd("r_sfixed64"), protoreflect.ValueOfInt64(-102)},
		{valFd("bool_val"), protoreflect.ValueOfBool(true)},
		{valFd("r_bool"), protoreflect.ValueOfBool(false)},
		{valFd("r_bool"), protoreflect.ValueOfBool(true)},
		{valFd("string_val"), protoreflect.ValueOfString("a")},
		{valFd("r_string"), protoreflect.ValueOfString("b")},
		{valFd("r_string"), protoreflect.ValueOfString("c")},
		{valFd("bytes_val"), protoreflect.ValueOfBytes([]byte("abc"))},
		{valFd("r_bytes"), protoreflect.ValueOfBytes([]byte("def"))},
		{valFd("r_bytes"), protoreflect.ValueOfBytes([]byte("ghi"))},
		{valFd("enum_val"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_A.Number())},
		{valFd("r_enum"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_B.Number())},
		{valFd("r_enum"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_C.Number())},
		{valFd("duration_val"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 10).ProtoReflect())},
		{valFd("r_duration"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 20).ProtoReflect())},
		{valFd("r_duration"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 30).ProtoReflect())},
		{valFd("timestamp_val"), protoreflect.ValueOfMessage(timestamppb.New(time.Unix(100, 10)).ProtoReflect())},
		{valFd("r_timestamp"), protoreflect.ValueOfMessage(timestamppb.New(time.Unix(200, 20)).ProtoReflect())},
		{valFd("r_timestamp"), protoreflect.ValueOfMessage(timestamppb.New(time.Unix(300, 30)).ProtoReflect())},
		{mapFd("m_int32_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_int32_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_int64_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_int64_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_uint32_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_uint32_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_uint64_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_uint64_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_sint32_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_sint32_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_sint64_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_sint64_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_fixed32_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_fixed32_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_fixed64_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_fixed64_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_sfixed32_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_sfixed32_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_sfixed64_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_sfixed64_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_bool_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_bool_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_string_string"), protoreflect.ValueOfString("a")},
		{mapFd("m_string_string"), protoreflect.ValueOfString("b")},
		{mapFd("m_string_enum"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_A.Number())},
		{mapFd("m_string_enum"), protoreflect.ValueOfEnum(querypb.ResultEnum_RESULT_ENUM_B.Number())},
		{mapFd("m_string_duration"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 10).ProtoReflect())},
		{mapFd("m_string_duration"), protoreflect.ValueOfMessage(durationpb.New(time.Second * 20).ProtoReflect())},
		{mapFd("m_string_timestamp"), protoreflect.ValueOfMessage(timestamppb.New(time.Unix(100, 10)).ProtoReflect())},
		{mapFd("m_string_timestamp"), protoreflect.ValueOfMessage(timestamppb.New(time.Unix(200, 20)).ProtoReflect())},
	}...)
}

func nestedResultLeafs(leafs []value) []value {
	// add all nested result leafs
	for range nestedMessagePaths {
		leafs = resultLeafs(leafs)
	}
	return leafs
}

func resultFd(f string) protoreflect.FieldDescriptor {
	md := (&querypb.Result{}).ProtoReflect().Descriptor()
	return md.Fields().ByName(protoreflect.Name(f))
}

func resultMapFd(f string) protoreflect.FieldDescriptor {
	return resultFd(f).MapValue()
}
