package node

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	gen_devicespb "github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func (n *Node) GetDevice(name string, opts ...resource.ReadOption) (*gen_devicespb.Device, error) {
	return n.devices.GetDevice(name, opts...)
}

func (n *Node) PullDevice(ctx context.Context, name string, opts ...resource.ReadOption) <-chan devicespb.DeviceChange {
	return n.devices.PullDevice(ctx, name, opts...)
}

func (n *Node) ListDevices(opts ...resource.ReadOption) []*gen_devicespb.Device {
	return n.devices.ListDevices(opts...)
}

func (n *Node) PullDevices(ctx context.Context, opts ...resource.ReadOption) <-chan devicespb.DevicesChange {
	return n.devices.PullDevices(ctx, opts...)
}
