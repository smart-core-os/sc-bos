package airthings

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	typespb "github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/local"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/trait/energystoragepb"
)

// announceDevice sets up and announces the traits supported by the device.
func (d *Driver) announceDevice(ctx context.Context, a node.Announcer, dev config.Device, loc *local.Location) error {
	for _, tn := range dev.Traits {
		// todo: case trait.BrightnessSensor: once it has a model, backed by "light" data
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

func sampleToAirQuality(in api.DeviceSampleResponseEnriched) *traits.AirQuality {
	dst := &traits.AirQuality{}
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

func sampleToAirTemperature(in api.DeviceSampleResponseEnriched) *traits.AirTemperature {
	dst := &traits.AirTemperature{}
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

func sampleToEnergyLevel(in api.DeviceSampleResponseEnriched) *traits.EnergyLevel {
	dst := &traits.EnergyLevel{}
	data := in.GetData()
	if v, ok := data.GetBatteryOk(); ok {
		dst.Quantity = &traits.EnergyLevel_Quantity{
			Percentage: *v,
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
	traits.AirTemperatureApiServer
}

func (s roAirTemperatureServer) UpdateAirTemperature(context.Context, *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	return nil, status.Errorf(codes.Unimplemented, "read-only")
}

type roEnergyStorageServer struct {
	traits.EnergyStorageApiServer
}

func (s roEnergyStorageServer) Charge(context.Context, *traits.ChargeRequest) (*traits.ChargeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "read-only")
}
