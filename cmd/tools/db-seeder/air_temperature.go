package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func SeedAirTemperature(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.AirTemperature)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	// Generate a random set point between 18 and 24 degrees with 0.5 degree accuracy
	randomNumber := 18 + rand.Float64()*6
	setPoint := math.Round(randomNumber*2) / 2

	for current.Before(now) {
		// Generate ambient temperature that varies +/- 2 degrees from set point
		ambientTemp := setPoint + (rand.Float64()*4 - 2)

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

		// Occasionally adjust the set point slightly to simulate realistic behavior
		if rand.Float64() < 0.1 { // 10% chance to adjust set point
			adjustment := rand.Float64() - 0.5 // -0.5 to +0.5 degree adjustment
			setPoint = math.Max(18, math.Min(24, setPoint+adjustment))
			setPoint = math.Round(setPoint*2) / 2 // Round to 0.5 degrees
		}

		current = current.Add(time.Duration(rand.Intn(60)) * time.Minute)
	}
	return nil
}
