package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"path"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/connect"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/task/serviceapi"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

// cloudCredentialSource returns the node's Connect credential for drivers,
// automations and zones, or nil when no cloud connection is configured. The
// returned Credential reads the connection state per call, so it follows
// certificate renewals and enrollment on a node that enrols after start-up.
//
// The test is on the config, not the connection: initCloud builds a cloud.Conn
// whether or not a cloud block was configured, so c.Cloud alone would never be
// nil and the documented "not configured" state would be unreachable. Compare
// the same test guarding the poll loop in Controller.Run.
func (c *Controller) cloudCredentialSource() connect.Credential {
	if c.SystemConfig.Cloud == nil || c.Cloud == nil {
		return nil // plain nil, not a typed nil: callers nil-check the interface
	}
	return cloudCredential{state: c.Cloud.State}
}

func (c *Controller) startDrivers(configs []driver.RawConfig) (*service.Map, error) {
	ctxServices := driver.Services{
		Logger:          c.Logger.Named("driver"),
		Node:            c.Node,
		ClientTLSConfig: c.ClientTLSConfig,
		CloudCredential: c.cloudCredentialSource(),
		HTTPMux:         c.Mux,
		Database:        c.Database,
	}

	m := service.NewMap(func(id, kind string) (service.Lifecycle, error) {
		driverServices := ctxServices
		driverServices.Config = &serviceConfigStore{store: c.ControllerConfig.Drivers(), id: id}
		driverServices.Logger = loggerWithServiceInfo(driverServices.Logger, id, kind)
		driverServices.Health = healthChecksForService(c.CheckRegistry, id, kind)
		driverServices.SystemCheck = newDriverSystemCheck(driverServices.Health, id, kind, driverServices.Logger)

		f, ok := c.SystemConfig.DriverFactories[kind]
		if !ok {
			return nil, fmt.Errorf("unsupported driver type %v", kind)
		}
		return f.New(driverServices), nil
	}, service.IdIsRequired)

	logger := c.Logger.Named("driver")
	for _, cfg := range configs {
		if _, _, err := m.Create(cfg.Name, cfg.Type, service.State{Active: !cfg.Disabled, Config: cfg.Raw}); err != nil {
			loggerWithServiceInfo(logger, cfg.Name, cfg.Type).Warn("Failed to create service", zap.Error(err))
		}
	}
	return m, nil
}

func (c *Controller) startAutomations(configs []auto.RawConfig) (*service.Map, error) {
	ctxServices := auto.Services{
		Logger:          c.Logger.Named("auto"),
		Node:            c.Node,
		Devices:         c.Devices,
		Database:        c.Database,
		Stores:          c.Stores,
		GRPCServices:    c.GRPC,
		CohortManager:   c.ManagerConn,
		ClientTLSConfig: c.ClientTLSConfig,
		CloudCredential: c.cloudCredentialSource(),
	}

	m := service.NewMap(func(id, kind string) (service.Lifecycle, error) {
		autoServices := ctxServices
		autoServices.Config = &serviceConfigStore{store: c.ControllerConfig.Automations(), id: id}
		autoServices.Logger = loggerWithServiceInfo(autoServices.Logger, id, kind)
		autoServices.Health = healthChecksForService(c.CheckRegistry, id, kind)

		f, ok := c.SystemConfig.AutoFactories[kind]
		if !ok {
			return nil, fmt.Errorf("unsupported automation type %v", kind)
		}
		return f.New(autoServices), nil
	}, service.IdIsRequired)

	logger := c.Logger.Named("auto")
	for _, cfg := range configs {
		if _, _, err := m.Create(cfg.Name, cfg.Type, service.State{Active: !cfg.Disabled, Config: cfg.Raw}); err != nil {
			loggerWithServiceInfo(logger, cfg.Name, cfg.Type).Warn("Failed to create service", zap.Error(err))
		}
	}
	return m, nil
}

// cloudCredential adapts the node's cloud connection to connect.Credential,
// presenting the current Connect leaf certificate, node id and API origin. It holds
// a state function rather than the *cloud.Conn itself so that all three accessors
// read the connection state per call - tracking certificate renewals and enrollment
// without reconnecting - and so the adapter can be exercised without opening a
// connection.
type cloudCredential struct{ state func() cloud.ConnState }

var _ connect.Credential = cloudCredential{}

func (c cloudCredential) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	reg := c.state().Registration
	if reg == nil {
		return nil, fmt.Errorf("cloud connection is not enrolled; no client certificate available")
	}
	return reg.TLSCertificate(), nil
}

func (c cloudCredential) NodeID() string {
	if reg := c.state().Registration; reg != nil {
		return reg.NodeID()
	}
	return ""
}

func (c cloudCredential) APIEndpoint() string {
	if reg := c.state().Registration; reg != nil {
		return reg.APIEndpoint
	}
	return ""
}

func (c *Controller) startSystems() (*service.Map, error) {
	grpcEndpoint, err := c.SystemConfig.ExternalGRPCEndpoint()
	if err != nil {
		return nil, err
	}
	var httpEndpoint string
	if hp, err := c.SystemConfig.ExternalHTTPEndpoint(); err == nil {
		httpEndpoint = hp
	}
	ctxServices := system.Services{
		ConfigDirs:       c.SystemConfig.ConfigDirs,
		DataDir:          c.SystemConfig.DataDir,
		Logger:           c.Logger.Named("system"),
		Node:             c.Node,
		HealthChecks:     devicesToHealthCheckCollection(c.DeviceStore),
		GRPCEndpoint:     grpcEndpoint,
		HTTPEndpoint:     httpEndpoint,
		Database:         c.Database,
		Stores:           c.Stores,
		Accounts:         c.Accounts,
		HTTPMux:          c.Mux,
		DownloadRouter:   c.DownloadRouter,
		TokenValidators:  c.TokenValidators,
		ReflectionServer: c.ReflectionServer,
		GRPCCerts:        c.GRPCCerts,
		PrivateKey:       c.PrivateKey,
		CohortManager:    c.ManagerConn,
		ClientTLSConfig:  c.ClientTLSConfig,
		LogLevel:         c.LogLevel,
	}
	if c.LogCapture != nil {
		ctxServices.AddLogCore = c.LogCapture.Add
	}
	if c.rebootCh != nil {
		rebootCh := c.rebootCh
		ctxServices.RequestReboot = func() {
			select {
			case rebootCh <- "":
			default:
			}
		}
	}
	m := service.NewMap(func(_, kind string) (service.Lifecycle, error) {
		f, ok := c.SystemConfig.SystemFactories[kind]
		if !ok {
			return nil, fmt.Errorf("unsupported system type %v", kind)
		}
		return f.New(ctxServices), nil
	}, service.IdIsKind)

	logger := c.Logger.Named("system")
	for kind, cfg := range c.SystemConfig.Systems {
		if _, _, err := m.Create("", kind, service.State{Active: !cfg.Disabled, Config: cfg.Raw}); err != nil {
			loggerWithServiceInfo(logger, kind, kind).Warn("Failed to create service", zap.Error(err))
		}
	}
	return m, nil
}

func (c *Controller) startZones(configs []zone.RawConfig) (*service.Map, error) {
	ctxServices := zone.Services{
		Logger:          c.Logger.Named("zone"),
		Node:            c.Node,
		ClientTLSConfig: c.ClientTLSConfig,
		CloudCredential: c.cloudCredentialSource(),
		HTTPMux:         c.Mux,
		DriverFactories: c.SystemConfig.DriverFactories,
	}

	m := service.NewMap(func(id, kind string) (service.Lifecycle, error) {
		zoneServices := ctxServices
		zoneServices.Config = &serviceConfigStore{store: c.ControllerConfig.Zones(), id: id}
		zoneServices.Logger = loggerWithServiceInfo(zoneServices.Logger, id, kind)
		zoneServices.Health = healthChecksForService(c.CheckRegistry, id, kind)

		f, ok := c.SystemConfig.ZoneFactories[kind]
		if !ok {
			return nil, fmt.Errorf("unsupported zone type %v", kind)
		}
		return f.New(zoneServices), nil
	}, service.IdIsRequired)

	logger := c.Logger.Named("zone")
	for _, cfg := range configs {
		if _, _, err := m.Create(cfg.Name, cfg.Type, service.State{Active: !cfg.Disabled, Config: cfg.Raw}); err != nil {
			loggerWithServiceInfo(logger, cfg.Name, cfg.Type).Warn("Failed to create service", zap.Error(err))
		}
	}
	return m, nil
}

func logServiceMapChanges(ctx context.Context, logger *zap.Logger, m *service.Map) {
	now, changes := m.GetAndListenState(ctx)
	for _, record := range now {
		logServiceRecordChange(logger, nil, record)
	}
	for change := range changes {
		logServiceRecordChange(logger, change.OldValue, change.NewValue)
	}
}

func logServiceRecordChange(logger *zap.Logger, oldVal, newVal *service.StateRecord) {
	switch {
	case newVal != nil:
		// the vars match the same fields passed to the services in startFoo
		logger = loggerWithServiceInfo(logger, newVal.Id, newVal.Kind)
	case oldVal != nil:
		logger = loggerWithServiceInfo(logger, oldVal.Id, oldVal.Kind)
	}
	switch {
	case oldVal == nil && newVal != nil: // created or initial snapshot
		if newVal.State.Err != nil {
			logger.Warn("Failed to configure service", zap.Error(newVal.State.Err))
		} else {
			logger.Debug("Created", zap.Bool("active", newVal.State.Active), zap.Bool("loading", newVal.State.Loading))
		}
	case newVal == nil: // removed
		logger.Debug("Removed")
	case oldVal == nil: // created (with no new value — should not normally happen)
	case !newVal.State.Active && newVal.State.Err != nil && oldVal.State.Err == nil: // error
		logger.Warn("Failed to load", zap.Error(newVal.State.Err))
	case oldVal.State.Active && !newVal.State.Active: // stopped
		logger.Debug("Stopped", zap.Error(newVal.State.Err))
	case newVal.State.Active && newVal.State.Loading && !newVal.State.NextAttemptTime.IsZero(): // retrying
		// rely on the service itself to log any issues that caused the retry
	case newVal.State.Active && newVal.State.Loading: // loading
		logger.Debug("Loading")
	case !oldVal.State.Active && newVal.State.Active || oldVal.State.Loading && !newVal.State.Loading: // started
		logger.Debug("Started")
	default:
		type state struct {
			Active, Loading bool
			Error           error
		}
		oldState := state{Active: oldVal.State.Active, Loading: oldVal.State.Loading, Error: oldVal.State.Err}
		newState := state{Active: newVal.State.Active, Loading: newVal.State.Loading, Error: newVal.State.Err}
		logger.Debug("Updated", zap.Any("old", oldState), zap.Any("new", newState))
	}
}

// Log field keys attached to service lifecycle loggers.
// Clients filter PullLogMessages on these keys; keep in sync with logFields in ui/ops/src/api/ui/log.js.
const (
	logFieldServiceID   = "service.id"
	logFieldServiceKind = "service.kind"
)

func loggerWithServiceInfo(logger *zap.Logger, id, kind string) *zap.Logger {
	return logger.With(zap.String(logFieldServiceID, id), zap.String(logFieldServiceKind, kind))
}

func healthChecksForService(r *healthpb.Registry, id, kind string) *healthpb.Checks {
	owner := fmt.Sprintf("%s:%s", kind, id)
	return r.ForOwner(owner)
}

// newDriverSystemCheck creates a driver-level system check registered under the driver's own name.
// Drivers should call MarkFailed/MarkRunning to reflect connectivity state, and must call
// Dispose in their stop handler. Returns nil if the check cannot be created.
func newDriverSystemCheck(health *healthpb.Checks, id, kind string, logger *zap.Logger) service.SystemCheck {
	check, err := health.NewFaultCheck(id, &healthpb.HealthCheck{
		Id:          "systemStatusCheck",
		DisplayName: "System Status Check",
		Description: fmt.Sprintf("Checks the %s driver is connected and operating correctly", kind),
	})
	if err != nil {
		logger.Warn("failed to create driver system check", zap.Error(err))
		return nil
	}
	return check
}

func devicesToHealthCheckCollection(d *devicespb.Collection) system.HealthCheckCollection {
	return (*devicesHealthCheckCollection)(d)
}

type devicesHealthCheckCollection devicespb.Collection

func (d *devicesHealthCheckCollection) MergeHealthChecks(name string, checks ...*healthpb.HealthCheck) error {
	_, err := (*devicespb.Collection)(d).Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
		dstDev := dst.(*devicespb.Device)
		dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, checks...)
	}), resource.WithCreateIfAbsent())
	return err
}

func (d *devicesHealthCheckCollection) RemoveHealthChecks(name string, ids ...string) error {
	_, err := (*devicespb.Collection)(d).Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, _ proto.Message) {
		dstDev := dst.(*devicespb.Device)
		for _, id := range ids {
			dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
		}
	}))
	if code := status.Code(err); code == codes.NotFound {
		err = nil
	}
	return err
}

func announceServices[M ~map[string]T, T any](c *Controller, name string, services *service.Map, factories M, store serviceapi.Store) node.Undo {
	srv := serviceapi.NewApi(services,
		serviceapi.WithKnownTypesFromMapKeys(factories),
		serviceapi.WithLogger(c.Logger.Named("serviceapi")),
		serviceapi.WithStore(store),
	)
	return announceNodeServer(c.Node, name, srv)
}

func announceAutoServices[M ~map[string]T, T any](c *Controller, services *service.Map, factories M) node.Undo {
	// special because the config name isn't the name we announce as
	srv := serviceapi.NewApi(services,
		serviceapi.WithKnownTypesFromMapKeys(factories),
		serviceapi.WithLogger(c.Logger.Named("serviceapi")),
		serviceapi.WithStore(c.ControllerConfig.Automations()),
	)
	return announceNodeServer(c.Node, "automations", srv)
}

func announceSystemServices[M ~map[string]T, T any](c *Controller, services *service.Map, factories M) node.Undo {
	// special because we don't support writing this config, yet
	// todo: support writing system config
	srv := serviceapi.NewApi(services,
		serviceapi.WithKnownTypesFromMapKeys(factories),
		serviceapi.WithLogger(c.Logger.Named("serviceapi")),
	)
	return announceNodeServer(c.Node, "systems", srv)
}

func announceNodeServer(n *node.Node, base string, srv servicespb.ServicesApiServer) node.Undo {
	var undos []node.Undo
	undos = append(undos, n.Announce(base,
		node.HasServer(servicespb.RegisterServicesApiServer, srv),
		node.HasDeviceType(metadatapb.Metadata_SERVICE),
	))
	if n.Name() != "" {
		undos = append(undos, n.Announce(path.Join(n.Name(), base),
			node.HasServer(servicespb.RegisterServicesApiServer, srv),
			node.HasDeviceType(metadatapb.Metadata_SERVICE),
		))
	}
	return node.UndoAll(undos...)
}

type serviceConfigStore struct {
	store serviceapi.Store
	id    string
}

func (s *serviceConfigStore) UpdateConfig(ctx context.Context, data []byte) error {
	return s.store.SaveConfig(ctx, s.id, "", data)
}
