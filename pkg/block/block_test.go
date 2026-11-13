package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slices"
)

func ExampleDiff() {
	type object struct {
		Name string `json:"name"`
		Addr int    `json:"addr"`
		Mode string `json:"mode"`
	}

	type config struct {
		Objects []object `json:"objects"`
	}

	before := config{
		Objects: []object{
			{Name: "foo", Addr: 1, Mode: "auto"},
			{Name: "bar", Addr: 2, Mode: "manual"},
			{Name: "baz", Addr: 3, Mode: "auto"},
		},
	}
	after := config{
		Objects: []object{
			{Name: "foo", Addr: 1, Mode: "manual"}, // Mode changed
			// bar removed
			{Name: "baz", Addr: 22, Mode: "auto"},  // Addr changed
			{Name: "new", Addr: 4, Mode: "manual"}, // new object
		},
	}
	blocks := []Block{
		{
			Path: []string{"objects"},
			Key:  "name",
		},
	}

	patches, err := Diff(before, after, blocks)
	if err != nil {
		panic(err)
	}
	// sort patches for deterministic output
	slices.SortFunc(patches, func(i, j Patch) int {
		return ComparePaths(i.Path, j.Path)
	})

	for _, patch := range patches {
		encoded, err := json.Marshal(patch)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(encoded))
	}
	// Output:
	// {"path":"/objects[name=\"bar\"]","deleted":true}
	// {"path":"/objects[name=\"baz\"]","value":{"addr":22,"mode":"auto","name":"baz"}}
	// {"path":"/objects[name=\"foo\"]","value":{"addr":1,"mode":"manual","name":"foo"}}
	// {"path":"/objects[name=\"new\"]","value":{"addr":4,"mode":"manual","name":"new"}}
}

func ExampleApplyPatches() {
	type hvac struct {
		Setpoint float64 `json:"setpoint"`
		Heat     bool    `json:"heat"`
	}
	type space struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
		HVAC *hvac  `json:"hvac,omitempty"`
	}
	type house struct {
		Addr     string  `json:"addr"`
		Codeword string  `json:"codeword"`
		Spaces   []space `json:"spaces"`
	}

	base := house{
		Addr:     "123 Road",
		Codeword: "please",
		Spaces: []space{
			{Name: "kitchen", Mode: "auto", HVAC: &hvac{Setpoint: 20.0, Heat: true}},
			{Name: "bedroom", Mode: "manual", HVAC: &hvac{Setpoint: 22.0, Heat: false}},
		},
	}
	// all field references are the JSON field names
	patches := []Patch{
		// replace all top-level fields, without disturbing 'spaces'
		{
			Path: nil,
			Value: map[string]any{
				"addr":   "123b Road",
				"spaces": Ignore{},
			},
		},
		// replaces an entire sub-object of an array element
		{
			Path: []PathSegment{{Field: "spaces"}, {ArrayKey: "name", ArrayElem: "kitchen"}, {Field: "hvac"}},
			Value: map[string]any{
				"setpoint": 21.0,
				"heat":     false,
			},
		},
		// deletes an array element
		{
			Path:    []PathSegment{{Field: "spaces"}, {ArrayKey: "name", ArrayElem: "bedroom"}},
			Deleted: true,
		},
		// adds a new array element (because it references a non-existing key)
		{
			Path: []PathSegment{{Field: "spaces"}, {ArrayKey: "name", ArrayElem: "livingroom"}},
			Value: map[string]any{
				"name": "livingroom",
				"mode": "auto",
				"hvac": map[string]any{
					"setpoint": 23.0,
					"heat":     true,
				},
			},
		},
	}
	patched, err := ApplyPatches(base, patches)
	if err != nil {
		panic(err)
	}
	patchedJSON, err := json.MarshalIndent(patched, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(patchedJSON))
	// Output:
	// {
	//   "addr": "123b Road",
	//   "codeword": "",
	//   "spaces": [
	//     {
	//       "name": "kitchen",
	//       "mode": "auto",
	//       "hvac": {
	//         "setpoint": 21,
	//         "heat": false
	//       }
	//     },
	//     {
	//       "name": "livingroom",
	//       "mode": "auto",
	//       "hvac": {
	//         "setpoint": 23,
	//         "heat": true
	//       }
	//     }
	//   ]
	// }
}

func TestDiff(t *testing.T) {
	type testCase struct {
		a      any
		b      any
		schema []Block
		expect []Patch
	}

	cases := map[string]testCase{
		"Empty": {
			a:      nil,
			b:      nil,
			schema: nil,
			expect: nil,
		},
		"Primitive": {
			a:      "foo",
			b:      "bar",
			schema: nil,
			expect: []Patch{{Value: "bar"}},
		},
		"EqualPrimitives": {
			a:      "foo",
			b:      "foo",
			expect: []Patch{},
		},
		"EqualMaps": {
			a: map[string]any{
				"foo": "bar",
			},
			b: map[string]any{
				"foo": "bar",
			},
			expect: []Patch{},
		},
		"EqualSlice": {
			a:      []any{"hello", "world"},
			b:      []any{"hello", "world"},
			expect: []Patch{},
		},
		"TopLevelFields": {
			a: map[string]any{
				"foo": "bar",
				"baz": "qux",
			},
			b: map[string]any{
				"foo": "bar",
				"baz": "waldo",
			},
			schema: nil,
			expect: []Patch{{Value: map[string]any{"foo": "bar", "baz": "waldo"}}},
		},
		"NestedFields": {
			a: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
				},
			},
			b: map[string]any{
				"foo": map[string]any{
					"bar": "qux",
				},
			},
			schema: nil,
			expect: []Patch{{Value: map[string]any{"foo": map[string]any{"bar": "qux"}}}},
		},
		"NestedFields_IrrelevantSplit": {
			a: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
				},
			},
			b: map[string]any{
				"foo": map[string]any{
					"bar": "qux",
				},
			},
			schema: []Block{{Path: []string{"foo", "fooooo"}}},
			expect: []Patch{{Value: map[string]any{"foo": map[string]any{"bar": "qux", "fooooo": Ignore{}}}}},
		},
		"SplitField": {
			a: map[string]any{
				"foo": "bar",
				"baz": "qux",
			},
			b: map[string]any{
				"foo": "barbar",
				"baz": "qux",
			},
			schema: []Block{{Path: []string{"foo"}}},
			expect: []Patch{
				{
					Path:  []PathSegment{{Field: "foo"}},
					Value: "barbar",
				},
			},
		},
		"SplitNestedField": {
			a: map[string]any{
				"props": map[string]any{
					"foo": "bar",
					"baz": "qux",
				},
			},
			b: map[string]any{
				"props": map[string]any{
					"foo": "barbar",
					"baz": "qux",
				},
			},
			schema: []Block{{Path: []string{"props", "foo"}}},
			expect: []Patch{
				{
					Path:  []PathSegment{{Field: "props"}, {Field: "foo"}},
					Value: "barbar",
				},
			},
		},
		"SplitFieldAdded": {
			a: map[string]any{
				"baz": "qux",
			},
			b: map[string]any{
				"foo": "barbar",
				"baz": "qux",
			},
			schema: []Block{{Path: []string{"foo"}}},
			expect: []Patch{
				{
					Path:  []PathSegment{{Field: "foo"}},
					Value: "barbar",
				},
			},
		},
		"SplitFieldRemoved": {
			a: map[string]any{
				"foo": "bar",
				"baz": "qux",
			},
			b: map[string]any{
				"baz": "qux",
			},
			schema: []Block{{Path: []string{"foo"}}},
			expect: []Patch{
				{
					Path:    []PathSegment{{Field: "foo"}},
					Deleted: true,
				},
			},
		},
		"SplitFieldNested": {
			a: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
					"qux": "waldo",
				},
			},
			b: map[string]any{
				"foo": map[string]any{
					"bar": "bazbaz",
					"qux": "quxqux",
				},
			},
			schema: []Block{
				{
					Path: []string{"foo"},
					Blocks: []Block{
						{Path: []string{"bar"}},
					},
				},
			},
			expect: []Patch{
				{
					Path: []PathSegment{{Field: "foo"}},
					Value: map[string]any{
						"bar": Ignore{},
						"qux": "quxqux",
					},
				},
				{
					Path:  []PathSegment{{Field: "foo"}, {Field: "bar"}},
					Value: "bazbaz",
				},
			},
		},
		"Array": {
			// because of the JSON roundtrip, the numbers must be float64
			a: map[string]any{
				"foo": []any{
					map[string]any{"id": float64(1), "addr": "foo"},
					map[string]any{"id": float64(2), "addr": "bar"},
				},
			},
			b: map[string]any{
				"foo": []any{
					map[string]any{"id": float64(1), "addr": "foo"},
					map[string]any{"id": float64(2), "addr": "baz"},
				},
			},
			schema: []Block{
				{
					Path: []string{"foo"},
					Key:  "id",
				},
			},
			expect: []Patch{
				{
					Path:  []PathSegment{{Field: "foo"}, {ArrayKey: "id", ArrayElem: json.Number("2")}},
					Value: map[string]any{"id": json.Number("2"), "addr": "baz"},
				},
			},
		},
		"ArraySplit": {
			a: map[string]any{
				"drivers": []any{
					map[string]any{
						"name": "driver-1",
						"foo":  "foo-1",
					},
					map[string]any{
						"name": "driver-2",
					},
				},
			},
			b: map[string]any{
				"drivers": []any{
					map[string]any{
						"name": "driver-1",
						"foo":  "oof-1",
					},
					map[string]any{
						"name": "driver-2",
						"foo":  "foo-2",
						"bar":  "bar-2",
					},
				},
			},
			schema: []Block{
				{
					Path: []string{"drivers"},
					Key:  "name",
					Blocks: []Block{
						{Path: []string{"foo"}},
					},
				},
			},
			expect: []Patch{
				{
					Path: []PathSegment{
						{Field: "drivers"},
						{ArrayKey: "name", ArrayElem: "driver-1"},
						{Field: "foo"},
					},
					Value: "oof-1",
				},
				{
					Path: []PathSegment{
						{Field: "drivers"},
						{ArrayKey: "name", ArrayElem: "driver-2"},
					},
					Value: map[string]any{
						"name": "driver-2",
						"foo":  Ignore{},
						"bar":  "bar-2",
					},
				},
				{
					Path: []PathSegment{
						{Field: "drivers"},
						{ArrayKey: "name", ArrayElem: "driver-2"},
						{Field: "foo"},
					},
					Value: "foo-2",
				},
			},
		},
		// regression test for panic("cannot mark fields in nil map")
		"ArraySplitAddWithBlock": {
			a: map[string]any{
				"array": []any{
					map[string]any{"name": "entry1", "foo": "foo1"},
				},
			},
			b: map[string]any{
				"array": []any{
					map[string]any{"name": "entry1", "foo": "foo1"},
					map[string]any{"name": "entry2", "foo": "foo2"},
				},
			},
			schema: []Block{
				{
					Path: []string{"array"},
					Key:  "name",
					Blocks: []Block{
						{
							Path: []string{"foo"},
						},
					},
				},
			},
			expect: []Patch{
				{
					Path:  Path{{Field: "array"}, {ArrayKey: "name", ArrayElem: "entry2"}},
					Value: map[string]any{"name": "entry2", "foo": Ignore{}},
				},
				{
					Path:  Path{{Field: "array"}, {ArrayKey: "name", ArrayElem: "entry2"}, {Field: "foo"}},
					Value: "foo2",
				},
			},
		},
		"ArraySplitByKey": {
			a: map[string]any{
				"drivers": []any{
					map[string]any{
						"type": "a",
						"name": "driver-1",
						"objects": []any{
							map[string]any{"id": float64(1), "addr": "foo"},
							map[string]any{"id": float64(2), "addr": "bar"},
						},
					},
					map[string]any{
						"type":     "b",
						"name":     "driver-2",
						"settings": map[string]any{"mode": "on"},
					},
				},
			},
			b: map[string]any{
				"drivers": []any{
					map[string]any{
						"type": "a",
						"name": "driver-1",
						"objects": []any{
							map[string]any{"id": float64(1), "addr": "foo2"},
							map[string]any{"id": float64(2), "addr": "bar"},
						},
					},
					map[string]any{
						"type":     "b",
						"name":     "driver-2",
						"settings": map[string]any{"mode": "off"},
					},
				},
			},
			schema: []Block{
				{
					Path:    []string{"drivers"},
					Key:     "name",
					TypeKey: "type",
					BlocksByType: map[string][]Block{
						"a": {
							{
								Path: []string{"objects"},
								Key:  "id",
							},
						},
						"b": {
							{
								Path: []string{"settings"},
							},
						},
					},
				},
			},
			expect: []Patch{
				{
					Path: []PathSegment{
						{Field: "drivers"},
						{ArrayKey: "name", ArrayElem: "driver-1"},
						{Field: "objects"},
						{ArrayKey: "id", ArrayElem: json.Number("1")},
					},
					Value: map[string]any{"id": json.Number("1"), "addr": "foo2"},
				},
				{
					Path: []PathSegment{
						{Field: "drivers"},
						{ArrayKey: "name", ArrayElem: "driver-2"},
						{Field: "settings"},
					},
					Value: map[string]any{"mode": "off"},
				},
			},
		},
		"ChangeType_MapToSlice": {
			a: map[string]any{
				"hello": "world",
			},
			b: []any{
				"hello", "world",
			},
			expect: []Patch{
				{
					Path:  nil,
					Value: []any{"hello", "world"},
				},
			},
		},
		"ChangeType_SliceToMap": {
			a: []any{
				"hello", "world",
			},
			b: map[string]any{
				"hello": "world",
			},
			expect: []Patch{
				{
					Path:  nil,
					Value: map[string]any{"hello": "world"},
				},
			},
		},
		"ChangeType_PrimitiveToMap": {
			a: "hello",
			b: map[string]any{
				"hello": "world",
			},
			expect: []Patch{
				{
					Path:  nil,
					Value: map[string]any{"hello": "world"},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches, err := Diff(tc.a, tc.b, tc.schema)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			slices.SortFunc(patches, func(i, j Patch) int {
				return ComparePaths(i.Path, j.Path)
			})
			if diff := cmp.Diff(tc.expect, patches, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyPatch(t *testing.T) {
	base := map[string]any{
		"foo": "bar",
		"baz": []any{1.0, 2.0, 3.0},
		"qux": map[string]any{
			"flub": map[string]any{
				"garply": "waldo",
			},
			"objects": []any{
				map[string]any{"name": "foo", "address": "1.2.3"},
				map[string]any{"name": "bar", "address": "4.5.6"},
			},
		},
	}

	type testCase struct {
		patch  Patch
		expect any
		err    error
	}

	cases := map[string]testCase{
		"ReplaceTopLevel": {
			patch: Patch{
				Path: nil,
				Value: map[string]any{
					"replaced": "value",
				},
			},
			expect: map[string]any{
				"replaced": "value",
			},
		},
		"ReplaceScalar": {
			patch: Patch{
				Path:  []PathSegment{{Field: "foo"}},
				Value: "replaced",
			},
			expect: map[string]any{
				"foo": "replaced",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
			},
		},
		"ReplaceTopLevelPoint": {
			patch: Patch{
				Path: nil,
				Value: map[string]any{
					"newfoo": "replaced",
					"qux":    Ignore{},
				},
			},
			expect: map[string]any{
				"newfoo": "replaced",
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
			},
		},
		"ReplaceArrayElem": {
			patch: Patch{
				Path:  []PathSegment{{Field: "qux"}, {Field: "objects"}, {ArrayKey: "name", ArrayElem: "foo"}},
				Value: map[string]any{"name": "foo", "address": "7.8.9"},
			},
			expect: map[string]any{
				"foo": "bar",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "7.8.9"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
			},
		},
		"AddArrayElem": {
			patch: Patch{
				Path:  []PathSegment{{Field: "qux"}, {Field: "objects"}, {ArrayKey: "name", ArrayElem: "baz"}},
				Value: map[string]any{"name": "baz", "address": "7.8.9"},
			},
			expect: map[string]any{
				"foo": "bar",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
						map[string]any{"name": "baz", "address": "7.8.9"},
					},
				},
			},
		},
		"AddField": {
			patch: Patch{
				Path:  []PathSegment{{Field: "newfield"}},
				Value: "newfieldvalue",
			},
			expect: map[string]any{
				"foo": "bar",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
				"newfield": "newfieldvalue",
			},
		},
		"AddNestedField": {
			patch: Patch{
				Path:  []PathSegment{{Field: "newfield1"}, {Field: "newfield2"}},
				Value: "newfieldvalue",
			},
			expect: map[string]any{
				"foo": "bar",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
				"newfield1": map[string]any{
					"newfield2": "newfieldvalue",
				},
			},
		},
		"AddNestedFieldInArray": {
			patch: Patch{
				Path:  []PathSegment{{Field: "newarray"}, {ArrayKey: "name", ArrayElem: "newelem"}, {Field: "newfield"}},
				Value: "newfieldvalue",
			},
			expect: map[string]any{
				"foo": "bar",
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
				"newarray": []any{
					map[string]any{"name": "newelem", "newfield": "newfieldvalue"},
				},
			},
		},
		"DeleteField": {
			patch: Patch{
				Path:    []PathSegment{{Field: "foo"}},
				Deleted: true,
			},
			expect: map[string]any{
				"baz": []any{1.0, 2.0, 3.0},
				"qux": map[string]any{
					"flub": map[string]any{
						"garply": "waldo",
					},
					"objects": []any{
						map[string]any{"name": "foo", "address": "1.2.3"},
						map[string]any{"name": "bar", "address": "4.5.6"},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dst := clone(base)
			dst, err := ApplyPatches(dst, []Patch{tc.patch})
			if !errors.Is(err, tc.err) {
				t.Errorf("expected error %v, got %v", tc.err, err)
			}
			if diff := cmp.Diff(tc.expect, dst); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}

			// test idempotence by applying the same patch again
			if tc.err == nil {
				dst, err = ApplyPatches(dst, []Patch{tc.patch})
				if err != nil {
					t.Errorf("second patch failed: %v", err)
				}
				if diff := cmp.Diff(tc.expect, dst); diff != "" {
					t.Errorf("unexpected result after second patch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestPatch_UnmarshalJSON(t *testing.T) {
	original := `
{
	"path": ["a", {"id": "theid"}],
	"value": {
		"foo": "foo",
	   	"bar": {
	 		"barbar": {"$block": "ignore"}
		}
	}
}
`
	expect := Patch{
		Path: []PathSegment{
			{Field: "a"},
			{ArrayKey: "id", ArrayElem: "theid"},
		},
		Value: map[string]any{
			"foo": "foo",
			"bar": map[string]any{
				"barbar": Ignore{},
			},
		},
	}

	var decoded Patch
	err := json.Unmarshal([]byte(original), &decoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diff := cmp.Diff(expect, decoded); diff != "" {
		t.Errorf("unexpected decode (-want +got):\n%s", diff)
	}

}

// tests that applying all the diffs between a and b to a results in b
func TestApplyPatch_Consistency(t *testing.T) {
	type testCase struct {
		a, b      any
		blockSets [][]Block // allows us to try multiple splits and check they all work
	}
	cases := map[string]testCase{
		"Primitive": {
			a:         "foo",
			b:         "bar",
			blockSets: nil,
		},
		"Object": {
			a: map[string]any{
				"foo": "bar",
				"baz": "qux",
			},
			b: map[string]any{
				"foofoo": "barbar",
				"baz":    "qux2",
			},
			blockSets: [][]Block{
				{},
				{{Path: []string{"baz"}}},
				{{Path: []string{"foo"}}},
				{{Path: []string{"foo", "baz"}}},
				{{Path: []string{"foofoo"}}},
			},
		},
		"Array": {
			a: map[string]any{
				"objects": []any{
					map[string]any{"type": "a", "name": "a-1", "addr": "foo"},
					map[string]any{"type": "b", "name": "b-1", "id": 1.0},
					map[string]any{"type": "b", "name": "b-2", "id": 2.0},
				},
			},
			b: map[string]any{
				"objects": []any{
					map[string]any{"type": "a", "name": "a-1", "addr": "oof"},
					map[string]any{"type": "a", "name": "a-2", "addr": "foo2"},
					map[string]any{"type": "b", "name": "b-2", "id": 3.0},
				},
			},
			blockSets: [][]Block{
				{},
				{{Path: []string{"objects"}}},
				{{Path: []string{"objects"}, Key: "name"}},
				{{Path: []string{"objects"}, Key: "name", TypeKey: "type", BlocksByType: map[string][]Block{
					"a": {{Path: []string{"addr"}}},
					"b": {{Path: []string{"id"}}},
				}}},
			},
		},
	}

	// ignore the order of entries in maps
	mapLess := func(a, b any) bool {
		aMap, okA := a.(map[string]any)
		bMap, okB := b.(map[string]any)
		if !okA || !okB {
			return false
		}
		nameA, okA := aMap["name"].(string)
		nameB, okB := bMap["name"].(string)
		if okA && okB {
			return nameA < nameB
		}
		return false
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for i, blocks := range tc.blockSets {
				patches, err := Diff(tc.a, tc.b, blocks)
				if err != nil {
					t.Errorf("blocks %d unexpected diff error: %v", i, err)
				}
				dst, err := ApplyPatches(tc.a, patches)
				if err != nil {
					t.Errorf("blocks %d unexpected patch error: %v", i, err)
				}
				if diff := cmp.Diff(tc.b, dst, cmpopts.SortSlices(mapLess)); diff != "" {
					t.Errorf("blocks %d unexpected result (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

func clone(dst any) any {
	switch dst := dst.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(dst))
		for k, v := range dst {
			cloned[k] = clone(v)
		}
		return cloned

	case []any:
		cloned := make([]any, len(dst))
		for i, v := range dst {
			cloned[i] = clone(v)
		}
		return cloned

	default:
		return dst
	}
}
