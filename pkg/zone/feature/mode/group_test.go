package mode

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/mode/config"
)

func TestGroup_mergeModeValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Root
		all  map[string]*traits.ModeValues
		one  *traits.ModeValues
	}{
		{
			"renamed",
			config.Root{
				Modes: map[string][]config.Option{
					"bms.occupancy": {
						{
							Name: "yes",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "occupied",
								}},
								{OptionSource: config.OptionSource{
									Devices: []string{"dev03"},
									Mode:    "present",
									Value:   "yup",
								}},
							},
						},
						{
							Name: "no",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "unoccupied",
								}},
								{OptionSource: config.OptionSource{
									Devices: []string{"dev03"},
									Mode:    "present",
									Value:   "nope",
								}},
							},
						},
					},
				},
			},
			map[string]*traits.ModeValues{
				"dev01": {Values: map[string]string{"occupancy": "occupied"}},
				"dev02": {Values: map[string]string{"occupancy": "occupied"}},
				"dev03": {Values: map[string]string{"present": "yup"}},
			},
			&traits.ModeValues{
				Values: map[string]string{"bms.occupancy": "yes"},
			},
		}, {
			"mixed",
			config.Root{
				Modes: map[string][]config.Option{
					"occupancy": {
						{
							Name: "occupied",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "occupied",
								}},
							},
						},
						{
							Name: "unoccupied",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "unoccupied",
								}},
							},
						},
					},
				},
			},
			map[string]*traits.ModeValues{
				"dev01": {Values: map[string]string{"occupancy": "occupied"}},
				"dev02": {Values: map[string]string{"occupancy": "unoccupied"}},
			},
			&traits.ModeValues{
				Values: map[string]string{"occupancy": MixedValue},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{cfg: tt.cfg}
			if diff := cmp.Diff(tt.one, g.mergeModeValues(tt.all), protocmp.Transform()); diff != "" {
				t.Fatalf("mergeModeValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroup_unmergeModeValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Root
		one  *traits.ModeValues
		all  map[string]*traits.ModeValues
	}{
		{
			"renamed",
			config.Root{
				Modes: map[string][]config.Option{
					"bms.occupancy": {
						{
							Name: "yes",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "occupied",
								}},
								{OptionSource: config.OptionSource{
									Devices: []string{"dev03"},
									Mode:    "present",
									Value:   "yup",
								}},
							},
						},
						{
							Name: "no",
							Sources: []config.SourceOrString{
								{OptionSource: config.OptionSource{
									Devices: []string{"dev01", "dev02"},
									Mode:    "occupancy",
									Value:   "unoccupied",
								}},
								{OptionSource: config.OptionSource{
									Devices: []string{"dev03"},
									Mode:    "present",
									Value:   "nope",
								}},
							},
						},
					},
				},
			},
			&traits.ModeValues{
				Values: map[string]string{"bms.occupancy": "yes"},
			},
			map[string]*traits.ModeValues{
				"dev01": {Values: map[string]string{"occupancy": "occupied"}},
				"dev02": {Values: map[string]string{"occupancy": "occupied"}},
				"dev03": {Values: map[string]string{"present": "yup"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{cfg: tt.cfg}
			if diff := cmp.Diff(tt.all, g.unmergeModeValues(tt.one), protocmp.Transform()); diff != "" {
				t.Fatalf("unmergeModeValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
