package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// peakPowerKW is the office's peak electrical load in kilowatts.
const peakPowerKW = 50.0

func SeedMeter(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, meterpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	usage := float32(0)
	prevTime := current

	for current.Before(now) {
		load := officeLoad(current)
		elapsed := current.Sub(prevTime)

		// Power scales from 5% standby at night to 100% at peak occupancy.
		instantPowerKW := (0.05 + 0.95*load) * peakPowerKW
		kWhIncrement := float32(instantPowerKW * elapsed.Hours())
		// Add ±5% noise to simulate real meter variation.
		kWhIncrement *= 0.95 + rand.Float32()*0.10
		usage += kWhIncrement

		payload, err := proto.Marshal(&meterpb.MeterReading{
			Usage: usage,
		})
		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)
		if err != nil {
			return err
		}

		prevTime = current
		current = current.Add(intervalForLoad(load))
	}
	return nil
}
