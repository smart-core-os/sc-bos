package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/proxy/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/node/alltraits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const DriverName = "proxy"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer:       services.Node,
		clientTLSConfig: services.ClientTLSConfig,
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[config.Root](d.Clear))
	d.logger = services.Logger.Named(DriverName)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	logger          *zap.Logger
	announcer       node.Announcer
	clientTLSConfig *tls.Config // base config used to dial nodes

	proxies []*proxy // all the nodes we proxy
}

func (d *Driver) Clear() {
	var err error

	// close all existing connections and unregister all proxied traits
	for _, p := range d.proxies {
		err = multierr.Append(err, p.Close())
	}
	d.proxies = nil
	if err != nil {
		d.logger.Warn("Failed to cleanly close existing proxies", zap.Error(err))
	}
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	// todo: support incremental updates to the config, i.e. a nodes trait list has updated
	var allErrs error

	d.Clear() // close existing proxies

	// For each node we create a proxy instance which manages the discovery of children exposed by that node.
	for _, n := range cfg.Nodes {
		tlsConfig := proxyTLSConfig(d.clientTLSConfig, n)
		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		}
		if n.OAuth2 != nil {
			httpClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
			creds, err := newOAuth2Credentials(*n.OAuth2, httpClient)
			if err != nil {
				allErrs = multierr.Append(allErrs, fmt.Errorf("oauth2 credentials for %s: %w", n.Host, err))
				continue
			}
			dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(creds))
		}
		conn, err := grpc.NewClient(n.Host, dialOpts...)
		if err != nil {
			// dial shouldn't fail, connections are lazy. If we do see an error here make sure we surface it!
			allErrs = multierr.Append(allErrs, fmt.Errorf("dial %v %w", n.Host, err))
			continue
		}

		ctx, shutdown := context.WithCancel(ctx)
		proxy := &proxy{
			config:    n,
			conn:      conn,
			announcer: d.announcer,
			skipChild: n.SkipChild,
			logger:    d.logger.Named(n.Host),
			shutdown:  shutdown,
		}
		d.proxies = append(d.proxies, proxy)

		// list, announce, and subscribe to updates to the list of children on the server
		if len(n.Children) > 0 {
			proxy.announceExplicitChildren(n.Children)
		} else {
			go func() {
				err := proxy.AnnounceChildren(ctx)
				if errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					d.logger.Warn("Announcing children error", zap.Error(err))
				}
			}()
		}
	}
	return allErrs
}

// proxyTLSConfig overlays any node specific TLS config onto the controller managed TLS config.
func proxyTLSConfig(tlsConfig *tls.Config, n config.Node) *tls.Config {
	if n.TLS.InsecureNoClientCert || n.TLS.InsecureSkipVerify {
		tlsConfig = tlsConfig.Clone()
		if n.TLS.InsecureSkipVerify {
			tlsConfig.InsecureSkipVerify = true
			tlsConfig.VerifyConnection = nil
		}
		if n.TLS.InsecureNoClientCert {
			tlsConfig.Certificates = nil
			tlsConfig.GetClientCertificate = nil
		}
	}
	return tlsConfig
}

// proxy manages updates to the announced traits of any proxied devices for a single node.
// At a high level it subscribes to changes in the nodes children,
// when new children are added it announces them on this node,
// when children are removed it removes them from this node too.
type proxy struct {
	config    config.Node
	conn      *grpc.ClientConn // used if the proxy updates its children
	skipChild bool             // if true we don't announce the child trait on this node
	announcer node.Announcer

	logger   *zap.Logger
	shutdown context.CancelFunc
}

// AnnounceChildren queries the nodes Parent trait for children and syncs those children with the announcer.
// AnnounceChildren blocks until either we give up getting children or ctx is done.
// A best effort is made to fetch children and updates, trying PullChildren and ListChildren as needed.
// Network level errors will be retried. If the server responds with codes.Unimplemented for both Pull and List calls
// then AnnounceChildren will give up and return an error.
func (p *proxy) AnnounceChildren(ctx context.Context) error {
	changes := make(chan *gen.PullDevicesResponse_Change)
	defer close(changes)

	go p.announceChanges(changes)

	fetcher := &childrenFetcher{name: p.config.Name, client: gen.NewDevicesApiClient(p.conn)}
	return pull.Changes[*gen.PullDevicesResponse_Change](ctx, fetcher, changes, pull.WithLogger(p.logger))
}

func (p *proxy) announceExplicitChildren(children []config.Child) {
	for _, c := range children {
		p.announceTraits(nil, c.Name, c.Traits)
	}
}

func (p *proxy) announceChanges(changes <-chan *gen.PullDevicesResponse_Change) {
	announced := announcedTraits{}
	defer announced.deleteAll()
	for change := range changes {
		p.announceChange(announced, change)
	}
}

func (p *proxy) announceChange(announced announcedTraits, change *gen.PullDevicesResponse_Change) {
	needAnnouncing := announced.updateChild(change.OldValue, change.NewValue)
	childName := change.GetNewValue().GetName() // nil safe way to get the name
	p.announceTraits(announced, childName, needAnnouncing)
}

// Announces traitNames for a childName.
// If announced is non-nil, the undo functions for the announcements are stored in it.
func (p *proxy) announceTraits(announced announcedTraits, childName string, traitNames []trait.Name) {
	for _, tn := range traitNames {
		services := alltraits.ServiceDesc(tn)
		if len(services) == 0 {
			p.logger.Warn(fmt.Sprintf("remote child implements unknown trait %s", tn))
			continue
		}

		features := []node.Feature{node.HasServices(p.conn, services...)}
		if !p.skipChild {
			features = append(features, node.HasTrait(tn))
		} else {
			features = append(features, node.HasNoAutoMetadata())
		}

		undo := p.announcer.Announce(childName, features...)
		if announced != nil {
			announced.add(childName, tn, undo)
		}
	}
}

func (p *proxy) Close() error {
	p.shutdown()
	return p.conn.Close()
}

// childTrait is used as a map key to uniquely identify a child+trait pair.
type childTrait struct {
	name  string
	trait trait.Name
}

// announcedTraits is a helper type representing child traits that have been announced already.
// This tracks the node.Undo so we can clean up when traits need to be forgotten.
type announcedTraits map[childTrait]node.Undo

func (a announcedTraits) add(name string, tn trait.Name, undo node.Undo) {
	a[childTrait{name: name, trait: tn}] = undo
}

// updateChild compares oldChild and newChild and undoes and deletes any child traits that no longer exist.
// oldChild and/or newChild may be nil.
// updateChild returns any traits that newChild has that `a` does not know about.
func (a announcedTraits) updateChild(oldChild, newChild *gen.Device) []trait.Name {
	if oldChild != nil && newChild == nil {
		a.deleteChild(oldChild)
		return nil
	}

	if newChild == nil {
		return nil // both old and new are nil, nothing to do
	}

	var needAnnouncing []trait.Name
	var needDeleting map[trait.Name]struct{}
	if oldChild != nil {
		needDeleting = make(map[trait.Name]struct{}, len(oldChild.Metadata.Traits))
		for _, t := range oldChild.Metadata.Traits {
			needDeleting[trait.Name(t.Name)] = struct{}{}
		}
	}
	for _, t := range newChild.Metadata.Traits {
		tn := trait.Name(t.Name)
		delete(needDeleting, tn)

		key := childTrait{
			name:  newChild.Name,
			trait: tn,
		}
		if _, ok := a[key]; ok {
			continue // we already know the child has this trait, don't announce and don't remove
		}

		// announce a new client trait
		needAnnouncing = append(needAnnouncing, tn)
	}
	if oldChild != nil {
		for tn := range needDeleting {
			a.deleteChildTrait(oldChild.Name, tn)
		}
	}
	return needAnnouncing
}

// deleteChild undoes and removes all the traits child has.
func (a announcedTraits) deleteChild(child *gen.Device) {
	for _, t := range child.Metadata.Traits {
		a.deleteChildTrait(child.Name, trait.Name(t.Name))
	}
}

func (a announcedTraits) deleteChildTrait(name string, tn trait.Name) {
	key := childTrait{name: name, trait: tn}
	old, ok := a[key]
	if ok {
		old()
		delete(a, key)
	}
}

func (a announcedTraits) deleteAll() {
	for k, undo := range a {
		undo()
		delete(a, k)
	}
}
