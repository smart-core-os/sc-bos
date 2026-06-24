// Package config loads the Supervisor's configuration from a JSON file, falling back to built-in
// defaults for a missing file or any option the file omits.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// A rough approximation of rules for these identifiers - excludes invalid characters but doesn't exactly match the
// required structure. Good enough to catch accidental misconfigurations.
var (
	imageRepoPattern = regexp.MustCompile(`^[a-zA-Z0-9._/:-]+$`)
	unitNamePattern  = regexp.MustCompile(`^[a-zA-Z0-9._:@-]+$`)
)

// Config is the Supervisor's runtime configuration. Every option is optional in the config file; an
// omitted one keeps the value from Default.
type Config struct {
	// Socket is the Unix socket path the SupervisorApi listens on.
	Socket string `json:"socket,omitempty"`
	// StateDir holds durable state: the update record at <StateDir>/state.json and downloaded artefacts
	// under <StateDir>/staging.
	StateDir string `json:"stateDir,omitempty"`
	// ImageRepo is the image repository whose :current/:previous tags the installer swaps.
	ImageRepo string `json:"imageRepo,omitempty"`
	// Unit is the systemd unit (and container name) for BOS that the Supervisor restarts.
	Unit string `json:"unit,omitempty"`
	// CommitDeadline bounds how long reconcile waits for BOS to Commit a new version before rolling back.
	// It is a Go duration string, e.g. "2m" or "90s".
	CommitDeadline jsontypes.Duration `json:"commitDeadline,omitempty"`
	// AllowInsecureDownloads permits artefact download URLs that do not use HTTPS. It defaults to false
	// (HTTPS only); enable it only for development against a plain-HTTP update server such as cloudsim.
	AllowInsecureDownloads bool `json:"allowInsecureDownloads,omitempty"`
}

// Default returns the configuration used when no config file is present, and for any option a present
// file omits.
func Default() Config {
	return Config{
		Socket:                 "/run/sc-bos-supervisor/supervisor.sock",
		StateDir:               "/var/lib/sc-bos-supervisor",
		ImageRepo:              "localhost/smartcore/bos",
		Unit:                   "sc-bos",
		CommitDeadline:         jsontypes.Duration{Duration: 2 * time.Minute},
		AllowInsecureDownloads: false,
	}
}

// Load reads configuration from path, overlaying any options it sets onto the defaults. A missing file
// is not an error - the defaults are returned. A file that cannot be opened, parsed, or whose values are
// invalid (unknown field, unparseable duration, non-positive commitDeadline, a required field set to
// empty, or a malformed imageRepo or unit) is an error.
func Load(path string) (Config, error) {
	cfg := Default()

	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("open config %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields() // a typo'd option is a misconfiguration, not silently ignored
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	// check for plausible config values
	for _, f := range []struct {
		name  string
		value string
		valid *regexp.Regexp // nil = only the non-empty check applies
		what  string         // description of a valid value, for the error message
	}{
		{name: "socket", value: cfg.Socket},
		{name: "stateDir", value: cfg.StateDir},
		{name: "imageRepo", value: cfg.ImageRepo, valid: imageRepoPattern, what: "a valid image repository path"},
		{name: "unit", value: cfg.Unit, valid: unitNamePattern, what: "a valid systemd unit name"},
	} {
		if f.value == "" {
			return Config{}, fmt.Errorf("invalid config %s: %s must not be empty", path, f.name)
		}
		if f.valid != nil && !f.valid.MatchString(f.value) {
			return Config{}, fmt.Errorf("invalid config %s: %s %q is not %s", path, f.name, f.value, f.what)
		}
	}

	// A zero or negative deadline arms time.NewTimer with a non-positive duration, which fires
	// immediately and rolls back every update; reject it rather than silently self-sabotaging.
	if cfg.CommitDeadline.Duration <= 0 {
		return Config{}, fmt.Errorf("invalid config %s: commitDeadline must be positive, got %s", path, cfg.CommitDeadline.Duration)
	}
	return cfg, nil
}
