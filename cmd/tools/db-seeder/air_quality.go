package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func SeedAirQuality(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.AirQualitySensor)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	aq := profile.AirQuality

	// Stateful accumulators — CO2 and VOC build up with occupancy and decay with ventilation.
	co2Level := aq.CO2Baseline
	vocLevel := aq.VOCBaseline

	for current.Before(now) {
		load := profile.Load(current)

		// CO2: exponential approach toward occupancy-driven target.
		// Ventilation rate increases with occupancy (HVAC works harder when occupied).
		ventRate := aq.CO2VentBase + load*aq.CO2VentRange
		targetCO2 := aq.CO2Baseline + load*aq.CO2PeakAbove
		co2Level += (targetCO2-co2Level)*ventRate + rand.NormFloat64()*20
		co2Level = clampFloat64(co2Level, aq.CO2Baseline*0.875, aq.CO2Baseline+aq.CO2PeakAbove*1.25)

		// VOC: same exponential approach pattern.
		targetVOC := aq.VOCBaseline + load*aq.VOCPeakAbove
		vocLevel += (targetVOC-vocLevel)*aq.VOCApproachRate + rand.NormFloat64()*10
		vocLevel = clampFloat64(vocLevel, aq.VOCBaseline*0.4, aq.VOCBaseline+aq.VOCPeakAbove*1.25)

		airPressure := float32(aq.PressureHPa + rand.NormFloat64()*aq.PressureNoise)
		airChange := float32(aq.AirChangeBase + load*aq.AirChangeRange)

		// Infection risk correlates with occupancy and CO2 concentration.
		infection := float32(clampFloat64(load*0.5*(co2Level/1000)*100, 0, 100))

		// Score: decreases with elevated CO2 and VOC.
		co2Score := (co2Level - aq.CO2Baseline) / (aq.CO2PeakAbove * 1.25) * 50
		vocScore := vocLevel / (aq.VOCBaseline + aq.VOCPeakAbove) * 50
		score := float32(clampFloat64(100-co2Score-vocScore, 0, 100))

		comfort := airqualitysensorpb.AirQuality_COMFORTABLE
		if score < aq.ComfortScoreThreshold {
			comfort = airqualitysensorpb.AirQuality_UNCOMFORTABLE
		}

		// Particulates: low in a well-filtered office, slight increase with activity.
		particulate1 := float32(5 + rand.Float64()*5 + load*5)
		particulate25 := float32(8 + rand.Float64()*7 + load*5)
		particulate10 := float32(10 + rand.Float64()*10 + load*5)

		co2 := float32(co2Level)
		voc := float32(vocLevel)

		payload, err := proto.Marshal(&airqualitysensorpb.AirQuality{
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

		current = current.Add(aq.Interval)
	}
	return nil
}
