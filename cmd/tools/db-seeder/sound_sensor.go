package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
)

func SeedSoundSensor(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, soundsensorpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	// Start at night-time HVAC-hum baseline.
	baseSoundLevel := 22.0

	for current.Before(now) {
		load := officeLoad(current)

		// Target dB anchored to office load: 22 dB (quiet night) → 60 dB (busy day).
		targetDb := 22 + load*38

		// Random walk toward the load-based target.
		baseSoundLevel += (targetDb-baseSoundLevel)*0.2 + rand.NormFloat64()*2.0
		baseSoundLevel = clampFloat64(baseSoundLevel, 15, 75)

		soundLevel := float32(baseSoundLevel)

		payload, err := proto.Marshal(&soundsensorpb.SoundLevel{
			SoundPressureLevel: &soundLevel,
		})
		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)
		if err != nil {
			return err
		}

		current = current.Add(intervalForLoad(load))
	}
	return nil
}
