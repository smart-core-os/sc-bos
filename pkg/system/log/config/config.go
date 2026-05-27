// Package config defines configuration for the log system.
package config

// Root is the configuration for the log system plugin.
type Root struct {
	// LogFilePath is a glob pattern used to discover log files on disk for metadata
	// and download. For example: "/var/log/sc-bos/*.log".
	// Leave empty to disable file-based features (metadata, download URLs).
	LogFilePath string `json:"logFilePath,omitempty"`

	// LogDir is an optional directory path. When set, GetDownloadLogUrl with
	// include_rotated=true returns all regular files in this directory.
	// Also used for metadata (file count, total size) in preference to LogFilePath.
	LogDir string `json:"logDir,omitempty"`

	// BufCap is the ring-buffer capacity for in-memory log message retention.
	// Defaults to 1000. Maximum is 10000.
	BufCap int `json:"bufCap,omitempty"`
}

const maxBufCap = 10000

// BufCapOrDefault returns BufCap clamped to [1, maxBufCap].
// If BufCap is <= 0 the default (1000) is used.
func (r Root) BufCapOrDefault() int {
	if r.BufCap <= 0 {
		return 1000
	}
	if r.BufCap > maxBufCap {
		return maxBufCap
	}
	return r.BufCap
}
