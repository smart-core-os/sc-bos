package goproto

import (
	"testing"
)

func TestGenerator_Has(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
		flag Generator
		want bool
	}{
		{
			name: "empty generator has no flags",
			gen:  0,
			flag: GenRouter,
			want: false,
		},
		{
			name: "router only has router",
			gen:  GenRouter,
			flag: GenRouter,
			want: true,
		},
		{
			name: "router only does not have wrapper",
			gen:  GenRouter,
			flag: GenWrapper,
			want: false,
		},
		{
			name: "wrapper only has wrapper",
			gen:  GenWrapper,
			flag: GenWrapper,
			want: true,
		},
		{
			name: "wrapper only does not have router",
			gen:  GenWrapper,
			flag: GenRouter,
			want: false,
		},
		{
			name: "combined has router",
			gen:  GenRouter | GenWrapper,
			flag: GenRouter,
			want: true,
		},
		{
			name: "combined has wrapper",
			gen:  GenRouter | GenWrapper,
			flag: GenWrapper,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.Has(tt.flag); got != tt.want {
				t.Errorf("Generator.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_String(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
		want string
	}{
		{
			name: "empty generator",
			gen:  0,
			want: "basic",
		},
		{
			name: "router only",
			gen:  GenRouter,
			want: "router",
		},
		{
			name: "wrapper only",
			gen:  GenWrapper,
			want: "wrapper",
		},
		{
			name: "router and wrapper",
			gen:  GenRouter | GenWrapper,
			want: "router+wrapper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.String(); got != tt.want {
				t.Errorf("Generator.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupByGeneratorSet(t *testing.T) {
	tests := []struct {
		name           string
		fileGenerators map[string]Generator
		wantGroups     map[Generator]int // map of generator to expected file count
	}{
		{
			name:           "empty input",
			fileGenerators: map[string]Generator{},
			wantGroups:     map[Generator]int{},
		},
		{
			name: "single group",
			fileGenerators: map[string]Generator{
				"file1.proto": GenRouter | GenWrapper,
				"file2.proto": GenRouter | GenWrapper,
			},
			wantGroups: map[Generator]int{
				GenRouter | GenWrapper: 2,
			},
		},
		{
			name: "multiple groups",
			fileGenerators: map[string]Generator{
				"file1.proto": GenRouter | GenWrapper,
				"file2.proto": GenWrapper,
				"file3.proto": 0,
				"file4.proto": GenRouter | GenWrapper,
			},
			wantGroups: map[Generator]int{
				GenRouter | GenWrapper: 2,
				GenWrapper:             1,
				0:                      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupByGeneratorSet(tt.fileGenerators)

			// Check bucket count
			if len(got) != len(tt.wantGroups) {
				t.Errorf("groupByGeneratorSet() returned %d groups, want %d", len(got), len(tt.wantGroups))
			}

			// Check file count per bucket
			for gen, wantCount := range tt.wantGroups {
				gotCount := len(got[gen])
				if gotCount != wantCount {
					t.Errorf("group %s has %d files, want %d", gen, gotCount, wantCount)
				}
			}
		})
	}
}
