package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func SeedElectric(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.Electric)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	for current.Before(now) {
		// Use the same time-of-day scaling as the mock driver
		tod := float32(scale.NineToFive.At(current))

		// Generate electric demand values matching mock driver patterns
		currentVal := float32Between(20, 40) * tod // 20-40A range scaled by time of day
		voltage := float32Between(238, 243)        // 238-243V range
		powerFactor := float32Between(0.7, 1.3)    // 0.7-1.3 range

		apparentPower := currentVal * voltage
		realPower := apparentPower * powerFactor
		reactivePower := apparentPower * (1 - powerFactor)

		payload, err := proto.Marshal(&traits.ElectricDemand{
			Current:       currentVal,
			Voltage:       &voltage,
			PowerFactor:   &powerFactor,
			ApparentPower: &apparentPower,
			RealPower:     &realPower,
			ReactivePower: &reactivePower,
		})

		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)
		if err != nil {
			return err
		}

		// Random interval between 1-30 minutes, matching mock driver's pattern
		interval := time.Duration(1+rand.Intn(29)) * time.Minute
		current = current.Add(interval)
	}

	return nil
}

func float32Between(min, max float32) float32 {
	return min + (max-min)*rand.Float32()
}
