package generator

import (
	"fmt"
	"os/exec"
	"strings"
)

// LogLevel controls the verbosity of output.
type LogLevel int

const (
	LogLevelQuiet   LogLevel = iota // No output except errors
	LogLevelInfo                    // Progress information (default)
	LogLevelVerbose                 // Detailed information
	LogLevelDebug                   // Debug information
)

// Context provides the execution context for code generation steps.
type Context struct {
	RootDir string
	Config
}

// Info prints progress information (shown by default).
func (c *Context) Info(format string, args ...interface{}) {
	if c.LogLevel >= LogLevelInfo {
		fmt.Printf(format+"\n", args...)
	}
}

// Verbose prints detailed information (shown with -v).
func (c *Context) Verbose(format string, args ...interface{}) {
	if c.LogLevel >= LogLevelVerbose {
		fmt.Printf(format+"\n", args...)
	}
}

// Debug prints debug information (shown with -vv).
func (c *Context) Debug(format string, args ...interface{}) {
	if c.LogLevel >= LogLevelDebug {
		fmt.Printf(format+"\n", args...)
	}
}

// Step represents a code generation step.
type Step struct {
	ID   string // Short identifier used for command-line selection
	Desc string // Human-readable description
	Run  func(*Context) error
}

type Config struct {
	LogLevel LogLevel
	DryRun   bool
}

func Run(cfg Config, steps []Step) error {
	rootDir, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repository root: %w", err)
	}

	ctx := &Context{
		RootDir: rootDir,
		Config:  cfg,
	}

	for _, s := range steps {
		ctx.Info("Running step: %s", s.Desc)
		if err := s.Run(ctx); err != nil {
			return fmt.Errorf("%s failed: %w", s.ID, err)
		}
	}

	return nil
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
