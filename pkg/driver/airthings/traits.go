package airthings

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/local"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	typespb "github.com/smart-core-os/sc-bos/sc-api/go/types"
)

// announceDevice sets up and announces the traits supported by the device.
func (d *Driver) announceDevice(ctx context.Context, a node.Announcer, dev config.Device, loc *local.Location) error {
	for _, tn := range dev.Traits {
		// todo: read the RSSI prop and link it with status
		switch trait.Name(tn) {
		case trait.AirQualitySensor:
			model := airqualitysensorpb.NewModel()
			client := airqualitysensorpb.WrapApi(airqualitysensorpb.NewModelServer(model))
			a.Announce(dev.Name, node.HasTrait(trait.AirQualitySensor, node.WithClients(client)))
			go d.pullSampleAirQuality(ctx, dev, loc, model)
		case trait.AirTemperature:
			model := airtemperaturepb.NewModel()
			client := airtemperaturepb.WrapApi(roAirTemperatureServer{airtemperaturepb.NewModelServer(model)})
			a.Announce(dev.Name, node.HasTrait(trait.AirTemperature, node.WithClients(client)))
			go d.pullSampleAirTemperature(ctx, dev, loc, model)
		case trait.BrightnessSensor:
			model := brightnesssensorpb.NewModel()
			client := brightnesssensorpb.WrapApi(brightnesssensorpb.NewModelServer(model))
			a.Announce(dev.Name, node.HasTrait(trait.BrightnessSensor, node.WithClients(client)))
			go d.pullSampleBrightness(ctx, dev, loc, model)
		case trait.EnergyStorage:
			model := energystoragepb.NewModel()
			client := energystoragepb.WrapApi(roEnergyStorageServer{energystoragepb.NewModelServer(model)})
			a.Announce(dev.Name, node.HasTrait(trait.EnergyStorage, node.WithClients(client)))
			go d.pullSampleEnergyLevel(ctx, dev, loc, model)
		default:
			return fmt.Errorf("unsupported trait %q", tn)
		}
	}
	return nil
}

func (d *Driver) pullSampleAirQuality(ctx context.Context, dev config.Device, loc *local.Location, model *airqualitysensorpb.Model) {
	initial, stream := loc.PullLatestSamples(ctx, dev.ID)
	_, _ = model.UpdateAirQuality(sampleToAirQuality(initial))
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-stream:
			_, _ = model.UpdateAirQuality(sampleToAirQuality(sample))
		}
	}
}

func sampleToAirQuality(in api.DeviceSampleResponseEnriched) *airqualitysensorpb.AirQuality {
	dst := &airqualitysensorpb.AirQuality{}
	data := in.GetData()
	if v, ok := data.GetAirExchangeRateOk(); ok {
		dst.AirChangePerHour = float64PtoFloat32P(v)
	}
	if v, ok := data.GetCo2Ok(); ok {
		dst.CarbonDioxideLevel = float64PtoFloat32P(v)
	}
	if v, ok := data.GetPm1Ok(); ok {
		dst.ParticulateMatter_1 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetPm25Ok(); ok {
		dst.ParticulateMatter_25 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetPm10Ok(); ok {
		dst.ParticulateMatter_10 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetPressureOk(); ok {
		dst.AirPressure = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetVirusRiskOk(); ok {
		dst.InfectionRisk = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetVocOk(); ok {
		*v.Float64 = (*v.Float64) / 1000 // convert from ppb to ppm
		dst.VolatileOrganicCompounds = float64PtoFloat32P(v.Float64)
	}

	// check the outdoor properties too
	if v, ok := data.GetOutdoorPm1Ok(); ok {
		dst.ParticulateMatter_1 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetOutdoorPm25Ok(); ok {
		dst.ParticulateMatter_25 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetOutdoorPm10Ok(); ok {
		dst.ParticulateMatter_10 = float64PtoFloat32P(v.Float64)
	}
	if v, ok := data.GetOutdoorPressureOk(); ok {
		dst.AirPressure = float64PtoFloat32P(v.Float64)
	}
	return dst
}

func (d *Driver) pullSampleAirTemperature(ctx context.Context, dev config.Device, loc *local.Location, model *airtemperaturepb.Model) {
	initial, stream := loc.PullLatestSamples(ctx, dev.ID)
	_, _ = model.UpdateAirTemperature(sampleToAirTemperature(initial))
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-stream:
			_, _ = model.UpdateAirTemperature(sampleToAirTemperature(sample))
		}
	}
}

func sampleToAirTemperature(in api.DeviceSampleResponseEnriched) *airtemperaturepb.AirTemperature {
	dst := &airtemperaturepb.AirTemperature{}
	data := in.GetData()
	if v, ok := data.GetTempOk(); ok {
		dst.AmbientTemperature = &typespb.Temperature{ValueCelsius: *v.Float64}
	}
	if v, ok := data.GetHumidityOk(); ok {
		dst.AmbientHumidity = float64PtoFloat32P(v.Float64)
	}

	// check the outdoor properties too
	if v, ok := data.GetOutdoorTempOk(); ok {
		dst.AmbientTemperature = &typespb.Temperature{ValueCelsius: *v.Float64}
	}
	if v, ok := data.GetOutdoorHumidityOk(); ok {
		dst.AmbientHumidity = float64PtoFloat32P(v.Float64)
	}
	return dst
}

func (d *Driver) pullSampleEnergyLevel(ctx context.Context, dev config.Device, loc *local.Location, model *energystoragepb.Model) {
	initial, stream := loc.PullLatestSamples(ctx, dev.ID)
	_, _ = model.UpdateEnergyLevel(sampleToEnergyLevel(initial))
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-stream:
			_, _ = model.UpdateEnergyLevel(sampleToEnergyLevel(sample))
		}
	}
}

func sampleToEnergyLevel(in api.DeviceSampleResponseEnriched) *energystoragepb.EnergyLevel {
	dst := &energystoragepb.EnergyLevel{}
	data := in.GetData()
	if v, ok := data.GetBatteryOk(); ok {
		dst.Quantity = &energystoragepb.EnergyLevel_Quantity{
			Percentage: *v,
		}
	}
	return dst
}

func (d *Driver) pullSampleBrightness(ctx context.Context, dev config.Device, loc *local.Location, model *brightnesssensorpb.Model) {
	initial, stream := loc.PullLatestSamples(ctx, dev.ID)
	_, _ = model.UpdateAmbientBrightness(sampleToAmbientBrightness(initial))
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-stream:
			_, _ = model.UpdateAmbientBrightness(sampleToAmbientBrightness(sample))
		}
	}
}

func sampleToAmbientBrightness(in api.DeviceSampleResponseEnriched) *brightnesssensorpb.AmbientBrightness {
	dst := &brightnesssensorpb.AmbientBrightness{}
	data := in.GetData()
	if v, ok := data.GetLightOk(); ok {
		if v.Float32 != nil {
			dst.BrightnessLux = *v.Float32
		}
	}
	return dst
}

func float64PtoFloat32P(in *float64) *float32 {
	if in == nil {
		return nil
	}
	v := float32(*in)
	return &v
}

type roAirTemperatureServer struct {
	airtemperaturepb.AirTemperatureApiServer
}

func (s roAirTemperatureServer) UpdateAirTemperature(context.Context, *airtemperaturepb.UpdateAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	return nil, status.Errorf(codes.Unimplemented, "read-only")
}

type roEnergyStorageServer struct {
	energystoragepb.EnergyStorageApiServer
}

func (s roEnergyStorageServer) Charge(context.Context, *energystoragepb.ChargeRequest) (*energystoragepb.ChargeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "read-only")
}
