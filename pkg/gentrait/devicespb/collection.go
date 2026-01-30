package devicespb

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/metadatapb"
)

// Collection is a list of unique devices.
type Collection struct {
	mu    sync.RWMutex
	names *resource.Collection // of *devicespb.Device, keyed by device name
}

func NewCollection(opts ...resource.Option) *Collection {
	return &Collection{
		names: resource.NewCollection(opts...),
	}
}

// ListDevices returns a slice containing all devices in the collection.
// The devices will be ordered by their name, ascending.
// Use opts for filter and project the returned devices.
func (c *Collection) ListDevices(opts ...resource.ReadOption) []*devicespb.Device {
	c.mu.RLock()
	res := c.names.List(opts...)
	c.mu.RUnlock()
	devices := make([]*devicespb.Device, 0, len(res))
	for _, r := range res {
		devices = append(devices, r.(*devicespb.Device))
	}
	return devices
}

type DevicesChange = resources.CollectionChange[*devicespb.Device]

// PullDevices returns a channel that will receive changes to the devices in the collection.
// Unless [resource.WithUpdatesOnly] is true, the channel will receive all current devices as ADDs.
func (c *Collection) PullDevices(ctx context.Context, opts ...resource.ReadOption) <-chan resources.CollectionChange[*devicespb.Device] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return resources.PullCollection[*devicespb.Device](ctx, c.names.Pull(ctx, opts...))
}

// GetDevice returns the device with the given name from the collection.
// If the device does not exist, an error with codes.NotFound will be returned.
func (c *Collection) GetDevice(name string, opts ...resource.ReadOption) (*devicespb.Device, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res, ok := c.names.Get(name, opts...)
	if !ok {
		return nil, status.Error(codes.NotFound, name)
	}
	return res.(*devicespb.Device), nil
}

type DeviceChange = resources.ValueChange[*devicespb.Device]

// PullDevice returns a channel that will receive changes to the device with the given name.
// If the device is deleted, the channel will close.
func (c *Collection) PullDevice(ctx context.Context, name string, opts ...resource.ReadOption) <-chan resources.ValueChange[*devicespb.Device] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return resources.PullValue[*devicespb.Device](ctx, c.names.PullID(ctx, name, opts...))
}

// Merge merges d with the existing device in the collection.
// An error will be returned if the device does not exist, and resource.WithCreateIfAbsent is not in opts.
// nil fields in d, like Metadata, will be interpreted as absent during the merge.
// This is different to how Update and normal proto merging works,
// where nil fields are interpreted as a delete instruction.
func (c *Collection) Merge(d *devicespb.Device, opts ...resource.WriteOption) (*devicespb.Device, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	opts = append([]resource.WriteOption{
		resource.InterceptBefore(mergeInterceptor),
	}, opts...)
	update, err := c.names.Update(d.Name, d, opts...)
	if err != nil {
		return nil, err
	}
	return update.(*devicespb.Device), nil
}

// Update updates the device with the given name in the collection.
func (c *Collection) Update(d *devicespb.Device, opts ...resource.WriteOption) (*devicespb.Device, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	update, err := c.names.Update(d.Name, d, opts...)
	if err != nil {
		return nil, err
	}
	return update.(*devicespb.Device), nil
}

// Delete removes the device with the given name from the collection.
func (c *Collection) Delete(name string, opts ...resource.WriteOption) (*devicespb.Device, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	old, err := c.names.Delete(name, opts...)
	var oldDevice *devicespb.Device
	if old != nil {
		oldDevice = old.(*devicespb.Device)
	}
	return oldDevice, err
}

func mergeInterceptor(old, new proto.Message) {
	oldVal := old.(*devicespb.Device)
	newVal := new.(*devicespb.Device)
	switch {
	case newVal.Metadata == nil:
		newVal.Metadata = oldVal.Metadata // no metadata in new, use old
	case oldVal.Metadata == nil:
		break // old has no metadata, keep new as is
	default:
		// both have metadata, merge them
		metadatapb.Merge(oldVal.Metadata, newVal.Metadata)
	}
}
