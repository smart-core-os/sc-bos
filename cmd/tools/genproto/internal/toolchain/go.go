package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// BuildPlugin builds the named Go package to a temporary directory and returns the binary path.
// The caller is responsible for calling the returned cleanup function to remove the temp dir.
// Use this for plugins that cannot be declared as go.mod tool directives without pulling in
// unwanted transitive dependencies.
func BuildPlugin(pkg string) (path string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "genproto-plugin-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmp) }

	name := filepath.Base(pkg)
	outPath := filepath.Join(tmp, name)
	cmd := exec.Command("go", "build", "-o", outPath, pkg)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("building %s: %w\n%s", pkg, err, out)
	}
	return outPath, cleanup, nil
}
