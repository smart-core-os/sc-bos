package logdownload

import (
	"path/filepath"
	"testing"
)

func TestIsUnderDir(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		fp, dir string
		want    bool
	}{
		{"/var/log/app.log", "/var/log", true},
		{"/var/log/sub/app.log", "/var/log", true},
		{"/var/logmalicious/app.log", "/var/log", false},
		{"/var/log", "/var/log", false},
		{"/etc/passwd", "/var/log", false},
		{"/var/log/app.log", "", true},
		{"relative/path", "", true},
		{sep + "var" + sep + "log" + sep + "app.log", "/var/log", true},
	}
	for _, tt := range tests {
		got := isUnderDir(tt.fp, tt.dir)
		if got != tt.want {
			t.Errorf("isUnderDir(%q, %q) = %v, want %v", tt.fp, tt.dir, got, tt.want)
		}
	}
}

func TestLogAllowedDir(t *testing.T) {
	tests := []struct {
		logFilePath, logDir, want string
	}{
		{"", "", ""},
		{"/var/log/app*.log", "", "/var/log"},
		{"/var/log/app*.log", "/var/log/archive", "/var/log/archive"},
		{"", "/var/log/archive", "/var/log/archive"},
		{"/var/log/", "", "/var/log"},
	}
	for _, tt := range tests {
		got := LogAllowedDir(tt.logFilePath, tt.logDir)
		if got != filepath.Clean(tt.want) && !(tt.want == "" && got == "") {
			t.Errorf("LogAllowedDir(%q, %q) = %q, want %q", tt.logFilePath, tt.logDir, got, tt.want)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"app.log.gz", "application/gzip"},
		{"app.log.zst", "application/zstd"},
		{"app.log.bz2", "application/x-bzip2"},
		{"app.unknown", "text/plain"},
		{"archive.zip", "application/zip"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := DetectContentType(tt.file)
			if got != tt.want {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.file, got, tt.want)
			}
		})
	}
}
