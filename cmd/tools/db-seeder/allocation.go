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

func SeedAllocation(ctx context.Context, db *pgxpool.Pool, name string, lookBack time.Duration) error {
	now := time.Now()
	current := now.Add(-lookBack)

	source := fmt.Sprintf("%s[%s]", name, allocationpb.TraitName)

	store, err := pgxstore.SetupStoreFromPool(ctx, source, db)
	if err != nil {
		return err
	}

	allocationTotal := int32(0)
	unallocationTotal := int32(0)

	for current.Before(now) {
		load := officeLoad(current)

		// Probability of ALLOCATED scales with office activity:
		// ~80% chance at peak, near 0% at night.
		var state allocationpb.Allocation_State
		if rand.Float64() < load*0.8 {
			state = allocationpb.Allocation_ALLOCATED
			allocationTotal += int32(rand.Intn(5) + 1)
		} else {
			state = allocationpb.Allocation_UNALLOCATED
			unallocationTotal += int32(rand.Intn(5) + 1)
		}

		payload, err := proto.Marshal(&allocationpb.Allocation{
			State:             state,
			GroupId:           ptr("GroupA"),
			AllocationTotal:   ptr(allocationTotal),
			UnallocationTotal: ptr(unallocationTotal),
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

func ptr[T any](v T) *T {
	return &v
}
