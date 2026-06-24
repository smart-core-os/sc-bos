package install

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
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

// MaxArtefactBytes is the default ceiling on a downloaded artefact, guarding a privileged root service
// against a wrong or oversized download filling its staging directory. A BOS container tarball is well
// under this.
const MaxArtefactBytes = 4 << 30 // 4 GiB

// DownloadAndVerify downloads the artefact at rawURL into destDir, verifying its SHA-256 against the
// hex-encoded wantSHA256. It returns the path to the downloaded file on success. On any failure (or if
// the body exceeds maxBytes) the partially-downloaded file is removed.
//
// The URL must use HTTPS, since it is a capability URL carrying its own authorisation, unless
// allowInsecure is true (a development-only escape hatch for plain-HTTP update servers). If client is
// nil, http.DefaultClient is used.
func DownloadAndVerify(ctx context.Context, client *http.Client, rawURL, wantSHA256, destDir string, maxBytes int64, allowInsecure bool) (string, error) {
	want, err := hex.DecodeString(wantSHA256)
	if err != nil || len(want) != sha256.Size {
		return "", fmt.Errorf("invalid sha256 %q: must be %d hex-encoded bytes", wantSHA256, sha256.Size)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" && !allowInsecure {
		return "", ErrInsecureURL
	}

	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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
	if subtle.ConstantTimeCompare(hasher.Sum(nil), want) != 1 {
		return "", fmt.Errorf("artefact checksum mismatch: got %s want %s", hex.EncodeToString(hasher.Sum(nil)), wantSHA256)
	}

	keep = true
	return f.Name(), nil
}
