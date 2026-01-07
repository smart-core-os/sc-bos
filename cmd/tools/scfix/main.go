// Command scfix provides a CLI tool for applying code fixes to sc-bos and dependent repos.
// It applies code transformations to update deprecated patterns to modern equivalents.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/goprotoimports"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/historyimports"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/optclients"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/protogopkg"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/protov1"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/protov1go"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/protov1js"
	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/wrap"
)

// fix associates a fixer.Fix with whether it's enabled by default.
type fix struct {
	Fix     fixer.Fix
	Enabled bool // whether this fix runs by default (when no -only flag specified)
}

// allFixes contains all available fixes and their default enabled state.
var allFixes = []fix{
	{Fix: optclients.Fix, Enabled: true},
	{Fix: historyimports.Fix, Enabled: true},
	{Fix: wrap.Fix, Enabled: false},
	{Fix: protov1.Fix, Enabled: false},
	{Fix: protov1go.Fix, Enabled: false},
	{Fix: protov1js.Fix, Enabled: false},
	{Fix: protogopkg.Fix, Enabled: false},
	{Fix: goprotoimports.Fix, Enabled: false},
}

// stringSliceFlag allows flags to be specified multiple times or as comma-separated values.
// Supports both -flag a,b and -flag a -flag b styles.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			*f = append(*f, trimmed)
		}
	}
	return nil
}

func main() {
	var (
		quiet     = flag.Bool("q", false, "quiet mode - only show errors")
		verbose   = flag.Bool("v", false, "verbose output")
		dryRun    = flag.Bool("dry-run", false, "dry run mode - don't apply changes")
		onlyFixes stringSliceFlag
		skipFixes stringSliceFlag
		listFixes = flag.Bool("list", false, "list available fixes and exit")
	)

	flag.Var(&onlyFixes, "only", "run only specified fixes by ID (can be comma-separated or specified multiple times)")
	flag.Var(&skipFixes, "skip", "skip specified fixes by ID (can be comma-separated or specified multiple times)")
	flag.Parse()

	if *listFixes {
		fmt.Println("Available fixes:")
		for _, rf := range allFixes {
			fmt.Printf("  %s - %s\n", rf.Fix.ID, rf.Fix.Desc)
		}
		return
	}

	if len(onlyFixes) > 0 && len(skipFixes) > 0 {
		fmt.Fprintf(os.Stderr, "Error: cannot use both -only and -skip flags together\n")
		os.Exit(1)
	}

	fixes := filterFixes(onlyFixes, skipFixes)

	if len(fixes) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no fixes to run\n")
		os.Exit(1)
	}

	cfg := fixer.Config{
		Quiet:       *quiet,
		VerboseMode: *verbose,
		DryRun:      *dryRun,
	}

	totalChanges, err := fixer.Run(cfg, fixes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Printf("Dry run complete. Would have made %d change(s).\n", totalChanges)
	} else if !*quiet {
		fmt.Printf("Successfully applied %d change(s).\n", totalChanges)
	}
}

// filterFixes returns the fixes to run based on only/skip flags.
func filterFixes(only, skip []string) []fixer.Fix {
	// -only: run only specified fixes
	if len(only) > 0 {
		all := make([]fixer.Fix, len(allFixes))
		for i, f := range allFixes {
			all[i] = f.Fix
		}
		return filterByIDs(all, only, true)
	}

	// -skip: run default fixes except those skipped
	if len(skip) > 0 {
		enabled := filterEnabled()
		return filterByIDs(enabled, skip, false)
	}

	// No flags: return only fixes enabled by default
	return filterEnabled()
}

// filterEnabled returns all fixes that are enabled by default.
func filterEnabled() []fixer.Fix {
	var result []fixer.Fix
	for _, f := range allFixes {
		if f.Enabled {
			result = append(result, f.Fix)
		}
	}
	return result
}

func filterByIDs(fixes []fixer.Fix, ids []string, include bool) []fixer.Fix {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	var result []fixer.Fix
	for _, fix := range fixes {
		if include == idSet[fix.ID] {
			result = append(result, fix)
		}
	}
	return result
}
