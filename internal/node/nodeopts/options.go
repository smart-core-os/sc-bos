// Package nodeopts provides common and private options for the node package.
package nodeopts

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/internal/router"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Option interface {
	apply(*Struct)
}

type optionFunc func(*Struct)

func (f optionFunc) apply(o *Struct) {
	f(o)
}

// WithStore sets the Store used by the Node to Store its announced devices.
func WithStore(store Store) Option {
	return optionFunc(func(o *Struct) {
		o.Store = store
	})
}

// WithRouter sets the Router used by the Node to route gRPC clients.
func WithRouter(r *router.Router) Option {
	return optionFunc(func(o *Struct) {
		o.Router = r
	})
}

// WithClientConnWrapper sets a decorator applied to the connection returned by Node.ClientConn.
// Every in-process caller obtains its connection from that method, so the wrapper sees all
// in-process traffic. Used to audit writes that never reach the gRPC server's interceptors.
func WithClientConnWrapper(wrapper func(grpc.ClientConnInterface) grpc.ClientConnInterface) Option {
	return optionFunc(func(o *Struct) {
		o.ClientConnWrapper = wrapper
	})
}

// Join combines multiple options into a single struct.
func Join(opts ...Option) Struct {
	var o Struct
	for _, opt := range opts {
		opt.apply(&o)
	}
	return o
}

// Struct contains all options for a Node as a struct for easy access.
type Struct struct {
	Store             Store
	Router            *router.Router
	ClientConnWrapper func(grpc.ClientConnInterface) grpc.ClientConnInterface
}

func (s Struct) apply(o *Struct) {
	if s.Store != nil {
		o.Store = s.Store
	}
	if s.ClientConnWrapper != nil {
		o.ClientConnWrapper = s.ClientConnWrapper
	}
}

// Store describes how a node stores its announced devices.
type Store interface {
	GetDevice(name string, opts ...resource.ReadOption) (*devicespb.Device, error)
	PullDevice(ctx context.Context, name string, opts ...resource.ReadOption) <-chan devicespb.DeviceChange
	ListDevices(opts ...resource.ReadOption) []*devicespb.Device
	PullDevices(ctx context.Context, opts ...resource.ReadOption) <-chan devicespb.DevicesChange
	Update(d *devicespb.Device, opts ...resource.WriteOption) (*devicespb.Device, error)
	Delete(name string, opts ...resource.WriteOption) (*devicespb.Device, error)
}
