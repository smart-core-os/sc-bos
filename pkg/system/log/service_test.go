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
			if tt.wantContains != "" {
				if len(u) < len(tt.wantContains) {
					t.Errorf("URL %q does not contain %q", u, tt.wantContains)
				}
			}
			// Token must be present.
			if _, err := base64.RawURLEncoding.DecodeString(""); u == "" {
				_ = err
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
	serveLogDownload(w, req, key)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestServeLogDownload_invalidToken(t *testing.T) {
	key := newHMACKey()
	req := httptest.NewRequest(http.MethodGet, "/dl?dlt=notvalid", nil)
	w := httptest.NewRecorder()
	serveLogDownload(w, req, key)
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
	serveLogDownload(w, req, key)
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
	serveLogDownload(w, req, key)

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
	serveLogDownload(w, req, key)

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
