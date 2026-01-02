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
		wantPresent []childTrait
		wantAbsent  []childTrait
	}{
		{
			name: "new child",
			change: &gen.PullDevicesResponse_Change{
				NewValue: &gen.Device{
					Name: "child01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
						},
					},
				},
			},
			wantPresent: []childTrait{
				{name: "child01", trait: trait.OnOff},
				{name: "child01", trait: trait.Hail},
			},
		},
		{
			name: "child adds new trait",
			change: &gen.PullDevicesResponse_Change{
				NewValue: &gen.Device{
					Name: "child01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
							{Name: trait.Light.String()},
						},
					},
				},
			},
			wantPresent: []childTrait{
				{name: "child01", trait: trait.OnOff},
				{name: "child01", trait: trait.Hail},
				{name: "child01", trait: trait.Light},
			},
		},
		{
			name: "child removes trait",
			change: &gen.PullDevicesResponse_Change{
				OldValue: &gen.Device{
					Name: "child01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Hail.String()},
							{Name: trait.Light.String()},
						},
					},
				},
				NewValue: &gen.Device{
					Name: "child01",
					Metadata: &traits.Metadata{
						Traits: []*traits.TraitMetadata{
							{Name: trait.OnOff.String()},
							{Name: trait.Light.String()},
						},
					},
				},
			},
			wantPresent: []childTrait{
				{name: "child01", trait: trait.OnOff},
				{name: "child01", trait: trait.Light},
			},
			wantAbsent: []childTrait{
				{name: "child01", trait: trait.Hail},
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
