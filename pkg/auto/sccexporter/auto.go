// Package sccexporter exports device telemetry from an on-premise Smart Core
// instance to Smart Core Connect (SCC). It discovers devices by trait, polls the
// typed trait API on a schedule, and publishes each device's data to the Connect
// telemetry (Event Grid MQTT) broker as UDMI: per-device pointset telemetry plus
// periodic device-metadata (discovery). Only the Meter trait is supported today.
//
// See docs/connect-telemetry-ingest.md for the topic grammar and payload contract.
package sccexporter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/sccexporter/config"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const AutoName = "sccexporter"

var Factory auto.Factory = factory{}

type factory struct{}

type AutoImpl struct {
	*service.Service[config.Root]
	auto.Services

	meterClient     meterpb.MeterApiClient
	meterInfoClient meterpb.MeterInfoClient

	// pointNaming is the resolved payload naming mode for the current config; it
	// selects raw vs DBO point keys when collectors are built. Zero value ⇒ DBO.
	pointNaming dbo.Naming
}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &AutoImpl{
		Services: services,
	}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ParseConfig))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

func (a *AutoImpl) initialiseClients(n *node.Node) {
	a.meterClient = meterpb.NewMeterApiClient(n.ClientConn())
	a.meterInfoClient = meterpb.NewMeterInfoClient(n.ClientConn())
}

func (a *AutoImpl) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *AutoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	a.initialiseClients(a.Node)
	a.pointNaming = dbo.Naming(cfg.PointNaming)

	grp, autoCtx := errgroup.WithContext(ctx)

	pub, err := newPublisher(autoCtx, cfg.Mqtt, a.CloudCredential, a.Logger)
	if err != nil {
		a.Logger.Error("failed to create mqtt client", zap.Error(err))
		return err
	}

	allDevices := make(map[string]*device)
	t := a.now()
	iterationCount := 0
	grp.Go(func() error {
		for {
			next := cfg.Mqtt.SendInterval.Next(t)
			select {
			case <-autoCtx.Done():
				return nil
			case <-time.After(time.Until(next)):
				t = a.now()
			}

			// Publish discovery on the first run and then every N cycles; refresh
			// the device list on the same cadence.
			publishDiscovery := (iterationCount % *cfg.Mqtt.MetadataInterval) == 0
			iterationCount++

			if publishDiscovery {
				// Refresh into a fresh map and only swap on success, so a transient
				// ListDevices failure retains the previous export set rather than
				// blanking it until the next discovery cycle (potentially minutes away).
				refreshed := make(map[string]*device)
				if err := a.refreshDevices(autoCtx, cfg.Traits, refreshed); err != nil {
					a.Logger.Error("error refreshing device list; retaining previous set", zap.Error(err))
				} else {
					allDevices = refreshed
				}
			}

			a.publishCycle(autoCtx, cfg, pub, allDevices, publishDiscovery)
		}
	})

	// applyConfig returns immediately; background tasks run until ctx is cancelled
	// (on reconfigure or stop), after which we disconnect cleanly.
	go func() {
		err := grp.Wait()
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		pub.close(disconnectCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			a.Logger.Error("sccexporter automation stopped with error", zap.Error(err))
		}
	}()

	return nil
}

// publishCycle publishes telemetry (and, when publishDiscovery is set, discovery)
// for every device, bounding concurrency to avoid overwhelming slow devices.
func (a *AutoImpl) publishCycle(ctx context.Context, cfg config.Root, pub *publisher, devices map[string]*device, publishDiscovery bool) {
	var wg errgroup.Group
	wg.SetLimit(100)
	for _, dev := range devices {
		wg.Go(func() error {
			a.publishDevice(ctx, cfg, pub, dev, publishDiscovery)
			return nil
		})
	}
	_ = wg.Wait()
}

func (a *AutoImpl) publishDevice(ctx context.Context, cfg config.Root, pub *publisher, dev *device, publishDiscovery bool) {
	// Bound the trait fetches so a slow device can't stall the cycle.
	fetchCtx, cancel := context.WithTimeout(ctx, cfg.FetchTimeout.Duration)
	defer cancel()

	now := a.now()

	if ev, ok := dev.buildTelemetry(fetchCtx, now, a.Logger); ok {
		a.publishJSON(ctx, pub, pointsetTopic(cfg.Mqtt.TopicPrefix, dev.name), ev, "telemetry", dev.name)
	}

	if publishDiscovery {
		a.publishJSON(ctx, pub, discoveryTopic(cfg.Mqtt.TopicPrefix, dev.name), dev.buildDiscovery(now), "discovery", dev.name)
	}
}

func (a *AutoImpl) publishJSON(ctx context.Context, pub *publisher, topic string, payload any, kind, deviceName string) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		a.Logger.Error("failed to marshal "+kind, zap.String("device", deviceName), zap.Error(err))
		return
	}
	if err := pub.publish(ctx, topic, bytes); err != nil {
		a.Logger.Warn("failed to publish "+kind, zap.String("device", deviceName), zap.String("topic", topic), zap.Error(err))
	}
}

// refreshDevices discovers all devices implementing the configured traits and
// (re)builds the export set.
func (a *AutoImpl) refreshDevices(ctx context.Context, traits []string, allDevices map[string]*device) error {
	for _, traitName := range traits {
		if err := a.getAllTraitImplementors(ctx, trait.Name(traitName), allDevices); err != nil {
			a.Logger.Error("failed to get devices for trait", zap.String("trait", traitName), zap.Error(err))
			return err
		}
	}
	return nil
}

// getAllTraitImplementors populates devices with those implementing traitName,
// attaching a collector for the trait.
func (a *AutoImpl) getAllTraitImplementors(ctx context.Context, traitName trait.Name, devices map[string]*device) error {
	resp, err := a.Services.Devices.ListDevices(ctx, &devicespb.ListDevicesRequest{
		Query: &devicespb.Device_Query{
			Conditions: []*devicespb.Device_Query_Condition{
				{
					Field: "metadata.traits.name",
					Value: &devicespb.Device_Query_Condition_StringEqual{
						StringEqual: string(traitName),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, deviceInfo := range resp.Devices {
		collector := a.newCollector(ctx, traitName, deviceInfo.Name)
		if collector == nil {
			a.Logger.Warn("trait is configured but not supported",
				zap.String("trait", string(traitName)), zap.String("device", deviceInfo.Name))
			continue
		}
		dev, ok := devices[deviceInfo.Name]
		if !ok {
			dev = &device{name: deviceInfo.Name, metaData: deviceInfo.Metadata}
			devices[deviceInfo.Name] = dev
		}
		dev.collectors = append(dev.collectors, collector)
	}
	return nil
}

// newCollector builds a collector for traitName, or nil if the trait is not
// supported for export.
func (a *AutoImpl) newCollector(ctx context.Context, traitName trait.Name, deviceName string) traitCollector {
	switch traitName {
	case meterpb.TraitName:
		c := &meterCollector{name: deviceName, client: a.meterClient, naming: a.pointNaming}
		// Fetch the reading support once; its units and production capability drive
		// both the telemetry point selection and the discovery inventory.
		if support, err := a.meterInfoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: deviceName}); err != nil {
			a.Logger.Warn("failed to get meter info", zap.String("device", deviceName), zap.Error(err))
		} else {
			c.support = support
		}
		return c
	default:
		return nil
	}
}
