package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
)

func SeedAllocation(ctx context.Context, db *pgxpool.Pool, name string, profile *OfficeProfile, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, allocationpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	allocationTotal := int32(0)
	unallocationTotal := int32(0)

	for !current.After(now) {
		load := profile.Load(current)

		// Probability of an ALLOCATED event scales with office activity.
		var state allocationpb.Allocation_State
		if rand.Float64() < load*profile.Allocation.MaxProbability {
			state = allocationpb.Allocation_ALLOCATED
			allocationTotal += int32(rand.Intn(5) + 1)
		} else {
			state = allocationpb.Allocation_UNALLOCATED
			unallocationTotal += int32(rand.Intn(5) + 1)
		}

		payload, err := proto.Marshal(&allocationpb.Allocation{
			State:             state,
			GroupId:           new("GroupA"),
			AllocationTotal:   new(allocationTotal),
			UnallocationTotal: new(unallocationTotal),
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
