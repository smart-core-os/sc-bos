package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// A missing file is not an error: the built-in defaults are returned.
func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	require.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

// A present-but-invalid file is an error.
func TestLoad_InvalidIsError(t *testing.T) {
	for name, content := range map[string]string{
		"malformed json": "{not json",
		"unknown field":  `{"nope": 1}`,
		"bad duration":   `{"commitDeadline": "soon"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(writeConfig(t, content))
			assert.Error(t, err)
		})
	}
}

// Options the file omits keep their defaults; only the ones it sets are overlaid.
func TestLoad_PartialOverlaysDefaults(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"unit": "custom-bos", "commitDeadline": "30s"}`))
	require.NoError(t, err)

	assert.Equal(t, "custom-bos", cfg.Unit)
	assert.Equal(t, 30*time.Second, cfg.CommitDeadline)
	assert.Equal(t, Default().Socket, cfg.Socket)
	assert.Equal(t, Default().StateDir, cfg.StateDir)
	assert.Equal(t, Default().ImageRepo, cfg.ImageRepo)
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
	require.NoError(t, err)

	assert.Equal(t, Config{
		Socket:                 "/tmp/s.sock",
		StateDir:               "/tmp/state",
		ImageRepo:              "localhost/foo/bar",
		Unit:                   "foo",
		CommitDeadline:         90 * time.Second,
		AllowInsecureDownloads: true,
	}, cfg)
}

// allowInsecureDownloads defaults to false and is overlaid only when the file sets it.
func TestLoad_AllowInsecureDownloads(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"unit": "foo"}`))
	require.NoError(t, err)
	assert.False(t, cfg.AllowInsecureDownloads, "defaults to false when omitted")

	cfg, err = Load(writeConfig(t, `{"allowInsecureDownloads": true}`))
	require.NoError(t, err)
	assert.True(t, cfg.AllowInsecureDownloads)
}
