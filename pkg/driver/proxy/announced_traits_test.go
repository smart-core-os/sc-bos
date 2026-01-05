package proxy

import (
	"testing"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func TestAnnouncedTraits_UpdateDevice(t *testing.T) {
	tests := []struct {
		name           string
		initial        announcedTraits
		oldDevice      *gen.Device
		newDevice      *gen.Device
		wantAnnouncing []trait.Name
		wantRemaining  int
		wantDeleted    int
	}{
		{
			name:      "nil old, new device added",
			initial:   announcedTraits{},
			oldDevice: nil,
			newDevice: &gen.Device{
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
			oldDevice:      nil,
			newDevice:      nil,
			wantAnnouncing: nil,
			wantRemaining:  0,
		},
		{
			name: "old device removed",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			oldDevice: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
					},
				},
			},
			newDevice:      nil,
			wantAnnouncing: nil,
			wantRemaining:  0,
			wantDeleted:    2,
		},
		{
			name: "traits added to existing device",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
			},
			oldDevice: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
					},
				},
			},
			newDevice: &gen.Device{
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
			name: "traits removed from existing device",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}:      func() {},
				deviceTrait{name: "device1", trait: "Light"}:      func() {},
				deviceTrait{name: "device1", trait: "Brightness"}: func() {},
			},
			oldDevice: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
						{Name: "Brightness"},
					},
				},
			},
			newDevice: &gen.Device{
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

			got := a.updateDevice(tt.oldDevice, tt.newDevice)

			if !traitNamesEqual(got, tt.wantAnnouncing) {
				t.Errorf("updateDevice() = %v, want %v", got, tt.wantAnnouncing)
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
		checkKey      *deviceTrait
	}{
		{
			name: "deleteDevice removes all device traits",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
				deviceTrait{name: "device2", trait: "OnOff"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteDevice(&gen.Device{
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
			checkKey:      &deviceTrait{name: "device2", trait: "OnOff"},
		},
		{
			name: "deleteDeviceTrait removes single trait",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteDeviceTrait("device1", "OnOff")
				return 0
			},
			wantRemaining: 1,
			wantDeleted:   1,
			checkKey:      &deviceTrait{name: "device1", trait: "Light"},
		},
		{
			name: "deleteDeviceTrait non-existent does not panic",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			operation: func(a announcedTraits) int {
				a.deleteDeviceTrait("device1", "NonExistent")
				return 0
			},
			wantRemaining: 1,
			wantDeleted:   0,
		},
		{
			name: "deleteAll removes everything",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
				deviceTrait{name: "device2", trait: "OnOff"}: func() {},
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

	key := deviceTrait{name: "device1", trait: "OnOff"}
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
