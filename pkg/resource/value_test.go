package resource

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestValue_Pull(t *testing.T) {
	t.Run("SeedValue", func(t *testing.T) {
		now := time.UnixMilli(0)
		clock := clockFunc(func() time.Time {
			return now
		})

		v := NewValue(WithInitialValue(val(1)), WithClock(clock))

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := v.Pull(ctx, WithBackpressure(false))

		// first value when not using UpdatesOnly should say it's not an update
		seed := waitForChan(t, changes, time.Second)
		want := &ValueChange{
			ChangeTime:    now,
			Value:         val(1),
			SeedValue:     true,
			LastSeedValue: true,
		}
		if diff := cmp.Diff(want, seed, protocmp.Transform()); diff != "" {
			t.Fatalf("Seed Value (-want,+got)\n%s", diff)
		}

		// second value should be an update
		v.Set(val(2))
		next := waitForChan(t, changes, time.Second)
		want = &ValueChange{
			ChangeTime:    now,
			Value:         val(2),
			SeedValue:     false,
			LastSeedValue: false,
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

		v := NewValue(WithInitialValue(val(1)), WithClock(clock))

		ctx, stop := context.WithCancel(context.Background())
		t.Cleanup(stop)
		changes := v.Pull(ctx, WithBackpressure(false), WithUpdatesOnly(true))

		// with updates only, there should be no waiting event
		noEmitWithin(t, changes, 50*time.Millisecond)

		// first value should be an update
		v.Set(val(2))
		change := waitForChan(t, changes, time.Second)
		want := &ValueChange{
			ChangeTime:    now,
			Value:         val(2),
			SeedValue:     false,
			LastSeedValue: false,
		}
		if diff := cmp.Diff(want, change, protocmp.Transform()); diff != "" {
			t.Fatalf("Value (-want,+got)\n%s", diff)
		}
	})

	t.Run("doesnt panic with no initial value", func(t *testing.T) {
		v := NewValue()

		res, err := v.Set(val(2))

		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(val(2), res, protocmp.Transform()); diff != "" {
			t.Fatalf("Set response (-want,+got)\n%s", diff)
		}

		if diff := cmp.Diff(val(2), v.Get(), protocmp.Transform()); diff != "" {
			t.Fatalf("Get response (-want,+got)\n%s", diff)
		}
	})
}
