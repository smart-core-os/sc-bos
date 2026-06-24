package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomArtefact(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 4096)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
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
	require.NoError(t, err)
	assert.Empty(t, entries, "staging dir should contain no leftover files")
}

func TestDownloadAndVerify_Success(t *testing.T) {
	data := randomArtefact(t)
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	path, err := DownloadAndVerify(context.Background(), client, url, hexSum(data), dir, MaxArtefactBytes, false)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(got, data), "downloaded bytes should match served bytes")
}

func TestDownloadAndVerify_ChecksumMismatch(t *testing.T) {
	data := randomArtefact(t)
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, hexSum([]byte("different")), dir, MaxArtefactBytes, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_TooLarge(t *testing.T) {
	data := randomArtefact(t) // 4096 bytes
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, hexSum(data), dir, 1024, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_HTTPError(t *testing.T) {
	url, client := tlsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	dir := t.TempDir()

	_, err := DownloadAndVerify(context.Background(), client, url, hexSum(nil), dir, MaxArtefactBytes, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assertNoStagedFiles(t, dir)
}

func TestDownloadAndVerify_RejectsPlainHTTP(t *testing.T) {
	_, err := DownloadAndVerify(context.Background(), nil, "http://example.com/artefact.tar", hexSum(nil), t.TempDir(), MaxArtefactBytes, false)
	assert.ErrorIs(t, err, ErrInsecureURL)
}

func TestDownloadAndVerify_AllowsPlainHTTPWhenPermitted(t *testing.T) {
	data := randomArtefact(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	srv.Client().Transport.(*http.Transport).DisableKeepAlives = true
	dir := t.TempDir()

	path, err := DownloadAndVerify(context.Background(), srv.Client(), srv.URL, hexSum(data), dir, MaxArtefactBytes, true)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(got, data), "downloaded bytes should match served bytes")
}
