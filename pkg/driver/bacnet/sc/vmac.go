package sc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

// VMAC is a BACnet/SC 6-octet virtual MAC address (ASHRAE 135-2020 Clause AB.1.5.2).
type VMAC [6]byte

// BroadcastVMAC is the local broadcast virtual address. A node sends to this
// address to have the hub distribute a message to all connected nodes.
var BroadcastVMAC = VMAC{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// zeroVMAC is reserved and must not be used as a node address (Clause AB.1.5.2).
var zeroVMAC = VMAC{}

// RandomVMAC returns a cryptographically random VMAC that is neither the
// reserved zero address nor the broadcast address.
func RandomVMAC() (VMAC, error) {
	for {
		var v VMAC
		if _, err := rand.Read(v[:]); err != nil {
			return VMAC{}, err
		}
		if v != zeroVMAC && v != BroadcastVMAC {
			return v, nil
		}
	}
}

// ParseVMAC parses 12 hex characters, optionally colon or hyphen separated,
// e.g. "010203040506" or "01:02:03:04:05:06".
func ParseVMAC(s string) (VMAC, error) {
	clean := strings.NewReplacer(":", "", "-", "", " ", "").Replace(s)
	b, err := hex.DecodeString(clean)
	if err != nil {
		return VMAC{}, fmt.Errorf("vmac %q: %w", s, err)
	}
	if len(b) != 6 {
		return VMAC{}, fmt.Errorf("vmac %q: expected 6 octets, got %d", s, len(b))
	}
	var v VMAC
	copy(v[:], b)
	return v, nil
}

func (v VMAC) String() string {
	return hex.EncodeToString(v[:])
}

func (v VMAC) isBroadcast() bool { return v == BroadcastVMAC }

// addressToVMAC derives the destination VMAC for a gobacnet address.
// A broadcast address maps to BroadcastVMAC; otherwise the 6-octet MAC of the
// address (populated from the originating VMAC of received messages) is used.
func addressToVMAC(addr bactypes.Address) VMAC {
	if addr.IsBroadcast() || addr.IsSubBroadcast() {
		return BroadcastVMAC
	}
	var v VMAC
	if len(addr.Mac) == 6 {
		copy(v[:], addr.Mac)
	}
	return v
}

// vmacToAddress builds a gobacnet address that carries vmac in its MAC field.
// net/adr describe the (optional) downstream BACnet network the device sits on,
// taken from the NPDU source when the device is reached via a router.
func vmacToAddress(vmac VMAC, net uint16, adr []byte) bactypes.Address {
	a := bactypes.Address{
		MacLen: 6,
		Mac:    append([]byte(nil), vmac[:]...),
		Net:    net,
	}
	if len(adr) > 0 {
		a.Adr = append([]byte(nil), adr...)
		a.Len = uint8(len(adr))
	}
	return a
}
