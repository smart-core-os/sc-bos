package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A missing file is not an error: the built-in defaults are returned.
func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(Default(), cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

// A present-but-invalid file is an error.
func TestLoad_InvalidIsError(t *testing.T) {
	for name, content := range map[string]string{
		"malformed json":            "{not json",
		"unknown field":             `{"nope": 1}`,
		"unknown nested field":      `{"podman": {"nope": 1}}`,
		"flat image repo":           `{"imageRepo": "localhost/foo"}`,
		"bad duration":              `{"commitDeadline": "soon"}`,
		"zero deadline":             `{"commitDeadline": "0s"}`,
		"empty socket":              `{"socket": ""}`,
		"empty state dir":           `{"stateDir": ""}`,
		"empty installer":           `{"installer": ""}`,
		"unknown installer":         `{"installer": "docker"}`,
		"unknown socket group":      `{"socketGroup": "sc-bos-no-such-group"}`,
		"empty image repo":          `{"installer": "podman", "podman": {"imageRepo": ""}}`,
		"empty unit":                `{"installer": "podman", "podman": {"unit": ""}}`,
		"image repo space":          `{"installer": "podman", "podman": {"imageRepo": "bad repo"}}`,
		"image repo punctuation":    `{"installer": "podman", "podman": {"imageRepo": "foo;bar"}}`,
		"unit slash":                `{"installer": "podman", "podman": {"unit": "bad/unit"}}`,
		"unit space":                `{"installer": "podman", "podman": {"unit": "bad unit"}}`,
		"unit punctuation":          `{"installer": "podman", "podman": {"unit": "bad!unit"}}`,
		"empty rc root":             `{"installer": "freebsdrc", "freebsdrc": {"root": ""}}`,
		"relative rc root":          `{"installer": "freebsdrc", "freebsdrc": {"root": "releases"}}`,
		"empty rc service":          `{"installer": "freebsdrc", "freebsdrc": {"service": ""}}`,
		"rc service hyphen":         `{"installer": "freebsdrc", "freebsdrc": {"service": "sc-bos"}}`,
		"rc service space":          `{"installer": "freebsdrc", "freebsdrc": {"service": "sc bos"}}`,
		"rc service shell metachar": `{"installer": "freebsdrc", "freebsdrc": {"service": "sc_bos;reboot"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, content)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

// Only the selected installer's block is checked, so the other may hold values it would reject.
func TestLoad_ChecksOnlySelectedInstaller(t *testing.T) {
	for name, content := range map[string]string{
		"bad podman under freebsdrc": `{"installer": "freebsdrc", "podman": {"imageRepo": "bad repo", "unit": ""}}`,
		"bad freebsdrc under podman": `{"installer": "podman", "freebsdrc": {"root": "releases", "service": "sc-bos"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, content)); err != nil {
				t.Errorf("want nil error, got %v", err)
			}
		})
	}
}

// Registry hosts with ports, multi-component repos, and instance/typed units are accepted.
func TestLoad_AcceptsValidImageRepoAndUnit(t *testing.T) {
	for name, podman := range map[string]string{
		"repo host and port": `{"imageRepo": "registry.example.com:5000/foo/bar"}`,
		"repo well-known":    `{"imageRepo": "docker.io/library/nginx"}`,
		"repo bare name":     `{"imageRepo": "bos"}`,
		"unit instance":      `{"unit": "getty@tty1.service"}`,
		"unit typed":         `{"unit": "sc-bos.service"}`,
	} {
		t.Run(name, func(t *testing.T) {
			content := fmt.Sprintf(`{"installer": "podman", "podman": %s}`, podman)
			if _, err := Load(writeConfig(t, content)); err != nil {
				t.Errorf("want nil error, got %v", err)
			}
		})
	}
}

// socketGroup accepts a group name or a numeric GID that resolves on this machine.
func TestLoad_SocketGroup(t *testing.T) {
	gid := strconv.Itoa(os.Getgid())
	g, err := user.LookupGroupId(gid)
	if err != nil {
		t.Skipf("cannot look up the current group: %v", err)
	}
	for name, group := range map[string]string{
		"name": g.Name,
		"gid":  gid,
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := Load(writeConfig(t, fmt.Sprintf(`{"socketGroup": %q}`, group)))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SocketGroup != group {
				t.Errorf("SocketGroup: want %q, got %q", group, cfg.SocketGroup)
			}
			got, err := LookupGroup(group)
			if err != nil {
				t.Fatal(err)
			}
			if got != os.Getgid() {
				t.Errorf("LookupGroup(%q): want %d, got %d", group, os.Getgid(), got)
			}
		})
	}
}

// Options the file omits keep their defaults; only the ones it sets are overlaid, including within an
// installer block.
func TestLoad_PartialOverlaysDefaults(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"podman": {"unit": "custom-bos"}, "commitDeadline": "30s"}`))
	if err != nil {
		t.Fatal(err)
	}

	want := Default()
	want.Podman.Unit = "custom-bos"
	want.CommitDeadline = jsontypes.Duration{Duration: 30 * time.Second}
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

// A full file sets every option.
func TestLoad_FullConfig(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{
		"socket": "/tmp/s.sock",
		"stateDir": "/tmp/state",
		"commitDeadline": "90s",
		"allowInsecureDownloads": true,
		"installer": "freebsdrc",
		"podman": {
			"imageRepo": "localhost/foo/bar",
			"unit": "foo"
		},
		"freebsdrc": {
			"root": "/opt/bos/releases",
			"service": "bos"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	want := Config{
		Socket:                 "/tmp/s.sock",
		StateDir:               "/tmp/state",
		CommitDeadline:         jsontypes.Duration{Duration: 90 * time.Second},
		AllowInsecureDownloads: true,
		Installer:              InstallerFreeBSDRC,
		Podman:                 Podman{ImageRepo: "localhost/foo/bar", Unit: "foo"},
		FreeBSDRC:              FreeBSDRC{Root: "/opt/bos/releases", Service: "bos"},
	}
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

// allowInsecureDownloads defaults to false and is overlaid only when the file sets it.
func TestLoad_AllowInsecureDownloads(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"stateDir": "/tmp/state"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowInsecureDownloads {
		t.Error("AllowInsecureDownloads: want false when omitted, got true")
	}

	cfg, err = Load(writeConfig(t, `{"allowInsecureDownloads": true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowInsecureDownloads {
		t.Error("AllowInsecureDownloads: want true, got false")
	}
}
