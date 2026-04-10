package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"
	"math/rand/v2"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/azureiot"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func main() {
	root := node.New("example-azureiot")

	model := airqualitysensorpb.NewModel(airqualitysensorpb.WithInitialAirQuality(iaq()))
	root.Announce("IAQ-001",
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model))),
		node.HasTrait(trait.AirQualitySensor),
	)

	l, _ := zap.NewDevelopment()
	services := auto.Services{
		Node:   root,
		Logger: l,
	}
	cs := "<< CONNECTION STRING >>"
	cfg := fmt.Sprintf(`{
	"type": "azureiot",
	"name": "example-azureiot",
	"pollInterval": "5s",
	"devices": [
		{
			"connectionString": %q,
			"children": [
				{"name": "IAQ-001", "traits": ["smartcore.traits.AirQualitySensor"]}
			]
		}
	]
}`, cs)
	a := azureiot.Factory.New(services)
	_, err := a.Configure([]byte(cfg))
	if err != nil {
		panic(err)
	}
	_, err = a.Start()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_, err := model.UpdateAirQuality(iaq())
			if err != nil {
				panic(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func iaq() *airqualitysensorpb.AirQuality {
	return &airqualitysensorpb.AirQuality{
		CarbonDioxideLevel:       randFP(400, 1000),
		VolatileOrganicCompounds: randFP(0.1, 0.23),
		AirPressure:              randFP(950, 1100),
		InfectionRisk:            randFP(1, 5),
		ParticulateMatter_1:      randFP(0, 10),
		ParticulateMatter_25:     randFP(0, 10),
		ParticulateMatter_10:     randFP(0, 10),
		AirChangePerHour:         randFP(1.8, 3),
	}
}

func randFP(from, to float32) *float32 {
	return new(from + (to-from)*rand.Float32())
}
