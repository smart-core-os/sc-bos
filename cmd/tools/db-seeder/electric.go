package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func SeedElectric(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.Electric)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	e := profile.Electric

	for !current.After(now) {
		// Use the same time-of-day scaling as the mock driver for current.
		tod := float32(scale.NineToFive.At(current))
		load := profile.Load(current)

		// NightCurrentA is the baseline (HVAC/servers always on); PeakCurrentA is the additional load.
		currentVal := e.NightCurrentA + e.PeakCurrentA*tod

		voltage := float32Between(e.VoltageMin, e.VoltageMax)

		// Power factor is bounded [0,1]; higher at full load (more resistive loads switched on).
		powerFactor := float32Between(
			e.PFBase+float32(load)*e.PFRange*0.5,
			e.PFBase+float32(load)*e.PFRange,
		)

		apparentPower := currentVal * voltage
		realPower := apparentPower * powerFactor
		reactivePower := apparentPower * (1 - powerFactor)

		payload, err := proto.Marshal(&electricpb.ElectricDemand{
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
