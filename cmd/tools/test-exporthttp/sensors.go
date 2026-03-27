package main

import (
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func announceOccupancy(root node.Announcer, name string, val int32) error {
	model := occupancysensorpb.NewModel()
	_, err := model.SetOccupancy(&occupancysensorpb.Occupancy{PeopleCount: val, Confidence: 1})
	if err != nil {
		return err
	}
	root.Announce(name,
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
		node.HasTrait(trait.OccupancySensor))
	return nil
}

func announceTemperature(root node.Announcer, name string, celsius float64) error {
	model := airtemperaturepb.NewModel()
	_, err := model.UpdateAirTemperature(&airtemperaturepb.AirTemperature{AmbientTemperature: &typespb.Temperature{ValueCelsius: celsius}})
	if err != nil {
		return err
	}
	root.Announce(name,
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(model))),
		node.HasTrait(trait.AirTemperature))
	return nil
}

func announceAirQuality(root node.Announcer, name string, val float32) error {
	model := airqualitysensorpb.NewModel()
	_, err := model.UpdateAirQuality(&airqualitysensorpb.AirQuality{CarbonDioxideLevel: &val})
	if err != nil {
		return err
	}
	root.Announce(name,
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model))),
		node.HasTrait(trait.AirQualitySensor))
	return nil
}
