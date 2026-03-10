package metadatapb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func TestModel_MergeMetadata(t *testing.T) {
	tests := []struct {
		name   string
		before *traits.Metadata
		update *traits.Metadata
		want   *traits.Metadata
	}{
		{
			name:   "apply to empty",
			before: &traits.Metadata{},
			update: &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}},
			want:   &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}},
		},
		{
			name:   "apply to different group",
			before: &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}},
			update: &traits.Metadata{Appearance: &traits.Metadata_Appearance{Description: "Foo"}},
			want:   &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}, Appearance: &traits.Metadata_Appearance{Description: "Foo"}},
		},
		{
			name:   "update group",
			before: &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}},
			update: &traits.Metadata{Membership: &traits.Metadata_Membership{Subsystem: "Lights"}},
			want:   &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device", Subsystem: "Lights"}},
		},
		{
			name:   "overwrite group",
			before: &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Test Device"}},
			update: &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Real Device"}},
			want:   &traits.Metadata{Membership: &traits.Metadata_Membership{Group: "Real Device"}},
		},
		{
			name:   "add trait",
			before: &traits.Metadata{},
			update: &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait"}}},
			want:   &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait"}}},
		},
		{
			name:   "add another trait",
			before: &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait"}}},
			update: &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "AnotherTrait"}}},
			want:   &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "AnotherTrait"}, {Name: "SuperTrait"}}},
		},
		{
			name:   "add trait meta",
			before: &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait", More: map[string]string{"one": "1", "two": "2"}}}},
			update: &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait", More: map[string]string{"two": "II", "three": "3"}}}},
			want:   &traits.Metadata{Traits: []*traits.TraitMetadata{{Name: "SuperTrait", More: map[string]string{"one": "1", "two": "II", "three": "3"}}}},
		},
	}

	for _, tt := range tests {
		// these tests must run in sequence
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(resource.WithInitialValue(tt.before))
			got, err := m.MergeMetadata(tt.update)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("MergeMetadata (-want,+got)\n%s", diff)
			}
		})
	}
}
