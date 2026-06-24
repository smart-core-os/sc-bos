// Package config loads the Supervisor's configuration from a JSON file, falling back to built-in
// defaults for a missing file or any option the file omits.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Config is the Supervisor's runtime configuration.
type Config struct {
	// Socket is the Unix socket path the SupervisorApi listens on.
	Socket string
	// StateDir holds durable state: the update record at <StateDir>/state.json and downloaded artefacts
	// under <StateDir>/staging.
	StateDir string
	// ImageRepo is the image repository whose :current/:previous tags the installer swaps.
	ImageRepo string
	// Unit is the systemd unit (and container name) for BOS that the Supervisor restarts.
	Unit string
	// CommitDeadline bounds how long reconcile waits for BOS to Commit a new version before rolling back.
	CommitDeadline time.Duration
	// AllowInsecureDownloads permits artefact download URLs that do not use HTTPS. It defaults to false
	// (HTTPS only); enable it only for development against a plain-HTTP update server such as cloudsim.
	AllowInsecureDownloads bool
}

// Default returns the configuration used when no config file is present, and for any option a present
// file omits.
func Default() Config {
	return Config{
		Socket:                 "/run/sc-bos-supervisor/supervisor.sock",
		StateDir:               "/var/lib/sc-bos-supervisor",
		ImageRepo:              "localhost/smartcore/bos",
		Unit:                   "sc-bos",
		CommitDeadline:         2 * time.Minute,
		AllowInsecureDownloads: false,
	}
}

// file mirrors Config for JSON decoding. Every field is optional; an omitted (or empty) one keeps its
// default. CommitDeadline is a Go duration string, e.g. "2m" or "90s".
type file struct {
	Socket                 string `json:"socket"`
	StateDir               string `json:"stateDir"`
	ImageRepo              string `json:"imageRepo"`
	Unit                   string `json:"unit"`
	CommitDeadline         string `json:"commitDeadline"`
	AllowInsecureDownloads *bool  `json:"allowInsecureDownloads"`
}

// Load reads configuration from path, overlaying any options it sets onto the defaults. A missing file
// is not an error - the defaults are returned. A file that cannot be opened, parsed, or whose values are
// invalid (unknown field, unparseable duration) is an error.
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

	var raw file
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields() // a typo'd option is a misconfiguration, not silently ignored
	if err := dec.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if raw.Socket != "" {
		cfg.Socket = raw.Socket
	}
	if raw.StateDir != "" {
		cfg.StateDir = raw.StateDir
	}
	if raw.ImageRepo != "" {
		cfg.ImageRepo = raw.ImageRepo
	}
	if raw.Unit != "" {
		cfg.Unit = raw.Unit
	}
	if raw.CommitDeadline != "" {
		d, err := time.ParseDuration(raw.CommitDeadline)
		if err != nil {
			return Config{}, fmt.Errorf("parse config %s: commitDeadline: %w", path, err)
		}
		cfg.CommitDeadline = d
	}
	if raw.AllowInsecureDownloads != nil {
		cfg.AllowInsecureDownloads = *raw.AllowInsecureDownloads
	}
	return cfg, nil
}
