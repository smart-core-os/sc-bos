// Command genproto generates code from proto definitions.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/goproto"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/toolchain"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/uiproto"
)

// stringSliceFlag allows flags to be specified multiple times or as comma-separated values.
// Supports both -flag a,b and -flag a -flag b styles.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	// Split by comma to support -flag a,b style
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
		debug     = flag.Bool("vv", false, "debug output (implies -v)")
		dryRun    = flag.Bool("dry-run", false, "dry run mode - don't execute commands")
		onlySteps stringSliceFlag
		skipSteps stringSliceFlag
		listSteps = flag.Bool("list", false, "list available steps and exit")
	)

	flag.Var(&onlySteps, "only", "run only specified steps by ID (can be comma-separated or specified multiple times)")
	flag.Var(&skipSteps, "skip", "skip specified steps by ID (can be comma-separated or specified multiple times)")
	flag.Parse()

	allSteps := []generator.Step{
		toolchain.Step,
		goproto.Step,
		uiproto.Step,
	}

	// List steps if requested
	if *listSteps {
		fmt.Println("Available steps:")
		for _, step := range allSteps {
			fmt.Printf("  %s - %s\n", step.ID, step.Desc)
		}
		return
	}

	// Validate that only one of -only or -skip is used
	if len(onlySteps) > 0 && len(skipSteps) > 0 {
		fmt.Fprintf(os.Stderr, "Error: cannot use both -only and -skip flags together\n")
		os.Exit(1)
	}

	// Filter steps based on flags
	steps := filterSteps(allSteps, onlySteps, skipSteps)

	if len(steps) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no steps to run\n")
		os.Exit(1)
	}

	// Determine log level
	logLevel := generator.LogLevelInfo // default
	if *quiet {
		logLevel = generator.LogLevelQuiet
	} else if *debug {
		logLevel = generator.LogLevelDebug
	} else if *verbose {
		logLevel = generator.LogLevelVerbose
	}

	cfg := generator.Config{
		LogLevel: logLevel,
		DryRun:   *dryRun,
	}

	if err := generator.Run(cfg, steps); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// filterSteps returns the steps to run based on only/skip flags.
// Steps are matched by their ID field for easier command-line usage.
func filterSteps(allSteps []generator.Step, only, skip []string) []generator.Step {
	// If only specific steps are requested
	if len(only) > 0 {
		var result []generator.Step
		for _, stepID := range only {
			found := false
			for _, step := range allSteps {
				if step.ID == stepID {
					result = append(result, step)
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "Warning: step with ID %q not found\n", stepID)
			}
		}
		return result
	}

	// If skipping specific steps
	if len(skip) > 0 {
		skipMap := make(map[string]bool)
		for _, s := range skip {
			skipMap[s] = true
		}

		var result []generator.Step
		for _, step := range allSteps {
			if !skipMap[step.ID] {
				result = append(result, step)
			}
		}
		return result
	}

	// Return all steps if no filter specified
	return allSteps
}
