package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

// ErrInsecureURL is returned when an artefact URL does not use HTTPS.
var ErrInsecureURL = errors.New("artefact URL must use https")

// MaxArtefactBytes is the default size limit for artefact downloads.
const MaxArtefactBytes = 4 << 30 // 4 GiB

// DownloadAndVerify downloads the artefact at srcURL into destDir, verifying it against want (a
// type-prefixed checksum, e.g. sha256 or md5). It returns the path to the downloaded file on success.
// On any failure (or if the body exceeds maxBytes) the partially-downloaded file is removed.
//
// The URL must use HTTPS, since it is a capability URL carrying its own authorisation, unless
// allowInsecure is true; this holds across redirects, so an https URL that redirects to http is
// rejected rather than followed over cleartext.
// If client is nil, http.DefaultClient is used.
func DownloadAndVerify(ctx context.Context, client *http.Client, srcURL string, want checksum.Checksum, destDir string, maxBytes int64, allowInsecure bool) (string, error) {
	hasher, err := want.Algo.NewHash()
	if err != nil {
		return "", fmt.Errorf("checksum: %w", err)
	}

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
	if !allowInsecure {
		// http.Client follows redirects without re-checking the scheme, so an https capability URL that
		// redirects to http:// would otherwise leak it over cleartext. Enforce https on every hop. Copy
		// the client so a shared one (e.g. http.DefaultClient) is not mutated; a custom CheckRedirect also
		// replaces the default 10-hop limit, so re-apply it.
		c := *client
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return ErrInsecureURL
			}
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		}
		client = &c
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

	// Read one byte past the ceiling so an artefact of exactly maxBytes is accepted but a larger one is
	// detected rather than silently truncated.
	n, err := io.Copy(io.MultiWriter(f, hasher), io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("write artefact: %w", err)
	}
	if n > maxBytes {
		return "", fmt.Errorf("artefact exceeds maximum size of %d bytes", maxBytes)
	}
	if got := hasher.Sum(nil); !bytes.Equal(got, want.Digest) {
		return "", fmt.Errorf("artefact checksum mismatch: got %s want %s", checksum.Format(want.Algo, got), want.String())
	}

	keep = true
	return f.Name(), nil
}
