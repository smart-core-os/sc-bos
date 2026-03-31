package resource

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

func TestCollection_Pull(t *testing.T) {
	t.Run("SeedValue", func(t *testing.T) {
		now := time.UnixMilli(0)
		clock := clockFunc(func() time.Time {
			return now
		})

		c := NewCollection(WithClock(clock))
		c.Add("three", val(1))
		c.Add("one", val(1))

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := c.Pull(ctx, WithBackpressure(false))

		// first value when not using UpdatesOnly should say it's not an update
		seed := waitForChan(t, changes, time.Second)
		want := &CollectionChange{
			Id:            "one",
			ChangeTime:    now,
			ChangeType:    typespb.ChangeType_ADD,
			NewValue:      val(1),
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
			ChangeType:    typespb.ChangeType_ADD,
			NewValue:      val(1),
			SeedValue:     true,
			LastSeedValue: true,
		}
		if diff := cmp.Diff(want, seed, protocmp.Transform()); diff != "" {
			t.Fatalf("Seed Value (-want,+got)\n%s", diff)
		}

		// second value should be an update
		c.Update("one", val(2))
		next := waitForChan(t, changes, time.Second)
		want = &CollectionChange{
			Id:         "one",
			ChangeTime: now,
			ChangeType: typespb.ChangeType_UPDATE,
			OldValue:   val(1),
			NewValue:   val(2),
		}
		if diff := cmp.Diff(want, next, protocmp.Transform()); diff != "" {
			t.Fatalf("Next Value (-want,+got)\n%s", diff)
		}

		// testing that adding also doesn't report as a SeedValue
		c.Update("two", val(1), WithCreateIfAbsent())
		next = waitForChan(t, changes, time.Second)
		want = &CollectionChange{
			Id:         "two",
			ChangeTime: now,
			ChangeType: typespb.ChangeType_ADD,
			NewValue:   val(1),
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
		c.Add("one", val(1))

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := c.Pull(ctx, WithBackpressure(false), WithUpdatesOnly(true))

		// with updates only, there should be no waiting event
		noEmitWithin(t, changes, 50*time.Millisecond)

		// first value should be an update
		c.Update("one", val(2))
		change := waitForChan(t, changes, time.Second)
		want := &CollectionChange{
			Id:         "one",
			ChangeTime: now,
			ChangeType: typespb.ChangeType_UPDATE,
			OldValue:   val(1),
			NewValue:   val(2),
		}
		if diff := cmp.Diff(want, change, protocmp.Transform()); diff != "" {
			t.Fatalf("Value (-want,+got)\n%s", diff)
		}
	})
}
