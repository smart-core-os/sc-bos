package install

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

func randomArtefact(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 4096)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

// sum returns the sha256 checksum of b, the algorithm used by most tests.
func sum(b []byte) checksum.Checksum {
	h := sha256.Sum256(b)
	return checksum.Checksum{Algo: checksum.SHA256, Digest: h[:]}
}

// tlsServer starts an HTTPS test server (DownloadAndVerify requires https) and returns a client that
// trusts it. Keep-alives are disabled so connection goroutines exit promptly under synctest.
func tlsServer(t *testing.T, h http.HandlerFunc) (string, *http.Client) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	client := srv.Client()
	client.Transport.(*http.Transport).DisableKeepAlives = true
	return srv.URL, client
}

func assertNoStagedFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("staging dir should contain no leftover files, got %d", len(entries))
	}
}

func TestDownloadAndVerify_Success(t *testing.T) {
	data := randomArtefact(t)
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	path, err := DownloadAndVerify(context.Background(), client, url, sum(data), dir, MaxArtefactBytes, false)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("downloaded bytes should match served bytes")
	}
}

func TestDownloadAndVerify_MD5(t *testing.T) {
	data := randomArtefact(t)
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	md5Sum := md5.Sum(data)
	want := checksum.Checksum{Algo: checksum.MD5, Digest: md5Sum[:]}
	path, err := DownloadAndVerify(context.Background(), client, url, want, dir, MaxArtefactBytes, false)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("downloaded bytes should match served bytes")
	}
}

func TestDownloadAndVerify_ChecksumMismatch(t *testing.T) {
	data := randomArtefact(t)
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, sum([]byte("different")), dir, MaxArtefactBytes, false)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("want checksum mismatch error, got %v", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_TooLarge(t *testing.T) {
	data := randomArtefact(t) // 4096 bytes
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, sum(data), dir, 1024, false)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("want size error, got %v", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_HTTPError(t *testing.T) {
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, sum(nil), dir, MaxArtefactBytes, false)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("want 404 error, got %v", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_RejectsPlainHTTP(t *testing.T) {
	_, err := DownloadAndVerify(context.Background(), nil, "http://example.com/artefact.tar", sum(nil), t.TempDir(), MaxArtefactBytes, false)
	if !errors.Is(err, ErrInsecureURL) {
		t.Errorf("want ErrInsecureURL, got %v", err)
	}
}

func TestDownloadAndVerify_RejectsRedirectToPlainHTTP(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(randomArtefact(t))
	}))
	t.Cleanup(plain.Close)

	url, client := tlsServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, sum(nil), dir, MaxArtefactBytes, false)
	if !errors.Is(err, ErrInsecureURL) {
		t.Errorf("want ErrInsecureURL, got %v", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_AllowsPlainHTTPWhenPermitted(t *testing.T) {
	data := randomArtefact(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	srv.Client().Transport.(*http.Transport).DisableKeepAlives = true
	dir := t.TempDir()

	path, err := DownloadAndVerify(context.Background(), srv.Client(), srv.URL, sum(data), dir, MaxArtefactBytes, true)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("downloaded bytes should match served bytes")
	}
}
