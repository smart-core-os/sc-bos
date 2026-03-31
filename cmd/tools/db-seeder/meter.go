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

func SeedMeter(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
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
		load := profile.Load(current)
		elapsed := current.Sub(prevTime)

		// Power scales from StandbyFactor at night to full PeakPowerKW at peak occupancy.
		m := profile.Meter
		instantPowerKW := (m.StandbyFactor + (1-m.StandbyFactor)*load) * m.PeakPowerKW
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
		current = current.Add(profile.IntervalForLoad(load))
	}
	return nil
}
