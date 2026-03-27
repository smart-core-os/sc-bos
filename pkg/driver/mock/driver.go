package mock

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/auto"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bookingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	fanspeedpb2 "github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hailpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/parentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/pressurepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
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

func (_ factory) New(services driver.Services) service.Lifecycle {
	return NewDriver(services)
}

func (_ factory) ConfigBlocks() []block.Block {
	return config.Blocks
}

func NewDriver(services driver.Services) *Driver {
	d := &Driver{
		announcer: services.Node,
		known:     make(map[deviceTrait]node.Undo),
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[config.Root](d.Clean))
	d.logger = services.Logger.Named(DriverName)
	return d
}

type Driver struct {
	*service.Service[config.Root]

	logger    *zap.Logger
	announcer node.Announcer
	known     map[deviceTrait]node.Undo
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

func (d *Driver) applyConfig(_ context.Context, cfg config.Root) error {
	toUndo := maps.Clone(d.known)
	for _, device := range cfg.Devices {
		var undos []node.Undo
		dt := deviceTrait{name: device.Name}

		// the device is still in the config, don't delete it
		delete(toUndo, dt)

		if u, ok := d.known[dt]; ok {
			undos = append(undos, u)
		}
		undos = append(undos, d.announcer.Announce(dt.name, node.HasMetadata(device.Metadata)))

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
				mockFeatures, slc := newMockClient(traitMd, device.Name, d.logger)
				if len(mockFeatures) == 0 {
					d.logger.Sugar().Warnf("Cannot create mock client %s::%s", dt.name, dt.trait)
				} else {
					features = append(features, mockFeatures...)

					// start any mock trait automations - e.g. updating occupancy sensors
					if slc != nil {
						_, err := slc.Start()
						if err != nil {
							d.logger.Sugar().Warnf("Unable to start mock trait automation %s::%s %v", dt.name, dt.trait, err)
						} else {
							undo = append(undo, func() {
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

	return nil
}

func newMockClient(traitMd *metadatapb.TraitMetadata, deviceName string, logger *zap.Logger) ([]node.Feature, service.Lifecycle) {
	switch trait.Name(traitMd.Name) {
	case trait.AirQualitySensor:
		model := airqualitysensorpb.NewModel(airqualitysensorpb.WithInitialAirQuality(auto.GetAirQualityState()))
		return []node.Feature{node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model)))}, auto.AirQualitySensorAuto(model)
	case trait.AirTemperature:
		model := airtemperaturepb.NewModel()
		return []node.Feature{node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(model)))}, auto.AirTemperatureAuto(model)
	case allocationpb.TraitName:
		model := allocationpb.NewModel()
		return []node.Feature{node.HasServer(allocationpb.RegisterAllocationApiServer, allocationpb.AllocationApiServer(allocationpb.NewModelServer(model)))}, auto.AllocationAuto(model)
	case trait.Booking:
		return []node.Feature{node.HasServer(bookingpb.RegisterBookingApiServer, bookingpb.BookingApiServer(bookingpb.NewModelServer(bookingpb.NewModel())))}, nil
	case trait.BrightnessSensor:
		// todo: return []node.Feature{node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, ...)}, nil
		return nil, nil
	case trait.Channel:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Count:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Electric:
		model := electricpb.NewModel()
		return []node.Feature{node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(electricpb.NewModelServer(model)))}, auto.Electric(model)
	case trait.Emergency:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
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
		return []node.Feature{node.HasServer(energystoragepb.RegisterEnergyStorageApiServer, energystoragepb.EnergyStorageApiServer(energystoragepb.NewModelServer(model)))}, auto.EnergyStorage(model, kind)
	case trait.EnterLeaveSensor:
		model := enterleavesensorpb.NewModel()
		return []node.Feature{node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(model)))}, auto.EnterLeaveAuto(model)
	case trait.ExtendRetract:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.FanSpeed:
		presets := []fanspeedpb2.Preset{
			{Name: "off", Percentage: 0},
			{Name: "low", Percentage: 15},
			{Name: "med", Percentage: 40},
			{Name: "high", Percentage: 75},
			{Name: "full", Percentage: 100},
		}
		model := fanspeedpb2.NewModel(fanspeedpb2.WithPresets(presets...))
		return []node.Feature{node.HasServer(fanspeedpb2.RegisterFanSpeedApiServer, fanspeedpb2.FanSpeedApiServer(fanspeedpb2.NewModelServer(model)))}, auto.FanSpeed(model, presets...)
	case trait.Hail:
		return []node.Feature{node.HasServer(hailpb.RegisterHailApiServer, hailpb.HailApiServer(hailpb.NewModelServer(hailpb.NewModel())))}, nil
	case trait.InputSelect:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Light:
		server := lightpb.NewModelServer(lightpb.NewModel(
			lightpb.WithPreset(0, &lightpb.LightPreset{Name: "off", Title: "Off"}),
			lightpb.WithPreset(40, &lightpb.LightPreset{Name: "low", Title: "Low"}),
			lightpb.WithPreset(60, &lightpb.LightPreset{Name: "med", Title: "Normal"}),
			lightpb.WithPreset(80, &lightpb.LightPreset{Name: "high", Title: "High"}),
			lightpb.WithPreset(100, &lightpb.LightPreset{Name: "full", Title: "Full"}),
		))
		return []node.Feature{
			node.HasServer(lightpb.RegisterLightApiServer, lightpb.LightApiServer(server)),
			node.HasServer(lightpb.RegisterLightInfoServer, lightpb.LightInfoServer(server)),
		}, nil
	case trait.LockUnlock:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Metadata:
		return []node.Feature{node.HasServer(metadatapb.RegisterMetadataApiServer, metadatapb.MetadataApiServer(metadatapb.NewModelServer(metadatapb.NewModel())))}, nil
	case trait.Microphone:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Mode:
		return mockMode(traitMd, deviceName, logger)
	case trait.MotionSensor:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.OccupancySensor:
		model := occupancysensorpb.NewModel()
		return []node.Feature{node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model)))}, auto.OccupancySensorAuto(model)
	case trait.OnOff:
		return []node.Feature{node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(onoffpb.NewModel(resource.WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_OFF})))))}, nil
	case trait.OpenClose:
		return mockOpenClose(traitMd, deviceName, logger)
	case trait.Parent:
		return []node.Feature{node.HasServer(parentpb.RegisterParentApiServer, parentpb.ParentApiServer(parentpb.NewModelServer(parentpb.NewModel())))}, nil
	case trait.Publication:
		return []node.Feature{node.HasServer(publicationpb.RegisterPublicationApiServer, publicationpb.PublicationApiServer(publicationpb.NewModelServer(publicationpb.NewModel())))}, nil
	case trait.Ptz:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Speaker:
		// todo: return []node.Feature{node.HasServer(...)}, nil
		return nil, nil
	case trait.Vending:
		return []node.Feature{node.HasServer(vendingpb.RegisterVendingApiServer, vendingpb.VendingApiServer(vendingpb.NewModelServer(vendingpb.NewModel())))}, nil

	case accesspb.TraitName:
		model := accesspb.NewModel()
		return []node.Feature{node.HasServer(accesspb.RegisterAccessApiServer, accesspb.AccessApiServer(accesspb.NewModelServer(model)))}, auto.Access(model)
	case buttonpb.TraitName:
		return []node.Feature{node.HasServer(buttonpb.RegisterButtonApiServer, buttonpb.ButtonApiServer(buttonpb.NewModelServer(buttonpb.NewModel(buttonpb.ButtonState_UNPRESSED))))}, nil
	case emergencylightpb.TraitName:
		model := emergencylightpb.NewModel()
		model.SetLastDurationTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		model.SetLastFunctionalTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		return []node.Feature{node.HasServer(emergencylightpb.RegisterEmergencyLightApiServer, emergencylightpb.EmergencyLightApiServer(emergencylightpb.NewModelServer(model)))}, nil
	case fluidflowpb.TraitName:
		model := fluidflowpb.NewModel()
		return []node.Feature{node.HasServer(fluidflowpb.RegisterFluidFlowApiServer, fluidflowpb.FluidFlowApiServer(fluidflowpb.NewModelServer(model)))}, auto.FluidFlow(model)
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
		}, auto.MeterAuto(model)
	case pressurepb.TraitName:
		model := pressurepb.NewModel()
		return []node.Feature{node.HasServer(pressurepb.RegisterPressureApiServer, pressurepb.PressureApiServer(pressurepb.NewModelServer(model)))}, auto.Pressure(model)
	case securityeventpb.TraitName:
		model := securityeventpb.NewModel()
		return []node.Feature{node.HasServer(securityeventpb.RegisterSecurityEventApiServer, securityeventpb.SecurityEventApiServer(securityeventpb.NewModelServer(model)))}, auto.SecurityEventAuto(model)
	case soundsensorpb.TraitName:
		model := soundsensorpb.NewModel()
		return []node.Feature{node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(soundsensorpb.NewModelServer(model)))}, auto.SoundSensorAuto(model)
	case statuspb.TraitName:
		model := statuspb.NewModel()
		// set an initial value or Pull methods can hang
		_, _ = model.UpdateProblem(&statuspb.StatusLog_Problem{Name: deviceName, Level: statuspb.StatusLog_NOMINAL})
		return []node.Feature{node.HasServer(statuspb.RegisterStatusApiServer, statuspb.StatusApiServer(statuspb.NewModelServer(model)))}, auto.Status(model, deviceName)
	case temperaturepb.TraitName:
		model := temperaturepb.NewModel()
		return []node.Feature{node.HasServer(temperaturepb.RegisterTemperatureApiServer, temperaturepb.TemperatureApiServer(temperaturepb.NewModelServer(model)))}, auto.TemperatureAuto(model)
	case transportpb.TraitName:
		model := transportpb.NewModel()
		maxFloor := 10
		if m, ok := traitMd.GetMore()["numFloors"]; ok {
			mi, err := strconv.Atoi(m)
			maxFloor = mi
			if err != nil {
				logger.Error("failed to parse maxFloor", zap.Error(err))
				return nil, nil
			}
		}
		return []node.Feature{node.HasServer(transportpb.RegisterTransportApiServer, transportpb.TransportApiServer(transportpb.NewModelServer(model)))}, auto.TransportAuto(model, maxFloor)
	case udmipb.TraitName:
		return []node.Feature{node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(auto.NewUdmiServer(logger, deviceName)))}, nil
	case wastepb.TraitName:
		model := wastepb.NewModel()
		return []node.Feature{node.HasServer(wastepb.RegisterWasteApiServer, wastepb.WasteApiServer(wastepb.NewModelServer(model)))}, auto.WasteRecordsAuto(model)
	}

	return nil, nil
}
