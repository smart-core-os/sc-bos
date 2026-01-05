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

	// For each node we create a proxy instance which manages the discovery of devices exposed by that node.
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
			config:     n,
			conn:       conn,
			announcer:  d.announcer,
			skipDevice: n.ShouldSkipDevice(),
			logger:     d.logger.Named(n.Host),
			shutdown:   shutdown,
		}
		d.proxies = append(d.proxies, proxy)

		// list, announce, and subscribe to updates to the list of devices on the server
		if len(n.GetDevices()) > 0 {
			proxy.announceExplicitDevices(n.GetDevices())
		} else {
			go func() {
				err := proxy.AnnounceDevices(ctx)
				if errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					d.logger.Warn("Announcing devices error", zap.Error(err))
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
// At a high level it subscribes to changes in the node's devices,
// when new devices are added, it announces them on this node,
// when devices are removed, it removes them from this node too.
type proxy struct {
	config     config.Node
	conn       *grpc.ClientConn // used if the proxy updates its devices
	skipDevice bool             // if true we don't announce the device trait on this node
	announcer  node.Announcer

	logger   *zap.Logger
	shutdown context.CancelFunc
}

// AnnounceDevices queries the DevicesApi for all devices on that node.
// AnnounceDevices blocks until either we give up getting devices or ctx is done.
// A best effort is made to fetch devices and updates, trying PullDevices and ListDevices as needed.
// Network level errors will be retried. If the server responds with codes.Unimplemented for both Pull and List calls
// then AnnounceDevices will give up and return an error.
func (p *proxy) AnnounceDevices(ctx context.Context) error {
	changes := make(chan *gen.PullDevicesResponse_Change)
	defer close(changes)

	go p.announceChanges(changes)

	fetcher := &deviceFetcher{name: p.config.Name, client: gen.NewDevicesApiClient(p.conn)}
	return pull.Changes[*gen.PullDevicesResponse_Change](ctx, fetcher, changes, pull.WithLogger(p.logger))
}

func (p *proxy) announceExplicitDevices(devices []config.Device) {
	for _, c := range devices {
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
	needAnnouncing := announced.updateDevice(change.OldValue, change.NewValue)
	deviceName := change.GetNewValue().GetName() // nil safe way to get the name
	p.announceTraits(announced, deviceName, needAnnouncing)
}

// Announces traitNames for a deviceName.
// If announced is non-nil, the undo functions for the announcements are stored in it.
func (p *proxy) announceTraits(announced announcedTraits, deviceName string, traitNames []trait.Name) {
	for _, tn := range traitNames {
		services := alltraits.ServiceDesc(tn)
		if len(services) == 0 {
			p.logger.Warn(fmt.Sprintf("remote device implements unknown trait %s", tn))
			continue
		}

		features := []node.Feature{node.HasServices(p.conn, services...)}
		if !p.skipDevice {
			features = append(features, node.HasTrait(tn))
		} else {
			features = append(features, node.HasNoAutoMetadata())
		}

		undo := p.announcer.Announce(deviceName, features...)
		if announced != nil {
			announced.add(deviceName, tn, undo)
		}
	}
}

func (p *proxy) Close() error {
	p.shutdown()
	return p.conn.Close()
}

// deviceTrait is used as a map key to uniquely identify a device+trait pair.
type deviceTrait struct {
	name  string
	trait trait.Name
}

// announcedTraits is a helper type representing device traits that have been announced already.
// This tracks the node.Undo so we can clean up when traits need to be forgotten.
type announcedTraits map[deviceTrait]node.Undo

func (a announcedTraits) add(name string, tn trait.Name, undo node.Undo) {
	a[deviceTrait{name: name, trait: tn}] = undo
}

// updateDevice compares oldDevice and newDevice and undoes and deletes any device traits that no longer exist.
// oldDevice and/or newDevice may be nil.
// updateDevice returns any traits that newDevice has that `a` does not know about.
func (a announcedTraits) updateDevice(oldDevice, newDevice *gen.Device) []trait.Name {
	if oldDevice != nil && newDevice == nil {
		a.deleteDevice(oldDevice)
		return nil
	}

	if newDevice == nil {
		return nil // both old and new are nil, nothing to do
	}

	var needAnnouncing []trait.Name
	var needDeleting map[trait.Name]struct{}
	if oldDevice != nil {
		needDeleting = make(map[trait.Name]struct{}, len(oldDevice.Metadata.Traits))
		for _, t := range oldDevice.Metadata.Traits {
			needDeleting[trait.Name(t.Name)] = struct{}{}
		}
	}
	for _, t := range newDevice.Metadata.Traits {
		tn := trait.Name(t.Name)
		delete(needDeleting, tn)

		key := deviceTrait{
			name:  newDevice.Name,
			trait: tn,
		}
		if _, ok := a[key]; ok {
			continue // we already know the device has this trait, don't announce and don't remove
		}

		// announce a new client trait
		needAnnouncing = append(needAnnouncing, tn)
	}
	if oldDevice != nil {
		for tn := range needDeleting {
			a.deleteDeviceTrait(oldDevice.Name, tn)
		}
	}
	return needAnnouncing
}

// deleteDevice undoes and removes all the traits the device has.
func (a announcedTraits) deleteDevice(device *gen.Device) {
	for _, t := range device.Metadata.Traits {
		a.deleteDeviceTrait(device.Name, trait.Name(t.Name))
	}
}

func (a announcedTraits) deleteDeviceTrait(name string, tn trait.Name) {
	key := deviceTrait{name: name, trait: tn}
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
