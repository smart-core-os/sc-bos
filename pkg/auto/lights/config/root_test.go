package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestProcess(t *testing.T) {
	tests := []struct {
		name string
		dd   *DaylightDimming
		want []LevelThreshold
	}{
		{
			name: "empty",
			dd:   nil,
			want: nil,
		},
		{
			name: "has thresholds",
			dd: &DaylightDimming{
				// this is invalid, but ignored
				Segments: &ThresholdSegments{
					MaxLevel: 2500,
				},
				Thresholds: []LevelThreshold{
					{BelowLux: 500, LevelPercent: 43},
					{BelowLux: 250, LevelPercent: 50},
				},
			},
			want: []LevelThreshold{
				{BelowLux: 500, LevelPercent: 43},
				{BelowLux: 250, LevelPercent: 50},
			},
		},
		{
			name: "generates thresholds",
			dd: &DaylightDimming{
				// this is invalid, but ignored
				Segments: &ThresholdSegments{
					Steps:  2,
					MinLux: 100,
					MaxLux: 1000,
				},
			},
			want: []LevelThreshold{
				{BelowLux: 1000, LevelPercent: 0},
				{BelowLux: 550, LevelPercent: 100},
			},
		},
		{
			name: "custom levels",
			dd: &DaylightDimming{
				// this is invalid, but ignored
				Segments: &ThresholdSegments{
					Steps:    2,
					MinLux:   100,
					MaxLux:   1000,
					MinLevel: 25,
					MaxLevel: 75,
				},
			},
			want: []LevelThreshold{
				{BelowLux: 1000, LevelPercent: 25},
				{BelowLux: 550, LevelPercent: 75},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dd.process()
			if err != nil {
				t.Fatalf("%s", err)
			}
			var got []LevelThreshold
			if tt.dd != nil {
				got = tt.dd.Thresholds
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("thresholds (-want,+got)\n%s", diff)
			}
		})
	}
}

func TestRoot_MarshalJSON(t *testing.T) {
	t.Run("empty daylight dimming", func(t *testing.T) {
		root := Root{
			Config: auto.Config{
				Name: "test",
				Type: "lights",
			},
			Mode: Mode{
				DaylightDimming: &DaylightDimming{},
			},
		}
		bin, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"name":"test","type":"lights","unoccupiedOffDelay":"0s","daylightDimming":{}}`
		if string(bin) != want {
			t.Fatalf("got %q, want %q", string(bin), want)
		}
	})
}

func TestRoot_modeDefaults(t *testing.T) {
	t.Run("from json", func(t *testing.T) {
		raw := `{
	"name":"test",
	"type":"lights",
	"daylightDimming": {
		"thresholds": [
			{
				"belowLux": 29713,
				"levelPercent": 1
			}
		]
	},
	"unoccupiedOffDelay": "30s",
	"modes": [
		{"name":"short","unoccupiedOffDelay":"1m","onLevelPercent":80},
		{"name":"long","unoccupiedOffDelay":"2m","daylightDimming":{}},
		{"name":"inherit","unoccupiedOffDelay":"0s","daylightDimming":{}}
	]
}`
		root, err := Read([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		var eighty float32 = 80
		want := []ModeOption{
			{
				Name: "short",
				Mode: Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: 1 * time.Minute},
					OnLevelPercent:     &eighty,
					DaylightDimming: &DaylightDimming{
						Thresholds: []LevelThreshold{
							{BelowLux: 29713, LevelPercent: 1},
						},
					},
				},
			},
			{
				Name: "long",
				Mode: Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: 2 * time.Minute},
					DaylightDimming:    &DaylightDimming{},
				},
			},
			{
				Name: "inherit",
				Mode: Mode{
					UnoccupiedOffDelay: jsontypes.Duration{Duration: 30 * time.Second},
					DaylightDimming:    &DaylightDimming{},
				},
			},
		}
		if diff := cmp.Diff(want, root.Modes); diff != "" {
			t.Fatalf("root (-want,+got)\n%s", diff)
		}
	})
}
