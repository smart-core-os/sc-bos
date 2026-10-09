package config

// DefaultPath is the config file the Supervisor reads when no other path is given.
const DefaultPath = "/usr/local/etc/sc-bos-supervisor/config.json"

const (
	defaultSocket    = "/var/run/sc-bos-supervisor/supervisor.sock"
	defaultStateDir  = "/var/db/sc-bos-supervisor"
	defaultInstaller = InstallerFreeBSDRC
)
