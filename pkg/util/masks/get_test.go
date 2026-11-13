package masks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/internal/testproto"
)

func TestResponseFilter_Filter(t *testing.T) {
	type nested = testproto.TestAllTypes_NestedMessage
	full := func() *testproto.TestAllTypes {
		return &testproto.TestAllTypes{
			DefaultInt32:         1,
			DefaultString:        "str",
			DefaultNestedMessage: &nested{A: 2, Corecursive: &testproto.TestAllTypes{DefaultInt32: 3, DefaultString: "deep"}},
			RepeatedString:       []string{"a", "b"},
			RepeatedNestedMessage: []*nested{
				{A: 4, Corecursive: &testproto.TestAllTypes{DefaultInt32: 5}},
				{A: 6},
			},
			MapStringString: map[string]string{"x": "1", "y": "2"},
			MapInt32Int32:   map[int32]int32{1: 10, 2: 20},
			MapStringNestedMessage: map[string]*nested{
				"x": {A: 7, Corecursive: &testproto.TestAllTypes{DefaultInt32: 8}},
				"y": {A: 9},
			},
		}
	}

	tests := []struct {
		name  string
		paths []string
		want  *testproto.TestAllTypes
	}{
		{
			name:  "top level scalar",
			paths: []string{"default_int32"},
			want:  &testproto.TestAllTypes{DefaultInt32: 1},
		},
		{
			name:  "whole message",
			paths: []string{"default_nested_message"},
			want:  &testproto.TestAllTypes{DefaultNestedMessage: full().DefaultNestedMessage},
		},
		{
			name:  "nested message field",
			paths: []string{"default_nested_message.corecursive.default_string"},
			want: &testproto.TestAllTypes{DefaultNestedMessage: &nested{
				Corecursive: &testproto.TestAllTypes{DefaultString: "deep"},
			}},
		},
		{
			name:  "path past a scalar",
			paths: []string{"default_int32.x"},
			want:  &testproto.TestAllTypes{DefaultInt32: 1},
		},
		{
			name:  "whole map",
			paths: []string{"map_string_string"},
			want:  &testproto.TestAllTypes{MapStringString: map[string]string{"x": "1", "y": "2"}},
		},
		{
			name:  "map key",
			paths: []string{"map_string_string.x"},
			want:  &testproto.TestAllTypes{MapStringString: map[string]string{"x": "1"}},
		},
		{
			name:  "missing map key",
			paths: []string{"map_string_string.missing"},
			want:  &testproto.TestAllTypes{},
		},
		{
			name:  "path past a map scalar value",
			paths: []string{"map_string_string.x.deeper"},
			want:  &testproto.TestAllTypes{MapStringString: map[string]string{"x": "1"}},
		},
		{
			name:  "int map key",
			paths: []string{"map_int32_int32.2"},
			want:  &testproto.TestAllTypes{MapInt32Int32: map[int32]int32{2: 20}},
		},
		{
			name:  "map message value field",
			paths: []string{"map_string_nested_message.x.a"},
			want:  &testproto.TestAllTypes{MapStringNestedMessage: map[string]*nested{"x": {A: 7}}},
		},
		{
			name:  "path past a repeated scalar",
			paths: []string{"repeated_string.x"},
			want:  &testproto.TestAllTypes{RepeatedString: []string{"a", "b"}},
		},
		{
			name:  "repeated message field",
			paths: []string{"repeated_nested_message.a"},
			want:  &testproto.TestAllTypes{RepeatedNestedMessage: []*nested{{A: 4}, {A: 6}}},
		},
		{
			name:  "several paths",
			paths: []string{"default_string", "map_string_nested_message.y", "repeated_nested_message.corecursive"},
			want: &testproto.TestAllTypes{
				DefaultString:          "str",
				MapStringNestedMessage: map[string]*nested{"y": {A: 9}},
				RepeatedNestedMessage: []*nested{
					{Corecursive: &testproto.TestAllTypes{DefaultInt32: 5}},
					{},
				},
			},
		},
		{
			name:  "empty mask clears everything",
			paths: []string{},
			want:  &testproto.TestAllTypes{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewResponseFilter(WithFieldMaskPaths(tt.paths...))

			got := full()
			filter.Filter(got)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("Filter (-want +got):\n%s", diff)
			}

			orig := full()
			clone := filter.FilterClone(orig)
			if diff := cmp.Diff(tt.want, clone, protocmp.Transform()); diff != "" {
				t.Errorf("FilterClone (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(full(), orig, protocmp.Transform()); diff != "" {
				t.Errorf("FilterClone changed the original (-want +got):\n%s", diff)
			}
		})
	}
}

// fmutils.Filter panicked on a path past a repeated scalar, here FieldMask.paths (SCB-1469).
func TestResponseFilter_Filter_repeatedScalarPath(t *testing.T) {
	msg := &fieldmaskpb.FieldMask{Paths: []string{"a", "b"}}
	NewResponseFilter(WithFieldMaskPaths("paths.x")).Filter(msg)
	want := &fieldmaskpb.FieldMask{Paths: []string{"a", "b"}}
	if !proto.Equal(want, msg) {
		t.Errorf("Filter got %v, want %v", msg, want)
	}
}

func TestResponseFilter_noMask(t *testing.T) {
	msg := &testproto.TestAllTypes{DefaultInt32: 1}
	filter := NewResponseFilter()
	filter.Filter(msg)
	if msg.DefaultInt32 != 1 {
		t.Errorf("Filter with no mask changed the message: %v", msg)
	}
	if got := filter.FilterClone(msg); got != msg {
		t.Errorf("FilterClone with no mask returned a different message")
	}
}
