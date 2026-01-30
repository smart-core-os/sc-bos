package known

import (
	"errors"
	"fmt"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrUnknownDevice = fmt.Errorf("device %w", ErrNotFound)
)

// Map holds information about devices and objects.
// Map implements Context.
type Map struct {
	devicesByID   map[bactypes.ObjectInstance]*device
	devicesByName map[string]*device
}

func NewMap() *Map {
	return &Map{}
}

func (m *Map) StoreDevice(name string, d bactypes.Device, defaultWritePriority uint) {
	if m.devicesByID == nil {
		m.devicesByID = make(map[bactypes.ObjectInstance]*device)
		m.devicesByName = make(map[string]*device)
	}

	var item *device
	if e, ok := m.devicesByID[d.ID.Instance]; ok {
		item = e
	} else {
		item = &device{
			bacDevice:            d,
			objectsByID:          make(map[bactypes.ObjectID]*object),
			objectsByName:        make(map[string]*object),
			defaultWritePriority: defaultWritePriority,
		}
		m.devicesByID[d.ID.Instance] = item
	}

	item.names = append(item.names, name)
	m.devicesByName[name] = item
}

func (m *Map) StoreObject(d bactypes.Device, name string, o bactypes.Object) error {
	item, ok := m.devicesByID[d.ID.Instance]
	if !ok {
		return ErrUnknownDevice
	}
	obj := &object{
		bacObject: o,
		name:      name,
	}
	item.objectsByID[o.ID] = obj
	item.objectsByName[name] = obj
	return nil
}

func (m *Map) DeleteDevice(device bactypes.Device) {
	m.DeleteDeviceByID(device.ID.Instance)
}

func (m *Map) DeleteDeviceByID(id bactypes.ObjectInstance) {
	d, ok := m.devicesByID[id]
	if !ok {
		return
	}
	delete(m.devicesByID, d.bacDevice.ID.Instance)
	for _, name := range d.names {
		delete(m.devicesByName, name)
	}
}

func (m *Map) DeleteDeviceByName(name string) {
	d, ok := m.devicesByName[name]
	if !ok {
		return
	}
	for i, s := range d.names {
		if s == name {
			d.names = append(d.names[:i], d.names[i+1:]...)
			break
		}
	}
	delete(m.devicesByName, name)
	if len(d.names) == 0 {
		delete(m.devicesByID, d.bacDevice.ID.Instance)
	}
}

func (m *Map) DeleteObject(device bactypes.Device, object bactypes.Object) {
	m.DeleteObjectByID(device, object.ID)
}

func (m *Map) DeleteObjectByID(device bactypes.Device, id bactypes.ObjectID) {
	d, ok := m.devicesByID[device.ID.Instance]
	if !ok {
		return
	}
	o, ok := d.objectsByID[id]
	if !ok {
		return
	}
	delete(d.objectsByID, o.bacObject.ID)
	delete(d.objectsByName, o.name)
}

func (m *Map) DeleteObjectByName(device bactypes.Device, name string) {
	d, ok := m.devicesByID[device.ID.Instance]
	if !ok {
		return
	}
	o, ok := d.objectsByName[name]
	if !ok {
		return
	}
	delete(d.objectsByID, o.bacObject.ID)
	delete(d.objectsByName, o.name)
}

func (m *Map) Clear() {
	m.devicesByID = nil
	m.devicesByName = nil
}

func (m *Map) ListObjects(device bactypes.Device) ([]bactypes.Object, error) {
	d, ok := m.devicesByID[device.ID.Instance]
	if !ok {
		return nil, ErrUnknownDevice
	}
	if len(d.objectsByID) == 0 {
		return nil, nil
	}
	res := make([]bactypes.Object, 0, len(d.objectsByID))
	for _, o := range d.objectsByID {
		res = append(res, o.bacObject)
	}
	return res, nil
}

func (m *Map) LookupDeviceByID(id bactypes.ObjectInstance) (bactypes.Device, error) {
	d, ok := m.devicesByID[id]
	if !ok {
		return bactypes.Device{}, ErrNotFound
	}
	return d.bacDevice, nil
}

func (m *Map) LookupDeviceByName(name string) (bactypes.Device, error) {
	d, ok := m.devicesByName[name]
	if !ok {
		return bactypes.Device{}, ErrNotFound
	}
	return d.bacDevice, nil
}

func (m *Map) LookupObjectByID(device bactypes.Device, id bactypes.ObjectID) (bactypes.Object, error) {
	d, ok := m.devicesByID[device.ID.Instance]
	if !ok {
		return bactypes.Object{}, ErrUnknownDevice
	}
	o, ok := d.objectsByID[id]
	if !ok {
		return bactypes.Object{}, ErrNotFound
	}
	return o.bacObject, nil
}

func (m *Map) LookupObjectByName(device bactypes.Device, name string) (bactypes.Object, error) {
	d, ok := m.devicesByID[device.ID.Instance]
	if !ok {
		return bactypes.Object{}, ErrUnknownDevice
	}
	o, ok := d.objectsByName[name]
	if !ok {
		return bactypes.Object{}, ErrNotFound
	}
	return o.bacObject, nil
}

func (m *Map) GetDeviceDefaultWritePriority(id bactypes.ObjectInstance) uint {
	d, ok := m.devicesByID[id]
	if !ok {
		return 0
	}
	return d.defaultWritePriority
}

type device struct {
	bacDevice bactypes.Device
	names     []string

	objectsByID          map[bactypes.ObjectID]*object
	objectsByName        map[string]*object
	defaultWritePriority uint
}

type object struct {
	bacObject bactypes.Object
	name      string
}
