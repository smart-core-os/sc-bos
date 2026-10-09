// Package config loads the Supervisor's configuration from a JSON file, falling back to built-in
// defaults for a missing file or any option the file omits.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// Installers the Supervisor can use to apply updates, selected by Config.Installer.
const (
	// InstallerPodman installs BOS as a container image run by a systemd unit.
	InstallerPodman = "podman"
	// InstallerFreeBSDRC installs BOS as a release folder run by an rc(8) service.
	InstallerFreeBSDRC = "freebsdrc"
)

// A rough approximation of rules for these identifiers - excludes invalid characters but doesn't exactly match the
// required structure. Good enough to catch accidental misconfigurations.
var (
	imageRepoPattern = regexp.MustCompile(`^[a-zA-Z0-9._/:-]+$`)
	unitNamePattern  = regexp.MustCompile(`^[a-zA-Z0-9._:@-]+$`)
	rcServicePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

// Config is the Supervisor's runtime configuration. Every option is optional in the config file; an
// omitted one keeps the value from Default.
type Config struct {
	// Socket is the Unix socket path the SupervisorApi listens on.
	Socket string `json:"socket,omitempty"`
	// SocketGroup is a group name or group ID that overrides the owning group of the socket file.
	// If unset, the default group is used.
	SocketGroup string `json:"socketGroup,omitempty"`
	// StateDir holds durable state: the update record at <StateDir>/state.json and downloaded artefacts
	// under <StateDir>/staging.
	StateDir string `json:"stateDir,omitempty"`
	// CommitDeadline bounds how long reconcile waits for BOS to Commit a new version before rolling back.
	// It is a Go duration string, e.g. "2m" or "90s".
	CommitDeadline jsontypes.Duration `json:"commitDeadline,omitempty"`
	// AllowInsecureDownloads permits artefact download URLs that do not use HTTPS. It defaults to false
	// (HTTPS only); enable it only for development against a plain-HTTP update server such as cloudsim.
	AllowInsecureDownloads bool `json:"allowInsecureDownloads,omitempty"`
	// Installer selects how updates are applied: InstallerPodman or InstallerFreeBSDRC.
	Installer string `json:"installer,omitempty"`
	// Podman configures the installer used when Installer is InstallerPodman.
	Podman Podman `json:"podman,omitzero"`
	// FreeBSDRC configures the installer used when Installer is InstallerFreeBSDRC.
	FreeBSDRC FreeBSDRC `json:"freebsdrc,omitzero"`
}

// Podman configures the Podman installer.
type Podman struct {
	// ImageRepo is the image repository whose :current/:previous tags the installer swaps.
	ImageRepo string `json:"imageRepo,omitempty"`
	// Unit is the systemd unit (and container name) for BOS that the Supervisor restarts.
	Unit string `json:"unit,omitempty"`
}

// FreeBSDRC configures the FreeBSD rc(8) installer.
type FreeBSDRC struct {
	// Root is the absolute path of the folder holding one folder per release and the current and
	// previous links. Installing a release deletes anything else in it.
	Root string `json:"root,omitempty"`
	// Service is the rc(8) service for BOS that the Supervisor restarts.
	Service string `json:"service,omitempty"`
}

// Default returns the configuration used when no config file is present, and for any option a present
// file omits. The socket, state and installer defaults depend on the OS the Supervisor is built for.
func Default() Config {
	return Config{
		Socket:                 defaultSocket,
		StateDir:               defaultStateDir,
		CommitDeadline:         jsontypes.Duration{Duration: 5 * time.Minute},
		AllowInsecureDownloads: false,
		Installer:              defaultInstaller,
		Podman: Podman{
			ImageRepo: "localhost/smartcore/bos",
			Unit:      "sc-bos",
		},
		FreeBSDRC: FreeBSDRC{
			Root:    "/usr/local/sc-bos/releases",
			Service: "sc_bos",
		},
	}
}

// Load reads configuration from path, overlaying any options it sets onto the defaults. A missing file
// is not an error - the defaults are returned. A file that cannot be opened, parsed, or whose values are
// invalid is an error. Invalid values include an unknown field, an unparseable or non-positive
// commitDeadline, an empty socket or stateDir, a socketGroup that does not resolve, an unknown
// installer, or a malformed option in the selected installer's block. The block of an installer that
// is not selected is not checked.
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
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.Socket == "" {
		return errors.New("socket must not be empty")
	}
	if c.StateDir == "" {
		return errors.New("stateDir must not be empty")
	}

	switch c.Installer {
	case InstallerPodman:
		if !imageRepoPattern.MatchString(c.Podman.ImageRepo) {
			return fmt.Errorf("podman.imageRepo %q is not a valid image repository path", c.Podman.ImageRepo)
		}
		if !unitNamePattern.MatchString(c.Podman.Unit) {
			return fmt.Errorf("podman.unit %q is not a valid systemd unit name", c.Podman.Unit)
		}
	case InstallerFreeBSDRC:
		if !filepath.IsAbs(c.FreeBSDRC.Root) {
			return fmt.Errorf("freebsdrc.root %q is not an absolute path", c.FreeBSDRC.Root)
		}
		if !rcServicePattern.MatchString(c.FreeBSDRC.Service) {
			return fmt.Errorf("freebsdrc.service %q is not a valid rc service name", c.FreeBSDRC.Service)
		}
	default:
		return fmt.Errorf("installer %q is not %q or %q", c.Installer, InstallerPodman, InstallerFreeBSDRC)
	}

	if c.SocketGroup != "" {
		if _, err := LookupGroup(c.SocketGroup); err != nil {
			return fmt.Errorf("socketGroup: %w", err)
		}
	}

	// A zero or negative deadline arms time.NewTimer with a non-positive duration, which fires
	// immediately and rolls back every update; reject it rather than silently self-sabotaging.
	if c.CommitDeadline.Duration <= 0 {
		return fmt.Errorf("commitDeadline must be positive, got %s", c.CommitDeadline.Duration)
	}
	return nil
}

// LookupGroup returns the GID of group, which is a group name or a numeric GID.
func LookupGroup(group string) (int, error) {
	g, err := user.LookupGroup(group)
	if err != nil {
		if _, convErr := strconv.Atoi(group); convErr != nil {
			return 0, err
		}
		if g, err = user.LookupGroupId(group); err != nil {
			return 0, err
		}
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, fmt.Errorf("group %q has non-numeric gid %q", group, g.Gid)
	}
	return gid, nil
}
