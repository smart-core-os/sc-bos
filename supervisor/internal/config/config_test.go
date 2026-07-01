package config

import (
	"os"
	"path/filepath"
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
		"malformed json": "{not json",
		"unknown field":  `{"nope": 1}`,
		"bad duration":   `{"commitDeadline": "soon"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, content)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

// Options the file omits keep their defaults; only the ones it sets are overlaid.
func TestLoad_PartialOverlaysDefaults(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"unit": "custom-bos", "commitDeadline": "30s"}`))
	if err != nil {
		t.Fatal(err)
	}

	want := Default()
	want.Unit = "custom-bos"
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
		"imageRepo": "localhost/foo/bar",
		"unit": "foo",
		"commitDeadline": "90s",
		"allowInsecureDownloads": true
	}`))
	if err != nil {
		t.Fatal(err)
	}

	want := Config{
		Socket:                 "/tmp/s.sock",
		StateDir:               "/tmp/state",
		ImageRepo:              "localhost/foo/bar",
		Unit:                   "foo",
		CommitDeadline:         jsontypes.Duration{Duration: 90 * time.Second},
		AllowInsecureDownloads: true,
	}
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

// allowInsecureDownloads defaults to false and is overlaid only when the file sets it.
func TestLoad_AllowInsecureDownloads(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"unit": "foo"}`))
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
