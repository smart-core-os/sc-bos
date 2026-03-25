package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const (
	tempOccupiedMin  = 21.0 // °C comfort setpoint lower bound
	tempOccupiedMax  = 22.0 // °C comfort setpoint upper bound
	tempSetbackMin   = 16.0 // °C unoccupied setback lower bound
	tempSetbackMax   = 18.0 // °C unoccupied setback upper bound
	tempMaxRatePerStep = 0.5 // max °C change per 15-min HVAC step
)

func SeedAirTemperature(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.AirTemperature)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	// Initialise to setback values (building is cold at start of lookback).
	currentSetpoint := tempSetbackMin + rand.Float64()*(tempSetbackMax-tempSetbackMin)
	currentAmbient := currentSetpoint + rand.NormFloat64()*0.5

	for current.Before(now) {
		load := officeLoad(current)

		// Choose target setpoint based on occupancy level.
		var targetSetpoint float64
		if load > 0.2 {
			targetSetpoint = tempOccupiedMin + rand.Float64()*(tempOccupiedMax-tempOccupiedMin)
		} else {
			targetSetpoint = tempSetbackMin + rand.Float64()*(tempSetbackMax-tempSetbackMin)
		}

		// Move setpoint toward target at HVAC ramp speed.
		diff := targetSetpoint - currentSetpoint
		if diff > tempMaxRatePerStep {
			diff = tempMaxRatePerStep
		} else if diff < -tempMaxRatePerStep {
			diff = -tempMaxRatePerStep
		}
		currentSetpoint += diff

		// Ambient temperature lags the setpoint (thermal mass of the building).
		targetAmbient := currentSetpoint + rand.NormFloat64()*0.5
		currentAmbient += (targetAmbient - currentAmbient) * 0.3

		setPoint := currentSetpoint
		ambientTemp := currentAmbient

		payload, err := proto.Marshal(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{
				ValueCelsius: ambientTemp,
			},
			TemperatureGoal: &traits.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &types.Temperature{ValueCelsius: setPoint},
			},
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
