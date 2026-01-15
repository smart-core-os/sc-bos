package auto

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	gen_allocationpb "github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
)

func AllocationAuto(model *allocationpb.Model) service.Lifecycle {
	states := maps.Values(gen_allocationpb.Allocation_State_value)

	states = states[1:] // remove the UNKNOWN state

	actors := []*actorpb.Actor{
		nil,
		{DisplayName: "Scott Lang", Ids: map[string]string{"card": "1234567890"}},
		{DisplayName: "Hope Van Dyne", Ids: map[string]string{"card": "0987654321"}},
		{DisplayName: "Janet Van Dyne", Ids: map[string]string{"card": "1234567890"}},
	}

	groupIds := []string{
		"GroupA",
		"GroupB",
		"GroupC",
	}

	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)

			allocationTotal := int32(0)
			unallocationTotal := int32(0)

			for {
				allocation := gen_allocationpb.Allocation_State(states[rand.IntN(len(states))])

				switch allocation {
				case gen_allocationpb.Allocation_ALLOCATED:
					allocationTotal++
				case gen_allocationpb.Allocation_UNALLOCATED:
					unallocationTotal++
				}

				state := &gen_allocationpb.Allocation{
					State:             allocation,
					Actor:             actors[rand.IntN(len(actors))],
					GroupId:           &groupIds[rand.IntN(len(groupIds))],
					AllocationTotal:   &allocationTotal,
					UnallocationTotal: &unallocationTotal,
				}
				model.UpdateAllocation(state)
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{}) // call configure to ensure we load when start is called.
	return slc
}
