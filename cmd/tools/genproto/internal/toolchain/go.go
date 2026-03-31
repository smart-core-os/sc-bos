package toolchain

import (
	"os/exec"
	"strings"
)

// GetGoToolPath returns the full path to the specified Go tool, like go tool -n {tool}.
func GetGoToolPath(tool string) (string, error) {
	output, err := exec.Command("go", "tool", "-n", tool).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
