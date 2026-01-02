package proxy

import (
	"testing"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func TestAnnouncedTraits_UpdateChild(t *testing.T) {
	tests := []struct {
		name           string
		initial        announcedTraits
		oldChild       *gen.Device
		newChild       *gen.Device
		wantAnnouncing []trait.Name
		wantRemaining  int
		wantDeleted    int
	}{
		{
			name:     "nil old, new child added",
			initial:  announcedTraits{},
			oldChild: nil,
			newChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
					},
				},
			},
			wantAnnouncing: []trait.Name{"OnOff", "Light"},
			wantRemaining:  0,
		},
		{
			name:           "both nil",
			initial:        announcedTraits{},
			oldChild:       nil,
			newChild:       nil,
			wantAnnouncing: nil,
			wantRemaining:  0,
		},
		{
			name: "old child removed",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}: func() {},
				childTrait{name: "device1", trait: "Light"}: func() {},
			},
			oldChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
					},
				},
			},
			newChild:       nil,
			wantAnnouncing: nil,
			wantRemaining:  0,
			wantDeleted:    2,
		},
		{
			name: "traits added to existing child",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}: func() {},
			},
			oldChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
					},
				},
			},
			newChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
						{Name: "Brightness"},
					},
				},
			},
			wantAnnouncing: []trait.Name{"Light", "Brightness"},
			wantRemaining:  1,
		},
		{
			name: "traits removed from existing child",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}:      func() {},
				childTrait{name: "device1", trait: "Light"}:      func() {},
				childTrait{name: "device1", trait: "Brightness"}: func() {},
			},
			oldChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
						{Name: "Brightness"},
					},
				},
			},
			newChild: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
					},
				},
			},
			wantAnnouncing: nil,
			wantRemaining:  1,
			wantDeleted:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := make(announcedTraits)
			deleteCount := 0
			for k := range tt.initial {
				a[k] = func() { deleteCount++ }
			}

			got := a.updateChild(tt.oldChild, tt.newChild)

			if !traitNamesEqual(got, tt.wantAnnouncing) {
				t.Errorf("updateChild() = %v, want %v", got, tt.wantAnnouncing)
			}

			if len(a) != tt.wantRemaining {
				t.Errorf("remaining traits = %d, want %d", len(a), tt.wantRemaining)
			}

			if deleteCount != tt.wantDeleted {
				t.Errorf("deleted traits = %d, want %d", deleteCount, tt.wantDeleted)
			}
		})
	}
}

func TestAnnouncedTraits_DeleteOperations(t *testing.T) {
	tests := []struct {
		name          string
		initial       announcedTraits
		operation     func(announcedTraits) int
		wantRemaining int
		wantDeleted   int
		checkKey      *childTrait
	}{
		{
			name: "deleteChild removes all child traits",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}: func() {},
				childTrait{name: "device1", trait: "Light"}: func() {},
				childTrait{name: "device2", trait: "OnOff"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteChild(&gen.Device{
					Name: "device1",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: "OnOff"},
							{Name: "Light"},
						},
					},
				})
				return 0
			},
			wantRemaining: 1,
			wantDeleted:   2,
			checkKey:      &childTrait{name: "device2", trait: "OnOff"},
		},
		{
			name: "deleteChildTrait removes single trait",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}: func() {},
				childTrait{name: "device1", trait: "Light"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteChildTrait("device1", "OnOff")
				return 0
			},
			wantRemaining: 1,
			wantDeleted:   1,
			checkKey:      &childTrait{name: "device1", trait: "Light"},
		},
		{
			name: "deleteChildTrait non-existent does not panic",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "Light"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteChildTrait("device1", "NonExistent")
				return 0
			},
			wantRemaining: 1,
			wantDeleted:   0,
		},
		{
			name: "deleteAll removes everything",
			initial: announcedTraits{
				childTrait{name: "device1", trait: "OnOff"}: func() {},
				childTrait{name: "device1", trait: "Light"}: func() {},
				childTrait{name: "device2", trait: "OnOff"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteAll()
				return 0
			},
			wantRemaining: 0,
			wantDeleted:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCount := 0
			a := make(announcedTraits)
			for k := range tt.initial {
				a[k] = func() { deleteCount++ }
			}

			tt.operation(a)

			if len(a) != tt.wantRemaining {
				t.Errorf("remaining traits = %d, want %d", len(a), tt.wantRemaining)
			}

			if deleteCount != tt.wantDeleted {
				t.Errorf("deleted traits = %d, want %d", deleteCount, tt.wantDeleted)
			}

			if tt.checkKey != nil {
				if _, ok := a[*tt.checkKey]; !ok {
					t.Errorf("expected %v to still exist", *tt.checkKey)
				}
			}
		})
	}
}

func TestAnnouncedTraits_Add(t *testing.T) {
	a := announcedTraits{}
	called := false
	undo := func() { called = true }

	a.add("device1", "OnOff", undo)

	if len(a) != 1 {
		t.Errorf("len(a) = %d, want 1", len(a))
	}

	key := childTrait{name: "device1", trait: "OnOff"}
	if fn, ok := a[key]; !ok {
		t.Error("trait not added")
	} else {
		fn()
		if !called {
			t.Error("undo function not called")
		}
	}
}

func traitNamesEqual(a, b []trait.Name) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	seen := make(map[trait.Name]int)
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}
