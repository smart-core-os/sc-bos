package audit

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/download"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// harness exposes the audit Setup through a typed gRPC client (for
// GetDownloadLogUrl) and a fetch function (for following the returned URL).
type harness struct {
	client logpb.LogApiClient
	fetch  func(t *testing.T, u string) *http.Response
}

func newHarness(t *testing.T, filename string) *harness {
	t.Helper()
	setup, err := NewSetup(filename, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("NewSetup: %v", err)
	}
	t.Cleanup(func() { _ = setup.Close() })

	router := newDownloadRouter(t)
	srv := setup.NewModelServer(router)

	n := node.New("test")
	n.Announce(n.Name(), node.HasServer(logpb.RegisterLogApiServer, logpb.LogApiServer(srv)))

	fetch := func(t *testing.T, u string) *http.Response {
		t.Helper()
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatalf("parse URL %q: %v", u, err)
		}
		target := parsed.RequestURI()
		if target == "" {
			target = parsed.Path
		}
		req := httptest.NewRequest("GET", target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Result()
	}
	return &harness{
		client: logpb.NewLogApiClient(n.ClientConn()),
		fetch:  fetch,
	}
}

func newDownloadRouter(t *testing.T) *download.Router {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return download.NewRouter(
		download.NewHMACSigner(key),
		download.WithBaseURL("/download"),
		download.WithTTL(15*time.Minute),
	)
}

// writeFile writes name with content into dir, replacing any existing file.
// NewSetup creates the audit file eagerly but only opens it via lumberjack on
// first write; tests never call Setup.Write, so it is safe to overwrite the
// file content here.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	fp := filepath.Join(dir, name)
	if err := os.WriteFile(fp, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", fp, err)
	}
	return fp
}

func TestAuditDownload_SingleFile(t *testing.T) {
	dir := t.TempDir()
	auditFile := filepath.Join(dir, "audit.log")
	h := newHarness(t, auditFile)
	writeFile(t, dir, "audit.log", "audit body")

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	if got := len(resp.Files); got != 1 {
		t.Fatalf("len(Files) = %d, want 1", got)
	}
	fl := resp.Files[0]
	if fl.Filename != "audit.log" {
		t.Errorf("Filename = %q, want audit.log", fl.Filename)
	}
	if want := int64(len("audit body")); fl.SizeBytes != want {
		t.Errorf("SizeBytes = %d, want %d", fl.SizeBytes, want)
	}
	// ContentType is derived from the host's mime DB (mime.TypeByExtension)
	// and varies across CI runners; only assert it is set.
	if fl.ContentType == "" {
		t.Error("ContentType is empty")
	}

	httpResp := h.fetch(t, fl.Url)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("fetch status = %d, body=%s", httpResp.StatusCode, body)
	}
	if ct := httpResp.Header.Get("Content-Type"); ct == "" {
		t.Error("response Content-Type header is empty")
	}
	if cd := httpResp.Header.Get("Content-Disposition"); !strings.Contains(cd, "audit.log") {
		t.Errorf("Content-Disposition = %q, want to contain audit.log", cd)
	}
	body, _ := io.ReadAll(httpResp.Body)
	if string(body) != "audit body" {
		t.Errorf("body = %q, want %q", body, "audit body")
	}
}

// With no filename configured, GetDownloadLogUrl is not wired and returns
// codes.Unimplemented (matches log system's behaviour when LogFilePath="").
func TestAuditDownload_Disabled(t *testing.T) {
	h := newHarness(t, "")

	_, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := status.Code(err); got != codes.Unimplemented {
		t.Errorf("status code = %s, want Unimplemented", got)
	}
}

// A tampered URL token is rejected with 401.
func TestAuditDownload_TamperedURL(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, filepath.Join(dir, "audit.log"))
	writeFile(t, dir, "audit.log", "secret")

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	good := resp.Files[0].Url
	tampered := good[:len(good)-1] + altChar(good[len(good)-1])

	httpResp := h.fetch(t, tampered)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", httpResp.StatusCode)
	}
}

// An expired URL is rejected. Uses testing/synctest to advance virtual time
// past the configured TTL without sleeping in real time.
func TestAuditDownload_ExpiredURL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		h := newHarness(t, filepath.Join(dir, "audit.log"))
		writeFile(t, dir, "audit.log", "ages ago")

		resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
		if err != nil {
			t.Fatalf("GetDownloadLogUrl: %v", err)
		}
		urlStr := resp.Files[0].Url

		time.Sleep(2*time.Hour + time.Minute)

		httpResp := h.fetch(t, urlStr)
		defer httpResp.Body.Close()
		if httpResp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", httpResp.StatusCode)
		}
	})
}

func altChar(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}
