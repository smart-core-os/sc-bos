// Package toolchain provides access to commands and tools for working with proto files.
package toolchain

import (
	"fmt"
	"os"
	"os/exec"
)

// RunProtomod executes protomod with the given arguments.
// If workDir is empty, the command runs from the current working directory.
func RunProtomod(workDir string, args ...string) error {
	cmdArgs := append([]string{"tool", "protomod"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running protomod: %w", err)
	}

	return nil
}
