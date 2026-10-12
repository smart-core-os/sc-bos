//go:build unix && !aix

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// testHost is a releases root and state folder named by a config file.
type testHost struct {
	config   string
	root     string
	stateDir string
}

func newTestHost(t *testing.T, installer string) testHost {
	t.Helper()
	dir := t.TempDir()
	h := testHost{
		config:   filepath.Join(dir, "config.json"),
		root:     filepath.Join(dir, "releases"),
		stateDir: filepath.Join(dir, "state"),
	}
	cfg := map[string]any{
		"installer": installer,
		"stateDir":  h.stateDir,
		"freebsdrc": map[string]string{"root": h.root, "service": "sc_bos"},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.config, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return h
}

// writeRelease writes a release tarball for version on this host's platform and returns its path.
func writeRelease(t *testing.T, version string) string {
	t.Helper()
	manifest := fmt.Sprintf(`{"schemaVersion": 1, "version": %q, "os": %q, "arch": %q, "entrypoint": "bin/sc-bos", "ui": {"ops": "ui/ops"}}`,
		version, runtime.GOOS, runtime.GOARCH)
	entries := []struct {
		name string
		mode int64
		body string
	}{
		{"manifest.json", 0o644, manifest},
		{"bin/", 0o755, ""},
		{"bin/sc-bos", 0o755, "binary " + version},
		{"ui/", 0o755, ""},
		{"ui/ops/", 0o755, ""},
		{"ui/ops/index.html", 0o644, "<html></html>"},
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: tar.TypeReg, Mode: e.mode, Size: int64(len(e.body))}
		if strings.HasSuffix(e.name, "/") {
			hdr.Typeflag = tar.TypeDir
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), version+".tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runInstall runs install-release with args and returns what it wrote to stdout. It fails the test if
// the command writes to stderr.
func runInstall(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	err := installRelease(args, &out, &errOut, zap.NewNop())
	if errOut.Len() > 0 {
		t.Errorf("unexpected stderr: %s", errOut.String())
	}
	return out.String(), err
}

func assertCurrent(t *testing.T, root, want string) {
	t.Helper()
	got, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("current -> %q, want %q", got, want)
	}
}

// A fresh root gets the release and a current link, and a second run with the same tarball changes nothing.
func TestInstallRelease_FreshThenRepeat(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	tarball := writeRelease(t, "v2026.10.1")

	out, err := runInstall(t, "-config", h.config, tarball)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if want := "installed v2026.10.1; current -> v2026.10.1 (was: none)\n"; out != want {
		t.Errorf("first install output = %q, want %q", out, want)
	}
	assertCurrent(t, h.root, "v2026.10.1")

	out, err = runInstall(t, "-config", h.config, tarball)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if want := "v2026.10.1 is already current\n"; out != want {
		t.Errorf("second install output = %q, want %q", out, want)
	}
	assertCurrent(t, h.root, "v2026.10.1")
	if _, err := os.Lstat(filepath.Join(h.root, "previous")); !os.IsNotExist(err) {
		t.Errorf("previous link exists (err %v), want none", err)
	}
}

func TestInstallRelease_Upgrade(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	if _, err := runInstall(t, "-config", h.config, writeRelease(t, "v1")); err != nil {
		t.Fatal(err)
	}
	out, err := runInstall(t, "-config", h.config, writeRelease(t, "v2"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "installed v2; current -> v2 (was: v1)\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
	assertCurrent(t, h.root, "v2")
}

func TestInstallRelease_PodmanInstaller(t *testing.T) {
	h := newTestHost(t, "podman")
	_, err := runInstall(t, "-config", h.config, writeRelease(t, "v1"))
	if err == nil || !strings.Contains(err.Error(), `needs the "freebsdrc" installer`) {
		t.Fatalf("error = %v, want an error naming the freebsdrc installer", err)
	}
	if _, err := os.Stat(h.root); !os.IsNotExist(err) {
		t.Errorf("releases root exists (err %v), want it untouched", err)
	}
}

// An unfinished Supervisor update stops the install before anything changes.
func TestInstallRelease_UnfinishedUpdate(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	if err := os.MkdirAll(h.stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	state := []byte(`{"target": {"version": "v3", "url": "https://example.com/v3.tar.gz", "checksum": "sha256:00"}}`)
	stateFile := filepath.Join(h.stateDir, "state.json")
	if err := os.WriteFile(stateFile, state, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := runInstall(t, "-config", h.config, writeRelease(t, "v1"))
	if err == nil || !strings.Contains(err.Error(), "unfinished update to v3") {
		t.Fatalf("error = %v, want an unfinished update error", err)
	}
	if _, err := os.Stat(h.root); !os.IsNotExist(err) {
		t.Errorf("releases root exists (err %v), want it untouched", err)
	}
	got, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, state) {
		t.Errorf("state file changed to %s", got)
	}
}

// After a finished update, installing a different version warns that the Supervisor's committed version is
// out of date. Installing the committed version doesn't.
func TestInstallRelease_SettledCommit(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	if err := os.MkdirAll(h.stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	state := `{"target": {"version": "v1", "url": "https://example.com/v1.tar.gz", "checksum": "sha256:00"}, "committed": "v1", "finishTime": "2026-10-01T10:00:00Z"}`
	if err := os.WriteFile(filepath.Join(h.stateDir, "state.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := runInstall(t, "-config", h.config, writeRelease(t, "v1")); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := installRelease([]string{"-config", h.config, writeRelease(t, "v2")}, &out, &errOut, zap.NewNop()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "still records v1 as the committed version") {
		t.Errorf("stderr = %q, want a warning naming v1", errOut.String())
	}
	assertCurrent(t, h.root, "v2")
}

func TestInstallRelease_Args(t *testing.T) {
	h := newTestHost(t, "freebsdrc")
	for _, args := range [][]string{
		{"-config", h.config},
		{"-config", h.config, "a.tar.gz", "b.tar.gz"},
	} {
		if _, err := runInstall(t, args...); err == nil || !strings.Contains(err.Error(), "exactly one release tarball") {
			t.Errorf("installRelease(%q) error = %v, want a usage error", args, err)
		}
	}
}
