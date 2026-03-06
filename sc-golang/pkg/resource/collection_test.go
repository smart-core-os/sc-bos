package resource

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func TestCollection_Pull(t *testing.T) {
	t.Run("SeedValue", func(t *testing.T) {
		now := time.UnixMilli(0)
		clock := clockFunc(func() time.Time {
			return now
		})

		c := NewCollection(WithClock(clock))
		c.Add("three", &traits.OnOff{State: traits.OnOff_ON})
		c.Add("one", &traits.OnOff{State: traits.OnOff_ON})

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := c.Pull(ctx, WithBackpressure(false))

		// first value when not using UpdatesOnly should say it's not an update
		seed := waitForChan(t, changes, time.Second)
		want := &CollectionChange{
			Id:            "one",
			ChangeTime:    now,
			ChangeType:    types.ChangeType_ADD,
			NewValue:      &traits.OnOff{State: traits.OnOff_ON},
			SeedValue:     true,
			LastSeedValue: false,
		}
		if diff := cmp.Diff(want, seed, protocmp.Transform()); diff != "" {
			t.Fatalf("Seed Value (-want,+got)\n%s", diff)
		}
		// second value is still a seed value, but should say its the last seed value
		seed = waitForChan(t, changes, time.Second)
		want = &CollectionChange{
			Id:            "three",
			ChangeTime:    now,
			ChangeType:    types.ChangeType_ADD,
			NewValue:      &traits.OnOff{State: traits.OnOff_ON},
			SeedValue:     true,
			LastSeedValue: true,
		}
		if diff := cmp.Diff(want, seed, protocmp.Transform()); diff != "" {
			t.Fatalf("Seed Value (-want,+got)\n%s", diff)
		}

		// second value should be an update
		c.Update("one", &traits.OnOff{State: traits.OnOff_OFF})
		next := waitForChan(t, changes, time.Second)
		want = &CollectionChange{
			Id:         "one",
			ChangeTime: now,
			ChangeType: types.ChangeType_UPDATE,
			OldValue:   &traits.OnOff{State: traits.OnOff_ON},
			NewValue:   &traits.OnOff{State: traits.OnOff_OFF},
		}
		if diff := cmp.Diff(want, next, protocmp.Transform()); diff != "" {
			t.Fatalf("Next Value (-want,+got)\n%s", diff)
		}

		// testing that adding also doesn't report as a SeedValue
		c.Update("two", &traits.OnOff{State: traits.OnOff_ON}, WithCreateIfAbsent())
		next = waitForChan(t, changes, time.Second)
		want = &CollectionChange{
			Id:         "two",
			ChangeTime: now,
			ChangeType: types.ChangeType_ADD,
			NewValue:   &traits.OnOff{State: traits.OnOff_ON},
		}
		if diff := cmp.Diff(want, next, protocmp.Transform()); diff != "" {
			t.Fatalf("Next Value (-want,+got)\n%s", diff)
		}
	})

	t.Run("SeedValue updatesOnly", func(t *testing.T) {
		now := time.UnixMilli(0)
		clock := clockFunc(func() time.Time {
			return now
		})

		c := NewCollection(WithClock(clock))
		c.Add("one", &traits.OnOff{State: traits.OnOff_ON})

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := c.Pull(ctx, WithBackpressure(false), WithUpdatesOnly(true))

		// with updates only, there should be no waiting event
		noEmitWithin(t, changes, 50*time.Millisecond)

		// first value should be an update
		c.Update("one", &traits.OnOff{State: traits.OnOff_OFF})
		change := waitForChan(t, changes, time.Second)
		want := &CollectionChange{
			Id:         "one",
			ChangeTime: now,
			ChangeType: types.ChangeType_UPDATE,
			OldValue:   &traits.OnOff{State: traits.OnOff_ON},
			NewValue:   &traits.OnOff{State: traits.OnOff_OFF},
		}
		if diff := cmp.Diff(want, change, protocmp.Transform()); diff != "" {
			t.Fatalf("Value (-want,+got)\n%s", diff)
		}
	})
}
