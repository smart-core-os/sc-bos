package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// ErrInsecureURL is returned when an artefact URL does not use HTTPS.
var ErrInsecureURL = errors.New("artefact URL must use https")

// MaxArtefactBytes is the default size limit for artefact downloads.
const MaxArtefactBytes = 4 << 30 // 4 GiB

// DownloadAndVerify downloads the artefact at srcURL into destDir, verifying its SHA-256 against
// wantSHA256 (the raw 32-byte digest). It returns the path to the downloaded file on success. On any
// failure (or if the body exceeds maxBytes) the partially-downloaded file is removed.
//
// The URL must use HTTPS, since it is a capability URL carrying its own authorisation, unless
// allowInsecure is true.
// If client is nil, http.DefaultClient is used.
func DownloadAndVerify(ctx context.Context, client *http.Client, srcURL string, wantSHA256 []byte, destDir string, maxBytes int64, allowInsecure bool) (string, error) {
	u, err := url.Parse(srcURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" && !allowInsecure {
		return "", ErrInsecureURL
	}

	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download artefact: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download artefact: server returned status %d", resp.StatusCode)
	}

	f, err := os.CreateTemp(destDir, "artefact-*.tar")
	if err != nil {
		return "", fmt.Errorf("create staging file: %w", err)
	}
	// Clean up unless we hand the file back to the caller.
	keep := false
	defer func() {
		_ = f.Close()
		if !keep {
			_ = os.Remove(f.Name())
		}
	}()

	hasher := sha256.New()
	// Read one byte past the ceiling so an artefact of exactly maxBytes is accepted but a larger one is
	// detected rather than silently truncated.
	n, err := io.Copy(io.MultiWriter(f, hasher), io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("write artefact: %w", err)
	}
	if n > maxBytes {
		return "", fmt.Errorf("artefact exceeds maximum size of %d bytes", maxBytes)
	}
	if !bytes.Equal(hasher.Sum(nil), wantSHA256) {
		return "", fmt.Errorf("artefact checksum mismatch: got %s want %s", hex.EncodeToString(hasher.Sum(nil)), hex.EncodeToString(wantSHA256))
	}

	keep = true
	return f.Name(), nil
}
