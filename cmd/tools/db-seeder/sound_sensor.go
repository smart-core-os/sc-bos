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

func SeedSoundSensor(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, soundsensorpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	s := profile.Sound

	// Start at the night-time baseline.
	baseSoundLevel := s.NightDB

	for current.Before(now) {
		load := profile.Load(current)

		// Target dB is anchored to the load-based range defined by NightDB and PeakDB.
		targetDb := s.NightDB + load*(s.PeakDB-s.NightDB)

		// Random walk toward the load target.
		baseSoundLevel += (targetDb-baseSoundLevel)*s.WalkFactor + rand.NormFloat64()*s.NoiseSigma
		baseSoundLevel = clampFloat64(baseSoundLevel, s.ClampMin, s.ClampMax)

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

		current = current.Add(profile.IntervalForLoad(load))
	}
	return nil
}
