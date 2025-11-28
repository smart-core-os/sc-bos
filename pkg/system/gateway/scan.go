package gateway

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

// scanRemoteHub calls scanRemoteNode for the hub and all enrolled nodes placing the results in a cohort.
// This blocks until ctx is done.
func (s *System) scanRemoteHub(ctx context.Context, c *cohort, hubConn *grpc.ClientConn) {
	hubClient := gen.NewHubApiClient(hubConn)
	hubAddr := hubConn.Target()
	if strings.HasPrefix(hubAddr, "dialchan:") || strings.HasPrefix(hubAddr, "passthrough:") {
		hubAddr = s.hub.Target()
	}
	hubNode := newRemoteNode(hubAddr, hubConn)
	hubNode.isHub = true
	c.Nodes.Set(hubNode)

	s.scanRemoteNode(ctx, "hub", hubNode)

	s.retry(ctx, "pull enrolled nodes", func(ctx context.Context) (task.Next, error) {
		return s.pullEnrolledNodes(ctx, hubClient, c)
	}, zap.String("node", hubNode.addr))
}

// scanLocalHub calls scanRemoteNode for all enrolled nodes placing the results in a cohort.
// This blocks until ctx is done.
func (s *System) scanLocalHub(ctx context.Context, c *cohort, hubClient gen.HubApiClient) {
	s.retry(ctx, "pull enrolled nodes", func(ctx context.Context) (task.Next, error) {
		return s.pullEnrolledNodes(ctx, hubClient, c)
	}, zap.String("node", "local"))
}

// scanRemoteNode starts background tasks to collect information about a remote node.
// Errors encountered during tasks are logged and the tasks are retried until ctx is done.
// nodeName is a human-readable name for the node, used in logs to distinguish between remote nodes.
func (s *System) scanRemoteNode(ctx context.Context, nodeName string, n *remoteNode) {
	go s.poll(ctx, func(ctx context.Context) error {
		return s.reflectNode(ctx, n)
	}, zap.String("task", "reflect"), zap.String("remoteAddr", n.addr), zap.String("nodeName", nodeName))

	go s.retry(ctx, "pull self", func(ctx context.Context) (task.Next, error) {
		return s.pullSelf(ctx, n)
	}, zap.String("remoteAddr", n.addr), zap.String("nodeName", nodeName))

	go s.retry(ctx, "pull systems", func(ctx context.Context) (task.Next, error) {
		return s.pullSystems(ctx, n)
	}, zap.String("remoteAddr", n.addr), zap.String("nodeName", nodeName))

	go s.retry(ctx, "pull devices", func(ctx context.Context) (task.Next, error) {
		return s.pullDevices(ctx, n)
	}, zap.String("remoteAddr", n.addr), zap.String("nodeName", nodeName))
}

// pullSelf updates node.Self with metadata about itself.
func (s *System) pullSelf(ctx context.Context, node *remoteNode) (task.Next, error) {
	mdClient := traits.NewMetadataApiClient(node.conn)
	mdStream, err := mdClient.PullMetadata(ctx, &traits.PullMetadataRequest{}) // no name means "self"
	if err != nil {
		return neverSucceeded, err
	}

	for msgReceived := false; ; msgReceived = true {
		cs, err := mdStream.Recv()
		if err != nil {
			if msgReceived {
				return someSuccess, err
			}
			return neverSucceeded, err
		}

		if len(cs.Changes) == 0 {
			continue
		}
		c := cs.Changes[len(cs.Changes)-1]
		md := c.Metadata
		// note: we don't fetch health checks for the self node,
		// this will be handled by the devices query
		node.Self.Set(remoteDesc{name: md.Name, md: md})
	}
}

// pullSystems populates node.Systems using the ServicesApi of node.
func (s *System) pullSystems(ctx context.Context, node *remoteNode) (task.Next, error) {
	client := gen.NewServicesApiClient(node.conn)
	stream, err := client.PullServices(ctx, &gen.PullServicesRequest{Name: "systems"})
	if err != nil {
		return neverSucceeded, err
	}

	var systems remoteSystems

	for msgReceived := false; ; msgReceived = true {
		msg, err := stream.Recv()
		if err != nil {
			if msgReceived {
				return someSuccess, err
			}
			return neverSucceeded, err
		}

		oldSystems := systems
		systems.msgRecvd = true
		for _, c := range msg.Changes {
			id := cmp.Or(c.GetNewValue().GetId(), c.GetOldValue().GetId())
			if id == "" {
				continue // not sure what happened, all services should have an id
			}
			if id == LegacyName {
				id = Name // legacy support for the old system name
			}
			switch id {
			case Name:
				systems.gateway = c.GetNewValue()
			}
		}

		// while we aren't using proto.Equal here, that's fine as the pointers would have changed if we got an update
		if oldSystems != systems {
			node.Systems.Set(systems)
		}
	}
}

// pullDevices uses node's DevicesApi to collect the list of devices and metadata about them.
// pullDevices blocks while the DevicesApi stream is active.
func (s *System) pullDevices(ctx context.Context, node *remoteNode) (task.Next, error) {
	client := gen.NewDevicesApiClient(node.conn)
	stream, err := client.PullDevices(ctx, &gen.PullDevicesRequest{})
	if err != nil {
		return neverSucceeded, err
	}

	for msgReceived := false; ; msgReceived = true {
		msg, err := stream.Recv()
		if err != nil {
			if msgReceived {
				return someSuccess, err
			}
			return neverSucceeded, err
		}

		for _, c := range msg.Changes {
			if c.NewValue == nil {
				node.Devices.Remove(remoteDesc{name: c.OldValue.Name})
				continue // device deleted
			}
			// Set covers both add and update cases
			node.Devices.Set(remoteDesc{name: c.NewValue.Name, md: c.NewValue.Metadata, health: c.NewValue.HealthChecks})
		}
	}
}

// pullEnrolledNodes calls scanRemoteNode for all enrolled nodes in the hub.
func (s *System) pullEnrolledNodes(ctx context.Context, hubClient gen.HubApiClient, cohort *cohort) (task.Next, error) {
	stream, err := hubClient.PullHubNodes(ctx, &gen.PullHubNodesRequest{})
	if err != nil {
		return neverSucceeded, err
	}

	// keyed by enrolled node address, just like in the cohort
	tasks := tasks{}
	defer tasks.callAll()

	for msgReceived := false; ; msgReceived = true {
		msg, err := stream.Recv()
		if err != nil {
			if msgReceived {
				return someSuccess, err
			}
			return neverSucceeded, err
		}

		for _, c := range msg.Changes {
			// todo: handle the update case instead of treating it as delete+add
			if c.OldValue != nil {
				tasks.remove(c.OldValue.Address)
			}
			if c.NewValue == nil {
				continue
			}

			var stops []func()
			stopAll := func() {
				for _, f := range stops {
					f()
				}
			}

			node := c.NewValue

			// check whether we should process this node at all
			if node.Name == s.self.Name() {
				continue // this is us, don't re-scan ourselves
			}
			if cohort.ShouldIgnore(node.Address) {
				continue
			}

			nodeCtx, stop := context.WithCancel(ctx)
			stops = append(stops, stop)

			conn, err := s.newClient(node.Address)
			if err != nil {
				// An error here means there's something wrong with the params we passed to grpc.NewClient.
				// It's unlikely that retrying will help, so skip this node and log an error.
				s.logger.Error("failed to dial remote node", zap.String("address", node.Address), zap.Error(err))
				stopAll()
				continue
			}
			stops = append(stops, func() { conn.Close() })

			remoteNode := newRemoteNode(node.Address, conn)
			// Notify others about the remote node before we scan it.
			// We do it this way to give others a chance to subscribe to the node before we start updating it
			cohort.Nodes.Set(remoteNode)
			s.scanRemoteNode(nodeCtx, cmp.Or(node.Name, node.Address), remoteNode)

			stops = append(stops, func() { cohort.Nodes.Remove(remoteNode) })
			tasks[node.Address] = stopAll
		}
	}
}

// reflectNode uses the reflection service to collect information about the services the node implements.
func (s *System) reflectNode(ctx context.Context, node *remoteNode) error {
	client := reflectionpb.NewServerReflectionClient(node.conn)

	streamCtx, stopStream := context.WithCancel(ctx)
	defer stopStream()
	stream, err := client.ServerReflectionInfo(streamCtx)
	if err != nil {
		return err
	}

	services, err := reflectionapi.ListServices(stream)
	if err != nil {
		return err
	}

	remoteServices := make([]protoreflect.ServiceDescriptor, 0, len(services))

	// reuse our own descriptor of the service if we have it
	services = slices.DeleteFunc(services, func(svc *reflectionpb.ServiceResponse) bool {
		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(svc.Name))
		if err != nil {
			return false
		}
		serviceDesc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return false
		}
		remoteServices = append(remoteServices, serviceDesc)
		return true
	})

	// collect all the type information for the services the node implements
	seenFiles := make(map[string]struct{})
	fileSet := &descriptorpb.FileDescriptorSet{}
	for _, svc := range services {
		serviceFiles, err := reflectionapi.FileContainingSymbol(stream, svc.Name)
		if err != nil {
			return err
		}
		for _, fileBytes := range serviceFiles {
			fileDescPB := &descriptorpb.FileDescriptorProto{}
			err := proto.Unmarshal(fileBytes, fileDescPB)
			if err != nil {
				return fmt.Errorf("unmarshal file descriptor: %w", err)
			}
			if _, ok := seenFiles[fileDescPB.GetName()]; ok {
				continue // skip duplicates
			}
			seenFiles[fileDescPB.GetName()] = struct{}{}
			fileSet.File = append(fileSet.File, fileDescPB)
		}
	}

	stopStream() // don't need the stream anymore, we have all the info we need

	files, err := protodesc.NewFiles(fileSet)
	if err != nil {
		return fmt.Errorf("create file set: %w", err)
	}
	for _, svc := range services {
		desc, err := files.FindDescriptorByName(protoreflect.FullName(svc.Name))
		if err != nil {
			// shouldn't have an error here as files is computed from the same set of services
			return fmt.Errorf("service %q not found in files: %w", svc.Name, err)
		}

		serviceDesc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return fmt.Errorf("descriptor %q is not a service, got %T", desc.FullName(), desc)
		}

		remoteServices = append(remoteServices, serviceDesc)
	}

	// we now can fully describe the remote service
	node.Services.Replace(remoteServices)

	return nil
}

// retry semantics for long-running tasks that can succeed and fail at the same time
const (
	neverSucceeded = task.Normal
	someSuccess    = task.ResetBackoff
)
