package lights

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func Test_processState(t *testing.T) {
	now := time.Unix(0, 0)
	autoStartTime := now.Add(-time.Hour)

	t.Run("empty", func(t *testing.T) {
		readState := NewReadState(autoStartTime)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)
		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNoMoreCalls()
	})

	t.Run("turn on when occupied", func(t *testing.T) {
		readState := NewReadState(autoStartTime)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{State: traits.Occupancy_OCCUPIED}

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 100,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("ignore non-relevant occupancy", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		// to ensure the automation start time doesn't consider it unoccupied
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 2 * time.Hour}
		readState.Config.OccupancySensors = []string{"pir02"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{State: traits.Occupancy_OCCUPIED}

		ttl, err := processState(context.Background(), readState, writeState, actions)
		// automation start time should expire after one hour
		assertNoErrAndTtl(t, ttl, err, time.Hour)
		actions.assertNoMoreCalls()
	})

	t.Run("turns lights off when unoccupied", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-20 * time.Minute)),
		}

		ttl, err := processState(context.Background(), readState, writeState, actions)

		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 0,
			},
		})
		actions.assertNoMoreCalls()
	})
	t.Run("pir ttl", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)
		now := time.Unix(0, 0)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-5 * time.Minute)),
		}

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("Got error %v", err)
		}
		if ttl != 5*time.Minute {
			t.Fatalf("TTL want %v, got %v", 5*time.Minute, ttl)
		}
		actions.assertNoMoreCalls()
	})

	t.Run("threshold brightness", func(t *testing.T) {
		dd := config.DaylightDimming{
			Thresholds: []config.LevelThreshold{
				{BelowLux: 10, LevelPercent: 100},
				{BelowLux: 200, LevelPercent: 70},
				{BelowLux: 1000, LevelPercent: 50},
				{BelowLux: 10_000, LevelPercent: 30},
				{BelowLux: 30_000, LevelPercent: 1},
				{LevelPercent: 0},
			},
		}
		tests := []struct {
			name string
			want float32
			lux  []float32
		}{
			{"no readings", 100, []float32{}},
			{"0 reading", 100, []float32{0}},
			{"average", 100, []float32{3, 4, 5}},
			{"just below threshold", 100, []float32{9.999}},
			{"on threshold", 70, []float32{10}},
			{"50%", 50, []float32{1000 - 1}},
			{"30%", 30, []float32{10_000 - 1}},
			{"1%", 1, []float32{30_000 - 1}},
			{"off", 0, []float32{30_000}},
			{"very bright", 0, []float32{100_000}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				readState := NewReadState(now)
				writeState := NewWriteState(time.Now())
				actions := newTestActions(t)

				readState.Config.DaylightDimming = &dd
				readState.Config.OccupancySensors = []string{"pir01"}
				readState.Config.Lights = []string{"light01"}
				readState.Occupancy["pir01"] = &traits.Occupancy{State: traits.Occupancy_OCCUPIED}

				readState.Config.BrightnessSensors = make([]string, len(tt.lux))
				for i, lux := range tt.lux {
					name := fmt.Sprintf("bri%02d", i)
					readState.Config.BrightnessSensors[i] = name
					readState.AmbientBrightness[name] = &traits.AmbientBrightness{BrightnessLux: lux}
				}

				ttl, err := processState(context.Background(), readState, writeState, actions)
				assertNoErrAndTtl(t, ttl, err, 0)
				actions.assertNextCall(&traits.UpdateBrightnessRequest{
					Name: "light01",
					Brightness: &traits.Brightness{
						LevelPercent: tt.want,
					},
				})
			})
		}
	})

	// Start of button tests

	t.Run("toggle pressed currently on", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 100},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 0 {
			t.Fatalf("Error, ttl not equal 0 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 0,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("toggle pressed currently half on", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01", "light02"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.Brightness["light02"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 50},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 0 {
			t.Fatalf("Error, ttl not equal 0 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light02",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("toggle pressed currently off", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 10*time.Minute {
			t.Fatalf("Error, ttl not equal 10 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 100,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("no op on ButtonState_PRESSED", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_PRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now.Add(-time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("toggle pressed dont action", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("toggle pressed in past dont action", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}

		readState.Buttons["toggleButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now.Add(-time.Minute)),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("on button pressed and off", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.OnButtons = []string{"onButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["onButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 10*time.Minute {
			t.Fatalf("Error, ttl not equal 10 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 100,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("on button pressed in the past", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.OnButtons = []string{"onButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["onButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now.Add(-time.Minute)),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 9*time.Minute {
			t.Fatalf("Error, ttl not equal 9 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 100,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("on button pressed and on", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.OnButtons = []string{"onButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["onButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 100},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 10*time.Minute {
			t.Fatalf("Error, ttl not equal 10 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 100},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("off button pressed and on", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.OffButtons = []string{"offButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["offButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 100},
		}
		writeState.LastButtonAction = now.Add(-5 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 0,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("off button pressed and off", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.OffButtons = []string{"offButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		readState.Buttons["offButton01"] = &buttonpb.ButtonState{
			State:             buttonpb.ButtonState_UNPRESSED,
			StateChangeTime:   timestamppb.New(now),
			MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
		}

		writeState.Brightness["light01"] = Value[*traits.Brightness]{
			At: now,
			V:  &traits.Brightness{LevelPercent: 0},
		}

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)

		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       "light01",
			Brightness: &traits.Brightness{LevelPercent: 0},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("button ttl", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}

		writeState.LastButtonOnTime = now.Add(-time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 9*time.Minute {
			t.Fatalf("Error, ttl not equal 9 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}
		actions.assertNoMoreCalls()
	})

	t.Run("button+pir ttl, button last", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}

		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-15 * time.Minute)),
		}
		writeState.LastButtonOnTime = now.Add(-time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 9*time.Minute {
			t.Fatalf("Error, ttl not equal 5 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}
		actions.assertNoMoreCalls()
	})

	t.Run("button+pir ttl, pir last", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}

		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-5 * time.Minute)),
		}
		writeState.LastButtonOnTime = now.Add(-15 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 5*time.Minute {
			t.Fatalf("Error, ttl not equal 5 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}
		actions.assertNoMoreCalls()
	})

	t.Run("button+pir ttl, both old", func(t *testing.T) {
		readState := testReadState(autoStartTime, now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.ToggleButtons = []string{"toggleButton01"}
		readState.Config.Lights = []string{"light01"}
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}

		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-15 * time.Minute)),
		}
		writeState.LastButtonOnTime = now.Add(-15 * time.Minute)

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 0,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("set on level from default mode", func(t *testing.T) {
		startTime := time.Date(2023, 4, 26, 0, 0, 0, 0, time.UTC)
		readState := NewReadState(startTime)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{State: traits.Occupancy_OCCUPIED}
		var onLevel float32 = 78
		readState.Config.Mode = config.Mode{
			OnLevelPercent: &onLevel,
		}

		ttl, err := processState(context.Background(), readState, writeState, actions)
		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: onLevel,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("set off level from default mode", func(t *testing.T) {
		now := time.Unix(0, 0)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.Now = func() time.Time { return now }
		readState.Config.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{
			State:           traits.Occupancy_UNOCCUPIED,
			StateChangeTime: timestamppb.New(now.Add(-20 * time.Minute)),
		}
		var offLevel float32 = 12
		readState.Config.Mode = config.Mode{
			OffLevelPercent: &offLevel,
		}

		ttl, err := processState(context.Background(), readState, writeState, actions)

		assertNoErrAndTtl(t, ttl, err, 0)
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: offLevel,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("set on level from mode at time", func(t *testing.T) {
		now := time.Unix(0, 0)
		now = now.In(time.UTC)
		readState := NewReadState(now)
		writeState := NewWriteState(time.Now())
		actions := newTestActions(t)

		readState.Config.OccupancySensors = []string{"pir01"}
		readState.Config.Lights = []string{"light01"}
		readState.Occupancy["pir01"] = &traits.Occupancy{State: traits.Occupancy_OCCUPIED}
		var onLevel, fullLevel float32 = 78, 100
		readState.Config.Mode = config.Mode{
			OnLevelPercent: &fullLevel,
		}
		readState.Config.Modes = []config.ModeOption{
			{
				Name:  "testMode",
				Start: jsontypes.MustParseSchedule("10 0 * * *"),
				End:   jsontypes.MustParseSchedule("18 0 * * *"),
				Mode: config.Mode{
					OnLevelPercent: &onLevel,
				},
			},
		}
		readState.Config.Now = func() time.Time { return now }

		ttl, err := processState(context.Background(), readState, writeState, actions)
		if ttl != 10*time.Minute {
			t.Fatalf("Error, ttl not equal 10 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 100,
			},
		})

		now = now.Add(10 * time.Minute)
		ttl, err = processState(context.Background(), readState, writeState, actions)
		if ttl != 8*time.Minute {
			t.Fatalf("Error, ttl not equal 8 minutes, got %s", ttl.String())
		}
		if err != nil {
			t.Fatalf("Error want <nil>, got %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: onLevel,
			},
		})
		actions.assertNoMoreCalls()
	})

	t.Run("reassert level on mode change", func(t *testing.T) {
		startTime := time.Unix(0, 0).In(time.UTC)
		readState := NewReadState(now)

		now := startTime.Add(time.Minute)
		readState.Config.Now = func() time.Time { return now }
		readState.Config.Modes = []config.ModeOption{
			{
				Name: "a",
				Mode: config.Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: time.Hour},
					OnLevelPercent:     asPtr[float32](33),
				},
			},
			{
				Name: "b",
				Mode: config.Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: time.Hour},
					OnLevelPercent:     asPtr[float32](66),
				},
			},
		}
		readState.Modes = &traits.ModeValues{
			Values: map[string]string{
				ModeValueKey: "a",
			},
		}
		readState.Config.ToggleButtons = []string{"button01"}
		readState.Config.Lights = []string{"light01"}
		readState.Buttons = map[string]*buttonpb.ButtonState{
			"button01": {
				State:           buttonpb.ButtonState_UNPRESSED,
				StateChangeTime: timestamppb.New(now),
				MostRecentGesture: &buttonpb.ButtonState_Gesture{
					Id:        "foo",
					Kind:      buttonpb.ButtonState_Gesture_CLICK,
					StartTime: timestamppb.New(now),
					EndTime:   timestamppb.New(now),
				},
			},
		}
		writeState := NewWriteState(startTime)

		actions := newTestActions(t)

		// check that we use mode a
		_, err := processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 33,
			},
		})

		// switch to mode b
		readState.Modes.Values[ModeValueKey] = "b"
		// check that we use mode b
		_, err = processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 66,
			},
		})
		actions.assertNoMoreCalls()
	})
	t.Run("do nothing when mode disables auto", func(t *testing.T) {
		startTime := time.Unix(0, 0).In(time.UTC)
		readState := NewReadState(now)

		now := startTime.Add(time.Minute)
		readState.Config.Now = func() time.Time { return now }
		readState.Config.Modes = []config.ModeOption{
			{
				Name: "a",
				Mode: config.Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: time.Hour},
					OnLevelPercent:     asPtr[float32](33),
				},
			},
			{
				Name: "b",
				Mode: config.Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: time.Hour},
					OnLevelPercent:     asPtr[float32](66),
				},
				DisableAuto: true,
			},
		}
		readState.Modes = &traits.ModeValues{
			Values: map[string]string{
				ModeValueKey: "a",
			},
		}
		readState.Config.ToggleButtons = []string{"button01"}
		readState.Config.Lights = []string{"light01"}
		readState.Buttons = map[string]*buttonpb.ButtonState{
			"button01": {
				State:           buttonpb.ButtonState_UNPRESSED,
				StateChangeTime: timestamppb.New(now),
				MostRecentGesture: &buttonpb.ButtonState_Gesture{
					Id:        "foo",
					Kind:      buttonpb.ButtonState_Gesture_CLICK,
					StartTime: timestamppb.New(now),
					EndTime:   timestamppb.New(now),
				},
			},
		}
		writeState := NewWriteState(startTime)

		actions := newTestActions(t)

		// check that we use mode a
		_, err := processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 33,
			},
		})

		// switch to mode b
		now = now.Add(time.Minute)
		readState.Modes.Values[ModeValueKey] = "b"
		// check that we use mode b
		_, err = processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		actions.assertNoMoreCalls()

		// switch back to mode a
		now = now.Add(time.Minute)
		readState.Modes.Values[ModeValueKey] = "a"
		_, err = processState(context.Background(), readState, writeState, actions)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		actions.assertNextCall(&traits.UpdateBrightnessRequest{
			Name: "light01",
			Brightness: &traits.Brightness{
				LevelPercent: 33,
			},
		})
		actions.assertNoMoreCalls()
	})
}

func assertNoErrAndTtl(t *testing.T, ttl time.Duration, err error, targetTtl time.Duration) {
	t.Helper()
	if ttl != targetTtl {
		t.Fatalf("TTL want %v, got %v", targetTtl, ttl)
	}
	if err != nil {
		t.Fatalf("Error want <nil>, got %v", err)
	}
}

func assertErrorAndTtl(t *testing.T, ttl time.Duration, err error, targetTtl time.Duration, targetErr error) {
	t.Helper()
	if ttl.Nanoseconds() < targetTtl.Nanoseconds() {
		t.Fatalf("TTL want %v, got %v, got ttl less than target TTL", targetTtl, ttl)
	}

	if !errors.Is(err, targetErr) {
		t.Fatalf("Error want %v, got %v", targetErr, err)
	}

}

func newTestActions(t *testing.T) *testActions {
	return &testActions{t: t}
}

type testActions struct {
	t *testing.T

	m        sync.Mutex
	calls    []any
	nextCall int // updated via assertNextCall

	err map[string]error

	brightnessCalls []*traits.UpdateBrightnessRequest
}

func (ta *testActions) assertNoMoreCalls() {
	ta.t.Helper()

	ta.m.Lock()
	defer ta.m.Unlock()
	if len(ta.calls) > ta.nextCall {
		callStr := ""
		for i, call := range ta.calls[ta.nextCall:] {
			callStr += fmt.Sprintf("  [%d] %+v\n", i, call)
		}
		ta.t.Fatalf("Call count want 0, got %d\n%s", len(ta.calls)-ta.nextCall, callStr)
	}
}

func (ta *testActions) assertNextCall(req any) {
	ta.t.Helper()

	ta.m.Lock()
	defer ta.m.Unlock()
	if len(ta.calls) <= ta.nextCall {
		ta.t.Fatalf("Call count want >%d, got %d", ta.nextCall, len(ta.calls))
	}
	call := ta.calls[ta.nextCall]
	ta.nextCall++

	if diff := cmp.Diff(req, call, protocmp.Transform()); diff != "" {
		ta.t.Fatalf("Next call (+want, -got)\n%s", diff)
	}
}

func (ta *testActions) assertNextBrightnessUpdates(level float32, names ...string) {
	ta.t.Helper()
	if len(names) == 0 {
		ta.t.Fatal("assertNextBrightnessUpdates called with no names")
	}

	for _, name := range names {
		ta.assertNextCall(&traits.UpdateBrightnessRequest{
			Name:       name,
			Brightness: &traits.Brightness{LevelPercent: level},
		})
	}
}

func (ta *testActions) nextCallReturnsError(err error, names ...string) {
	ta.t.Helper()
	if len(names) == 0 {
		ta.t.Fatal("nextCallReturnsError called with no names")
	}

	ta.m.Lock()
	defer ta.m.Unlock()
	if ta.err == nil {
		ta.err = make(map[string]error)
	}
	for _, name := range names {
		ta.err[name] = err
	}
}

func (ta *testActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	ta.m.Lock()
	defer ta.m.Unlock()
	ta.calls = append(ta.calls, req)
	ta.brightnessCalls = append(ta.brightnessCalls, req)
	err := ta.err[req.Name]
	if err != nil {
		delete(ta.err, req.Name)

		return err
	}

	state.Brightness[req.Name] = Value[*traits.Brightness]{
		At: now,
		V:  req.Brightness,
	}

	return nil
}

func Test_activeMode(t *testing.T) {
	startTime := time.Date(2023, 4, 26, 0, 0, 0, 0, time.UTC)
	cfg := NewReadState(startTime)
	cfg.Config.Mode = config.Mode{}
	// These modes basically look like this:
	//    [-a-----] [-d------]         [-f---]
	//      [-b-------]    [-e-------]
	//  [-c--------------]
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "a",
		Start: jsontypes.MustParseSchedule("10, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("20, 0, 1, 1, ?"),
	})
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "b",
		Start: jsontypes.MustParseSchedule("12, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("25, 0, 1, 1, ?"),
	})
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "c",
		Start: jsontypes.MustParseSchedule("5, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("28, 0, 1, 1, ?"),
	})
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "d",
		Start: jsontypes.MustParseSchedule("22, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("30, 0, 1, 1, ?"),
	})
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "e",
		Start: jsontypes.MustParseSchedule("29, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("35, 0, 1, 1, ?"),
	})
	cfg.Config.Modes = append(cfg.Config.Modes, config.ModeOption{
		Name:  "f",
		Start: jsontypes.MustParseSchedule("40, 0, 1, 1, ?"),
		End:   jsontypes.MustParseSchedule("45, 0, 1, 1, ?"),
	})

	tests := []struct {
		name     string
		now      int
		wantMode string
		wantWake time.Duration
	}{
		{"before all", 0, "default", 5 * time.Minute},
		{"c start", 5, "c", 5 * time.Minute},
		{"after c start", 9, "c", 1 * time.Minute},
		{"after a start", 11, "a", 1 * time.Minute},
		{"after b start", 13, "a", 7 * time.Minute},
		{"after a end", 21, "b", 1 * time.Minute},
		{"after d start", 23, "b", 2 * time.Minute},
		{"after b end", 26, "c", 2 * time.Minute},
		{"c end", 28, "d", 1 * time.Minute},
		{"after e start", 30, "e", 5 * time.Minute},
		{"after e end", 36, "default", 4 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2023, 1, 1, 0, tt.now, 0, 0, time.UTC)
			gotMode, gotWake := activeMode(now, cfg)
			if gotMode.Name != tt.wantMode {
				t.Errorf("activeMode() mode got = %v, want %v", gotMode.Name, tt.wantMode)
			}
			if gotWake != tt.wantWake {
				t.Errorf("activeMode() wake got = %v, want %v", gotWake, tt.wantWake)
			}
		})
	}
}

func testReadState(start time.Time, now time.Time) *ReadState {
	rs := NewReadState(start)
	rs.Config.Now = func() time.Time {
		return now
	}
	return rs
}

func asPtr[T any](v T) *T {
	return &v
}
