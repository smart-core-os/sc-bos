// Package connecttelemetry exports device telemetry from an on-premise Smart Core
// instance to Smart Core Connect (SCC). It discovers devices by trait, polls the
// typed trait API on a schedule, and publishes each device's data to the Connect
// telemetry (Event Grid MQTT) broker as UDMI: per-device pointset telemetry plus
// periodic device-metadata (discovery). The Meter trait is exported for every
// device that implements it; the ResourceUse trait only for the local node, which
// publishes its own host resources.
//
// When health is enabled it also publishes the health checks this node evaluates,
// over the same connection.
//
// See docs/connect-telemetry-ingest.md for the topic grammar and payload contract.
package connecttelemetry

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/connecttelemetry/config"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const AutoName = "connecttelemetry"

var Factory auto.Factory = factory{}

type factory struct{}

type AutoImpl struct {
	*service.Service[config.Root]
	auto.Services

	meterClient     meterpb.MeterApiClient
	meterInfoClient meterpb.MeterInfoClient

	resourceUseClient resourceusepb.ResourceUseApiClient

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
	a.resourceUseClient = resourceusepb.NewResourceUseApiClient(n.ClientConn())
}

func (a *AutoImpl) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *AutoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	health := cfg.HealthEnabled()
	if health && a.LocalHealthChecks == nil {
		// as config.ParseConfig does for a config with no traits and health off
		if len(cfg.Traits) == 0 {
			return errors.New("nothing to publish: health is the only output, but this node's health checks are unavailable")
		}
		a.Logger.Warn("health publishing is enabled but this node's health checks are unavailable; not publishing health")
		health = false
	}

	a.initialiseClients(a.Node)
	a.pointNaming = dbo.Naming(cfg.PointNaming)

	grp, autoCtx := errgroup.WithContext(ctx)

	pub, err := newPublisher(autoCtx, cfg.Mqtt, a.CloudCredential, a.SystemCheck, a.Logger)
	if err != nil {
		a.Logger.Error("failed to create mqtt client", zap.Error(err))
		return err
	}

	if len(cfg.Traits) > 0 {
		a.runTelemetry(autoCtx, grp, cfg, pub)
	}
	if health {
		a.runHealth(autoCtx, grp, cfg, pub)
	}

	// applyConfig returns immediately; background tasks run until ctx is cancelled
	// (on reconfigure or stop), after which we disconnect cleanly.
	go func() {
		err := grp.Wait()
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		pub.close(disconnectCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			a.Logger.Error("connecttelemetry automation stopped with error", zap.Error(err))
		}
	}()

	return nil
}

// telemetrySink is where telemetry and discovery go, a *publisher outside tests.
type telemetrySink interface {
	// awaitConnection blocks until the connection is up, or ctx is done.
	awaitConnection(ctx context.Context) error
	// publish sends payload to topic at the configured QoS.
	publish(ctx context.Context, topic string, payload []byte) error
}

// runTelemetry starts polling the configured traits and publishing their telemetry and discovery.
func (a *AutoImpl) runTelemetry(autoCtx context.Context, grp *errgroup.Group, cfg config.Root, sink telemetrySink) {
	allDevices := make(map[string]*device)
	refreshDue := true
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

			// Owe every device its discovery on the first cycle and then every N
			// cycles, and refresh the device list on the same cadence. Skipped cycles
			// count, keeping the cadence on the schedule; whatever is owed goes on the
			// next cycle that reaches the broker.
			if iterationCount%*cfg.Mqtt.MetadataInterval == 0 {
				refreshDue = true
				for _, dev := range allDevices {
					dev.discoveryOwed = true
				}
			}
			iterationCount++

			// Wait for the broker before refreshing or fetching, so the metadata and
			// readings are current when they go, but only until the next send, which
			// takes over from this one.
			waitCtx, cancel := context.WithTimeout(autoCtx, cfg.Mqtt.SendInterval.Next(t).Sub(a.now()))
			err := sink.awaitConnection(waitCtx)
			cancel()
			if err != nil {
				if autoCtx.Err() != nil {
					return nil
				}
				a.Logger.Warn("broker not connected; skipping telemetry cycle", zap.Error(err))
				continue
			}

			if refreshDue {
				// Refresh into a fresh map and only swap on success, so a transient
				// ListDevices failure retains the previous export set rather than
				// blanking it. A failed refresh is retried next cycle.
				refreshed := make(map[string]*device)
				if err := a.refreshDevices(autoCtx, cfg, refreshed); err != nil {
					a.Logger.Error("error refreshing device list; retaining previous set", zap.Error(err))
				} else {
					allDevices = refreshed
					refreshDue = false
				}
			}

			a.publishCycle(autoCtx, cfg, sink, allDevices)
		}
	})
}

// runHealth starts publishing this node's own health checks.
func (a *AutoImpl) runHealth(autoCtx context.Context, grp *errgroup.Group, cfg config.Root, pub *publisher) {
	hp := newHealthPublisher(pub, a.LocalHealthChecks, a.Node, cfg.Mqtt.TopicPrefix,
		cfg.Health.ManifestInterval.Duration, cfg.Health.MaxPublishRate, a.Logger.Named("health"))
	grp.Go(func() error { return hp.run(autoCtx) })
}

// publishCycle publishes telemetry, and any discovery owed, for every device,
// bounding concurrency to avoid overwhelming slow devices.
func (a *AutoImpl) publishCycle(ctx context.Context, cfg config.Root, sink telemetrySink, devices map[string]*device) {
	var wg errgroup.Group
	wg.SetLimit(maxConcurrentFetches)
	for _, dev := range devices {
		wg.Go(func() error {
			a.publishDevice(ctx, cfg, sink, dev)
			return nil
		})
	}
	_ = wg.Wait()
}

func (a *AutoImpl) publishDevice(ctx context.Context, cfg config.Root, sink telemetrySink, dev *device) {
	// Bound the trait fetches so a slow device can't stall the cycle.
	fetchCtx, cancel := context.WithTimeout(ctx, cfg.FetchTimeout.Duration)
	defer cancel()

	now := a.now()

	if ev, ok := dev.buildTelemetry(fetchCtx, now, a.Logger); ok {
		a.publishJSON(ctx, sink, pointsetTopic(cfg.Mqtt.TopicPrefix, dev.name), ev, "telemetry", dev.name)
	}

	if dev.discoveryOwed {
		if a.publishJSON(ctx, sink, discoveryTopic(cfg.Mqtt.TopicPrefix, dev.name), dev.buildDiscovery(now), "discovery", dev.name) {
			dev.discoveryOwed = false
		}
	}
}

// publishJSON reports whether payload was published.
func (a *AutoImpl) publishJSON(ctx context.Context, sink telemetrySink, topic string, payload any, kind, deviceName string) bool {
	bytes, err := json.Marshal(payload)
	if err != nil {
		a.Logger.Error("failed to marshal "+kind, zap.String("device", deviceName), zap.Error(err))
		return false
	}
	if err := sink.publish(ctx, topic, bytes); err != nil {
		a.Logger.Warn("failed to publish "+kind, zap.String("device", deviceName), zap.String("topic", topic), zap.Error(err))
		return false
	}
	return true
}

// refreshDevices discovers all devices implementing the configured traits and
// (re)builds the export set.
func (a *AutoImpl) refreshDevices(ctx context.Context, cfg config.Root, allDevices map[string]*device) error {
	for _, traitName := range cfg.Traits {
		if err := a.getAllTraitImplementors(ctx, trait.Name(traitName), cfg.FetchTimeout.Duration, allDevices); err != nil {
			a.Logger.Error("failed to get devices for trait", zap.String("trait", traitName), zap.Error(err))
			return err
		}
	}
	return nil
}

// getAllTraitImplementors populates devices with those implementing traitName,
// attaching a collector for the trait.
func (a *AutoImpl) getAllTraitImplementors(ctx context.Context, traitName trait.Name, fetchTimeout time.Duration, devices map[string]*device) error {
	conditions := []*devicespb.Device_Query_Condition{
		{
			Field: "metadata.traits.name",
			Value: &devicespb.Device_Query_Condition_StringEqual{
				StringEqual: string(traitName),
			},
		},
	}
	if traitName == resourceusepb.TraitName {
		// The reporting node publishes its own resources: a gateway sees its
		// cohort's ResourceUse too, but leaves those nodes to publish their own.
		conditions = append(conditions, &devicespb.Device_Query_Condition{
			Field: "name",
			Value: &devicespb.Device_Query_Condition_StringEqual{
				StringEqual: a.Node.Name(),
			},
		})
	}
	var deviceInfos []*devicespb.Device
	req := &devicespb.ListDevicesRequest{
		Query:    &devicespb.Device_Query{Conditions: conditions},
		PageSize: listDevicesPageSize,
	}
	for {
		resp, err := a.Services.Devices.ListDevices(ctx, req)
		if err != nil {
			return err
		}
		deviceInfos = append(deviceInfos, resp.Devices...)
		req.PageToken = resp.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	// Building a collector fetches from the device, so bound each build as
	// publishDevice bounds its fetches; a slow device can't stall the refresh.
	type built struct {
		collector traitCollector
		supported bool
	}
	builds := make([]built, len(deviceInfos))
	var grp errgroup.Group
	grp.SetLimit(maxConcurrentFetches)
	for i, deviceInfo := range deviceInfos {
		grp.Go(func() error {
			fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
			defer cancel()
			builds[i].collector, builds[i].supported = a.newCollector(fetchCtx, traitName, deviceInfo.Name)
			return nil
		})
	}
	_ = grp.Wait()

	for i, deviceInfo := range deviceInfos {
		if !builds[i].supported {
			a.Logger.Warn("trait is configured but not supported",
				zap.String("trait", string(traitName)), zap.String("device", deviceInfo.Name))
			continue
		}
		dev, ok := devices[deviceInfo.Name]
		if !ok {
			dev = &device{name: deviceInfo.Name, metaData: deviceInfo.Metadata, discoveryOwed: true}
			devices[deviceInfo.Name] = dev
		}
		dev.collectors = append(dev.collectors, builds[i].collector)
	}
	return nil
}

// listDevicesPageSize is the devices server's maximum page size, so most sites
// list each trait in one round trip.
const listDevicesPageSize = 1000

// maxConcurrentFetches bounds how many devices are fetched from at once, to
// avoid overwhelming slow devices.
const maxConcurrentFetches = 100

// newCollector builds a collector for traitName. supported is false if the trait
// can't be exported.
func (a *AutoImpl) newCollector(ctx context.Context, traitName trait.Name, deviceName string) (_ traitCollector, supported bool) {
	switch traitName {
	case resourceusepb.TraitName:
		c := &resourceUseCollector{name: deviceName, client: a.resourceUseClient}
		// Seed the reading so discovery has an inventory even if the first
		// telemetry fetch fails.
		if r, err := a.resourceUseClient.GetResourceUse(ctx, &resourceusepb.GetResourceUseRequest{Name: deviceName}); err != nil {
			a.Logger.Warn("failed to get resource use", zap.String("device", deviceName), zap.Error(err))
		} else {
			c.last = r
		}
		return c, true
	case meterpb.TraitName:
		c := &meterCollector{name: deviceName, client: a.meterClient, naming: a.pointNaming}
		// Fetch the reading support once; its units and production capability drive
		// both the telemetry point selection and the discovery inventory.
		if support, err := a.meterInfoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: deviceName}); err != nil {
			a.Logger.Warn("failed to get meter info", zap.String("device", deviceName), zap.Error(err))
		} else {
			c.support = support
		}
		return c, true
	default:
		return nil, false
	}
}
