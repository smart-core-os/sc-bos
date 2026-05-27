package mock

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/auto"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/control"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bookingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/channelpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/countpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/extendretractpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hailpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/inputselectpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lockunlockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/microphonepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/motionsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/parentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/pressurepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/speakerpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/vendingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/wastepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
)

const DriverName = "mock"

var Factory driver.Factory = factory{}

type factory struct{}

func (factory) New(services driver.Services) service.Lifecycle {
	return NewDriver(services)
}

func (factory) ConfigBlocks() []block.Block {
	return config.Blocks
}

func NewDriver(services driver.Services) *Driver {
	d := &Driver{
		announcer:   services.Node,
		controller:  control.New(),
		systemCheck: services.SystemCheck,
		known:       make(map[deviceTrait]node.Undo),
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[config.Root](func() {
		if d.simCancel != nil {
			d.simCancel()
			d.simCancel = nil
		}
		d.Clean()
		if d.systemCheck != nil {
			d.systemCheck.Dispose()
		}
	}))
	d.logger = services.Logger.Named(DriverName)
	return d
}

type Driver struct {
	*service.Service[config.Root]

	logger      *zap.Logger
	announcer   node.Announcer
	controller  *control.Controller
	systemCheck service.SystemCheck
	simCancel   context.CancelFunc
	known       map[deviceTrait]node.Undo
}

type deviceTrait struct {
	name  string
	trait trait.Name
}

func (d *Driver) Clean() {
	for _, undo := range d.known {
		undo()
	}
	d.known = make(map[deviceTrait]node.Undo)
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	// Cancel any running health simulation before (re)configuring.
	if d.simCancel != nil {
		d.simCancel()
		d.simCancel = nil
	}

	toUndo := maps.Clone(d.known)
	for _, device := range cfg.Devices {
		var undos []node.Undo
		dt := deviceTrait{name: device.Name}

		// the device is still in the config, don't delete it
		delete(toUndo, dt)

		if u, ok := d.known[dt]; ok {
			undos = append(undos, u)
		}
		undos = append(undos, d.announcer.Announce(dt.name,
			node.HasMetadata(device.Metadata),
			node.HasServer(mockpb.RegisterMockDeviceApiServer, mockpb.MockDeviceApiServer(d.controller)),
		))

		for _, traitMd := range device.Traits {
			dt.trait = trait.Name(traitMd.Name)

			// the trait is still in the device config, don't delete it
			delete(toUndo, dt)

			var undo []node.Undo
			if u, ok := d.known[dt]; ok {
				undo = append(undo, u)
			}

			var features []node.Feature
			if _, ok := d.known[dt]; !ok {
				mockFeatures, slc, forceUndo := newMockClient(traitMd, device.Name, d.logger, d.controller)
				if len(mockFeatures) == 0 {
					d.logger.Sugar().Warnf("Cannot create mock client %s::%s", dt.name, dt.trait)
				} else {
					features = append(features, mockFeatures...)
					if forceUndo != nil {
						undo = append(undo, forceUndo)
					}

					// start any mock trait automations - e.g. updating occupancy sensors
					if slc != nil {
						_, err := slc.Start()
						if err != nil {
							d.logger.Sugar().Warnf("Unable to start mock trait automation %s::%s %v", dt.name, dt.trait, err)
						} else {
							lifecycleUndo := d.controller.RegisterLifecycle(device.Name, traitMd.Name, "default", slc)
							undo = append(undo, func() {
								lifecycleUndo()
								_, _ = slc.Stop()
							})
						}
					}
				}
			}
			features = append(features, node.HasTrait(dt.trait))
			undo = append(undo, d.announcer.Announce(dt.name, features...))
			d.known[dt] = node.UndoAll(undo...)
			undos = append(undos, undo...)
		}

		dt.trait = ""
		d.known[dt] = node.UndoAll(undos...)
	}

	for k, undo := range toUndo {
		undo()
		delete(d.known, k)
	}

	// Start driver-level health simulation if configured.
	if cfg.HealthCheck != nil && d.systemCheck != nil {
		p := cfg.HealthCheck.FaultProbability
		if p <= 0 {
			p = 0.15
		}
		if p > 1 {
			return fmt.Errorf("healthCheck.faultProbability must be between 0 and 1, got %g", p)
		}
		simCtx, cancel := context.WithCancel(ctx)
		d.simCancel = cancel
		d.systemCheck.MarkRunning()
		go runDriverHealthSimulation(simCtx, d.systemCheck, p)
	}

	return nil
}

func newMockClient(traitMd *metadatapb.TraitMetadata, deviceName string, logger *zap.Logger, ctrl *control.Controller) ([]node.Feature, service.Lifecycle, func()) {
	switch trait.Name(traitMd.Name) {
	case trait.AirQualitySensor:
		model := airqualitysensorpb.NewModel(airqualitysensorpb.WithInitialAirQuality(auto.GetAirQualityState()))
		return []node.Feature{node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model)))}, auto.AirQualitySensorAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *airqualitysensorpb.AirQuality { return new(airqualitysensorpb.AirQuality) },
				func(v *airqualitysensorpb.AirQuality, opts ...resource.WriteOption) error { _, err := model.UpdateAirQuality(v, opts...); return err }))
	case trait.AirTemperature:
		model := airtemperaturepb.NewModel()
		return []node.Feature{
				node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(model)))}, auto.AirTemperatureAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *airtemperaturepb.AirTemperature { return new(airtemperaturepb.AirTemperature) },
				func(v *airtemperaturepb.AirTemperature, opts ...resource.WriteOption) error { _, err := model.UpdateAirTemperature(v, opts...); return err }))
	case allocationpb.TraitName:
		model := allocationpb.NewModel()
		return []node.Feature{node.HasServer(allocationpb.RegisterAllocationApiServer, allocationpb.AllocationApiServer(allocationpb.NewModelServer(model)))}, auto.AllocationAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *allocationpb.Allocation { return new(allocationpb.Allocation) },
				func(v *allocationpb.Allocation, opts ...resource.WriteOption) error { model.UpdateAllocation(v, opts...); return nil }))
	case trait.Booking:
		return []node.Feature{node.HasServer(bookingpb.RegisterBookingApiServer, bookingpb.BookingApiServer(bookingpb.NewModelServer(bookingpb.NewModel())))}, nil, nil
	case trait.BrightnessSensor:
		model := brightnesssensorpb.NewModel()
		return []node.Feature{node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, brightnesssensorpb.BrightnessSensorApiServer(brightnesssensorpb.NewModelServer(model)))}, auto.BrightnessSensorAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *brightnesssensorpb.AmbientBrightness { return new(brightnesssensorpb.AmbientBrightness) },
				func(v *brightnesssensorpb.AmbientBrightness, opts ...resource.WriteOption) error {
					_, err := model.UpdateAmbientBrightness(v, opts...)
					return err
				}))
	case trait.Channel:
		model := channelpb.NewModel()
		return []node.Feature{node.HasServer(channelpb.RegisterChannelApiServer, channelpb.ChannelApiServer(channelpb.NewModelServer(model)))}, auto.ChannelAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *channelpb.Channel { return new(channelpb.Channel) },
				func(v *channelpb.Channel, opts ...resource.WriteOption) error { _, err := model.UpdateChosenChannel(v, opts...); return err }))
	case trait.Count:
		model := countpb.NewModel()
		return []node.Feature{node.HasServer(countpb.RegisterCountApiServer, countpb.CountApiServer(countpb.NewModelServer(model)))}, auto.CountAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *countpb.Count { return new(countpb.Count) },
				func(v *countpb.Count, opts ...resource.WriteOption) error { _, err := model.UpdateCount(v, opts...); return err }))
	case trait.Electric:
		model := electricpb.NewModel()
		return []node.Feature{node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(electricpb.NewModelServer(model)))}, auto.Electric(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *electricpb.ElectricDemand { return new(electricpb.ElectricDemand) },
				func(v *electricpb.ElectricDemand, opts ...resource.WriteOption) error { _, err := model.UpdateDemand(v, opts...); return err }))
	case trait.Emergency:
		model := emergencypb.NewModel()
		return []node.Feature{node.HasServer(emergencypb.RegisterEmergencyApiServer, emergencypb.EmergencyApiServer(emergencypb.NewModelServer(model)))}, nil, nil
	case trait.EnergyStorage:
		model := energystoragepb.NewModel()
		kind := auto.EnergyStorageDeviceTypeBattery
		if k, ok := traitMd.GetMore()["type"]; ok {
			switch auto.EnergyStorageDeviceType(k) {
			case auto.EnergyStorageDeviceTypeBattery:
				kind = auto.EnergyStorageDeviceTypeBattery
			case auto.EnergyStorageDeviceTypeEV:
				kind = auto.EnergyStorageDeviceTypeEV
			case auto.EnergyStorageDeviceTypeDrone:
				kind = auto.EnergyStorageDeviceTypeDrone
			default:
				logger.Sugar().Warnf("Unknown energy storage device type '%s' for %s, defaulting to battery", k, deviceName)
			}
		}
		return []node.Feature{node.HasServer(energystoragepb.RegisterEnergyStorageApiServer, energystoragepb.EnergyStorageApiServer(energystoragepb.NewModelServer(model)))}, auto.EnergyStorage(model, kind),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *energystoragepb.EnergyLevel { return new(energystoragepb.EnergyLevel) },
				func(v *energystoragepb.EnergyLevel, opts ...resource.WriteOption) error { _, err := model.UpdateEnergyLevel(v, opts...); return err }))
	case trait.EnterLeaveSensor:
		model := enterleavesensorpb.NewModel()
		return []node.Feature{node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(model)))}, auto.EnterLeaveAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *enterleavesensorpb.EnterLeaveEvent { return new(enterleavesensorpb.EnterLeaveEvent) },
				func(v *enterleavesensorpb.EnterLeaveEvent, opts ...resource.WriteOption) error { return model.CreateEnterLeaveEvent(v, opts...) }))
	case trait.ExtendRetract:
		model := extendretractpb.NewModel()
		return []node.Feature{node.HasServer(extendretractpb.RegisterExtendRetractApiServer, extendretractpb.ExtendRetractApiServer(extendretractpb.NewModelServer(model)))}, auto.ExtendRetractAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *extendretractpb.Extension { return new(extendretractpb.Extension) },
				func(v *extendretractpb.Extension, opts ...resource.WriteOption) error { _, err := model.UpdateExtension(v, opts...); return err }))
	case trait.FanSpeed:
		presets := []fanspeedpb.Preset{
			{Name: "off", Percentage: 0},
			{Name: "low", Percentage: 15},
			{Name: "med", Percentage: 40},
			{Name: "high", Percentage: 75},
			{Name: "full", Percentage: 100},
		}
		model := fanspeedpb.NewModel(fanspeedpb.WithPresets(presets...))
		return []node.Feature{node.HasServer(fanspeedpb.RegisterFanSpeedApiServer, fanspeedpb.FanSpeedApiServer(fanspeedpb.NewModelServer(model)))}, auto.FanSpeed(model, presets...),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *fanspeedpb.FanSpeed { return new(fanspeedpb.FanSpeed) },
				func(v *fanspeedpb.FanSpeed, opts ...resource.WriteOption) error { _, err := model.UpdateFanSpeed(v, opts...); return err }))
	case trait.Hail:
		return []node.Feature{node.HasServer(hailpb.RegisterHailApiServer, hailpb.HailApiServer(hailpb.NewModelServer(hailpb.NewModel())))}, nil, nil
	case healthpb.TraitName:
		model := healthpb.NewModel()
		return []node.Feature{node.HasServer(healthpb.RegisterHealthApiServer, healthpb.HealthApiServer(healthpb.NewModelServer(model)))}, auto.HealthAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *healthpb.HealthCheck { return new(healthpb.HealthCheck) },
				func(v *healthpb.HealthCheck, opts ...resource.WriteOption) error { _, err := model.UpdateHealthCheck(v, opts...); return err }))
	case trait.InputSelect:
		model := inputselectpb.NewModel()
		return []node.Feature{node.HasServer(inputselectpb.RegisterInputSelectApiServer, inputselectpb.InputSelectApiServer(inputselectpb.NewModelServer(model)))}, auto.InputSelectAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *inputselectpb.Input { return new(inputselectpb.Input) },
				func(v *inputselectpb.Input, opts ...resource.WriteOption) error { _, err := model.UpdateInput(v, opts...); return err }))
	case trait.Light:
		model := lightpb.NewModel(
			lightpb.WithPreset(0, &lightpb.LightPreset{Name: "off", Title: "Off"}),
			lightpb.WithPreset(40, &lightpb.LightPreset{Name: "low", Title: "Low"}),
			lightpb.WithPreset(60, &lightpb.LightPreset{Name: "med", Title: "Normal"}),
			lightpb.WithPreset(80, &lightpb.LightPreset{Name: "high", Title: "High"}),
			lightpb.WithPreset(100, &lightpb.LightPreset{Name: "full", Title: "Full"}),
		)
		server := lightpb.NewModelServer(model)
		return []node.Feature{
				node.HasServer(lightpb.RegisterLightApiServer, lightpb.LightApiServer(server)),
				node.HasServer(lightpb.RegisterLightInfoServer, lightpb.LightInfoServer(server)),
			}, auto.LightAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *lightpb.Brightness { return new(lightpb.Brightness) },
				func(v *lightpb.Brightness, opts ...resource.WriteOption) error { _, err := model.UpdateBrightness(v, opts...); return err }))
	case trait.LockUnlock:
		model := lockunlockpb.NewModel()
		return []node.Feature{node.HasServer(lockunlockpb.RegisterLockUnlockApiServer, lockunlockpb.LockUnlockApiServer(lockunlockpb.NewModelServer(model)))}, auto.LockUnlockAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *lockunlockpb.LockUnlock { return new(lockunlockpb.LockUnlock) },
				func(v *lockunlockpb.LockUnlock, opts ...resource.WriteOption) error { _, err := model.UpdateLockUnlock(v, opts...); return err }))
	case trait.Metadata:
		return []node.Feature{node.HasServer(metadatapb.RegisterMetadataApiServer, metadatapb.MetadataApiServer(metadatapb.NewModelServer(metadatapb.NewModel())))}, nil, nil
	case trait.Microphone:
		model := microphonepb.NewModel()
		return []node.Feature{node.HasServer(microphonepb.RegisterMicrophoneApiServer, microphonepb.MicrophoneApiServer(microphonepb.NewModelServer(model)))}, auto.MicrophoneAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *typespb.AudioLevel { return new(typespb.AudioLevel) },
				func(v *typespb.AudioLevel, opts ...resource.WriteOption) error { _, err := model.UpdateGain(v, opts...); return err }))
	case trait.Mode:
		// Mode has config-driven presets; there is no single resource type to unmarshal into.
		features, slc := mockMode(traitMd, deviceName, logger)
		return features, slc, nil
	case trait.MotionSensor:
		model := motionsensorpb.NewModel()
		return []node.Feature{node.HasServer(motionsensorpb.RegisterMotionSensorApiServer, motionsensorpb.MotionSensorApiServer(motionsensorpb.NewModelServer(model)))}, auto.MotionSensorAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *motionsensorpb.MotionDetection { return new(motionsensorpb.MotionDetection) },
				func(v *motionsensorpb.MotionDetection, opts ...resource.WriteOption) error { _, err := model.SetMotionDetection(v, opts...); return err }))
	case trait.OccupancySensor:
		model := occupancysensorpb.NewModel()
		return []node.Feature{node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model)))}, auto.OccupancySensorAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *occupancysensorpb.Occupancy { return new(occupancysensorpb.Occupancy) },
				func(v *occupancysensorpb.Occupancy, opts ...resource.WriteOption) error { _, err := model.SetOccupancy(v, opts...); return err }))
	case trait.OnOff:
		model := onoffpb.NewModel(resource.WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_OFF}))
		return []node.Feature{node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(model)))}, auto.OnOffAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *onoffpb.OnOff { return new(onoffpb.OnOff) },
				func(v *onoffpb.OnOff, opts ...resource.WriteOption) error { _, err := model.UpdateOnOff(v, opts...); return err }))
	case trait.OpenClose:
		// OpenClose positions are config-driven; there is no single resource type to unmarshal into.
		features, slc := mockOpenClose(traitMd, deviceName, logger)
		return features, slc, nil
	case trait.Parent:
		return []node.Feature{node.HasServer(parentpb.RegisterParentApiServer, parentpb.ParentApiServer(parentpb.NewModelServer(parentpb.NewModel())))}, nil, nil
	case trait.Publication:
		return []node.Feature{node.HasServer(publicationpb.RegisterPublicationApiServer, publicationpb.PublicationApiServer(publicationpb.NewModelServer(publicationpb.NewModel())))}, nil, nil
	case trait.Ptz:
		model := ptzpb.NewModel()
		return []node.Feature{node.HasServer(ptzpb.RegisterPtzApiServer, ptzpb.PtzApiServer(ptzpb.NewModelServer(model)))}, auto.PtzAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *ptzpb.Ptz { return new(ptzpb.Ptz) },
				func(v *ptzpb.Ptz, opts ...resource.WriteOption) error { _, err := model.UpdatePtz(v, opts...); return err }))
	case trait.Speaker:
		model := speakerpb.NewModel()
		return []node.Feature{node.HasServer(speakerpb.RegisterSpeakerApiServer, speakerpb.SpeakerApiServer(speakerpb.NewModelServer(model)))}, auto.SpeakerAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *typespb.AudioLevel { return new(typespb.AudioLevel) },
				func(v *typespb.AudioLevel, opts ...resource.WriteOption) error { _, err := model.UpdateVolume(v, opts...); return err }))
	case trait.Vending:
		return []node.Feature{node.HasServer(vendingpb.RegisterVendingApiServer, vendingpb.VendingApiServer(vendingpb.NewModelServer(vendingpb.NewModel())))}, nil, nil

	case accesspb.TraitName:
		model := accesspb.NewModel()
		return []node.Feature{node.HasServer(accesspb.RegisterAccessApiServer, accesspb.AccessApiServer(accesspb.NewModelServer(model)))}, auto.Access(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *accesspb.AccessAttempt { return new(accesspb.AccessAttempt) },
				func(v *accesspb.AccessAttempt, opts ...resource.WriteOption) error { _, err := model.UpdateLastAccessAttempt(v, opts...); return err }))
	case buttonpb.TraitName:
		model := buttonpb.NewModel(buttonpb.ButtonState_UNPRESSED)
		return []node.Feature{node.HasServer(buttonpb.RegisterButtonApiServer, buttonpb.ButtonApiServer(buttonpb.NewModelServer(model)))}, auto.ButtonAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *buttonpb.ButtonState { return new(buttonpb.ButtonState) },
				func(v *buttonpb.ButtonState, opts ...resource.WriteOption) error { _, err := model.UpdateButtonState(v, opts...); return err }))
	case emergencylightpb.TraitName:
		model := emergencylightpb.NewModel()
		model.SetLastDurationTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		model.SetLastFunctionalTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		return []node.Feature{node.HasServer(emergencylightpb.RegisterEmergencyLightApiServer, emergencylightpb.EmergencyLightApiServer(emergencylightpb.NewModelServer(model)))}, nil, nil
	case fluidflowpb.TraitName:
		model := fluidflowpb.NewModel()
		return []node.Feature{node.HasServer(fluidflowpb.RegisterFluidFlowApiServer, fluidflowpb.FluidFlowApiServer(fluidflowpb.NewModelServer(model)))}, auto.FluidFlow(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *fluidflowpb.FluidFlow { return new(fluidflowpb.FluidFlow) },
				func(v *fluidflowpb.FluidFlow, opts ...resource.WriteOption) error { _, err := model.UpdateFluidFlow(v, opts...); return err }))
	case meterpb.TraitName:
		var (
			unit string
			ok   bool
		)
		if unit, ok = traitMd.GetMore()["unit"]; !ok {
			unit = "kWh"
		}
		model := meterpb.NewModel()
		info := &meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
			ResourceSupport: &typespb.ResourceSupport{
				Readable:   true,
				Writable:   true,
				Observable: true,
			},
			UsageUnit: unit,
		}}
		return []node.Feature{
				node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(model))),
				node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(info)),
			}, auto.MeterAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *meterpb.MeterReading { return new(meterpb.MeterReading) },
				func(v *meterpb.MeterReading, opts ...resource.WriteOption) error { _, err := model.UpdateMeterReading(v, opts...); return err }))
	case pressurepb.TraitName:
		model := pressurepb.NewModel()
		return []node.Feature{node.HasServer(pressurepb.RegisterPressureApiServer, pressurepb.PressureApiServer(pressurepb.NewModelServer(model)))}, auto.Pressure(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *pressurepb.Pressure { return new(pressurepb.Pressure) },
				func(v *pressurepb.Pressure, opts ...resource.WriteOption) error { _, err := model.UpdatePressure(v, opts...); return err }))
	case securityeventpb.TraitName:
		model := securityeventpb.NewModel()
		return []node.Feature{node.HasServer(securityeventpb.RegisterSecurityEventApiServer, securityeventpb.SecurityEventApiServer(securityeventpb.NewModelServer(model)))}, auto.SecurityEventAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *securityeventpb.SecurityEvent { return new(securityeventpb.SecurityEvent) },
				func(v *securityeventpb.SecurityEvent, opts ...resource.WriteOption) error { _, err := model.AddSecurityEvent(v, opts...); return err }))
	case soundsensorpb.TraitName:
		model := soundsensorpb.NewModel()
		return []node.Feature{node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(soundsensorpb.NewModelServer(model)))}, auto.SoundSensorAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *soundsensorpb.SoundLevel { return new(soundsensorpb.SoundLevel) },
				func(v *soundsensorpb.SoundLevel, opts ...resource.WriteOption) error { _, err := model.UpdateSoundLevel(v, opts...); return err }))
	case statuspb.TraitName:
		model := statuspb.NewModel()
		// set an initial value or Pull methods can hang
		_, _ = model.UpdateProblem(&statuspb.StatusLog_Problem{Name: deviceName, Level: statuspb.StatusLog_NOMINAL})
		return []node.Feature{node.HasServer(statuspb.RegisterStatusApiServer, statuspb.StatusApiServer(statuspb.NewModelServer(model)))}, auto.Status(model, deviceName), nil
	case temperaturepb.TraitName:
		model := temperaturepb.NewModel()
		return []node.Feature{node.HasServer(temperaturepb.RegisterTemperatureApiServer, temperaturepb.TemperatureApiServer(temperaturepb.NewModelServer(model)))}, auto.TemperatureAuto(model),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *temperaturepb.Temperature { return new(temperaturepb.Temperature) },
				func(v *temperaturepb.Temperature, opts ...resource.WriteOption) error { _, err := model.UpdateTemperature(v, opts...); return err }))
	case transportpb.TraitName:
		model := transportpb.NewModel()
		maxFloor := 10
		if m, ok := traitMd.GetMore()["numFloors"]; ok {
			mi, err := strconv.Atoi(m)
			maxFloor = mi
			if err != nil {
				logger.Error("failed to parse maxFloor", zap.Error(err))
				return nil, nil, nil
			}
		}
		return []node.Feature{node.HasServer(transportpb.RegisterTransportApiServer, transportpb.TransportApiServer(transportpb.NewModelServer(model)))}, auto.TransportAuto(model, maxFloor),
			ctrl.Register(deviceName, traitMd.Name, forceJSON(func() *transportpb.Transport { return new(transportpb.Transport) },
				func(v *transportpb.Transport, opts ...resource.WriteOption) error { _, err := model.UpdateTransport(v, opts...); return err }))
	case udmipb.TraitName:
		return []node.Feature{node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(auto.NewUdmiServer(logger, deviceName)))}, nil, nil
	case wastepb.TraitName:
		model := wastepb.NewModel()
		return []node.Feature{node.HasServer(wastepb.RegisterWasteApiServer, wastepb.WasteApiServer(wastepb.NewModelServer(model)))}, auto.WasteRecordsAuto(model), nil
	}

	return nil, nil, nil
}

// forceJSON creates a ForceFunc that parses valueJSON into a new proto message and passes it to update.
// Only fields present in the JSON are written; absent fields are left unchanged (patch semantics).
func forceJSON[T proto.Message](newMsg func() T, update func(T, ...resource.WriteOption) error) control.ForceFunc {
	return func(valueJSON string) error {
		msg := newMsg()
		if err := protojson.Unmarshal([]byte(valueJSON), msg); err != nil {
			return err
		}
		return update(msg, resource.WithUpdateMask(setFieldMask(msg)))
	}
}

// setFieldMask returns a FieldMask covering every top-level field that is set in m.
func setFieldMask(m proto.Message) *fieldmaskpb.FieldMask {
	var paths []string
	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		paths = append(paths, string(fd.Name()))
		return true
	})
	return &fieldmaskpb.FieldMask{Paths: paths}
}

// runDriverHealthSimulation periodically randomises the driver's system-level health check state.
// With probability p it transitions to a fault state; otherwise healthy.
func runDriverHealthSimulation(ctx context.Context, check service.SystemCheck, p float64) {
	timer := time.NewTimer(randDuration(30*time.Second, 90*time.Second))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(randDuration(30*time.Second, 90*time.Second))
		if rand.Float64() < p {
			check.MarkFailed(fmt.Errorf("simulated connectivity failure"))
		} else {
			check.MarkRunning()
		}
	}
}

func randDuration(min, max time.Duration) time.Duration {
	return time.Duration(rand.Intn(int(max-min))) + min
}
