// Package opcua implements a Smart Core driver for OPC UA servers.
// It subscribes to OPC UA variable nodes and exposes their values through Smart Core traits
// including Meter, Electric, Transport, and UDMI.
//
// The driver creates an internal device instance for each configured device, which manages
// OPC UA subscriptions and routes value changes to the appropriate trait handlers.
package opcua

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const DriverName = "opcua"

var Factory driver.Factory = factory{}

type factory struct{}

type EventHandler interface {
	handleEvent(ctx context.Context, node *ua.NodeID, value any)
}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)

	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(services.Node),
		health:      services.Health,
		logger:      logger,
		systemCheck: services.SystemCheck,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ParseConfig),
		service.WithOnStop[config.Root](d.onStop),
		service.WithRetry[config.Root](
			service.RetryWithLogger(func(logContext service.RetryContext) {
				logContext.LogTo("applyConfig", logger)
			}),
			service.RetryWithInitialDelay(2*time.Second),
			service.RetryWithMinDelay(2*time.Second),
			service.RetryWithMaxDelay(30*time.Second),
		),
	)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	health    *healthpb.Checks
	logger    *zap.Logger

	systemCheck service.SystemCheck
	checks      []*healthpb.FaultCheck
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	a := d.announcer.Replace(ctx)

	d.dispose()

	// the parameters are workable or ParseConfig would have rejected them, but say so if they
	// look likely to overload the server or overflow its queues. Before connecting, so the
	// warning still lands when a misconfigured server is also unreachable.
	for _, w := range cfg.Conn.MonitoringWarnings() {
		d.logger.Warn("monitoring parameters may cause trouble: " + w)
	}

	opcClient, err := d.connectOpcClient(ctx, cfg)
	if err != nil {
		d.logger.Warn("Connect error", zap.Error(err))
		return err
	}

	client := NewClient(opcClient, d.logger, cfg.Conn)

	a.Announce(cfg.Name, node.HasMetadata(cfg.Meta))

	grp, ctx := errgroup.WithContext(ctx)
	for _, dev := range cfg.Devices {
		allFeatures := []node.Feature{node.HasMetadata(dev.Meta), node.HasDeviceType(metadatapb.Metadata_DEVICE)}

		faultCheck, err := d.health.NewFaultCheck(dev.Name, getDeviceHealthCheck(dev.Health.OccupantImpact.ToProto(), dev.Health.EquipmentImpact.ToProto()))
		if err != nil {
			d.logger.Error("failed to create device fault check", zap.String("device", dev.Name), zap.Error(err))
			return err
		}
		d.checks = append(d.checks, faultCheck)

		opcDev := newDevice(&dev, d.logger, client, faultCheck, d.systemCheck)

		for _, t := range dev.Traits {
			switch t.Kind {
			case meterpb.TraitName:
				m, err := newMeter(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", meterpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, m)
				allFeatures = append(allFeatures,
					node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(m)),
					node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(m)),
					node.HasTrait(meterpb.TraitName),
				)

			case transportpb.TraitName:
				tr, err := newTransport(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", transportpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, tr)
				allFeatures = append(allFeatures,
					node.HasServer(transportpb.RegisterTransportApiServer, transportpb.TransportApiServer(tr)),
					node.HasServer(transportpb.RegisterTransportInfoServer, transportpb.TransportInfoServer(tr)),
					node.HasTrait(transportpb.TraitName),
				)

			case udmipb.TraitName:
				u, err := newUdmi(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", udmipb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, u)
				allFeatures = append(allFeatures,
					node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(u)),
					node.HasTrait(udmipb.TraitName),
				)

			case trait.Electric:
				e, err := newElectric(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", trait.Electric), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, e)
				allFeatures = append(allFeatures,
					node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(e)),
					node.HasTrait(trait.Electric),
				)

			case healthpb.TraitName:
				h, err := newHealth(t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", healthpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, h)
				for _, check := range h.cfg.Checks {
					c := getDeviceErrorCheck(check)
					fc, err := d.health.NewFaultCheck(dev.Name, c)
					if err != nil {
						d.logger.Error("failed to create health fault check", zap.String("device", dev.Name), zap.String("check", check.Id), zap.Error(err))
						return err
					}
					h.errorChecks[check.Id] = fc
					d.checks = append(d.checks, fc)
				}
			default:
				d.logger.Error("unknown trait", zap.String("trait", t.Name))
			}
		}

		a.Announce(dev.Name, allFeatures...)
		dev := opcDev
		grp.Go(func() error {
			return dev.subscribe(ctx)
		})
	}

	go func() {
		err := grp.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			d.logger.Error("run error", zap.Error(err))
		}

		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = opcClient.Close(closeCtx); err != nil {
			d.logger.Warn("failed to close opcua client", zap.Error(err))
		}
	}()
	return nil
}

func (d *Driver) connectOpcClient(ctx context.Context, cfg config.Root) (*opcua.Client, error) {
	endpoint, opts, err := d.opcClientOptions(ctx, cfg.Conn)
	if err != nil {
		service.UpdateSystemCheck(d.systemCheck, err)
		return nil, err
	}

	opcClient, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		service.UpdateSystemCheck(d.systemCheck, err)
		d.logger.Error("error creating new client", zap.Error(err))
		return nil, err
	}

	err = opcClient.Connect(ctx)
	if err != nil {
		service.UpdateSystemCheck(d.systemCheck, err)
		d.logger.Error("error connecting to opc ua server", zap.Error(err))
		return nil, err
	}
	service.UpdateSystemCheck(d.systemCheck, nil)
	return opcClient, nil
}

// opcClientOptions works out how to connect to the server described by conn, returning the
// endpoint URL to dial and the client options that apply the configured security and
// credentials. The password, if any, is read from disk here rather than during config
// parsing so that a rotated secret is picked up by the driver's retry loop.
func (d *Driver) opcClientOptions(ctx context.Context, conn config.Conn) (string, []opcua.Option, error) {
	sec, err := conn.ResolveSecurity() // already validated during config.ParseConfig
	if err != nil {
		return "", nil, err
	}
	if sec.AnonymousInsecure() {
		// nothing to negotiate, so skip discovery and connect as the driver always has
		return conn.Endpoint, nil, nil
	}

	// note this dials the server without security, which a server that secures its
	// discovery endpoint will refuse
	endpoints, err := opcua.GetEndpoints(ctx, conn.Endpoint)
	if err != nil {
		d.logger.Error("error getting opc ua server endpoints", zap.String("endpoint", conn.Endpoint), zap.Error(err))
		return "", nil, fmt.Errorf("get endpoints %q: %w", conn.Endpoint, err)
	}
	ep, err := opcua.SelectEndpoint(endpoints, sec.PolicyURI, sec.Mode)
	if err != nil {
		d.logger.Error("no opc ua endpoint matches the configured security",
			zap.String("policy", sec.PolicyURI), zap.Stringer("mode", sec.Mode), zap.Error(err))
		return "", nil, fmt.Errorf("select endpoint: %w", err)
	}
	if ep.EndpointURL != conn.Endpoint {
		// we have to dial the URL the server advertises, strict servers reject a session
		// created against any other. Log it, because a server advertising a hostname this
		// controller can't resolve is the usual cause of a connect failure from here.
		d.logger.Info("using the endpoint url advertised by the opc ua server",
			zap.String("configured", conn.Endpoint), zap.String("advertised", ep.EndpointURL))
	}

	opts := []opcua.Option{
		// must come before the auth options: this creates the user identity token and sets
		// its policy id from the endpoint, AuthUsername only fills in an existing token
		opcua.SecurityFromEndpoint(ep, sec.TokenType),
		opcua.CertificateFile(sec.CertFile), // a no-op when empty
		opcua.PrivateKeyFile(sec.KeyFile),
	}
	if conn.Auth != nil {
		if sec.Mode == ua.MessageSecurityModeNone {
			d.logger.Warn("opc ua password will cross the network unencrypted, the configured security mode is None",
				zap.String("username", conn.Auth.Username))
		}
		pass, err := conn.Auth.Read()
		if err != nil {
			d.logger.Error("error reading opc ua password file", zap.String("passwordFile", conn.Auth.PasswordFile), zap.Error(err))
			return "", nil, fmt.Errorf("read password: %w", err)
		}
		opts = append(opts, opcua.AuthUsername(conn.Auth.Username, pass))
	} else {
		opts = append(opts, opcua.AuthAnonymous())
	}
	return ep.EndpointURL, opts, nil
}

func (d *Driver) onStop() {
	if d.systemCheck != nil {
		d.systemCheck.Dispose()
	}
	d.dispose()
}

func (d *Driver) dispose() {
	for _, c := range d.checks {
		if c != nil {
			c.Dispose()
		}
	}
	d.checks = nil
}
