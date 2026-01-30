package known

import bactypes "github.com/smart-core-os/gobacnet/types"

// Context describes what we know about a bacnet system.
type Context interface {
	ListObjects(device bactypes.Device) ([]bactypes.Object, error)
	LookupDeviceByID(id bactypes.ObjectInstance) (bactypes.Device, error)
	LookupDeviceByName(name string) (bactypes.Device, error)
	LookupObjectByID(device bactypes.Device, id bactypes.ObjectID) (bactypes.Object, error)
	LookupObjectByName(device bactypes.Device, name string) (bactypes.Object, error)
	GetDeviceDefaultWritePriority(id bactypes.ObjectInstance) uint
}
