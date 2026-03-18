package main

import (
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

func announceOccupancy(root node.Announcer, name string, val int32) error {
	model := occupancysensorpb.NewModel()
	_, err := model.SetOccupancy(&occupancysensorpb.Occupancy{PeopleCount: val, Confidence: 1})
	if err != nil {
		return err
	}
	client := node.WithClients(occupancysensorpb.WrapApi(occupancysensorpb.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.OccupancySensor, client))
	return nil
}

func announceTemperature(root node.Announcer, name string, celsius float64) error {
	model := airtemperaturepb.NewModel()
	_, err := model.UpdateAirTemperature(&airtemperaturepb.AirTemperature{AmbientTemperature: &types.Temperature{ValueCelsius: celsius}})
	if err != nil {
		return err
	}
	client := node.WithClients(airtemperaturepb.WrapApi(airtemperaturepb.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.AirTemperature, client))
	return nil
}

func announceAirQuality(root node.Announcer, name string, val float32) error {
	model := airqualitysensorpb.NewModel()
	_, err := model.UpdateAirQuality(&airqualitysensorpb.AirQuality{CarbonDioxideLevel: &val})
	if err != nil {
		return err
	}
	client := node.WithClients(airqualitysensorpb.WrapApi(airqualitysensorpb.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.AirQualitySensor, client))
	return nil
}
