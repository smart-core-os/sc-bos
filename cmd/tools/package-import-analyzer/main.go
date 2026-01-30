// Command package-import-analyzer analyzes GitHub repositories in an organization
// to find which packages from this repository are imported by other repos.
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Global flags
	verbose     = flag.Bool("verbose", false, "Enable verbose output")
	veryVerbose = flag.Bool("vv", false, "Enable very verbose output (includes cache hits/misses and API calls)")
)

func main() {
	flag.Parse()
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "scan":
		scanCmd.Parse(os.Args[2:])
		runScan()
	case "analyze":
		analyzeCmd.Parse(os.Args[2:])
		runAnalyze()
	default:
		printUsage()
		os.Exit(1)
	}
}
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  scan     Scan GitHub organization for package imports\n")
	fmt.Fprintf(os.Stderr, "  analyze  Analyze scan results\n\n")
	fmt.Fprintf(os.Stderr, "Use '%s <command> -h' for more information about a command.\n", os.Args[0])
}
