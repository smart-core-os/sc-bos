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
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bookingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	fanspeedpb2 "github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	hailpb2 "github.com/smart-core-os/sc-bos/pkg/proto/hailpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	lightpb2 "github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	metadatapb2 "github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/parentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/pressurepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/vendingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/wastepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	airqualitysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/airqualitysensorpb"
	airtemperaturepb2 "github.com/smart-core-os/sc-bos/pkg/trait/airtemperaturepb"
	electricpb2 "github.com/smart-core-os/sc-bos/pkg/trait/electricpb"
	enterleavesensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/enterleavesensorpb"
	occupancysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
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

			var traitOpts []node.TraitOption
			var undo []node.Undo
			if u, ok := d.known[dt]; ok {
				undo = append(undo, u)
			}

			if _, ok := d.known[dt]; !ok {
				clients, slc := newMockClient(traitMd, device.Name, d.logger)
				if len(clients) == 0 {
					d.logger.Sugar().Warnf("Cannot create mock client %s::%s", dt.name, dt.trait)
				} else {
					traitOpts = append(traitOpts, node.WithClients(clients...))

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
			undo = append(undo, d.announcer.Announce(dt.name, node.HasTrait(dt.trait, traitOpts...)))
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

func newMockClient(traitMd *metadatapb.TraitMetadata, deviceName string, logger *zap.Logger) ([]wrap.ServiceUnwrapper, service.Lifecycle) {
	switch trait.Name(traitMd.Name) {
	case trait.AirQualitySensor:
		model := airqualitysensorpb2.NewModel(airqualitysensorpb2.WithInitialAirQuality(auto.GetAirQualityState()))
		return []wrap.ServiceUnwrapper{airqualitysensorpb2.WrapApi(airqualitysensorpb2.NewModelServer(model))}, auto.AirQualitySensorAuto(model)
	case trait.AirTemperature:
		model := airtemperaturepb2.NewModel()
		return []wrap.ServiceUnwrapper{airtemperaturepb2.WrapApi(airtemperaturepb2.NewModelServer(model))}, auto.AirTemperatureAuto(model)
	case allocationpb.TraitName:
		model := allocationpb.NewModel()
		return []wrap.ServiceUnwrapper{allocationpb.WrapApi(allocationpb.NewModelServer(model))}, auto.AllocationAuto(model)
	case trait.Booking:
		return []wrap.ServiceUnwrapper{bookingpb.WrapApi(bookingpb.NewModelServer(bookingpb.NewModel()))}, nil
	case trait.BrightnessSensor:
		// todo: return []any{brightnesssensor.WrapApi(brightnesssensor.NewModelServer(brightnesssensor.NewModel()))}, nil
		return nil, nil
	case trait.Channel:
		// todo: return []any{channel.WrapApi(channel.NewModelServer(channel.NewModel())), nil
		return nil, nil
	case trait.Count:
		// todo: return []any{count.WrapApi(count.NewModelServer(count.NewModel())), nil
		return nil, nil
	case trait.Electric:
		model := electricpb2.NewModel()
		return []wrap.ServiceUnwrapper{electricpb2.WrapApi(electricpb2.NewModelServer(model))}, auto.Electric(model)
	case trait.Emergency:
		// todo: return []any{emergency.WrapApi(emergency.NewModelServer(emergency.NewModel()))}, nil
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
		return []wrap.ServiceUnwrapper{energystoragepb.WrapApi(energystoragepb.NewModelServer(model))}, auto.EnergyStorage(model, kind)
	case trait.EnterLeaveSensor:
		model := enterleavesensorpb2.NewModel()
		return []wrap.ServiceUnwrapper{enterleavesensorpb2.WrapApi(enterleavesensorpb2.NewModelServer(model))}, auto.EnterLeaveAuto(model)
	case trait.ExtendRetract:
		// todo: return []any{extendretract.WrapApi(extendretract.NewModelServer(extendretract.NewModel()))}, nil
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
		return []wrap.ServiceUnwrapper{fanspeedpb2.WrapApi(fanspeedpb2.NewModelServer(model))}, auto.FanSpeed(model, presets...)
	case trait.Hail:
		return []wrap.ServiceUnwrapper{hailpb2.WrapApi(hailpb2.NewModelServer(hailpb2.NewModel()))}, nil
	case trait.InputSelect:
		// todo: return []any{inputselect.WrapApi(inputselect.NewModelServer(inputselect.NewModel()))}, nil
		return nil, nil
	case trait.Light:
		server := lightpb2.NewModelServer(lightpb2.NewModel(
			lightpb2.WithPreset(0, &lightpb.LightPreset{Name: "off", Title: "Off"}),
			lightpb2.WithPreset(40, &lightpb.LightPreset{Name: "low", Title: "Low"}),
			lightpb2.WithPreset(60, &lightpb.LightPreset{Name: "med", Title: "Normal"}),
			lightpb2.WithPreset(80, &lightpb.LightPreset{Name: "high", Title: "High"}),
			lightpb2.WithPreset(100, &lightpb.LightPreset{Name: "full", Title: "Full"}),
		))
		return []wrap.ServiceUnwrapper{lightpb2.WrapApi(server), lightpb2.WrapInfo(server)}, nil
	case trait.LockUnlock:
		// todo: return []any{lockunlock.WrapApi(lockunlock.NewModelServer(lockunlock.NewModel()))}, nil
		return nil, nil
	case trait.Metadata:
		return []wrap.ServiceUnwrapper{metadatapb2.WrapApi(metadatapb2.NewModelServer(metadatapb2.NewModel()))}, nil
	case trait.Microphone:
		// todo: return []any{microphone.WrapApi(microphone.NewModelServer(microphone.NewModel()))}, nil
		return nil, nil
	case trait.Mode:
		return mockMode(traitMd, deviceName, logger)
	case trait.MotionSensor:
		// todo: return []any{motionsensor.WrapApi(motionsensor.NewModelServer(motionsensor.NewModel()))}, nil
		return nil, nil
	case trait.OccupancySensor:
		model := occupancysensorpb2.NewModel()
		return []wrap.ServiceUnwrapper{occupancysensorpb2.WrapApi(occupancysensorpb2.NewModelServer(model))}, auto.OccupancySensorAuto(model)
	case trait.OnOff:
		return []wrap.ServiceUnwrapper{onoffpb.WrapApi(onoffpb.NewModelServer(onoffpb.NewModel(resource.WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_OFF}))))}, nil
	case trait.OpenClose:
		return mockOpenClose(traitMd, deviceName, logger)
	case trait.Parent:
		return []wrap.ServiceUnwrapper{parentpb.WrapApi(parentpb.NewModelServer(parentpb.NewModel()))}, nil
	case trait.Publication:
		return []wrap.ServiceUnwrapper{publicationpb.WrapApi(publicationpb.NewModelServer(publicationpb.NewModel()))}, nil
	case trait.Ptz:
		// todo: return []any{ptz.WrapApi(ptz.NewModelServer(ptz.NewModel()))}, nil
		return nil, nil
	case trait.Speaker:
		// todo: return []any{speaker.WrapApi(speaker.NewModelServer(speaker.NewModel())), nil
		return nil, nil
	case trait.Vending:
		return []wrap.ServiceUnwrapper{vendingpb.WrapApi(vendingpb.NewModelServer(vendingpb.NewModel()))}, nil

	case accesspb.TraitName:
		model := accesspb.NewModel()
		return []wrap.ServiceUnwrapper{accesspb.WrapApi(accesspb.NewModelServer(model))}, auto.Access(model)
	case buttonpb.TraitName:
		return []wrap.ServiceUnwrapper{buttonpb.WrapApi(buttonpb.NewModelServer(buttonpb.NewModel(buttonpb.ButtonState_UNPRESSED)))}, nil
	case emergencylightpb.TraitName:
		model := emergencylightpb.NewModel()
		model.SetLastDurationTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		model.SetLastFunctionalTest(emergencylightpb.EmergencyTestResult_TEST_PASSED)
		return []wrap.ServiceUnwrapper{emergencylightpb.WrapApi(emergencylightpb.NewModelServer(model))}, nil
	case fluidflowpb.TraitName:
		model := fluidflowpb.NewModel()
		return []wrap.ServiceUnwrapper{fluidflowpb.WrapApi(fluidflowpb.NewModelServer(model))}, auto.FluidFlow(model)
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
			ResourceSupport: &types.ResourceSupport{
				Readable:   true,
				Writable:   true,
				Observable: true,
			},
			UsageUnit: unit,
		}}
		return []wrap.ServiceUnwrapper{meterpb.WrapApi(meterpb.NewModelServer(model)), meterpb.WrapInfo(info)}, auto.MeterAuto(model)
	case pressurepb.TraitName:
		model := pressurepb.NewModel()
		return []wrap.ServiceUnwrapper{pressurepb.WrapApi(pressurepb.NewModelServer(model))}, auto.Pressure(model)
	case securityeventpb.TraitName:
		model := securityeventpb.NewModel()
		return []wrap.ServiceUnwrapper{securityeventpb.WrapApi(securityeventpb.NewModelServer(model))}, auto.SecurityEventAuto(model)
	case soundsensorpb.TraitName:
		model := soundsensorpb.NewModel()
		return []wrap.ServiceUnwrapper{soundsensorpb.WrapApi(soundsensorpb.NewModelServer(model))}, auto.SoundSensorAuto(model)
	case statuspb.TraitName:
		model := statuspb.NewModel()
		// set an initial value or Pull methods can hang
		_, _ = model.UpdateProblem(&statuspb.StatusLog_Problem{Name: deviceName, Level: statuspb.StatusLog_NOMINAL})
		return []wrap.ServiceUnwrapper{statuspb.WrapApi(statuspb.NewModelServer(model))}, auto.Status(model, deviceName)
	case temperaturepb.TraitName:
		model := temperaturepb.NewModel()
		return []wrap.ServiceUnwrapper{temperaturepb.WrapApi(temperaturepb.NewModelServer(model))}, auto.TemperatureAuto(model)
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
		return []wrap.ServiceUnwrapper{transportpb.WrapApi(transportpb.NewModelServer(model))}, auto.TransportAuto(model, maxFloor)
	case udmipb.TraitName:
		return []wrap.ServiceUnwrapper{udmipb.WrapService(auto.NewUdmiServer(logger, deviceName))}, nil
	case wastepb.TraitName:
		model := wastepb.NewModel()
		return []wrap.ServiceUnwrapper{wastepb.WrapApi(wastepb.NewModelServer(model))}, auto.WasteRecordsAuto(model)
	}

	return nil, nil
}
