package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

func SeedAirTemperature(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.AirTemperature)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	tp := profile.Temperature

	// Initialise to setback values (building is cold at the start of the lookback period).
	currentSetpoint := tp.SetbackMin + rand.Float64()*(tp.SetbackMax-tp.SetbackMin)
	currentAmbient := currentSetpoint + rand.NormFloat64()*tp.AmbientNoiseSigma

	for current.Before(now) {
		load := profile.Load(current)

		// Choose target setpoint based on occupancy level.
		var targetSetpoint float64
		if load > tp.OccupiedThreshold {
			targetSetpoint = tp.OccupiedMin + rand.Float64()*(tp.OccupiedMax-tp.OccupiedMin)
		} else {
			targetSetpoint = tp.SetbackMin + rand.Float64()*(tp.SetbackMax-tp.SetbackMin)
		}

		// Move setpoint toward target at HVAC ramp speed.
		diff := clampFloat64(targetSetpoint-currentSetpoint, -tp.HVACRatePerStep, tp.HVACRatePerStep)
		currentSetpoint += diff

		// Ambient temperature lags the setpoint (thermal mass of the building).
		targetAmbient := currentSetpoint + rand.NormFloat64()*tp.AmbientNoiseSigma
		currentAmbient += (targetAmbient - currentAmbient) * tp.ThermalLagFactor

		setPoint := currentSetpoint
		ambientTemp := currentAmbient

		payload, err := proto.Marshal(&airtemperaturepb.AirTemperature{
			AmbientTemperature: &types.Temperature{
				ValueCelsius: ambientTemp,
			},
			TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
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

		current = current.Add(tp.Interval)
	}
	return nil
}
