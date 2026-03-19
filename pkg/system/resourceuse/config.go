package resourceuse

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// defaultHiddenFsTypes is the built-in list of filesystem types to exclude from disk
// usage reporting. These represent virtual, kernel-internal, or container-managed mounts
// that are never useful for capacity monitoring and may expose internal OS details.
var defaultHiddenFsTypes = []string{
	"devtmpfs", "devfs",  // device filesystems
	"tmpfs",              // memory-backed temp mounts (includes /run, /dev/shm, k8s secrets)
	"proc",               // Linux process filesystem
	"sysfs",              // Linux sysfs
	"cgroup", "cgroup2",  // control groups
	"pstore",             // persistent crash storage
	"bpf",                // BPF filesystem
	"tracefs",            // kernel tracing
	"debugfs",            // kernel debug
	"securityfs",         // Linux security modules (SELinux etc.)
	"configfs",           // kernel config interface
	"fusectl",            // FUSE control filesystem
	"hugetlbfs",          // huge pages
	"mqueue",             // POSIX message queues
	"overlay",            // Docker/container overlay filesystems
	"squashfs",           // snap packages (read-only compressed)
	"autofs",             // auto-mount filesystem
	"rpc_pipefs",         // NFS RPC
}

// defaultHiddenMountPrefixes is the built-in list of mount point path prefixes to exclude
// from disk usage reporting. This catches mounts that use a data filesystem type but still
// represent OS internals, secrets, or container storage.
var defaultHiddenMountPrefixes = []string{
	"/proc",                      // Linux kernel virtual filesystem
	"/sys",                       // Linux sysfs
	"/dev",                       // device filesystem (includes /dev/shm)
	"/run",                       // runtime data and secrets (includes /run/secrets)
	"/var/run",                   // legacy symlink to /run; kept for older systems that report the unresolved path
	"/snap",                      // snap package mounts
	"/var/lib/docker",            // Docker-managed storage
	"/System/Volumes/VM",         // macOS virtual memory
	"/System/Volumes/Preboot",    // macOS preboot environment
	"/System/Volumes/Update",     // macOS software update staging
	"/System/Volumes/xarts",      // macOS hardware attestation
	"/System/Volumes/iSCPreboot", // macOS initial preboot
	"/System/Volumes/Hardware",   // macOS hardware info
}

type Root struct {
	// PollInterval controls how often resource stats are collected.
	// Defaults to 10s if unset.
	PollInterval *jsontypes.Duration `json:"pollInterval,omitempty"`
	// HiddenFsTypes lists filesystem types to exclude from disk usage reporting.
	// If unset, defaultHiddenFsTypes is used.
	HiddenFsTypes []string `json:"hiddenFsTypes,omitempty"`
	// HiddenMountPrefixes lists mount point path prefixes to exclude from disk usage reporting.
	// If unset, defaultHiddenMountPrefixes is used.
	HiddenMountPrefixes []string `json:"hiddenMountPrefixes,omitempty"`
}

func (r Root) pollInterval() time.Duration {
	return r.PollInterval.Or(10 * time.Second)
}

func (r Root) effectiveHiddenFsTypes() map[string]struct{} {
	src := r.HiddenFsTypes
	if len(src) == 0 {
		src = defaultHiddenFsTypes
	}
	m := make(map[string]struct{}, len(src))
	for _, t := range src {
		m[t] = struct{}{}
	}
	return m
}

func (r Root) effectiveHiddenMountPrefixes() []string {
	if len(r.HiddenMountPrefixes) == 0 {
		return defaultHiddenMountPrefixes
	}
	return r.HiddenMountPrefixes
}
