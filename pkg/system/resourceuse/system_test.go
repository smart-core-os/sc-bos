package resourceuse

import (
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
)

func TestShouldHideDisk_FsType(t *testing.T) {
	cfg := Root{}
	fsTypes := cfg.effectiveHiddenFsTypes()
	prefixes := cfg.effectiveHiddenMountPrefixes()

	hidden := []string{
		"tmpfs", "devtmpfs", "devfs",
		"proc", "sysfs",
		"cgroup", "cgroup2",
		"overlay", "squashfs",
		"pstore", "bpf", "tracefs", "debugfs", "securityfs",
		"configfs", "fusectl", "hugetlbfs", "mqueue",
		"autofs", "rpc_pipefs",
	}
	for _, fstype := range hidden {
		p := disk.PartitionStat{Device: "x", Mountpoint: "/data", Fstype: fstype}
		if !shouldHideDisk(p, fsTypes, prefixes) {
			t.Errorf("expected fstype %q to be hidden", fstype)
		}
	}
}

func TestShouldHideDisk_MountPrefix(t *testing.T) {
	cfg := Root{}
	fsTypes := cfg.effectiveHiddenFsTypes()
	prefixes := cfg.effectiveHiddenMountPrefixes()

	cases := []struct {
		mountpoint string
		hidden     bool
	}{
		// OS-level virtual filesystems
		{"/proc", true},
		{"/proc/1/fd", true},
		{"/sys", true},
		{"/sys/fs/cgroup", true},
		{"/dev", true},
		{"/dev/shm", true},
		{"/dev/pts", true},

		// Runtime / secrets
		{"/run", true},
		{"/run/secrets", true},
		{"/run/secrets/kubernetes.io/serviceaccount", true},
		{"/var/run", true},
		{"/var/run/secrets", true},

		// Snap packages
		{"/snap", true},
		{"/snap/core/12345", true},

		// Docker
		{"/var/lib/docker", true},
		{"/var/lib/docker/overlay2/abc/merged", true},

		// macOS system volumes
		{"/System/Volumes/VM", true},
		{"/System/Volumes/Preboot", true},
		{"/System/Volumes/Update", true},
		{"/System/Volumes/xarts", true},
		{"/System/Volumes/iSCPreboot", true},
		{"/System/Volumes/Hardware", true},

		// Should NOT be hidden
		{"/", false},
		{"/home", false},
		{"/data", false},
		{"/mnt/storage", false},
		{"/var/log", false},
		{"/System/Volumes/Data", false}, // macOS user data volume — keep visible
	}

	for _, tc := range cases {
		p := disk.PartitionStat{Device: "/dev/sda1", Mountpoint: tc.mountpoint, Fstype: "ext4"}
		got := shouldHideDisk(p, fsTypes, prefixes)
		if got != tc.hidden {
			t.Errorf("mountpoint %q: got hidden=%v, want %v", tc.mountpoint, got, tc.hidden)
		}
	}
}

func TestShouldHideDisk_ConfigOverride(t *testing.T) {
	cfg := Root{
		HiddenFsTypes:       []string{"customfs"},
		HiddenMountPrefixes: []string{"/custom/secrets"},
	}
	fsTypes := cfg.effectiveHiddenFsTypes()
	prefixes := cfg.effectiveHiddenMountPrefixes()

	// Custom fstype is hidden
	p := disk.PartitionStat{Device: "x", Mountpoint: "/data", Fstype: "customfs"}
	if !shouldHideDisk(p, fsTypes, prefixes) {
		t.Error("expected custom fstype to be hidden")
	}

	// Custom mount prefix is hidden
	p = disk.PartitionStat{Device: "x", Mountpoint: "/custom/secrets/token", Fstype: "ext4"}
	if !shouldHideDisk(p, fsTypes, prefixes) {
		t.Error("expected custom mount prefix to be hidden")
	}

	// Default tmpfs is NOT hidden when overriding (config replaces defaults entirely)
	p = disk.PartitionStat{Device: "x", Mountpoint: "/run/secrets", Fstype: "tmpfs"}
	if shouldHideDisk(p, fsTypes, prefixes) {
		t.Error("default rules should not apply when config overrides are set")
	}
}