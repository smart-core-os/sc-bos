package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

func SeedMeter(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, meter.TraitName)

	incremental := rand.Float32() * 1_000

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	for current.Before(now) {
		incremental = incremental + rand.Float32()*1_000
		payload, err := proto.Marshal(&meterpb.MeterReading{
			Usage: incremental,
		})

		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)

		if err != nil {
			return err
		}

		current = current.Add(time.Duration(rand.Intn(60)) * time.Minute)

	}
	return nil
}
