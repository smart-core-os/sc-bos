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

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func SeedOccupancy(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, trait.OccupancySensor)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	for current.Before(now) {
		load := officeLoad(current)
		baseCount := load * 49
		count := int32(math.Round(baseCount + rand.NormFloat64()*2))
		if count < 0 {
			count = 0
		}

		state := traits.Occupancy_UNOCCUPIED
		if count > 0 {
			state = traits.Occupancy_OCCUPIED
		}

		payload, err := proto.Marshal(&traits.Occupancy{
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

		current = current.Add(intervalForLoad(load))
	}

	return nil
}
