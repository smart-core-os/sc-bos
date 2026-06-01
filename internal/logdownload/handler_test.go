package logdownload

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/download"
)

// FileDownloadHandler with a payload that is not a valid DownloadToken
// returns 400 — the only failure mode not exercised by the subsystem-level
// tests, which always supply valid tokens.
func TestFileDownloadHandler_CorruptPayload(t *testing.T) {
	h := &FileDownloadHandler{AllowedDir: func() string { return "" }}
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(download.ContextWithPayload(req.Context(), []byte("not-a-protobuf")))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestFileDownloadHandler_FilePath(t *testing.T) {
	dir := t.TempDir()
	fp := writeTestFile(t, dir, "app.log", "log body")

	t.Run("inside allowed dir", func(t *testing.T) {
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return dir }}, &DownloadToken{FilePath: fp})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Body.String(); got != "log body" {
			t.Errorf("body = %q, want %q", got, "log body")
		}
		if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="app.log"` {
			t.Errorf("Content-Disposition = %q", got)
		}
		if got := rec.Header().Get("Content-Type"); got == "" {
			t.Errorf("Content-Type missing")
		}
	})

	t.Run("outside allowed dir is forbidden", func(t *testing.T) {
		otherDir := t.TempDir()
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return otherDir }}, &DownloadToken{FilePath: fp})
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("empty allowed dir disables the check", func(t *testing.T) {
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return "" }}, &DownloadToken{FilePath: fp})
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("nil AllowedDir closure disables the check", func(t *testing.T) {
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: nil}, &DownloadToken{FilePath: fp})
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("missing file returns 404", func(t *testing.T) {
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return dir }}, &DownloadToken{FilePath: filepath.Join(dir, "missing.log")})
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestFileDownloadHandler_ZipFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeTestFile(t, dir, "a.log", "alpha")
	b := writeTestFile(t, dir, "b.log", "beta")

	t.Run("all files inside allowed dir", func(t *testing.T) {
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return dir }}, &DownloadToken{ZipFiles: []string{a, b}})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/zip" {
			t.Errorf("Content-Type = %q, want application/zip", got)
		}
		got := readZipEntries(t, rec.Body.Bytes())
		want := map[string]string{"a.log": "alpha", "b.log": "beta"}
		if len(got) != 2 || got["a.log"] != "alpha" || got["b.log"] != "beta" {
			t.Errorf("zip entries = %v, want %v", got, want)
		}
	})

	t.Run("files outside allowed dir are silently skipped", func(t *testing.T) {
		// ServeZip drops out-of-allowed entries without erroring the response,
		// so the client still receives a usable zip of the permitted files.
		other := writeTestFile(t, t.TempDir(), "leaked.log", "secret")
		rec := serveToken(t, &FileDownloadHandler{AllowedDir: func() string { return dir }}, &DownloadToken{ZipFiles: []string{a, other}})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		got := readZipEntries(t, rec.Body.Bytes())
		if _, leaked := got["leaked.log"]; leaked {
			t.Errorf("leaked.log appeared in zip (got entries=%v)", got)
		}
		if got["a.log"] != "alpha" {
			t.Errorf("expected a.log in zip, got entries=%v", got)
		}
	})
}

func writeTestFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func serveToken(t *testing.T, h *FileDownloadHandler, token *DownloadToken) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := proto.Marshal(token)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(download.ContextWithPayload(req.Context(), payload))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func readZipEntries(t *testing.T, body []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		out[f.Name] = string(b)
	}
	return out
}
