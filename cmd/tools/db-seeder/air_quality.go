package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func SeedAirQuality(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.AirQualitySensor)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	// Stateful accumulators — CO2 and VOC build up with occupancy and decay with ventilation.
	co2Level := 400.0   // ppm, starts at outdoor baseline
	vocLevel := 50.0    // µg/m³, starts at clean indoor baseline

	for current.Before(now) {
		load := officeLoad(current)

		// CO2: exponential approach toward occupancy-driven target.
		// Ventilation rate increases with occupancy (HVAC works harder when occupied).
		ventRate := 0.1 + load*0.3
		targetCO2 := 400 + load*800
		co2Level += (targetCO2 - co2Level) * ventRate
		co2Level += rand.NormFloat64() * 20
		co2Level = clampFloat64(co2Level, 350, 1500)

		// VOC: similar exponential approach.
		targetVOC := 50 + load*350
		vocLevel += (targetVOC - vocLevel) * 0.15
		vocLevel += rand.NormFloat64() * 10
		vocLevel = clampFloat64(vocLevel, 20, 500)

		// Air pressure: slow daily variation, no units change (hPa).
		airPressure := float32(1013 + rand.NormFloat64()*3)

		// Air changes per hour: more ventilation when occupied.
		airChange := float32(2 + load*4)

		// Infection risk correlates with occupancy and CO2 concentration.
		infection := float32(clampFloat64(load*0.5*(co2Level/1000)*100, 0, 100))

		// Score: decreases with high CO2 and VOC.
		score := float32(clampFloat64(100-(co2Level-400)/1600*50-vocLevel/500*50, 0, 100))

		// Comfort: good when air quality score is high.
		comfort := traits.AirQuality_COMFORTABLE
		if score < 70 {
			comfort = traits.AirQuality_UNCOMFORTABLE
		}

		// Particulates: relatively stable in a well-filtered office, slight increase with activity.
		particulate1 := float32(5 + rand.Float64()*5 + load*5)
		particulate25 := float32(8 + rand.Float64()*7 + load*5)
		particulate10 := float32(10 + rand.Float64()*10 + load*5)

		co2 := float32(co2Level)
		voc := float32(vocLevel)

		payload, err := proto.Marshal(&traits.AirQuality{
			CarbonDioxideLevel:       &co2,
			VolatileOrganicCompounds: &voc,
			AirPressure:              &airPressure,
			Comfort:                  comfort,
			InfectionRisk:            &infection,
			Score:                    &score,
			ParticulateMatter_1:      &particulate1,
			ParticulateMatter_10:     &particulate10,
			ParticulateMatter_25:     &particulate25,
			AirChangePerHour:         &airChange,
		})
		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)
		if err != nil {
			return err
		}

		current = current.Add(15 * time.Minute)
	}
	return nil
}
