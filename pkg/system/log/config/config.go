// Package config defines configuration for the log system.
package config

// Root is the configuration for the log system plugin.
type Root struct {
	// LogFilePath is a glob pattern used to discover log files on disk for metadata
	// and download. For example: "/var/log/sc-bos/*.log".
	// Leave empty to disable file-based features (metadata, download URLs).
	LogFilePath string `json:"logFilePath,omitempty"`

	// HTTPDownloadPath is the HTTP path at which the log file download handler
	// will be registered. Defaults to "/__/log/download".
	HTTPDownloadPath string `json:"httpDownloadPath,omitempty"`

	// HTTPDownloadURLBase is the scheme+host prefix used when constructing
	// download URLs returned by GetDownloadLogUrl. For example:
	// "https://controller.example.com".
	// If empty, the controller's HTTPEndpoint (from system services) is used as the base,
	// producing an absolute URL. Set this to override the auto-detected address.
	HTTPDownloadURLBase string `json:"httpDownloadUrlBase,omitempty"`

	// URLTTLSeconds is the default lifetime of a signed download URL in seconds.
	// Defaults to 900 (15 minutes).
	URLTTLSeconds int `json:"urlTtlSeconds,omitempty"`

	// BufCap is the ring-buffer capacity for in-memory log message retention.
	// Defaults to 1000. Maximum is 10000.
	BufCap int `json:"bufCap,omitempty"`
}

func (r Root) DownloadPath() string {
	if r.HTTPDownloadPath != "" {
		return r.HTTPDownloadPath
	}
	return "/__/log/download"
}

func (r Root) URLTTLSecondsOrDefault() int {
	if r.URLTTLSeconds > 0 {
		return r.URLTTLSeconds
	}
	return 900
}
