package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/trait"
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
		// Use the same time-of-day scaling as the mock driver for current.
		tod := float32(scale.NineToFive.At(current))
		load := officeLoad(current)

		// 4A baseline (HVAC/servers always on) plus up to 36A during peak occupancy.
		currentVal := 4 + 36*tod
		voltage := float32Between(238, 243)

		// Power factor is physically bounded [0,1]; higher during occupied hours.
		pfMin := float32(0.75 + load*0.10)
		pfMax := float32(0.85 + load*0.10)
		powerFactor := float32Between(pfMin, pfMax)

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

		current = current.Add(time.Duration(1+rand.Intn(29)) * time.Minute)
	}

	return nil
}

func float32Between(min, max float32) float32 {
	return min + (max-min)*rand.Float32()
}
