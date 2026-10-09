//go:build !freebsd

// The Supervisor runs on Linux and FreeBSD. Builds for any other OS, such as development and test builds on
// macOS, use the Linux defaults.

package config

// DefaultPath is the config file the Supervisor reads when no other path is given.
const DefaultPath = "/etc/sc-bos-supervisor/config.json"

const (
	defaultSocket    = "/run/sc-bos-supervisor/supervisor.sock"
	defaultStateDir  = "/var/lib/sc-bos-supervisor"
	defaultInstaller = InstallerPodman
)
