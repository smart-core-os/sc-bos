// Package fixer provides the core execution framework for code transformations.
package fixer

import (
	"fmt"
	"os/exec"
	"strings"
)

// Context provides the execution context for code transformation steps.
type Context struct {
	RootDir string
	Config
}

// Info prints progress information (shown by default).
func (c *Context) Info(format string, args ...interface{}) {
	if !c.Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// Verbose prints detailed information (shown with -verbose).
func (c *Context) Verbose(format string, args ...interface{}) {
	if c.VerboseMode {
		fmt.Printf(format+"\n", args...)
	}
}

// Fix represents a code transformation step.
type Fix struct {
	ID   string // Short identifier used for command-line selection
	Desc string // Human-readable description
	Run  func(*Context) (int, error)
}

type Config struct {
	Quiet       bool
	VerboseMode bool
	DryRun      bool
}

func Run(cfg Config, fixes []Fix) (int, error) {
	rootDir, err := findRepoRoot()
	if err != nil {
		return 0, fmt.Errorf("finding repository root: %w", err)
	}

	ctx := &Context{
		RootDir: rootDir,
		Config:  cfg,
	}

	totalChanges := 0
	for _, f := range fixes {
		ctx.Info("Running fix: %s", f.Desc)
		changes, err := f.Run(ctx)
		if err != nil {
			return totalChanges, fmt.Errorf("%s failed: %w", f.ID, err)
		}
		totalChanges += changes
		if changes > 0 {
			ctx.Verbose("  Applied %d change(s)", changes)
		}
	}

	return totalChanges, nil
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
