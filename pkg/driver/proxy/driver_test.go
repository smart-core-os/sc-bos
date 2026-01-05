package proxy

import (
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func Test_proxy_announceChange(t *testing.T) {
	tests := []struct {
		name        string
		change      *gen.PullDevicesResponse_Change
		wantPresent []deviceTrait
		wantAbsent  []deviceTrait
	}{
		{
			name: "new device",
			change: &gen.PullDevicesResponse_Change{
				NewValue: &gen.Device{
					Name: "device01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
						},
					},
				},
			},
			wantPresent: []deviceTrait{
				{name: "device01", trait: trait.OnOff},
				{name: "device01", trait: trait.Hail},
			},
		},
		{
			name: "device adds new trait",
			change: &gen.PullDevicesResponse_Change{
				NewValue: &gen.Device{
					Name: "device01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
							{Name: trait.Light.String()},
						},
					},
				},
			},
			wantPresent: []deviceTrait{
				{name: "device01", trait: trait.OnOff},
				{name: "device01", trait: trait.Hail},
				{name: "device01", trait: trait.Light},
			},
		},
		{
			name: "device removes trait",
			change: &gen.PullDevicesResponse_Change{
				OldValue: &gen.Device{
					Name: "device01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
							{Name: trait.Light.String()},
						},
					},
				},
				NewValue: &gen.Device{
					Name: "device01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Light.String()},
						},
					},
				},
			},
			wantPresent: []deviceTrait{
				{name: "device01", trait: trait.OnOff},
				{name: "device01", trait: trait.Light},
			},
			wantAbsent: []deviceTrait{
				{name: "device01", trait: trait.Hail},
			},
		},
	}

	announcer := &testAnnouncer{}
	proxy := &proxy{
		announcer: announcer,
		logger:    zap.NewNop(),
	}
	known := announcedTraits{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy.announceChange(known, tt.change)

			for _, ct := range tt.wantPresent {
				if _, ok := known[ct]; !ok {
					t.Errorf("expected %s:%s to be remembered, got %v", ct.name, ct.trait, known)
				}
			}

			for _, ct := range tt.wantAbsent {
				if _, ok := known[ct]; ok {
					t.Errorf("expected %s:%s to be forgotten, got %v", ct.name, ct.trait, known)
				}
			}
		})
	}
}

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

func TestAnnouncedTraits_DeleteDevice(t *testing.T) {
	tests := []struct {
		name          string
		initial       announcedTraits
		device        *gen.Device
		wantRemaining int
		wantDeleted   int
		checkKey      *deviceTrait
	}{
		{
			name: "removes all device traits",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
				deviceTrait{name: "device2", trait: "OnOff"}: func() {},
			},
			device: &gen.Device{
				Name: "device1",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{
						{Name: "OnOff"},
						{Name: "Light"},
					},
				},
			},
			wantRemaining: 1,
			wantDeleted:   2,
			checkKey:      &deviceTrait{name: "device2", trait: "OnOff"},
		},
		{
			name: "deletes nothing for device with no traits",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
			},
			device: &gen.Device{
				Name: "device2",
				Metadata: &traits.Metadata{
					Traits: []*traits.TraitMetadata{},
				},
			},
			wantRemaining: 1,
			wantDeleted:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCount := 0
			a := make(announcedTraits)
			for k := range tt.initial {
				a[k] = func() { deleteCount++ }
			}

			a.deleteDevice(tt.device)

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

func TestAnnouncedTraits_DeleteDeviceTrait(t *testing.T) {
	tests := []struct {
		name          string
		initial       announcedTraits
		deviceName    string
		traitName     trait.Name
		wantRemaining int
		wantDeleted   int
		checkKey      *deviceTrait
	}{
		{
			name: "removes single trait",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			deviceName:    "device1",
			traitName:     "OnOff",
			wantRemaining: 1,
			wantDeleted:   1,
			checkKey:      &deviceTrait{name: "device1", trait: "Light"},
		},
		{
			name: "non-existent trait does not panic",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			deviceName:    "device1",
			traitName:     "NonExistent",
			wantRemaining: 1,
			wantDeleted:   0,
		},
		{
			name: "non-existent device does not panic",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "Light"}: func() {},
			},
			deviceName:    "device2",
			traitName:     "Light",
			wantRemaining: 1,
			wantDeleted:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCount := 0
			a := make(announcedTraits)
			for k := range tt.initial {
				a[k] = func() { deleteCount++ }
			}

			a.deleteDeviceTrait(tt.deviceName, tt.traitName)

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

func TestAnnouncedTraits_DeleteAll(t *testing.T) {
	tests := []struct {
		name        string
		initial     announcedTraits
		wantDeleted int
	}{
		{
			name: "removes everything",
			initial: announcedTraits{
				deviceTrait{name: "device1", trait: "OnOff"}: func() {},
				deviceTrait{name: "device1", trait: "Light"}: func() {},
				deviceTrait{name: "device2", trait: "OnOff"}: func() {},
			},
			wantDeleted: 3,
		},
		{
			name:        "empty map remains empty",
			initial:     announcedTraits{},
			wantDeleted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCount := 0
			a := make(announcedTraits)
			for k := range tt.initial {
				a[k] = func() { deleteCount++ }
			}

			a.deleteAll()

			if len(a) != 0 {
				t.Errorf("remaining traits = %d, want 0", len(a))
			}

			if deleteCount != tt.wantDeleted {
				t.Errorf("deleted traits = %d, want %d", deleteCount, tt.wantDeleted)
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

type testAnnouncer []*announcement

func (t *testAnnouncer) Announce(name string, features ...node.Feature) node.Undo {
	an := &announcement{name: name, features: features}
	*t = append(*t, an)
	return func() {
		an.undone++
	}
}

type announcement struct {
	name     string
	features []node.Feature
	undone   int
}
