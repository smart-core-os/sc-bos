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
		chance := rand.Intn(2)
		// Randomly pick between ALLOCATED and UNALLOCATED
		state := allocationpb.Allocation_State(allocationpb.Allocation_State_value[allocationpb.Allocation_State_name[int32(chance+1)]])

		if state == allocationpb.Allocation_ALLOCATED {
			allocationTotal += int32(rand.Intn(5) + 1)
		} else {
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

		current = current.Add(time.Duration(rand.Intn(30)) * time.Minute)
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}
