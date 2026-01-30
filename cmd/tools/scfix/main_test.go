package main

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

func TestFilterFixes(t *testing.T) {
	// Create test fixtures
	testFixes := []fix{
		{Fix: fixer.Fix{ID: "fix1", Desc: "Fix 1"}, Enabled: true},
		{Fix: fixer.Fix{ID: "fix2", Desc: "Fix 2"}, Enabled: true},
		{Fix: fixer.Fix{ID: "fix3", Desc: "Fix 3"}, Enabled: false},
		{Fix: fixer.Fix{ID: "fix4", Desc: "Fix 4"}, Enabled: false},
	}

	tests := []struct {
		name     string
		only     []string
		skip     []string
		fixtures []fix
		wantIDs  []string
	}{
		{
			name:     "no flags - returns only default enabled fixes",
			only:     nil,
			skip:     nil,
			fixtures: testFixes,
			wantIDs:  []string{"fix1", "fix2"},
		},
		{
			name:     "only flag - returns only specified fixes regardless of default",
			only:     []string{"fix3", "fix1"},
			skip:     nil,
			fixtures: testFixes,
			wantIDs:  []string{"fix3", "fix1"},
		},
		{
			name:     "only flag with disabled fix",
			only:     []string{"fix4"},
			skip:     nil,
			fixtures: testFixes,
			wantIDs:  []string{"fix4"},
		},
		{
			name:     "skip flag - skips from default enabled fixes only",
			only:     nil,
			skip:     []string{"fix1"},
			fixtures: testFixes,
			wantIDs:  []string{"fix2"},
		},
		{
			name:     "skip flag - does not include disabled fixes",
			only:     nil,
			skip:     []string{"fix1"},
			fixtures: testFixes,
			wantIDs:  []string{"fix2"}, // fix3 and fix4 should NOT be included
		},
		{
			name:     "skip all default fixes - returns empty",
			only:     nil,
			skip:     []string{"fix1", "fix2"},
			fixtures: testFixes,
			wantIDs:  []string{},
		},
		{
			name:     "skip non-existent fix - returns all default fixes",
			only:     nil,
			skip:     []string{"nonexistent"},
			fixtures: testFixes,
			wantIDs:  []string{"fix1", "fix2"},
		},
		{
			name:     "only with all fixes",
			only:     []string{"fix1", "fix2", "fix3", "fix4"},
			skip:     nil,
			fixtures: testFixes,
			wantIDs:  []string{"fix1", "fix2", "fix3", "fix4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily replace global allFixes with test fixtures
			oldAllFixes := allFixes
			allFixes = tt.fixtures
			defer func() { allFixes = oldAllFixes }()

			got := filterFixes(tt.only, tt.skip)
			gotIDs := make([]string, len(got))
			for i, fix := range got {
				gotIDs[i] = fix.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("filterFixes() returned %d fixes, want %d\nGot: %v\nWant: %v",
					len(gotIDs), len(tt.wantIDs), gotIDs, tt.wantIDs)
				return
			}

			// Check that all expected IDs are present (order doesn't matter)
			gotSet := make(map[string]bool)
			for _, id := range gotIDs {
				gotSet[id] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotSet[wantID] {
					t.Errorf("filterFixes() missing expected fix %q\nGot: %v\nWant: %v",
						wantID, gotIDs, tt.wantIDs)
				}
			}
		})
	}
}

func TestFilterByIDs(t *testing.T) {
	testFixes := []fix{
		{Fix: fixer.Fix{ID: "fix1", Desc: "Fix 1"}, Enabled: true},
		{Fix: fixer.Fix{ID: "fix2", Desc: "Fix 2"}, Enabled: false},
		{Fix: fixer.Fix{ID: "fix3", Desc: "Fix 3"}, Enabled: true},
	}

	tests := []struct {
		name     string
		ids      []string
		include  bool
		fixtures []fix
		wantIDs  []string
	}{
		{
			name:     "include specified IDs",
			ids:      []string{"fix1", "fix3"},
			include:  true,
			fixtures: testFixes,
			wantIDs:  []string{"fix1", "fix3"},
		},
		{
			name:     "exclude specified IDs",
			ids:      []string{"fix1"},
			include:  false,
			fixtures: testFixes,
			wantIDs:  []string{"fix2", "fix3"},
		},
		{
			name:     "include single ID",
			ids:      []string{"fix2"},
			include:  true,
			fixtures: testFixes,
			wantIDs:  []string{"fix2"},
		},
		{
			name:     "include non-existent ID - returns nothing",
			ids:      []string{"nonexistent"},
			include:  true,
			fixtures: testFixes,
			wantIDs:  []string{},
		},
		{
			name:     "exclude all IDs",
			ids:      []string{"fix1", "fix2", "fix3"},
			include:  false,
			fixtures: testFixes,
			wantIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAllFixes := allFixes
			allFixes = tt.fixtures
			defer func() { allFixes = oldAllFixes }()

			// Convert fixtures to []fixer.Fix for filterByIDs
			fixes := make([]fixer.Fix, len(tt.fixtures))
			for i, f := range tt.fixtures {
				fixes[i] = f.Fix
			}

			got := filterByIDs(fixes, tt.ids, tt.include)
			gotIDs := make([]string, len(got))
			for i, fix := range got {
				gotIDs[i] = fix.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("filterByIDs() returned %d fixes, want %d\nGot: %v\nWant: %v",
					len(gotIDs), len(tt.wantIDs), gotIDs, tt.wantIDs)
				return
			}

			gotSet := make(map[string]bool)
			for _, id := range gotIDs {
				gotSet[id] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotSet[wantID] {
					t.Errorf("filterByIDs() missing expected fix %q\nGot: %v\nWant: %v",
						wantID, gotIDs, tt.wantIDs)
				}
			}
		})
	}
}
