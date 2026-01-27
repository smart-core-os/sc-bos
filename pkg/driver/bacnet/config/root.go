package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
)

// Root represents a full collection of related configuration properties
type Root struct {
	driver.BaseConfig

	LocalInterface string `json:"localInterface,omitempty"`
	LocalPort      uint16 `json:"localPort,omitempty"`

	MaxConcurrentTransactions uint8 `json:"maxConcurrentTransactions,omitempty"`

	Discovery                *Discovery `json:"discovery,omitempty"`
	ForceDiscovery           bool       `json:"forceDiscovery,omitempty"`
	IncludeDiscoveredDevices bool       `json:"includeDiscoveredDevices,omitempty"`

	DeviceNamePrefix string `json:"deviceNamePrefix"` // defaults to "bacnet/device/" if absent in json or using Defaults
	ObjectNamePrefix string `json:"objectNamePrefix"` // defaults to "obj/" if absent in json, or using Defaults

	DiscoverObjects bool `json:"discoverObjects,omitempty"`

	COV *COV `json:"cov,omitempty"`

	// Metadata is applied to all announced names.
	Metadata *traits.Metadata `json:"metadata,omitempty"`

	Devices []Device   `json:"devices,omitempty"`
	Traits  []RawTrait `json:"traits,omitempty"`

	// SystemHealth represents the system-level health check configuration. If not configured,
	// occupant and equipment impact will default to UNSPECIFIED.
	SystemHealth Health `json:"systemHealth,omitempty"`
}

// ReadFile reads from the named file a config Root.
func ReadFile(name string) (Root, error) {
	root := Defaults()
	bytes, err := os.ReadFile(name)
	if err != nil {
		return root, err
	}
	err = json.Unmarshal(bytes, &root)
	return root, err
}

// Read decodes r into a config Root.
func Read(r io.Reader) (Root, error) {
	root := Defaults()
	err := json.NewDecoder(r).Decode(&root)
	return root, err
}

// ReadBytes decodes bytes into a config Root.
func ReadBytes(data []byte) (root Root, err error) {
	return Read(bytes.NewReader(data))
}

func Defaults() Root {
	return Root{
		DeviceNamePrefix: "bacnet/device/",
		ObjectNamePrefix: "obj/",
	}
}

type Discovery struct {
	Min        int      `json:"min,omitempty"`
	Max        int      `json:"max,omitempty"`
	Chunk      int      `json:"chunk,omitempty"`
	ChunkDelay Duration `json:"chunkDelay,omitempty"`
}

type Device struct {
	Name  string                  `json:"name,omitempty"`
	Title string                  `json:"title,omitempty"`
	Comm  *Comm                   `json:"comm,omitempty"`
	ID    bactypes.ObjectInstance `json:"id,omitempty"`

	COV *COV `json:"cov,omitempty"`

	// Metadata applied to any traits sharing this devices name.
	Metadata *traits.Metadata `json:"metadata,omitempty"`

	DiscoverObjects      *bool    `json:"discoverObjects,omitempty"`
	Objects              []Object `json:"objects,omitempty"`
	DefaultWritePriority uint     `json:"defaultWritePriority,omitempty"`

	// Health contains settings for a BACnet device health check
	// If not configured, the occupant and equipment impact will default to UNSPECIFIED
	Health Health `json:"health"`
}

type Comm struct {
	IP *netip.AddrPort `json:"ip,omitempty"`
	// the destination BACnet network
	// this is used if the target BACnet device is not IP, but being proxied through a BACnet IP router
	// see: http://www.bacnetwiki.com/wiki/index.php?title=Network_Layer_Protocol_Data_Unit
	Destination *Destination `json:"destination,omitempty"`
}

func (c *Comm) ToAddress() (bactypes.Address, error) {
	var addr bactypes.Address
	if c.IP != nil {
		udpAddr := net.UDPAddrFromAddrPort(*c.IP)
		addr = bactypes.UDPToAddress(udpAddr)
	}

	if c.Destination != nil {
		addr.Net = c.Destination.Network
		if c.Destination.Address != "" {
			dadr, err := c.Destination.Address.Bytes()
			if err != nil {
				return bactypes.Address{}, nil
			}
			addr.Adr = dadr
			addr.Len = uint8(len(dadr))
		}
	}

	return addr, nil
}

// Destination representation BACnet NPDU destination settings
// see: http://www.bacnetwiki.com/wiki/index.php?title=Network_Layer_Protocol_Data_Unit
type Destination struct {
	// Network is the Destination Network 1-65534, or 65535 for broadcast
	// http://www.bacnetwiki.com/wiki/index.php?title=Destination_Network
	Network uint16 `json:"network,omitempty"`
	// Address is in dot-notation when multiple octets
	// could be upto 18, but in practice 6 or less
	// http://www.bacnetwiki.com/wiki/index.php?title=MAC_Layer_Address
	Address DestinationAddress `json:"address,omitempty"`
}

type DestinationAddress string

func (d DestinationAddress) Bytes() ([]byte, error) {
	octets := strings.Split(string(d), ".")
	var addr []byte
	if d == "" {
		return addr, nil
	}
	for i := 0; i < len(octets); i++ {
		octet := octets[i]
		value, err := strconv.ParseUint(octet, 10, 8)
		if err != nil {
			return nil, err
		}
		addr = append(addr, byte(value))
	}
	return addr, nil
}

var Blocks = []block.Block{
	{Path: []string{"metadata"}, Blocks: mdblock.Categories},
	{
		Path: []string{"devices"},
		Key:  "id",
		Blocks: []block.Block{
			{Path: []string{"metadata"}, Blocks: mdblock.Categories},
			{
				Path: []string{"objects"},
				Key:  "id",
				Blocks: []block.Block{
					{Path: []string{"metadata"}, Blocks: mdblock.Categories},
				},
			},
		},
	},
	{
		Path: []string{"traits"},
		Key:  "name",
		Blocks: []block.Block{
			{Path: []string{"metadata"}, Blocks: mdblock.Categories},
		},
	},
}
