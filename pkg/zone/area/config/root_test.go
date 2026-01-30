package config

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

func TestRoot_UnmarshalJSON(t *testing.T) {
	buf := []byte(`{
	"name": "test",
	"type": "area",
	"metadata": {
		"appearance": {"title": "TEST"}
	},
	"another": "prop"
}`)
	config := Root{}
	err := json.Unmarshal(buf, &config)
	if err != nil {
		t.Fatal(err)
	}

	want := Root{
		Config: zone.Config{
			Name: "test",
			Type: "area",
		},
		Self: Self{
			Metadata: &traits.Metadata{
				Appearance: &traits.Metadata_Appearance{Title: "TEST"},
			},
		},
		Raw: buf,
	}

	if diff := cmp.Diff(want, config, protocmp.Transform()); diff != "" {
		t.Fatalf("(-want,+got)\n%s", diff)
	}
}
