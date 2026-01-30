package node

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/smart-core-os/sc-bos/internal/router"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

type Service struct {
	routerService *router.Service
	conn          grpc.ClientConnInterface
}

func (s *Service) NameRoutable() bool {
	return s.routerService.KeyRoutable()
}

// ReflectedConnService returns a Service that routes requests to a ClientConn.
func ReflectedConnService(desc protoreflect.ServiceDescriptor, conn grpc.ClientConnInterface) *Service {
	return &Service{
		routerService: maybeNameRoutedService(desc),
		conn:          conn,
	}
}

// RegistryService returns a Service that routes requests to a local server implementation srv.
//
// The service described by desc must exist in the global protobuf registry, or an error will be returned.
func RegistryService(desc grpc.ServiceDesc, srv any) (*Service, error) {
	return RegistryConnService(desc, wrap.ServerToClient(desc, srv))
}

func RegistryConnService(desc grpc.ServiceDesc, conn grpc.ClientConnInterface) (*Service, error) {
	reflectSrvDesc, err := registryDescriptor(desc.ServiceName)
	if err != nil {
		return nil, err
	}
	return &Service{
		routerService: maybeNameRoutedService(reflectSrvDesc),
		conn:          conn,
	}, nil
}

// AnnounceService will make the service available on the node's router.
//
// If the gRPC service is already registered by a device, then requests that can't be routed to a device will be routed
// to the provided Service implementation.
// Otherwise, all requests for the service will be routed to the provided Service implementation.
func (n *Node) AnnounceService(srv *Service) (Undo, error) {
	serviceName := srv.routerService.Name()
	// service must be present in the router first
	err := n.SupportService(srv)
	if err != nil {
		return NilUndo, err
	}

	err = n.router.AddRoute(serviceName, "", srv.conn)
	if err != nil {
		return NilUndo, err
	}
	return func() {
		_ = n.router.DeleteRoute(serviceName, "")
	}, nil
}

// SupportService enables the node's router to handle requests for the service.
//
// No routes are added, so by default all requests will fail.
func (n *Node) SupportService(srv *Service) error {
	serviceName := srv.routerService.Name()
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.router.GetService(serviceName) == nil {
		err := n.router.AddService(srv.routerService)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetServiceInfo implements the grpc.ServiceRegistrar interface, reporting all the services supported
// by the node's router.
func (n *Node) GetServiceInfo() map[string]grpc.ServiceInfo {
	return n.router.GetServiceInfo()
}

func (n *Node) StreamServerInfo(method string) (grpc.StreamServerInfo, bool) {
	return n.router.StreamServerInfo(method)
}

func maybeNameRoutedService(desc protoreflect.ServiceDescriptor) *router.Service {
	srv, err := router.NewRoutedService(desc, "name")
	if err == nil {
		return srv
	}
	return router.NewUnroutedService(desc)
}

func registryDescriptor(serviceName string) (protoreflect.ServiceDescriptor, error) {
	reflectDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, err
	}
	reflectSrvDesc, ok := reflectDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%q is not a service descriptor (actually %T)", serviceName, reflectDesc)
	}
	return reflectSrvDesc, nil
}
