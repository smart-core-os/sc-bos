package toolchain

import (
	"fmt"
	"os"
	"os/exec"
)

// RunProtoc executes protoc directly with the given arguments.
func RunProtoc(workDir string, args ...string) error {
	cmd := exec.Command("protoc", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running protoc: %w", err)
	}
	return nil
}
