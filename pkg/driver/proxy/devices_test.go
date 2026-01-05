package proxy

import (
	"context"
	"testing"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	devicesmanage "github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func TestDeviceFetcher_Poll(t *testing.T) {
	tests := []struct {
		name      string
		initial   map[string]*gen.Device
		devices   []*gen.Device
		wantAdds  []string
		wantUpdts []string
		wantRems  []string
		wantErr   bool
	}{
		{
			name:     "empty response",
			initial:  nil,
			devices:  nil,
			wantAdds: nil,
		},
		{
			name:    "new devices added",
			initial: nil,
			devices: []*gen.Device{
				{Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
				{Name: "device2", Metadata: &traits.Metadata{
					Name: "device2",
				}},
			},
			wantAdds: []string{"device1", "device2"},
		},
		{
			name: "devices updated",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
			},
			devices: []*gen.Device{
				{Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}, {Name: "OnOff"}},
				}},
			},
			wantUpdts: []string{"device1"},
		},
		{
			name: "devices removed",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
				"device2": {Name: "device2", Metadata: &traits.Metadata{
					Name:   "device2",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
			},
			devices: []*gen.Device{
				{Name: "device1", Metadata: &traits.Metadata{
					Name: "device1",
				}},
			},
			wantRems: []string{"device2"},
		},
		{
			name: "no change if proto equal",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
			},
			devices: []*gen.Device{
				{Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
			},
		},
		{
			name:    "paginated response",
			initial: nil,
			devices: []*gen.Device{
				{Name: "device1", Metadata: &traits.Metadata{
					Name:   "device1",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
				{Name: "device2", Metadata: &traits.Metadata{
					Name:   "device2",
					Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}},
				}},
			},
			wantAdds: []string{"device1", "device2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := node.New("test")

			if len(tt.devices) > 0 {
				addDevices(n, tt.devices...)
			}

			client := gen.WrapDevicesApi(devicesmanage.NewServer(n))

			initial := tt.initial

			fetcher := &deviceFetcher{
				client: client,
				known:  initial,
			}

			ctx := context.Background()
			changes := make(chan *gen.PullDevicesResponse_Change, 10)

			err := fetcher.Poll(ctx, changes)
			close(changes)

			if (err != nil) != tt.wantErr {
				t.Errorf("Poll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var gotAdds []string
			var gotUpdts []string
			var gotRems []string

			for change := range changes {
				var name string
				switch change.Type {
				case types.ChangeType_ADD:
					name = change.NewValue.Name
					if name != "test" { // filter out the root node
						gotAdds = append(gotAdds, name)
					}
				case types.ChangeType_UPDATE:
					name = change.NewValue.Name
					if name != "test" {
						gotUpdts = append(gotUpdts, name)
					}
				case types.ChangeType_REMOVE:
					name = change.OldValue.Name
					if name != "test" {
						gotRems = append(gotRems, name)
					}
				}
			}

			if !stringSlicesEqual(gotAdds, tt.wantAdds) {
				t.Errorf("Poll() adds = %v, want %v", gotAdds, tt.wantAdds)
			}
			if !stringSlicesEqual(gotUpdts, tt.wantUpdts) {
				t.Errorf("Poll() updates = %v, want %v", gotUpdts, tt.wantUpdts)
			}
			if !stringSlicesEqual(gotRems, tt.wantRems) {
				t.Errorf("Poll() removes = %v, want %v", gotRems, tt.wantRems)
			}
		})
	}
}

func TestDeviceFetcher_Poll_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n := node.New("test")
	client := gen.WrapDevicesApi(devicesmanage.NewServer(n))

	fetcher := &deviceFetcher{
		client: client,
	}

	changes := make(chan *gen.PullDevicesResponse_Change, 10)

	err := fetcher.Poll(ctx, changes)
	if err == nil {
		t.Error("Poll() expected error, got nil")
	}
}

func TestDeviceFetcher_Pull(t *testing.T) {
	t.Run("streams changes", func(t *testing.T) {
		n := node.New("test")
		client := gen.WrapDevicesApi(devicesmanage.NewServer(n))

		fetcher := &deviceFetcher{
			client: client,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		changes := make(chan *gen.PullDevicesResponse_Change, 10)

		errCh := make(chan error, 1)
		go func() {
			errCh <- fetcher.Pull(ctx, changes)
		}()

		readNextChange := func() *gen.PullDevicesResponse_Change {
			for {
				change := <-changes
				if change.NewValue != nil && change.NewValue.Name != "test" {
					return change
				}
			}
		}

		n.Announce("device1", node.HasMetadata(&traits.Metadata{}))
		change1 := readNextChange()
		if change1.Type != types.ChangeType_ADD {
			t.Errorf("expected ADD change for device1, got %v", change1.Type)
		}
		if change1.NewValue.Name != "device1" {
			t.Errorf("expected device1, got %v", change1.NewValue.Name)
		}

		n.Announce("device1", node.HasMetadata(&traits.Metadata{
			Traits: []*traits.TraitMetadata{{Name: "OnOff"}},
		}), node.HasTrait(trait.OnOff))
		change2 := readNextChange()
		if change2.Type != types.ChangeType_ADD && change2.Type != types.ChangeType_UPDATE {
			t.Errorf("expected ADD or UPDATE change for device1 update, got %v", change2.Type)
		}

		cancel()
		<-errCh
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		n := node.New("test")
		client := gen.WrapDevicesApi(devicesmanage.NewServer(n))

		fetcher := &deviceFetcher{
			client: client,
		}

		changes := make(chan *gen.PullDevicesResponse_Change, 10)

		err := fetcher.Pull(ctx, changes)
		if err == nil {
			t.Error("Pull() expected error with canceled context, got nil")
		}
	})
}

func addDevices(n *node.Node, devices ...*gen.Device) {

	for _, device := range devices {
		opts := []node.Feature{
			node.HasMetadata(device.Metadata),
		}
		for _, t := range device.Metadata.Traits {
			opts = append(opts, node.HasTrait(trait.Name(t.Name)))
		}
		n.Announce(device.Name, opts...)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	seen := make(map[string]int)
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
