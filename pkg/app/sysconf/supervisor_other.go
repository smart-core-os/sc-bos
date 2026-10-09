//go:build !linux && !freebsd

package sysconf

// defaultSupervisorSocketPath is empty because the Supervisor only runs on Linux and FreeBSD. The
// integration stays disabled unless the config sets a socket path explicitly.
const defaultSupervisorSocketPath = ""
