package block

import (
	"encoding/json"
	"strings"
	"testing"
)

// 1308675217947965579 is above 2^53, so a float64 round-trip turns it into 1308675217947965700 (SCB-1427).
const bigID = 1308675217947965579

type bigNumDevice struct {
	Name     string          `json:"name"`
	DeviceID int64           `json:"deviceId"`
	Mode     string          `json:"mode"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

type bigNumConfig struct {
	Devices []bigNumDevice `json:"devices"`
	Nested  struct {
		ID   int64  `json:"id"`
		Note string `json:"note"`
	} `json:"nested"`
}

func TestDiffApply_LargeIntegers(t *testing.T) {
	const rawWant = `{"id":1308675217947965579}`
	before := bigNumConfig{
		Devices: []bigNumDevice{{Name: "d1", DeviceID: bigID, Mode: "auto", Raw: json.RawMessage(rawWant)}},
	}
	before.Nested.ID = bigID
	before.Nested.Note = "before"

	after := before
	after.Devices = []bigNumDevice{{Name: "d1", DeviceID: bigID, Mode: "manual", Raw: json.RawMessage(rawWant)}}
	after.Nested.Note = "after"

	schema := []Block{{Path: []string{"devices"}, Key: "name"}}
	patches, err := Diff(before, after, schema)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(patches) != 2 {
		t.Fatalf("got %d patches, want 2: %+v", len(patches), patches)
	}
	encoded, err := json.Marshal(patches)
	if err != nil {
		t.Fatalf("marshal patches: %v", err)
	}
	// once in each patch value, plus the raw message in the device patch
	if got := strings.Count(string(encoded), "1308675217947965579"); got != 3 {
		t.Errorf("patches mention the exact id %d times, want 3:\n%s", got, encoded)
	}

	// patches read back from disk must apply just the same.
	// That holds here because the array key is a string: ParsePath reads numeric selectors as float64,
	// so a logged patch keyed by an integer above 2^53 could still match a neighbouring element.
	var decoded []Patch
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal patches: %v", err)
	}

	for name, ps := range map[string][]Patch{"fromDiff": patches, "decoded": decoded} {
		t.Run(name, func(t *testing.T) {
			got, err := ApplyPatches(before, ps)
			if err != nil {
				t.Fatalf("ApplyPatches: %v", err)
			}
			if len(got.Devices) != 1 {
				t.Fatalf("got %d devices, want 1: %+v", len(got.Devices), got.Devices)
			}
			d := got.Devices[0]
			if d.DeviceID != bigID {
				t.Errorf("device id = %d, want %d", d.DeviceID, int64(bigID))
			}
			if d.Mode != "manual" {
				t.Errorf("device mode = %q, want manual", d.Mode)
			}
			if string(d.Raw) != rawWant {
				t.Errorf("device raw = %s, want %s", d.Raw, rawWant)
			}
			if got.Nested.ID != bigID {
				t.Errorf("nested id = %d, want %d", got.Nested.ID, int64(bigID))
			}
			if got.Nested.Note != "after" {
				t.Errorf("nested note = %q, want after", got.Nested.Note)
			}
		})
	}
}

// Array keys from Diff are json.Number, but paths parsed from their string form hold float64.
// Both must find the same element, or the patch would append a duplicate.
func TestApplyPatches_NumericKeyFromParsedPath(t *testing.T) {
	data := map[string]any{"objects": []any{map[string]any{"id": 2, "addr": "foo"}}}
	path, err := ParsePath(`/objects[id=2]/addr`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyPatches(data, []Patch{{Path: path, Value: "bar"}})
	if err != nil {
		t.Fatal(err)
	}
	objects := got["objects"].([]any)
	if len(objects) != 1 {
		t.Fatalf("got %d objects, want 1: %v", len(objects), objects)
	}
	if addr := objects[0].(map[string]any)["addr"]; addr != "bar" {
		t.Errorf("addr = %v, want bar", addr)
	}
}

// Distinct integer keys above 2^53 can share a float64, so they must not be compared as floats.
func TestApplyPatches_LargeIntegerKeys(t *testing.T) {
	type entry struct {
		ID   int64  `json:"id"`
		Addr string `json:"addr"`
	}
	type doc struct {
		Objects []entry `json:"objects"`
	}
	before := doc{Objects: []entry{{ID: bigID, Addr: "a"}, {ID: bigID + 1, Addr: "b"}}}
	after := doc{Objects: []entry{{ID: bigID, Addr: "a"}, {ID: bigID + 1, Addr: "b2"}}}
	patches, err := Diff(before, after, []Block{{Path: []string{"objects"}, Key: "id"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyPatches(before, patches)
	if err != nil {
		t.Fatal(err)
	}
	want := after
	if len(got.Objects) != 2 || got.Objects[0] != want.Objects[0] || got.Objects[1] != want.Objects[1] {
		t.Errorf("got %+v, want %+v", got.Objects, want.Objects)
	}
}
