package metadatapb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

func TestModel_MergeMetadata(t *testing.T) {
	tests := []struct {
		name   string
		before *Metadata
		update *Metadata
		want   *Metadata
	}{
		{
			name:   "apply to empty",
			before: &Metadata{},
			update: &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}},
			want:   &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}},
		},
		{
			name:   "apply to different group",
			before: &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}},
			update: &Metadata{Appearance: &Metadata_Appearance{Description: "Foo"}},
			want:   &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}, Appearance: &Metadata_Appearance{Description: "Foo"}},
		},
		{
			name:   "update group",
			before: &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}},
			update: &Metadata{Membership: &Metadata_Membership{Subsystem: "Lights"}},
			want:   &Metadata{Membership: &Metadata_Membership{Group: "Test Device", Subsystem: "Lights"}},
		},
		{
			name:   "overwrite group",
			before: &Metadata{Membership: &Metadata_Membership{Group: "Test Device"}},
			update: &Metadata{Membership: &Metadata_Membership{Group: "Real Device"}},
			want:   &Metadata{Membership: &Metadata_Membership{Group: "Real Device"}},
		},
		{
			name:   "add trait",
			before: &Metadata{},
			update: &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait"}}},
			want:   &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait"}}},
		},
		{
			name:   "add another trait",
			before: &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait"}}},
			update: &Metadata{Traits: []*TraitMetadata{{Name: "AnotherTrait"}}},
			want:   &Metadata{Traits: []*TraitMetadata{{Name: "AnotherTrait"}, {Name: "SuperTrait"}}},
		},
		{
			name:   "add trait meta",
			before: &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait", More: map[string]string{"one": "1", "two": "2"}}}},
			update: &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait", More: map[string]string{"two": "II", "three": "3"}}}},
			want:   &Metadata{Traits: []*TraitMetadata{{Name: "SuperTrait", More: map[string]string{"one": "1", "two": "II", "three": "3"}}}},
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

// A read_mask path naming a key in the more map used to panic in fmutils and exit the process (SCB-1469).
func TestModel_GetMetadata_moreKeyMask(t *testing.T) {
	model := NewModel(resource.WithInitialValue(&Metadata{
		Name: "dev",
		More: map[string]string{"x": "1", "y": "2"},
	}))
	got, err := model.GetMetadata(resource.WithReadMask(&fieldmaskpb.FieldMask{Paths: []string{"more.x"}}))
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	want := &Metadata{More: map[string]string{"x": "1"}}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("GetMetadata (-want +got):\n%s", diff)
	}
}
