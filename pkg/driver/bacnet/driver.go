package bacnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/adapt"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/ctxerr"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/merge"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const DriverName = "bacnet"

var Factory driver.Factory = factory{}

type factory struct{}

func (factory) New(services driver.Services) service.Lifecycle {
	return NewDriver(services)
}

func (factory) ConfigBlocks() []block.Block {
	return config.Blocks
}

// Driver brings BACnet devices into Smart Core.
type Driver struct {
	announcer *node.ReplaceAnnouncer // Any device we setup gets announced here
	logger    *zap.Logger

	*service.Service[config.Root]
	client *gobacnet.Client // How we interact with bacnet systems

	mu      sync.RWMutex
	devices *known.Map

	health      *healthpb.Checks
	systemCheck service.SystemCheck
	checks      []*healthpb.FaultCheck
	healthTasks []task.StopFn

	controllerMu      sync.Mutex
	controllerHealths map[string]*controllerHealth
}

func NewDriver(services driver.Services) *Driver {
	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(services.Node),
		devices:     known.NewMap(),
		health:      services.Health,
		systemCheck: services.SystemCheck,
		logger:      services.Logger.Named("bacnet"),
	}
	d.Service = service.New(service.MonoApply(d.applyConfig),
		service.WithParser(config.ReadBytes),
		service.WithOnStop[config.Root](func() {
			d.Clear()
			if d.systemCheck != nil {
				d.systemCheck.Dispose()
			}
		}))
	return d
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	// AnnounceContext only makes sense if using MonoApply, which we are in NewDriver
	rootAnnouncer := d.announcer.Replace(ctx)
	if cfg.Metadata != nil {
		rootAnnouncer = node.AnnounceFeatures(rootAnnouncer, node.HasMetadata(cfg.Metadata))
	}
	// we start fresh each time config is updated
	d.Clear()

	err := d.initClient(ctx, cfg, d.systemCheck)
	if err != nil {
		return err
	}

	d.controllerMu.Lock()
	d.controllerHealths = make(map[string]*controllerHealth)
	d.controllerMu.Unlock()

	devices := known.SyncContext(d.mu.RLocker(), d.devices)

	// setup all our devices and objects...
	for _, device := range cfg.Devices {
		// make sure to retry setting up devices in case they aren't yet online but might be in the future
		deviceName := adapt.DeviceName(device)
		logger := d.logger.With(zap.String("device", deviceName), zap.Uint32("deviceId", uint32(device.ID)),
			zap.Stringer("address", device.Comm.IP))

		// allow status reporting for this device
		scDeviceName := cfg.DeviceNamePrefix + deviceName

		// even if devices are offline, they still have metadata
		if device.Metadata != nil {
			rootAnnouncer.Announce(device.Name, node.HasMetadata(device.Metadata))
		}

		faultCheck, err := d.health.NewFaultCheck(scDeviceName, createDeviceHealthCheck(device.Health.OccupantImpact.ToProto(), device.Health.EquipmentImpact.ToProto()))
		if errors.Is(err, healthpb.ErrAlreadyExists) {
			logger.Warn("device health check already registered, health will not be tracked", zap.String("device", device.Name))
			faultCheck = nil
		} else if err != nil {
			logger.Error("failed to create device fault check", zap.String("device", device.Name), zap.Error(err))
			return err
		}

		if faultCheck != nil {
			d.checks = append(d.checks, faultCheck)
		}

		// For devices with a known IP, eagerly register them with their controller check.
		// WhoIs devices (no Comm.IP) are registered lazily inside configureDevice.
		var ctrlHealth *controllerHealth
		if device.Comm != nil && device.Comm.IP != nil {
			ctrlHealth, err = d.getOrCreateControllerCheck(ctx, cfg, device.Comm.IP.Addr())
			if err != nil {
				logger.Error("failed to create controller health check", zap.Error(err))
			} else {
				ctrlHealth.register(scDeviceName)
			}
		}

		go func() {
			// This is more complicated than I think it should be.
			// The issue is that the Context passed to Task is cancelled when the task returns.
			// We don't want this to happen as any announced names should live past the lifetime of the task.
			// To avoid this we have to split the cleanup of names from the cancellation of the task.
			cfgCtx := ctx
			cfgAnnouncer := node.NewReplaceAnnouncer(rootAnnouncer)
			cleanupLastAttempt := func() {}

			taskOpts := []task.Option{
				task.WithRetry(task.RetryUnlimited),
				task.WithBackoff(500*time.Millisecond, 30*time.Second),
				task.WithTimeout(10 * time.Second),
				// no WithErrorLogger as it's too noisy, we'll log errors ourselves
			}
			attempt := 0

			err := task.Run(cfgCtx, func(ctx context.Context) (task.Next, error) {
				attempt++
				// clean up any names that were announced during previous attempts
				cleanupLastAttempt()
				// make sure we can clean up announced names if the task is retried or the enclosing Service is stopped or reconfigured.
				var announceCtx context.Context
				announceCtx, cleanupLastAttempt = context.WithCancel(cfgCtx)
				announcer := cfgAnnouncer.Replace(announceCtx)

				// It's ok for configureDevices to receive the task context here as ctx is only used for queries
				err := d.configureDevice(ctx, announcer, cfg, device, devices, faultCheck, ctrlHealth, logger)

				if err != nil {
					if errors.Is(err, context.Canceled) {
						return task.Normal, err
					}

					switch attempt {
					case 1, 2:
						logger.Warn("Device offline? Will keep trying", zap.Error(err), zap.Int("attempt", attempt))
					case 3:
						logger.Warn("Device offline? Reducing logging.", zap.Error(err), zap.Int("attempt", attempt))
					default:
						if attempt%10 == 0 {
							logger.Debug("Device still offline? Will keep trying", zap.Error(err), zap.Int("attempt", attempt))
						}
					}
				} else {
					logger.Debug("Device configured successfully")
				}
				return task.Normal, err
			}, taskOpts...)
			if err != nil && !errors.Is(err, context.Canceled) {
				d.logger.Error("Cannot configure device", zap.Error(err))
			}
		}()
	}

	// Combine objects together into traits...
	for _, trait := range cfg.Traits {
		logger := d.logger.With(zap.Stringer("trait", trait.Kind), zap.String("name", trait.Name))
		faultCheck, err := d.health.NewFaultCheck(trait.Name, createTraitHealthCheck(trait.Kind, trait.Health.OccupantImpact.ToProto(), trait.Health.EquipmentImpact.ToProto()))
		if errors.Is(err, healthpb.ErrAlreadyExists) {
			logger.Warn("trait health check already registered, health will not be tracked", zap.Stringer("trait", trait.Kind), zap.String("name", trait.Name))
			faultCheck = nil
		} else if err != nil {
			logger.Error("failed to create trait fault check", zap.String("trait", trait.Name), zap.Error(err))
			return err
		}

		// special case health trait as it needs custom handling
		// as it doesn't announce to the node.Announcer in the same way as other traits
		if trait.Kind == healthpb.TraitName {
			h, err := merge.NewHealth(d.client, devices, d.health, faultCheck, trait, logger)

			if err != nil {
				logger.Error("failed to create a new health impl", zap.Error(err))
				if faultCheck != nil {
					faultCheck.Dispose()
				}
				return err
			}

			for _, check := range h.DeviceChecks {
				d.checks = append(d.checks, check)
			}

			stop, err := h.StartPoll(ctx)

			if err != nil {
				logger.Error("failed to start health polling", zap.Error(err))
				if faultCheck != nil {
					faultCheck.Dispose()
				}
				return err
			}

			d.healthTasks = append(d.healthTasks, stop)
			continue
		}

		impl, err := merge.IntoTrait(d.client, devices, faultCheck, trait, logger)
		if errors.Is(err, merge.ErrTraitNotSupported) {
			logger.Error("Cannot combine into trait, not supported")
			if faultCheck != nil {
				faultCheck.Dispose()
			}
			continue
		}
		if err != nil {
			logger.Error("Cannot combine into trait", zap.Error(err))
			if faultCheck != nil {
				faultCheck.Dispose()
			}
			continue
		}

		announcer := rootAnnouncer
		if trait.Metadata != nil {
			announcer = node.AnnounceFeatures(announcer, node.HasMetadata(trait.Metadata))
		}
		impl.AnnounceSelf(announcer)

		d.checks = append(d.checks, faultCheck)
	}

	return nil
}

func (d *Driver) initClient(ctx context.Context, cfg config.Root, systemCheck service.SystemCheck) error {
	client, err := gobacnet.NewClient(cfg.LocalInterface, int(cfg.LocalPort),
		gobacnet.WithMaxConcurrentTransactions(cfg.MaxConcurrentTransactions), gobacnet.WithLogLevel(logrus.InfoLevel))
	if err != nil {
		if systemCheck != nil {
			systemCheck.MarkFailed(fmt.Errorf("failed to create BACnet client: %w", err))
		}
		return err
	}
	d.client = client
	if address, err := client.LocalUDPAddress(); err != nil {
		if systemCheck != nil {
			systemCheck.MarkFailed(fmt.Errorf("failed to get local UDP address: %w", err))
		}
		return err
	} else {
		d.logger.Debug("bacnet client configured", zap.Stringer("local", address),
			zap.String("localInterface", cfg.LocalInterface), zap.Uint16("localPort", cfg.LocalPort))
	}
	if systemCheck != nil {
		systemCheck.MarkRunning()
	}
	return nil
}

func (d *Driver) configureDevice(ctx context.Context, rootAnnouncer node.Announcer, cfg config.Root, device config.Device, devices known.Context, deviceHealth *healthpb.FaultCheck, ctrlHealth *controllerHealth, logger *zap.Logger) error {
	deviceName := adapt.DeviceName(device)
	scDeviceName := cfg.DeviceNamePrefix + deviceName

	bacDevice, err := d.findDevice(ctx, device)
	if err != nil {
		if deviceHealth != nil {
			healthErr := &healthpb.HealthCheck_Error{
				SummaryText: "Cannot find device",
				DetailsText: fmt.Sprintf("net handshake: %v", ctxerr.Cause(ctx, err).Error()),
				Code:        statusToHealthCode(DeviceUnreachable),
			}
			deviceHealth.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
				State:     healthpb.HealthCheck_Reliability_UNRELIABLE,
				LastError: healthErr,
				Cause: &healthpb.HealthCheck_Reliability_Cause{
					Error:       healthErr,
					DisplayName: "deviceConfiguration",
				},
			})
		}
		if ctrlHealth != nil {
			ctrlHealth.setFailing(ctx, scDeviceName)
		}

		return fmt.Errorf("device comm handshake: %w", ctxerr.Cause(ctx, err))
	}

	// For WhoIs-discovered devices the controller IP is only known after findDevice succeeds.
	if ctrlHealth == nil {
		if udpAddr, addrErr := bacDevice.Addr.UDPAddr(); addrErr == nil {
			ip, ok := netip.AddrFromSlice(udpAddr.IP)
			if ok {
				ctrlHealth, _ = d.getOrCreateControllerCheck(ctx, cfg, ip.Unmap())
				if ctrlHealth != nil {
					ctrlHealth.register(scDeviceName)
				}
			}
		}
	}
	if ctrlHealth != nil {
		ctrlHealth.setOK(ctx, scDeviceName)
	}

	d.storeDevice(deviceName, bacDevice, device.DefaultWritePriority)

	if device.Metadata != nil {
		rootAnnouncer = node.AnnounceFeatures(rootAnnouncer, node.HasMetadata(device.Metadata))
	}

	// Wrap the per-request error callback to also update controller-level health.
	updateFn := func(rCtx context.Context, dh *healthpb.FaultCheck, name, request string, reqErr error) {
		updateRequestErrorStatus(rCtx, dh, name, request, reqErr)
		if ctrlHealth != nil {
			if reqErr != nil {
				ctrlHealth.setFailing(rCtx, scDeviceName)
			} else {
				ctrlHealth.setOK(rCtx, scDeviceName)
			}
		}
	}

	adapt.Device(scDeviceName, d.client, bacDevice, devices, deviceHealth, updateFn).AnnounceSelf(rootAnnouncer)

	// aka "[bacnet/devices/]{deviceName}/[obj/]"
	prefix := fmt.Sprintf("%s/%s", scDeviceName, cfg.ObjectNamePrefix)

	// Collect all the object that we will be announcing.
	// This will be a combination of configured objects and those we discover on the device.
	objects, err := d.fetchObjects(ctx, cfg, device, bacDevice)
	if err != nil {
		return fmt.Errorf("fetch objects: %w", ctxerr.Cause(ctx, err))
	}

	for _, object := range objects {
		co, bo := object.co, object.bo
		logger := logger.With(zap.Stringer("object", co))
		// Device types are handled separately
		if bo.ID.Type == objecttype.Device {
			// We're assuming that devices in the wild follow the spec
			// which says each network device has exactly one bacnet device.
			// We check for this explicitly to make sure our assumptions hold
			if bo.ID != bacDevice.ID {
				logger.Error("BACnet device with multiple advertised devices!")
			}
			continue
		}

		// no error, we added the device before we entered the loop so it should exist
		_ = d.storeObject(bacDevice, co, bo)

		impl, err := adapt.Object(prefix, d.client, bacDevice, co, deviceHealth, updateFn)
		if errors.Is(err, adapt.ErrNoDefault) {
			// logger.Debug("No default adaptation trait for object")
			continue
		}
		if errors.Is(err, adapt.ErrNoAdaptation) {
			logger.Error("No adaptation from object to trait", zap.Stringer("trait", co.Trait))
			continue
		}
		if err != nil {
			logger.Error("Error adapting object", zap.Error(err))
			continue
		}

		announcer := rootAnnouncer
		if object.co.Metadata != nil {
			announcer = node.AnnounceFeatures(announcer, node.HasMetadata(object.co.Metadata))
		}
		impl.AnnounceSelf(announcer)
	}

	return nil
}

func (d *Driver) storeObject(bacDevice bactypes.Device, co config.Object, bo *bactypes.Object) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.devices.StoreObject(bacDevice, adapt.ObjectName(co), *bo)
}

func (d *Driver) storeDevice(deviceName string, bacDevice bactypes.Device, defaultWritePriority uint) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devices.StoreDevice(deviceName, bacDevice, defaultWritePriority)
}

func (d *Driver) Clear() {
	// dispose system check before stopping devices and client
	d.dispose()

	d.mu.Lock()
	d.devices.Clear()
	d.mu.Unlock()
	if d.client != nil {
		// Important: without this, stopping the bacnet driver closes os.Stderr by default!
		if d.client.Log.Out == os.Stderr {
			d.client.Log.Out = io.Discard
		}
		d.client.Close()
		d.client = nil
	}
}

func (d *Driver) dispose() {
	for _, stop := range d.healthTasks {
		if stop != nil {
			stop()
		}
	}
	d.healthTasks = nil

	for _, check := range d.checks {
		if check != nil {
			check.Dispose()
		}
	}
	d.checks = nil

	d.controllerMu.Lock()
	for _, ch := range d.controllerHealths {
		ch.faultCheck.Dispose()
	}
	d.controllerHealths = nil
	d.controllerMu.Unlock()
}

// getOrCreateControllerCheck returns the controllerHealth for the given IP, creating it if needed.
// The returned controllerHealth is already stored in d.controllerHealths.
func (d *Driver) getOrCreateControllerCheck(ctx context.Context, cfg config.Root, ip netip.Addr) (*controllerHealth, error) {
	key := cfg.Name + "/" + ip.String()
	d.controllerMu.Lock()
	defer d.controllerMu.Unlock()
	if ch, ok := d.controllerHealths[key]; ok {
		return ch, nil
	}
	fc, err := d.health.NewFaultCheck(key, createControllerHealthCheck(cfg.SystemHealth.OccupantImpact.ToProto(), cfg.SystemHealth.EquipmentImpact.ToProto()))
	if err != nil {
		return nil, err
	}
	ch := newControllerHealth(fc, cfg.ControllerHealthThreshold)
	d.controllerHealths[key] = ch
	return ch, nil
}
