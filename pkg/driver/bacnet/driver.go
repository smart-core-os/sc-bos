package bacnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
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
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/status"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const DriverName = "bacnet"

var Factory driver.Factory = factory{}

type factory struct{}

func (_ factory) New(services driver.Services) service.Lifecycle {
	return NewDriver(services)
}

func (_ factory) ConfigBlocks() []block.Block {
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
}

func NewDriver(services driver.Services) *Driver {
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		devices:   known.NewMap(),
		logger:    services.Logger.Named("bacnet"),
	}
	d.Service = service.New(service.MonoApply(d.applyConfig),
		service.WithParser(config.ReadBytes),
		service.WithOnStop[config.Root](d.Clear))
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

	errs := d.initClient(cfg)
	if errs != nil {
		return errs
	}

	devices := known.SyncContext(d.mu.RLocker(), d.devices)
	statuses := statuspb.NewMap(rootAnnouncer)

	// setup all our devices and objects...
	for _, device := range cfg.Devices {
		// make sure to retry setting up devices in case they aren't yet online but might be in the future
		device := device
		deviceName := adapt.DeviceName(device)
		logger := d.logger.With(zap.String("device", deviceName), zap.Uint32("deviceId", uint32(device.ID)),
			zap.Stringer("address", device.Comm.IP))

		// allow status reporting for this device
		scDeviceName := cfg.DeviceNamePrefix + deviceName

		// even if devices are offline, they still have metadata
		if device.Metadata != nil {
			rootAnnouncer.Announce(device.Name, node.HasMetadata(device.Metadata))
		}

		statuses.UpdateProblem(scDeviceName, &gen_statuspb.StatusLog_Problem{
			Name:        scDeviceName + ":setup",
			Level:       gen_statuspb.StatusLog_NON_FUNCTIONAL,
			Description: "device has not yet been configured",
		})

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
				err := d.configureDevice(ctx, announcer, cfg, device, devices, statuses, logger)

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
		impl, err := merge.IntoTrait(d.client, devices, statuses, trait, logger)
		if errors.Is(err, merge.ErrTraitNotSupported) {
			logger.Error("Cannot combine into trait, not supported")
			continue
		}
		if err != nil {
			logger.Error("Cannot combine into trait", zap.Error(err))
			continue
		}
		announcer := rootAnnouncer
		if trait.Metadata != nil {
			announcer = node.AnnounceFeatures(announcer, node.HasMetadata(trait.Metadata))
		}
		impl.AnnounceSelf(announcer)
	}

	return errs
}

func (d *Driver) initClient(cfg config.Root) error {
	client, err := gobacnet.NewClient(cfg.LocalInterface, int(cfg.LocalPort),
		gobacnet.WithMaxConcurrentTransactions(cfg.MaxConcurrentTransactions), gobacnet.WithLogLevel(logrus.InfoLevel))
	if err != nil {
		return err
	}
	d.client = client
	if address, err := client.LocalUDPAddress(); err == nil {
		d.logger.Debug("bacnet client configured", zap.Stringer("local", address),
			zap.String("localInterface", cfg.LocalInterface), zap.Uint16("localPort", cfg.LocalPort))
	}
	return err
}

func (d *Driver) configureDevice(ctx context.Context, rootAnnouncer node.Announcer, cfg config.Root, device config.Device, devices known.Context, statuses *statuspb.Map, logger *zap.Logger) error {
	deviceName := adapt.DeviceName(device)
	scDeviceName := cfg.DeviceNamePrefix + deviceName

	bacDevice, err := d.findDevice(ctx, device)
	if err != nil {
		_, msg := status.SummariseRequestErrors("net handshake:",
			[]string{"MaxApduLengthAccepted", "SegmentationSupported", "VendorIdentifier"},
			multierr.Errors(err))
		statuses.UpdateProblem(scDeviceName, &gen_statuspb.StatusLog_Problem{
			Name:        scDeviceName + ":setup",
			Level:       gen_statuspb.StatusLog_OFFLINE,
			Description: msg,
		})
		return fmt.Errorf("device comm handshake: %w", ctxerr.Cause(ctx, err))
	}

	d.storeDevice(deviceName, bacDevice, device.DefaultWritePriority)

	if device.Metadata != nil {
		rootAnnouncer = node.AnnounceFeatures(rootAnnouncer, node.HasMetadata(device.Metadata))
	}

	// assume we're online unless we get an error
	statuses.UpdateProblem(scDeviceName, &gen_statuspb.StatusLog_Problem{
		Name:        scDeviceName + ":setup",
		Level:       gen_statuspb.StatusLog_NOMINAL,
		Description: "handshake successful",
	})
	adapt.Device(scDeviceName, d.client, bacDevice, devices, statuses).AnnounceSelf(rootAnnouncer)

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

		impl, err := adapt.Object(prefix, d.client, bacDevice, co, statuses)
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
