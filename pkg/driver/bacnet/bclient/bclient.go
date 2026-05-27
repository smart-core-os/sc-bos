// Package bclient defines the BACnet client interface used by the bacnet driver.
//
// The driver historically used the concrete *gobacnet.Client, which only speaks
// BACnet/IP (UDP). To allow alternative data links - notably BACnet/SC (secure
// websockets, see package sc) - the driver and its adapt/merge helpers depend on
// this interface instead. Both *gobacnet.Client and sc.Client satisfy it.
package bclient

import (
	"context"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

// Client is the subset of gobacnet.Client behaviour the bacnet driver relies on.
// It is transport agnostic: BACnet/IP and BACnet/SC implementations both satisfy it.
type Client interface {
	// ReadProperty reads a single property from a single object on the device.
	ReadProperty(ctx context.Context, dest bactypes.Device, rp bactypes.ReadPropertyData) (bactypes.ReadPropertyData, error)
	// ReadMultiProperty reads multiple properties from the device in a single request.
	ReadMultiProperty(ctx context.Context, dev bactypes.Device, rp bactypes.ReadMultipleProperty) (bactypes.ReadMultipleProperty, error)
	// WriteProperty writes a single property to a single object on the device.
	WriteProperty(ctx context.Context, dest bactypes.Device, wp bactypes.ReadPropertyData, priority uint) error
	// WhoIs broadcasts a Who-Is and collects the I-Am responses with device IDs in [low, high].
	WhoIs(ctx context.Context, low, high int) ([]bactypes.Device, error)
	// RemoteDevices reads device details for the given device instances at addr.
	RemoteDevices(ctx context.Context, addr bactypes.Address, ids ...bactypes.ObjectInstance) ([]bactypes.Device, error)
	// Objects enumerates the objects present on dev, populating dev.Objects.
	Objects(ctx context.Context, dev bactypes.Device) (bactypes.Device, error)
	// Close releases the client's network resources.
	Close()
}
