package confmerge

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/block"
)

// a made-up example of a config struct
type sampleConfig struct {
	Name    string         `json:"name"`
	Drivers []sampleDriver `json:"drivers"`
}

type sampleDriver struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Devices []sampleDevice `json:"devices"`
}

type sampleDevice struct {
	Name string `json:"name"`
	Addr string `json:"addr,omitempty"`
}

var sampleBlocks = []block.Block{
	{
		Path:    []string{"drivers"},
		Key:     "name",
		TypeKey: "type",
		BlocksByType: map[string][]block.Block{
			"driverType1": {
				{
					Path: []string{"devices"},
					Key:  "name",
					Blocks: []block.Block{
						{
							Path: []string{"addr"},
						},
					},
				},
			},
		},
	},
}

func TestBootConfig(t *testing.T) {
	store := &memStore{}
	mutate := func(mutator func(*sampleConfig)) {
		active, _, err := getActiveJSON[sampleConfig](store)
		if err != nil {
			t.Fatal(err)
		}
		mutator(&active)
		err = setActiveJSON(store, active)
		if err != nil {
			t.Fatal(err)
		}
	}

	// boot with an empty store - this should just return the local config
	conf, _, err := Merge(sampleConfig{
		Name: "test",
		Drivers: []sampleDriver{
			{
				Name: "driver1",
				Type: "driverType1",
				Devices: []sampleDevice{
					{Name: "device1", Addr: "addr1"},
					{Name: "device2", Addr: "addr2"},
				},
			},
		},
	}, store, sampleBlocks)
	if err != nil {
		t.Fatal(err)
	}

	expect := sampleConfig{
		Name: "test",
		Drivers: []sampleDriver{
			{
				Name: "driver1",
				Type: "driverType1",
				Devices: []sampleDevice{
					{Name: "device1", Addr: "addr1"},
					{Name: "device2", Addr: "addr2"},
				},
			},
		},
	}
	diff := cmp.Diff(expect, conf,
		cmpopts.EquateEmpty(),
		protocmp.Transform(),
	)
	if diff != "" {
		t.Errorf("(1) unexpected config (-want +got):\n%s", diff)
	}

	// modify the active config directly - this simulates making live changes
	// change the addr of a single device
	mutate(func(c *sampleConfig) {
		c.Drivers[0].Devices = []sampleDevice{
			{Name: "device1", Addr: "addr1"},
			{Name: "device2", Addr: "addr2changed"},
		}
	})

	// modify the local config with a patch adds another device to driver1
	conf, _, err = Merge(sampleConfig{
		Name: "test",
		Drivers: []sampleDriver{
			{
				Name: "driver1",
				Type: "driverType1",
				Devices: []sampleDevice{
					{Name: "device1", Addr: "addr1"},
					{Name: "device2", Addr: "addr2"},
					{Name: "device3", Addr: "addr3"},
				},
			},
		},
	}, store, sampleBlocks)
	if err != nil {
		t.Fatal(err)
	}

	// check that both changes from local config were applied to the active config
	expect = sampleConfig{
		Name: "test",
		Drivers: []sampleDriver{
			{
				Name: "driver1",
				Type: "driverType1",
				Devices: []sampleDevice{
					{Name: "device1", Addr: "addr1"},
					{Name: "device2", Addr: "addr2changed"},
					{Name: "device3", Addr: "addr3"},
				},
			},
		},
	}

	diff = cmp.Diff(expect, conf,
		cmpopts.EquateEmpty(),
		protocmp.Transform(),
	)
	if diff != "" {
		t.Errorf("(2) unexpected config (-want +got):\n%s", diff)
	}
}

type memStore struct {
	local  []byte
	active []byte
}

func (m *memStore) GetExternalConfig() ([]byte, error) {
	return bytes.Clone(m.local), nil
}

func (m *memStore) SetExternalConfig(c []byte) error {
	m.local = bytes.Clone(c)
	return nil
}

func (m *memStore) GetActiveConfig() ([]byte, error) {
	return bytes.Clone(m.active), nil
}

func (m *memStore) SetActiveConfig(c []byte) error {
	m.active = bytes.Clone(c)
	return nil
}
