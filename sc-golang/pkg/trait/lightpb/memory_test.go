package lightpb

import (
	"context"
	"testing"
	"time"

	"github.com/smart-core-os/sc-golang/internal/th"
	"github.com/smart-core-os/sc-golang/pkg/resource"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func TestMemoryDevice_UpdateBrightness(t *testing.T) {
	t.Run("tween", func(t *testing.T) {
		api := NewMemoryDevice()
		api.brightnessTick = 10 * time.Millisecond // give us a chance to see some updates
		ctx, done := context.WithCancel(th.Ctx)
		update := api.brightness.Pull(ctx, resource.WithUpdatesOnly(true))
		t.Cleanup(done)
		var updates []*resource.ValueChange
		updatesDone := make(chan struct{})
		go func() {
			defer close(updatesDone)
			for change := range update {
				updates = append(updates, change)
			}
		}()

		duration := 100 * time.Millisecond
		expectedInitialValue := &traits.Brightness{
			LevelPercent:       0,
			TargetLevelPercent: 50,
			BrightnessTween: &types.Tween{
				TotalDuration: durationpb.New(duration),
				Progress:      0,
			},
		}
		initialValue, err := api.UpdateBrightness(th.Ctx, &traits.UpdateBrightnessRequest{
			Brightness: &traits.Brightness{
				LevelPercent: 50,
				BrightnessTween: &types.Tween{
					TotalDuration: durationpb.New(duration),
				},
			},
		})
		if err != nil {
			t.Fatalf("got error %v", err)
		}
		if diff := cmp.Diff(expectedInitialValue, initialValue, protocmp.Transform()); diff != "" {
			t.Fatalf("initial value (-want, +got)\n%v", diff)
		}

		time.Sleep(duration * 2)
		done()
		<-updatesDone
		if len(updates) <= 1 {
			t.Fatalf("expected more than one change, got %v", len(updates))
		}

		for i, change := range updates[1 : len(updates)-1] {
			lastChange := updates[i]
			// changes happen in order
			if !change.ChangeTime.After(lastChange.ChangeTime) {
				t.Fatalf("changes out of order [%v] %v is after [%v] %v", i, lastChange, i+1, change)
			}
			// changes move the value closer to the target value
			brightness := change.Value.(*traits.Brightness)
			lastBrightness := lastChange.Value.(*traits.Brightness)
			if brightness.LevelPercent <= lastBrightness.LevelPercent {
				t.Fatalf("LevelPercent is moving backwards [%v] %v >= [%v] %v", i, lastBrightness.LevelPercent, i+1, brightness.LevelPercent)
			}
			// changes advance the progress
			if brightness.BrightnessTween.Progress <= lastBrightness.BrightnessTween.Progress {
				t.Fatalf("Progress is moving backwards [%v] %v >= [%v] %v", i, lastBrightness.BrightnessTween.Progress, i+1, brightness.BrightnessTween.Progress)
			}
			// check that the current value is close to what we'd expect given the progress made
			progress := brightness.BrightnessTween.Progress
			slop := float32(0.1)
			expectedValue := progress / 2
			if brightness.LevelPercent < expectedValue-slop || brightness.LevelPercent > expectedValue+slop {
				t.Fatalf("at progress %v expected value %v (%v-%v), got %v", progress, expectedValue, expectedValue-slop, expectedValue+slop, brightness.LevelPercent)
			}
		}

		// the last update should be equal to our expected
		lastUpdate := updates[len(updates)-1]
		lastValue := lastUpdate.Value.(*traits.Brightness)
		expectedLastValue := &traits.Brightness{
			LevelPercent: 50,
		}
		if diff := cmp.Diff(expectedLastValue, lastValue, protocmp.Transform()); diff != "" {
			t.Fatalf("last value (-want, +got)\n%v", diff)
		}
	})

	t.Run("tween interrupted", func(t *testing.T) {
		api := NewMemoryDevice()
		api.brightnessTick = 10 * time.Millisecond // give us a chance to see some updates
		ctx, done := context.WithCancel(th.Ctx)
		update := api.brightness.Pull(ctx)
		t.Cleanup(done)

		tweenStarted := make(chan struct{}, 3)
		var updates []*resource.ValueChange
		updatesDone := make(chan struct{})
		go func() {
			defer close(updatesDone)
			for change := range update {
				updates = append(updates, change)

				select {
				case tweenStarted <- struct{}{}:
				default:
				}
			}
		}()

		duration := 100 * time.Millisecond
		_, err := api.UpdateBrightness(th.Ctx, &traits.UpdateBrightnessRequest{
			Brightness: &traits.Brightness{
				LevelPercent:    100,
				BrightnessTween: &types.Tween{TotalDuration: durationpb.New(duration)},
			},
		})
		if err != nil {
			t.Fatalf("got error %v", err)
		}

		// wait for three updates, the tween has done some work
		<-tweenStarted
		<-tweenStarted
		<-tweenStarted

		finalValue, err := api.UpdateBrightness(th.Ctx, &traits.UpdateBrightnessRequest{
			Brightness: &traits.Brightness{LevelPercent: 10},
		})
		if err != nil {
			t.Fatalf("got error %v", err)
		}

		time.Sleep(duration * 2)

		// expect between 4 and 5 updates, 3-4 from the initial tweening, and one from the final update
		done()
		<-updatesDone
		updateCount := len(updates)
		if updateCount != 4 {
			t.Fatalf("update count, want 4, got %v", updateCount)
		}
		expectedFinalValue := &traits.Brightness{LevelPercent: 10}
		if diff := cmp.Diff(expectedFinalValue, finalValue, protocmp.Transform()); diff != "" {
			t.Fatalf("final value (-want,+got)\n%v", diff)
		}
	})
}
