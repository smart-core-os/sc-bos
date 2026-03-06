package cmp

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-golang/internal/testproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEqual(t *testing.T) {

	tests := []struct {
		name string
		cmp  []Value
		x, y proto.Message
		want bool
	}{
		{"nil,nil", nil, nil, nil, true},
		{"nil,{}", nil, nil, &testproto.TestAllTypes{}, false},
		{"{},nil", nil, &testproto.TestAllTypes{}, nil, false},
		{"{},{}", nil, &testproto.TestAllTypes{}, &testproto.TestAllTypes{}, true},
		{"{...},{...}", nil, allTypesFull(), allTypesFull(), true},
		{"{t=1},{t=2}=false", nil, setWellKnownTimes(allTypesFull(), time.Unix(1000, 0)), setWellKnownTimes(allTypesFull(), time.Unix(1001, 0)), false},
		{"{t=1},{t=2}=true", []Value{TimeValueWithin(1 * time.Second)}, setWellKnownTimes(allTypesFull(), time.Unix(1000, 0)), setWellKnownTimes(allTypesFull(), time.Unix(1001, 0)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := Equal(tt.cmp...)
			if eq(tt.x, tt.y) != tt.want {
				t.Errorf("%v Equal(x, y) != %v: x=%v y=%v", tt.name, tt.want, tt.x, tt.y)
			}
			if eq(tt.y, tt.x) != tt.want {
				t.Errorf("%v Equal(x, y) != %v: (rev) x=%v y=%v", tt.name, tt.want, tt.y, tt.x)
			}
		})
	}
}

func allTypesFull() *testproto.TestAllTypes {
	var optionalInt32 int32 = 1
	var optionalInt64 int64 = 2
	var optionalUint32 uint32 = 3
	var optionalUint64 uint64 = 4
	var optionalSint32 int32 = 5
	var optionalSint64 int64 = 6
	var optionalFixed32 uint32 = 7
	var optionalFixed64 uint64 = 8
	var optionalSfixed32 int32 = 9
	var optionalSfixed64 int64 = 10
	var optionalFloat float32 = 1.1
	var optionalDouble float64 = 1.2
	var optionalBool bool = true
	var optionalString string = "foo"

	return &testproto.TestAllTypes{
		DefaultInt32:         1,
		DefaultInt64:         2,
		DefaultUint32:        3,
		DefaultUint64:        4,
		DefaultSint32:        5,
		DefaultSint64:        6,
		DefaultFixed32:       7,
		DefaultFixed64:       8,
		DefaultSfixed32:      9,
		DefaultSfixed64:      10,
		DefaultFloat:         1.1,
		DefaultDouble:        1.2,
		DefaultBool:          true,
		DefaultString:        "foo",
		DefaultBytes:         []byte("bar"),
		OneofDefault:         &testproto.TestAllTypes_OneofDefaultInt32{OneofDefaultInt32: 11},
		DefaultNestedMessage: &testproto.TestAllTypes_NestedMessage{A: 12},
		DefaultForeignMessage: &testproto.ForeignMessage{
			C: 1,
			D: 2,
		},
		DefaultWellKnown:       wellKnownMessage(),
		DefaultNestedEnum:      testproto.TestAllTypes_BAZ,
		DefaultForeignEnum:     testproto.ForeignEnum_FOREIGN_BAR,
		RepeatedInt32:          []int32{1, 2},
		RepeatedInt64:          []int64{3, 4},
		RepeatedUint32:         []uint32{5, 6},
		RepeatedUint64:         []uint64{7, 8},
		RepeatedSint32:         []int32{9, 10},
		RepeatedSint64:         []int64{11, 12},
		RepeatedFixed32:        []uint32{13, 14},
		RepeatedFixed64:        []uint64{15, 16},
		RepeatedSfixed32:       []int32{17, 18},
		RepeatedSfixed64:       []int64{19, 20},
		RepeatedFloat:          []float32{1.1, 1.2},
		RepeatedDouble:         []float64{0.1, 0.2},
		RepeatedBool:           []bool{true, false},
		RepeatedString:         []string{"foo", "bar"},
		RepeatedBytes:          [][]byte{[]byte("baz"), []byte("qux")},
		RepeatedNestedMessage:  []*testproto.TestAllTypes_NestedMessage{{A: 1}, {A: 2}},
		RepeatedForeignMessage: []*testproto.ForeignMessage{{C: 1}, {D: 2}},
		RepeatedWellKnown: []*testproto.WellKnown{
			wellKnownMessage(),
			wellKnownMessage(),
		},
		RepeatedNestedEnum:     []testproto.TestAllTypes_NestedEnum{testproto.TestAllTypes_BAR, testproto.TestAllTypes_FOO},
		RepeatedForeignEnum:    []testproto.ForeignEnum{testproto.ForeignEnum_FOREIGN_BAR, testproto.ForeignEnum_FOREIGN_BAZ},
		MapInt32Int32:          map[int32]int32{1: 2},
		MapInt64Int64:          map[int64]int64{3: 4},
		MapUint32Uint32:        map[uint32]uint32{5: 6},
		MapUint64Uint64:        map[uint64]uint64{7: 8},
		MapSint32Sint32:        map[int32]int32{9: 10},
		MapSint64Sint64:        map[int64]int64{11: 12},
		MapFixed32Fixed32:      map[uint32]uint32{13: 14},
		MapFixed64Fixed64:      map[uint64]uint64{15: 16},
		MapSfixed32Sfixed32:    map[int32]int32{17: 18},
		MapSfixed64Sfixed64:    map[int64]int64{19: 20},
		MapInt32Float:          map[int32]float32{1: 1.1},
		MapInt32Double:         map[int32]float64{2: 2.1},
		MapBoolBool:            map[bool]bool{true: false},
		MapStringString:        map[string]string{"foo": "bar"},
		MapStringBytes:         map[string][]byte{"baz": []byte("qux")},
		MapStringNestedMessage: map[string]*testproto.TestAllTypes_NestedMessage{"1": {A: 1}},
		MapStringWellKnown:     map[string]*testproto.WellKnown{"2": wellKnownMessage()},
		OptionalInt32:          &optionalInt32,
		OptionalInt64:          &optionalInt64,
		OptionalUint32:         &optionalUint32,
		OptionalUint64:         &optionalUint64,
		OptionalSint32:         &optionalSint32,
		OptionalSint64:         &optionalSint64,
		OptionalFixed32:        &optionalFixed32,
		OptionalFixed64:        &optionalFixed64,
		OptionalSfixed32:       &optionalSfixed32,
		OptionalSfixed64:       &optionalSfixed64,
		OptionalFloat:          &optionalFloat,
		OptionalDouble:         &optionalDouble,
		OptionalBool:           &optionalBool,
		OptionalString:         &optionalString,
		OptionalBytes:          []byte("bytes"),
	}
}

func wellKnownMessage() *testproto.WellKnown {
	return &testproto.WellKnown{
		DefaultTimestamp: timestamppb.New(time.Unix(1, 2)),
		DefaultDuration:  durationpb.New(3 * time.Second),
	}
}

func setWellKnownTimes(base *testproto.TestAllTypes, t time.Time) *testproto.TestAllTypes {
	base = proto.Clone(base).(*testproto.TestAllTypes)
	base.DefaultWellKnown.DefaultTimestamp = timestamppb.New(t)
	for _, known := range base.RepeatedWellKnown {
		known.DefaultTimestamp = timestamppb.New(t)
	}
	message := base.DefaultNestedMessage
	if message.GetCorecursive() != nil {
		message.Corecursive = setWellKnownTimes(message.Corecursive, t)
	}
	for _, message := range base.RepeatedNestedMessage {
		if message.GetCorecursive() != nil {
			message.Corecursive = setWellKnownTimes(message.Corecursive, t)
		}
	}
	return base
}

func setWellKnownDurations(base *testproto.TestAllTypes, d time.Duration) *testproto.TestAllTypes {
	base = proto.Clone(base).(*testproto.TestAllTypes)
	base.DefaultWellKnown.DefaultDuration = durationpb.New(d)
	for _, known := range base.RepeatedWellKnown {
		known.DefaultDuration = durationpb.New(d)
	}
	message := base.DefaultNestedMessage
	if message.GetCorecursive() != nil {
		message.Corecursive = setWellKnownDurations(message.Corecursive, d)
	}
	for _, message := range base.RepeatedNestedMessage {
		if message.GetCorecursive() != nil {
			message.Corecursive = setWellKnownDurations(message.Corecursive, d)
		}
	}
	return base
}

func setFloatValues(base *testproto.TestAllTypes, f float32) *testproto.TestAllTypes {
	base = proto.Clone(base).(*testproto.TestAllTypes)
	base.DefaultFloat = f
	base.OptionalFloat = &f
	message := base.DefaultNestedMessage
	if message.GetCorecursive() != nil {
		message.Corecursive = setFloatValues(message.Corecursive, f)
	}
	for _, message := range base.RepeatedNestedMessage {
		if message.GetCorecursive() != nil {
			message.Corecursive = setFloatValues(message.Corecursive, f)
		}
	}
	for i, _ := range base.RepeatedFloat {
		base.RepeatedFloat[i] = f
	}
	return base
}
func setDoubleValues(base *testproto.TestAllTypes, f float64) *testproto.TestAllTypes {
	base = proto.Clone(base).(*testproto.TestAllTypes)
	base.DefaultDouble = f
	base.OptionalDouble = &f
	message := base.DefaultNestedMessage
	if message.GetCorecursive() != nil {
		message.Corecursive = setDoubleValues(message.Corecursive, f)
	}
	for _, message := range base.RepeatedNestedMessage {
		if message.GetCorecursive() != nil {
			message.Corecursive = setDoubleValues(message.Corecursive, f)
		}
	}
	for i, _ := range base.RepeatedDouble {
		base.RepeatedDouble[i] = f
	}
	return base
}
