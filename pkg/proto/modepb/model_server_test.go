package modepb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestModelServer_UpdateModeValues(t *testing.T) {
	m := NewModel()
	s := NewModelServer(m)
	newValues, err := s.UpdateModeValues(nil, &traits.UpdateModeValuesRequest{
		Relative: &traits.ModeValuesRelative{Values: map[string]int32{
			"temperature": 1,
			"spin":        -1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := &traits.ModeValues{
		Values: map[string]string{
			"temperature": "medium",
			"spin":        "fast",
		},
	}
	if diff := cmp.Diff(want, newValues, protocmp.Transform()); diff != "" {
		t.Errorf("response (-want, +got)\n%v", diff)
	}
}
