package node

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/wrap"
)

// Announcer defines the Announce method.
// Calling Announce signals that a given name exists and has the collection of features provided.
// Typically announcing the same name more than once will combine the features of the existing name with the new features.
type Announcer interface {
	// Announce signals that the name exists and has the given features.
	Announce(name string, features ...Feature) Undo
}

// SelfAnnouncer is a complement to node.Announcer allowing a type to announce itself.
type SelfAnnouncer interface {
	AnnounceSelf(a Announcer) Undo
}

// AnnouncerFunc allows adapting a func of the correct signature to implement Announcer
type AnnouncerFunc func(name string, features ...Feature) Undo

func (a AnnouncerFunc) Announce(name string, features ...Feature) Undo {
	return a(name, features...)
}

// SelfAnnouncerFunc allows adapting a func of the correct signature to implement SelfAnnouncer
type SelfAnnouncerFunc func(a Announcer) Undo

func (sa SelfAnnouncerFunc) AnnounceSelf(a Announcer) Undo {
	return sa(a)
}

// AnnounceWithNamePrefix returns an Announcer whose Announce method acts like `Announce(prefix+name, features...)`
func AnnounceWithNamePrefix(prefix string, a Announcer) Announcer {
	return AnnouncerFunc(func(name string, features ...Feature) Undo {
		return a.Announce(prefix+name, features...)
	})
}

// AnnounceContext returns a new Announcer that undoes any announcements when ctx is Done.
// This leaks a go routine if ctx is never done.
//
// Deprecated: AnnounceContext does not allow synchronising with the undos, which is usually necessary.
// Use AnnounceScope instead, or a ReplaceAnnouncer if multiple generations are required.
func AnnounceContext(ctx context.Context, a Announcer) Announcer {
	scoped, undo := AnnounceScope(a)
	_ = context.AfterFunc(ctx, undo)
	return scoped
}

// AnnounceScope returns a new scoped Announcer that will undo all announcements made to it when the returned Undo is called.
// The Undo will block until all undos are complete.
// Once the Undo is called, the Announcer is no longer valid and any further calls to Announce will do nothing.
//
// It is safe to call the Undo more than once - calls after the first will do nothing.
// It is safe to call the Undo returned by an individual Announce call, which will undo that announcement as normal,
// but it is not necessary to do so before calling the scope's Undo.
func AnnounceScope(parent Announcer) (Announcer, Undo) {
	s := &scope{
		parent: parent,
	}
	return s, s.undoAll
}

// scope wraps an Announcer to allow undoing all announcements at once.
type scope struct {
	parent Announcer

	mu    sync.Mutex
	done  bool
	undos []Undo
}

func (s *scope) Announce(name string, features ...Feature) Undo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return NilUndo
	}

	undo := s.parent.Announce(name, features...)

	undo = UndoOnce(undo)
	s.undos = append(s.undos, undo)
	return undo
}

// undoAll will undo all announcements made to this scope, blocking until all undos are complete.
// This method is idempotent and safe to call concurrently.
func (s *scope) undoAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return
	}
	for _, undo := range s.undos {
		undo()
	}
	s.done = true
	s.undos = nil
}

// ReplaceAnnouncer wraps an Announcer to allow replacing a set of announcements safely.
type ReplaceAnnouncer struct {
	parent      Announcer
	current     Announcer
	undoCurrent Undo
	mu          sync.Mutex
}

func NewReplaceAnnouncer(parent Announcer) *ReplaceAnnouncer {
	return &ReplaceAnnouncer{parent: parent}
}

// Replace will return a new Announcer that supercedes all previous Announcers returned by this ReplaceAnnouncer.
//
// Announcements made to the returned Announcer will be undone when ctx is cancelled.
// The announcer from the previous call to Replace has all its announcements undone - Replace blocks until this is
// complete.
func (r *ReplaceAnnouncer) Replace(ctx context.Context) Announcer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil {
		r.undoCurrent()
	}
	created, undo := AnnounceScope(r.parent)
	_ = context.AfterFunc(ctx, undo)
	r.current = created
	r.undoCurrent = undo
	return created
}

// AnnounceFeatures returns an Announcer that acts like `Announce(name, [moreFeatures..., features...])`
func AnnounceFeatures(a Announcer, moreFeatures ...Feature) Announcer {
	return AnnouncerFunc(func(name string, features ...Feature) Undo {
		// do it this way to avoid modifying modeFeatures if it happens to have capacity
		allFeatures := append([]Feature{}, moreFeatures...)
		allFeatures = append(allFeatures, features...)
		return a.Announce(name, allFeatures...)
	})
}

type announcement struct {
	name           string
	services       []service
	proxyTo        grpc.ClientConnInterface
	traits         []traitFeature
	metadata       []*traits.Metadata
	noAutoMetadata bool
	undo           []Undo
}

type traitFeature struct {
	name     trait.Name
	services []service
	metadata map[string]string
}

// Feature describes some aspect of a named device.
type Feature interface{ apply(a *announcement) }

// EmptyFeature is a Feature that means nothing.
// It can be embedded in custom Feature types to allow them to extend the capabilities of the Feature system.
type EmptyFeature struct{}

func (e EmptyFeature) apply(a *announcement) {
	// do nothing
}

type featureFunc func(a *announcement)

func (f featureFunc) apply(a *announcement) {
	f(a)
}

// HasClient indicates that the name implements non-trait apis as defined by these clients.
// The clients are still added to routers and all requests on the clients should accept a Name.
// If the node does not support routing for the API the client is for a message will be logged during announce.
//
// Panics if the service is not registered with the protobuf global registry.
func HasClient(clients ...wrap.ServiceUnwrapper) Feature {
	return featureFunc(func(a *announcement) {
		for _, c := range clients {
			conn, desc := c.UnwrapService()
			reflectDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(desc.ServiceName))
			if err != nil {
				panic(err)
			}
			reflectServiceDesc := reflectDesc.(protoreflect.ServiceDescriptor)

			a.services = append(a.services, service{desc: reflectServiceDesc, conn: conn, nameRouting: true})
		}
	})
}

// HasOptClient is like HasClient without logging when the client is not supported.
//
// Deprecated: Use HasClient instead, which behaves identically.
func HasOptClient(clients ...wrap.ServiceUnwrapper) Feature {
	return HasClient(clients...)
}

// HasServer registers a gRPC server type as routable for this announcement's name.
//
// Panics if the service is not registered with the protobuf global registry.
func HasServer[S any](register func(registrar grpc.ServiceRegistrar, srv S), srv S) Feature {
	return featureFunc(func(a *announcement) {
		// capture the descriptor of the registered service
		var registrar capturingRegistrar
		register(&registrar, srv)

		reflectDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(registrar.desc.ServiceName))
		if err != nil {
			panic(err)
		}
		reflectServiceDesc := reflectDesc.(protoreflect.ServiceDescriptor)

		a.services = append(a.services, service{
			desc:        reflectServiceDesc,
			conn:        wrap.ServerToClient(*registrar.desc, srv),
			nameRouting: true,
		})
	})
}

// HasServices indicates that conn serves the provided name-routable services, and that the name of this announcement
// is a valid name to use with each service.
//
// Panics if any of the provided services is not registered with the protobuf global registry.
func HasServices(conn grpc.ClientConnInterface, services ...grpc.ServiceDesc) Feature {
	var reflectedServices []protoreflect.ServiceDescriptor
	for _, s := range services {
		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(s.ServiceName))
		if err != nil {
			// this should never happen as the service desc is created by the user
			panic(err)
		}
		reflectedServices = append(reflectedServices, desc.(protoreflect.ServiceDescriptor))
	}
	return HasReflectedServices(conn, reflectedServices...)
}

// HasReflectedServices indicates that conn serves the provided name-routable services, and that the name of this announcement
// is a valid name to use with each service.
func HasReflectedServices(conn grpc.ClientConnInterface, services ...protoreflect.ServiceDescriptor) Feature {
	return featureFunc(func(a *announcement) {
		for _, s := range services {
			a.services = append(a.services, service{desc: s, conn: conn, nameRouting: true})
		}
	})
}

func HasProxy(conn grpc.ClientConnInterface) Feature {
	return featureFunc(func(a *announcement) {
		a.proxyTo = conn
	})
}

// HasTrait indicates that the device implements the named trait.
func HasTrait(name trait.Name, opt ...TraitOption) Feature {
	return featureFunc(func(a *announcement) {
		feature := traitFeature{name: name}
		for _, option := range opt {
			option(&feature)
		}
		a.traits = append(a.traits, feature)
	})
}

// HasMetadata merges the given metadata into any existing metadata held against the device name.
// Merging metadata is not commutative, two calls to Announce causes the second call to overwrite the first for all common fields.
// Announce accepts multiple HasMetadata features acting as if Announce were called in sequence with each HasMetadata feature.
//
// See metadata.Model.MergeMetadata for more details.
// If md is nil, does not adjust the announcement.
func HasMetadata(md *traits.Metadata) Feature {
	return featureFunc(func(a *announcement) {
		if md == nil {
			return // do nothing, helps callers to avoid nil checks
		}
		// We clone because if this is the first time the name has been associated with metadata,
		// then the passed md is used as is instead of cloning which can cause unexpected mutation
		// from the pov of the caller.
		a.metadata = append(a.metadata, proto.Clone(md).(*traits.Metadata))
	})
}

// HasNoAutoMetadata indicates that announcing the device should not announce automatic metadata for the device.
// Announcing will normally inspect all traits and announce basic metadata for them, this feature turns that off.
// HasMetadata will still be applied.
func HasNoAutoMetadata() Feature {
	return featureFunc(func(a *announcement) {
		a.noAutoMetadata = true
	})
}

// TraitOption controls how a Node behaves when presented with a new device trait.
type TraitOption func(t *traitFeature)

// WithClients indicates that the trait is implemented by these client instances.
// The clients will be added to the relevant routers when the trait is announced.
// If the node does not support routing for the API the client is for a message will be logged during announce.
//
// Panics if the service is not registered with the protobuf global registry.
func WithClients(clients ...wrap.ServiceUnwrapper) TraitOption {
	return func(t *traitFeature) {
		for _, c := range clients {
			conn, desc := c.UnwrapService()
			reflectDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(desc.ServiceName))
			if err != nil {
				panic(err)
			}
			reflectServiceDesc := reflectDesc.(protoreflect.ServiceDescriptor)
			t.services = append(t.services, service{desc: reflectServiceDesc, conn: conn, nameRouting: true})
		}
	}
}

// WithOptClients is the same as WithClients. It is retained for backwards compatibility, from when WithClients
// could fail because the trait was not supported by the node.
//
// Deprecated: Use WithClients instead. Services no longer have to be pre-supported by the node, so any correct
// ServiceUnwrapper will succeed.
func WithOptClients(clients ...wrap.ServiceUnwrapper) TraitOption {
	return WithClients(clients...)
}

type capturingRegistrar struct {
	desc *grpc.ServiceDesc
	impl any
}

func (r *capturingRegistrar) RegisterService(desc *grpc.ServiceDesc, impl any) {
	r.desc = desc
	r.impl = impl
}

type service struct {
	desc        protoreflect.ServiceDescriptor
	conn        grpc.ClientConnInterface // if nil, just ensure the service is registered but don't add any routes
	nameRouting bool
}
