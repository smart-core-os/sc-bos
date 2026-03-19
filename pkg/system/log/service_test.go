package log

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- JWT sign / parse round-trip ------------------------------------------------

func TestSignAndParseDownloadToken_roundtrip(t *testing.T) {
	key := newHMACKey()
	body := downloadClaims{FilePath: "/var/log/app.log"}
	expiry := time.Now().Add(5 * time.Minute)

	tokenStr, err := signDownloadToken(body, key, expiry)
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}

	got, err := parseDownloadToken(tokenStr, key)
	if err != nil {
		t.Fatalf("parseDownloadToken: %v", err)
	}
	if got.FilePath != body.FilePath {
		t.Errorf("FilePath = %q, want %q", got.FilePath, body.FilePath)
	}
}

func TestSignAndParseDownloadToken_zipFiles(t *testing.T) {
	key := newHMACKey()
	body := downloadClaims{ZipFiles: []string{"/a.log", "/b.log"}}
	expiry := time.Now().Add(time.Minute)

	tokenStr, err := signDownloadToken(body, key, expiry)
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	got, err := parseDownloadToken(tokenStr, key)
	if err != nil {
		t.Fatalf("parseDownloadToken: %v", err)
	}
	if len(got.ZipFiles) != 2 || got.ZipFiles[0] != "/a.log" {
		t.Errorf("ZipFiles = %v", got.ZipFiles)
	}
}

func TestParseDownloadToken_expired(t *testing.T) {
	key := newHMACKey()
	// Sign with a past expiry (more than 1-minute leeway ago).
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: "/x"}, key, time.Now().Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	_, err = parseDownloadToken(tokenStr, key)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestParseDownloadToken_wrongKey(t *testing.T) {
	key1 := newHMACKey()
	key2 := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: "/x"}, key1, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	_, err = parseDownloadToken(tokenStr, key2)
	if err == nil {
		t.Error("expected error for wrong key, got nil")
	}
}

// ---- buildDownloadURL -----------------------------------------------------------

func TestBuildDownloadURL(t *testing.T) {
	tests := []struct {
		name         string
		urlBase      string
		downloadPath string
		wantContains string
	}{
		{
			name:         "full base URL",
			urlBase:      "https://bos.example.com",
			downloadPath: "/api/logs/download",
			wantContains: "https://bos.example.com/api/logs/download",
		},
		{
			name:         "base URL with path already set",
			urlBase:      "https://bos.example.com/prefix",
			downloadPath: "/api/logs/download",
			wantContains: "https://bos.example.com/prefix",
		},
		{
			name:         "empty base falls back to downloadPath",
			urlBase:      "",
			downloadPath: "/api/logs/download",
			wantContains: "/api/logs/download",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := buildDownloadURL(tt.urlBase, tt.downloadPath, "tok123")
			if err != nil {
				t.Fatalf("buildDownloadURL: %v", err)
			}
			if u == "" {
				t.Error("got empty URL")
			}
			if tt.wantContains != "" && !containsString(u, tt.wantContains) {
				t.Errorf("URL %q does not contain %q", u, tt.wantContains)
			}
		})
	}
}

func TestBuildDownloadURL_tokenEmbedded(t *testing.T) {
	const tok = "sometoken"
	u, err := buildDownloadURL("https://host", "/dl", tok)
	if err != nil {
		t.Fatalf("buildDownloadURL: %v", err)
	}
	want := base64.RawURLEncoding.EncodeToString([]byte(tok))
	if len(u) == 0 {
		t.Fatal("empty URL")
	}
	// The raw URL-encoded token should appear in the query string.
	if !containsString(u, want) {
		t.Errorf("URL %q does not contain encoded token %q", u, want)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := range len(s) - len(sub) + 1 {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---- HTTP download handler ------------------------------------------------------

func TestServeLogDownload_missingToken(t *testing.T) {
	key := newHMACKey()
	req := httptest.NewRequest(http.MethodGet, "/dl", nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestServeLogDownload_invalidToken(t *testing.T) {
	key := newHMACKey()
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt=notvalid", nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestServeLogDownload_fileNotFound(t *testing.T) {
	key := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: "/nonexistent/path/file.log"}, key, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt="+encoded, nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServeLogDownload_singleFile(t *testing.T) {
	// Write a temporary log file.
	dir := t.TempDir()
	fp := filepath.Join(dir, "app.log")
	if err := os.WriteFile(fp, []byte("log content here"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	key := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: fp}, key, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt="+encoded, nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, dir)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "log content here" {
		t.Errorf("body = %q, want %q", got, "log content here")
	}
}

func TestServeLogDownload_zipDownload(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.log")
	f2 := filepath.Join(dir, "b.log")
	_ = os.WriteFile(f1, []byte("file a"), 0600)
	_ = os.WriteFile(f2, []byte("file b"), 0600)

	key := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{ZipFiles: []string{f1, f2}}, key, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt="+encoded, nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, dir)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("empty zip body")
	}
}

// ---- path validation (isUnderDir / allowedDir enforcement) ----------------------

func TestServeLogDownload_pathOutsideAllowedDir(t *testing.T) {
	// Create a real file outside the configured log directory.
	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	allowedDir := t.TempDir() // different directory

	key := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: secret}, key, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt="+encoded, nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, allowedDir)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (forbidden)", w.Code, http.StatusForbidden)
	}
}

func TestServeLogDownload_pathPrefixCollision(t *testing.T) {
	// Ensure "/var/logmalicious/file" is rejected when allowedDir is "/var/log".
	// We can't create that literal path in a test, so we use temp dirs that
	// reproduce the same prefix-collision pattern.
	parent := t.TempDir()
	allowedDir := filepath.Join(parent, "log")
	maliciousDir := filepath.Join(parent, "logmalicious")
	if err := os.MkdirAll(maliciousDir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	fp := filepath.Join(maliciousDir, "file.log")
	if err := os.WriteFile(fp, []byte("should not be served"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	key := newHMACKey()
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: fp}, key, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("signDownloadToken: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt="+encoded, nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key, allowedDir)

	if w.Code != http.StatusForbidden {
		t.Errorf("prefix-collision: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestIsUnderDir(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		fp, dir string
		want    bool
	}{
		{"/var/log/app.log", "/var/log", true},
		{"/var/log/sub/app.log", "/var/log", true},
		{"/var/logmalicious/app.log", "/var/log", false},
		{"/var/log", "/var/log", false},  // dir itself is not "under" dir
		{"/etc/passwd", "/var/log", false},
		{"/var/log/app.log", "", true},   // empty dir: check disabled
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
		got := logAllowedDir(tt.logFilePath, tt.logDir)
		if got != filepath.Clean(tt.want) && !(tt.want == "" && got == "") {
			t.Errorf("logAllowedDir(%q, %q) = %q, want %q", tt.logFilePath, tt.logDir, got, tt.want)
		}
	}
}

// ---- detectContentType ----------------------------------------------------------

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
			got := detectContentType(tt.file)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.file, got, tt.want)
			}
		})
	}
}
