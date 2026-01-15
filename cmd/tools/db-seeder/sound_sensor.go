package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	gen_soundsensorpb "github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
)

func SeedSoundSensor(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, soundsensorpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	// Start with a base sound level between 20-40 dB
	baseSoundLevel := 20 + rand.Float32()*20

	for current.Before(now) {
		// Generate sound level that varies +/- 2 dB from the current level
		soundLevel := baseSoundLevel + (rand.Float32()*4 - 2)

		// Ensure the sound level stays within a reasonable range (15-60 dB)
		if soundLevel < 15 {
			soundLevel = 15
		} else if soundLevel > 60 {
			soundLevel = 60
		}

		payload, err := proto.Marshal(&gen_soundsensorpb.SoundLevel{
			SoundPressureLevel: &soundLevel,
		})

		if err != nil {
			return err
		}

		_, _, err = store.Insert(ctx, current, payload)
		if err != nil {
			return err
		}

		// Update the base sound level slightly for next iteration
		baseSoundLevel = soundLevel

		current = current.Add(time.Duration(rand.Intn(60)) * time.Minute)
	}
	return nil
}
