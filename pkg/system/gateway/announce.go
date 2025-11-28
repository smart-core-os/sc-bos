package gateway

import (
	"context"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alphapb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/rx"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	scslices "github.com/smart-core-os/sc-bos/pkg/util/slices"
)

// announceCohort announces information about the cohort as if it were present on this system.
// This includes announcing remote names and apis via the local DevicesApi and other apis.
// This blocks until ctx is done.
func (s *System) announceCohort(ctx context.Context, c *cohort) {
	tasks := tasks{}
	defer tasks.callAll()

	table := &table{
		services:    &counts{m: make(map[string]int)},
		serviceUndo: newSyncMap[node.Undo](),
	}

	runAnnouncer := func(n *remoteNode) {
		nodeCtx, stopCtx := context.WithCancel(ctx)
		scope, stopScope := node.AnnounceScope(s.announcer)
		stop := func() {
			stopCtx()
			stopScope()
		}
		tasks[n.addr] = stop
		a := &announcer{
			System: s,
			logger: s.logger.With(
				zap.String("remoteAddr", n.addr),
				zap.Bool("isHub", n.isHub),
				// the remote node's name can change over time, can't add as a logger field
			),
			node:      n,
			Announcer: scope,
			table:     table,
		}
		go a.announceRemoteNode(nodeCtx)
	}

	nodes, nodeChanges := c.Nodes.Sub(ctx)
	for _, n := range nodes.All {
		runAnnouncer(n)
	}

	for nodeChange := range nodeChanges {
		if nodeChange.Old != nil {
			tasks.remove(nodeChange.Old.addr)
		}
		if nodeChange.New == nil {
			continue // was a deletion
		}
		runAnnouncer(nodeChange.New)
	}
}

// announcer provides the functionality for announcing parts of a cohort.
// Typically, the entrypoint is announceRemoteNode.
type announcer struct {
	*System
	logger *zap.Logger
	table  *table
	node   *remoteNode
	node.Announcer
}

// announceRemoteNode announces the remote node as if it were present on this system.
// This blocks until ctx is done.
// All side effects are undone when the method returns.
func (a *announcer) announceRemoteNode(ctx context.Context) {
	self, selfChanges := a.node.Self.Sub(ctx)

	// Remote nodes that are gateways will likely also be announcing all the same devices and apis we will.
	// To avoid circular routing we don't announce advertised devices for remote nodes that are gateways.
	// To avoid extra dynamic proxying we also don't announce reflected services for remote nodes that are gateways.
	// As being a gateway is something that can change,
	// we track and update our advertised devices and services when the gateway status changes.
	systems, systemChanges := a.node.Systems.Sub(ctx)
	isGateway := func() bool {
		// we are intentionally ignoring the loading state of the gateway system,
		// under the assumption that whether the remote gateway system has loaded or not it's
		// still intending to be a gateway eventually.
		return systems.gateway.GetActive()
	}
	wasGateway := isGateway()

	// The services we proxy are dependent on whether the remote node is a gateway or not.
	// These track and update our advertised services when the gateway status changes.
	var serviceChanges <-chan rx.Change[protoreflect.ServiceDescriptor]
	var stopServiceSub context.CancelFunc
	var undoServices tasks
	setupServiceSub := func() {
		var serviceCtx context.Context
		serviceCtx, stopServiceSub = context.WithCancel(ctx)
		var services *scslices.Sorted[protoreflect.ServiceDescriptor]
		services, serviceChanges = a.node.Services.Sub(serviceCtx)
		undoServices = a.announceRemoteServices(services.All)
	}
	closeServiceSub := func() {
		if stopServiceSub != nil {
			stopServiceSub()
		}
		undoServices.callAll()
		serviceChanges = nil
		undoServices = nil
	}
	defer closeServiceSub()

	// The devices we proxy are dependent on whether the remote node is a gateway or not.
	// All nodes have some devices that get proxied, but gateway nodes only proxy a subset of them.
	var deviceChanges <-chan rx.Change[remoteDesc]
	var stopDevicesSub context.CancelFunc
	// So we can undo the API proxy and metadata announcement separately.
	var undoDevices, undoMD, removeChecks tasks
	shouldProxyDevice := func(c remoteDesc) bool {
		if isGateway() {
			// only special names get proxied for gateway nodes
			suffix := strings.TrimPrefix(c.name, self.name+"/")
			return c.name == self.name || isFixedServiceName(suffix)
		} else {
			return true
		}
	}
	setupDevicesSub := func() {
		var devicesCtx context.Context
		devicesCtx, stopDevicesSub = context.WithCancel(ctx)
		var devices *scslices.Sorted[remoteDesc]
		devices, deviceChanges = a.node.Devices.Sub(devicesCtx)
		undoDevices, undoMD, removeChecks = a.announceNames(filter2(devices.All, func(_ int, v remoteDesc) bool {
			return shouldProxyDevice(v)
		}))
	}
	closeDevicesSub := func() {
		if stopDevicesSub != nil {
			stopDevicesSub()
		}
		undoDevices.callAll()
		undoMD.callAll()
		removeChecks.callAll()
		deviceChanges = nil
		undoDevices = nil
		undoMD = nil
		removeChecks = nil
	}
	renewDevicesSub := func() {
		closeDevicesSub()
		setupDevicesSub()
	}
	defer closeDevicesSub()

	// The types and services we return via reflection depends on whether the remote node is a gateway or not.
	// These update the reflection server when the gateway status changes.
	setupReflection := func() {
		a.reflection.Add(a.node.conn)
	}
	closeReflection := func() {
		a.reflection.Remove(a.node.conn)
	}
	defer closeReflection()

	// helper for switching between gateway and non-gateway mode
	switchGatewayMode := func(isGateway bool) {
		renewDevicesSub()
		if isGateway {
			closeServiceSub()
			closeReflection()
		} else {
			setupServiceSub()
			setupReflection()
		}
	}

	// todo: remove both these waits once we have something that can tell us the origin node of a device.

	// When the remote node is a gateway, the devices we proxy are dependent on the remote node's name.
	// See shouldProxyDevice, TL;DR we need the name to identify the remote node and its fixed service names.
	waitForFunc(ctx, &self, selfChanges, func(d remoteDesc) bool {
		return d.name != ""
	})
	// Announcing a node that hasn't, but will eventually, be classified as a gateway is expensive.
	// Delay our announcement a little bit to allow us to do an initial classification of the node,
	// at least until we have a response from the remote node.
	waitForFunc(ctx, &systems, systemChanges, func(s remoteSystems) bool {
		return s.msgRecvd
	})

	switchGatewayMode(isGateway())

	for {
		select {
		case <-ctx.Done():
			// a done ctx should clean up all the subscriptions and announced names
			return
		case self = <-selfChanges:
		case c, ok := <-serviceChanges:
			if !ok {
				continue // we stopped watching
			}
			if c.Type != rx.Add {
				undoServices.remove(string(c.Old.FullName()))
			}
			if c.Type != rx.Remove {
				undoServices[string(c.New.FullName())] = a.announceRemoteService(c.New)
			}
		case systems = <-systemChanges:
			isGateway := isGateway()
			if isGateway != wasGateway {
				a.logger.Debug("switching gateway mode", zap.Bool("isGateway", isGateway))
				switchGatewayMode(isGateway)
				wasGateway = isGateway
			}
		case c, ok := <-deviceChanges:
			if !ok {
				continue // we stopped watching
			}
			switch c.Type {
			case rx.Add:
				if !shouldProxyDevice(c.New) {
					continue
				}
				undoDevices[c.New.name] = a.announceProxy(c.New)
				undoMD[c.New.name] = a.announceMetadata(c.New)
				removeChecks[c.New.name] = a.announceHealthChecks(c.Old, c.New)
			case rx.Remove:
				undoDevices.remove(c.Old.name)
				undoMD.remove(c.Old.name)
				removeChecks.remove(c.Old.name)
			case rx.Update:
				if !shouldProxyDevice(c.New) {
					undoMD.remove(c.Old.name)
					removeChecks.remove(c.Old.name)
					continue
				}
				if !proto.Equal(c.Old.md, c.New.md) {
					undoMD.remove(c.Old.name)
					undoMD[c.New.name] = a.announceMetadata(c.New)
				}
				removeChecks[c.New.name] = a.announceHealthChecks(c.Old, c.New)
			}
		}
	}
}

// announceName allows this node to answer requests aimed at the given remoteDesc.
// This includes named RPCs (like trait requests) and DeviceApi / ParentApi requests that would include this name in their responses.
// The func returns undo functions for both the API proxy and metadata announcement.
func (a *announcer) announceName(d remoteDesc) (device, md, checks node.Undo) {
	return a.announceProxy(d), a.announceMetadata(d), a.announceHealthChecks(remoteDesc{}, d)
}

// announceProxy updates this node to proxy requests for the given remoteDesc.
func (a *announcer) announceProxy(d remoteDesc) node.Undo {
	if !shouldAnnounceName(d.name) {
		return node.NilUndo
	}
	return a.Announce(d.name, node.HasProxy(a.node.conn))
}

// announceMetadata updates this node to announce metadata for the given remoteDesc.
func (a *announcer) announceMetadata(d remoteDesc) node.Undo {
	if !shouldAnnounceName(d.name) {
		return node.NilUndo
	}
	return a.Announce(d.name, node.HasMetadata(d.md))
}

// announceHealthChecks updates this node to announce health checks for the given remoteDesc.
// Any checks that are in before but not in after are removed.
// The returned Undo will remove any added checks and is equivalent to calling announceHealthChecks(after, remoteDesc{}).
func (a *announcer) announceHealthChecks(before, after remoteDesc) node.Undo {
	// remove checks that are no longer present
	if len(before.health) > 0 {
		toRemove := make(map[string]struct{})
		for _, health := range before.health {
			toRemove[health.Id] = struct{}{}
		}
		for _, health := range after.health {
			delete(toRemove, health.Id)
		}
		err := a.checks.RemoveHealthChecks(before.name, slices.Collect(maps.Keys(toRemove))...)
		if err != nil {
			a.logger.Warn("failed to unannounce health checks for remote name",
				zap.String("name", before.name),
				zap.Strings("removedCheckIds", slices.Sorted(maps.Keys(toRemove))),
				zap.Error(err),
			)
		}
	}

	if len(after.health) == 0 {
		return node.NilUndo
	}

	addedIds := make([]string, 0, len(after.health))
	for _, c := range after.health {
		addedIds = append(addedIds, c.Id)
	}
	err := a.checks.MergeHealthChecks(after.name, after.health...) // will replace checks that already exist
	if err != nil {
		a.logger.Warn("failed to announce health checks for remote name",
			zap.String("name", after.name),
			zap.Strings("addedCheckIds", addedIds),
			zap.Error(err),
		)
	}
	return func() {
		err := a.checks.RemoveHealthChecks(after.name, addedIds...)
		if err != nil {
			a.logger.Warn("failed to unannounce health checks for remote name",
				zap.String("name", after.name),
				zap.Strings("removedCheckIds", addedIds),
				zap.Error(err),
			)
		}
	}
}

// shouldAnnounceName returns true if we should announce the given name.
func shouldAnnounceName(name string) bool {
	if name == "" {
		return false
	}
	// If the name is one of the ignored names, we don't announce it.
	if isFixedServiceName(name) {
		return false
	}
	return true
}

// announceNames calls announceName for each remoteDesc in seq, collecting the results into tasks keyed by remoteDesc.name.
func (a *announcer) announceNames(seq iter.Seq2[int, remoteDesc]) (devices, mds, checks tasks) {
	devices = tasks{}
	mds = tasks{}
	checks = tasks{}
	for _, d := range seq {
		devices[d.name], mds[d.name], checks[d.name] = a.announceName(d)
	}
	return devices, mds, checks
}

// announceRemoteService updates this node to respond to requests for the given remoteService.
func (a *announcer) announceRemoteService(rs protoreflect.ServiceDescriptor) node.Undo {
	if a.ignoreRemoteService(rs) {
		return node.NilUndo
	}
	name := string(rs.FullName())
	if a.table.services.Inc(name) == 1 {
		undo := a.announceRemoteServiceApis(rs)
		if undo != nil { // can be nil if there's nothing to undo
			a.table.serviceUndo.Set(name, undo)
		}
	}

	return func() {
		if a.table.services.Dec(name) == 0 {
			// we were the last to remove the service, clean everything up
			undo, ok := a.table.serviceUndo.Del(name)
			if ok {
				undo()
			}
		}
	}
}

// announceRemoteServices updates this node to respond to requests for each remoteService in seq.
func (a *announcer) announceRemoteServices(seq iter.Seq2[int, protoreflect.ServiceDescriptor]) tasks {
	dst := tasks{}
	for _, rs := range seq {
		name := string(rs.FullName())
		dst[name] = a.announceRemoteService(rs)
	}
	return dst
}

func (a *announcer) announceRemoteServiceApis(rs protoreflect.ServiceDescriptor) node.Undo {
	srv := node.ReflectedConnService(rs, a.node.conn)
	var undos []node.Undo

	// which type of proxying should each method use?
	switch {
	case srv.NameRoutable():
		// routes will be added by device announcements as this service is routable by name
		err := a.self.SupportService(srv)
		if err != nil {
			a.logger.Warn("failed to announce routable service",
				zap.String("service", string(rs.FullName())),
				zap.Error(err),
			)
			break
		}
		a.logger.Debug("routable service announced",
			zap.String("service", string(rs.FullName())),
		)
	case a.node.isHub:
		// route everything to the hub
		undo, err := a.self.AnnounceService(srv)
		if err != nil {
			a.logger.Warn("failed to announce non-routable hub service",
				zap.String("service", string(rs.FullName())),
				zap.Error(err),
			)
			break
		}
		a.logger.Debug("non-routable hub service announced",
			zap.String("service", string(rs.FullName())),
		)
		undos = append(undos, undo)
	default:
		// Found a non-routable service on a non-hub node.
		// We didn't think we had any of these so log it to remind us.
		a.logger.Warn("unable to announce non-routable service on non-hub node", zap.String("service", string(rs.FullName())))
		return nil
	}

	return node.UndoAll(undos...)
}

// ignoreRemoteService returns true for service descriptors that we shouldn't proxy automatically.
// All services that return true are proxied in some other way.
func (a *announcer) ignoreRemoteService(rs protoreflect.ServiceDescriptor) bool {
	name := string(rs.FullName())
	switch name {
	// services that are handled explicitly via other mechanisms
	case
		gen.DevicesApi_ServiceDesc.ServiceName,                // handled by the node outside the gateway service
		gen.EnrollmentApi_ServiceDesc.ServiceName,             // handled by the app controller during boot
		gen.ServicesApi_ServiceDesc.ServiceName,               // see announceServiceApi
		reflectionpb.ServerReflection_ServiceDesc.ServiceName, // see setup/closeReflection in announceRemoteNode
		reflectionv1alphapb.ServerReflection_ServiceDesc.ServiceName:
		return true
	}
	return false
}

// table tracks state across all remote nodes.
type table struct {
	services    *counts             // keyed by service full name, counts how many times we've seen services across all remote nodes
	serviceUndo *syncMap[node.Undo] // keyed by service full name
}

// syncMap is a simple synchronised map with string keys.
type syncMap[T any] struct {
	mu sync.RWMutex
	m  map[string]T
}

func newSyncMap[T any]() *syncMap[T] {
	return &syncMap[T]{m: make(map[string]T)}
}

func (r *syncMap[T]) Get(k string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.m[k]
	return v, ok
}

// Set adds or replaces a value in the map, returning the old value and true if the key was already present.
func (r *syncMap[T]) Set(k string, v T) (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.m[k]
	r.m[k] = v
	return old, ok
}

// Del removes a value from the map, returning the old value and true if the key was present.
func (r *syncMap[T]) Del(k string) (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.m[k]
	if ok {
		delete(r.m, k)
	}
	return v, ok
}

// counts acts like a synchronous multiset of strings.
type counts struct {
	mu sync.Mutex
	m  map[string]int
}

// Inc adds one to the count of k, returning the new count.
func (r *counts) Inc(k string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[k]++
	return r.m[k]
}

// Dec subtracts one from the count of k, returning the new count.
// Dec will never cause the count to become less than zero.
func (r *counts) Dec(k string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[k]--
	if r.m[k] <= 0 {
		delete(r.m, k)
	}
	return r.m[k]
}

var fixedServiceNames = []string{"automations", "drivers", "systems", "zones"}

func isFixedServiceName(name string) bool {
	_, found := slices.BinarySearch(fixedServiceNames, name)
	return found
}

// filter2 returns an iterator that yields only the elements of seq for which f returns true.
func filter2[K, V any](seq iter.Seq2[K, V], f func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range seq {
			if f(k, v) {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

const waitTimeout = 5 * time.Second

// waitForFunc returns when a value received from c satisfies the function f, or a timeout has expired.
// Each value received will be assigned to v, which is a pointer to the current value.
func waitForFunc[T any](ctx context.Context, v *T, c <-chan T, f func(T) bool) {
	if !f(*v) {
		ctx, cancel := context.WithTimeout(ctx, waitTimeout)
		defer cancel()
		_, _ = chans.RecvContextFunc(ctx, c, func(new T) error {
			*v = new
			if !f(new) {
				return chans.ErrSkip
			}
			return nil
		})
		cancel()
	}
}
