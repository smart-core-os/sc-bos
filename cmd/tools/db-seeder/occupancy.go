package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func SeedOccupancy(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.OccupancySensor)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	for current.Before(now) {
		load := profile.Load(current)
		count := max(int32(math.Round(load*float64(profile.Occupancy.MaxPeople)+rand.NormFloat64()*2)), 0)

		state := occupancysensorpb.Occupancy_UNOCCUPIED
		if count > 0 {
			state = occupancysensorpb.Occupancy_OCCUPIED
		}

		payload, err := proto.Marshal(&occupancysensorpb.Occupancy{
			PeopleCount:     count,
			State:           state,
			StateChangeTime: timestamppb.New(current),
			Confidence:      1,
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
