package main

import (
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	airqualitysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/airqualitysensorpb"
	airtemperaturepb2 "github.com/smart-core-os/sc-bos/pkg/trait/airtemperaturepb"
	occupancysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

func announceOccupancy(root node.Announcer, name string, val int32) error {
	model := occupancysensorpb2.NewModel()
	_, err := model.SetOccupancy(&traits.Occupancy{PeopleCount: val, Confidence: 1})
	if err != nil {
		return err
	}
	client := node.WithClients(occupancysensorpb2.WrapApi(occupancysensorpb2.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.OccupancySensor, client))
	return nil
}

func announceTemperature(root node.Announcer, name string, celsius float64) error {
	model := airtemperaturepb2.NewModel()
	_, err := model.UpdateAirTemperature(&traits.AirTemperature{AmbientTemperature: &types.Temperature{ValueCelsius: celsius}})
	if err != nil {
		return err
	}
	client := node.WithClients(airtemperaturepb2.WrapApi(airtemperaturepb2.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.AirTemperature, client))
	return nil
}

func announceAirQuality(root node.Announcer, name string, val float32) error {
	model := airqualitysensorpb2.NewModel()
	_, err := model.UpdateAirQuality(&traits.AirQuality{CarbonDioxideLevel: &val})
	if err != nil {
		return err
	}
	client := node.WithClients(airqualitysensorpb2.WrapApi(airqualitysensorpb2.NewModelServer(model)))
	root.Announce(name, node.HasTrait(trait.AirQualitySensor, client))
	return nil
}
