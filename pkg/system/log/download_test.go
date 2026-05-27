package log

import (
	"archive/zip"
	"bytes"
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

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/download"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/log/config"
)

// harness exposes the System through a typed gRPC client (for
// GetDownloadLogUrl) and a fetch function (for following the returned URL).
type harness struct {
	client  logpb.LogApiClient
	fetch   func(t *testing.T, u string) *http.Response
	reapply func(cfg config.Root) error
}

func newHarness(t *testing.T, cfg config.Root) *harness {
	t.Helper()
	n := node.New("test")
	router := newDownloadRouter(t)
	services := system.Services{
		Logger:         zap.NewNop(),
		Node:           n,
		DownloadRouter: router,
	}
	s := NewSystem(services)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := s.applyConfig(ctx, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
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
		client:  logpb.NewLogApiClient(n.ClientConn()),
		fetch:   fetch,
		reapply: func(cfg config.Root) error { return s.applyConfig(ctx, cfg) },
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

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	fp := filepath.Join(dir, name)
	if err := os.WriteFile(fp, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", fp, err)
	}
	return fp
}

// include_rotated=false → single (last-lexically) file streamed.
func TestSystem_Download_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.log.1", "first")
	writeFile(t, dir, "app.log.2", "second")
	writeFile(t, dir, "app.log.3", "third")

	h := newHarness(t, config.Root{
		LogFilePath: filepath.Join(dir, "app.log.*"),
	})

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	if got := len(resp.Files); got != 1 {
		t.Fatalf("len(Files) = %d, want 1", got)
	}
	fl := resp.Files[0]
	if fl.Filename != "app.log.3" {
		t.Errorf("Filename = %q, want app.log.3 (last lexically)", fl.Filename)
	}
	if want := int64(len("third")); fl.SizeBytes != want {
		t.Errorf("SizeBytes = %d, want %d", fl.SizeBytes, want)
	}
	if fl.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want text/plain", fl.ContentType)
	}

	httpResp := h.fetch(t, fl.Url)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("fetch status = %d, body=%s", httpResp.StatusCode, body)
	}
	if ct := httpResp.Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("response Content-Type = %q, want text/plain", ct)
	}
	if cd := httpResp.Header.Get("Content-Disposition"); !strings.Contains(cd, "app.log.3") {
		t.Errorf("Content-Disposition = %q, want to contain app.log.3", cd)
	}
	body, _ := io.ReadAll(httpResp.Body)
	if string(body) != "third" {
		t.Errorf("body = %q, want %q", body, "third")
	}
}

// include_rotated=true with LogDir → zip of every regular file in LogDir.
func TestSystem_Download_RotatedZip_LogDir(t *testing.T) {
	dir := t.TempDir()
	want := map[string]string{
		"a.log":    "alpha",
		"b.log.gz": "compressed",
		"c.log":    "gamma",
	}
	for name, body := range want {
		writeFile(t, dir, name, body)
	}

	h := newHarness(t, config.Root{
		LogFilePath: filepath.Join(dir, "*.log"),
		LogDir:      dir,
	})

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{IncludeRotated: true})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	if got := len(resp.Files); got != 1 {
		t.Fatalf("len(Files) = %d, want 1", got)
	}
	fl := resp.Files[0]
	if fl.Filename != "logs.zip" {
		t.Errorf("Filename = %q, want logs.zip", fl.Filename)
	}
	if fl.ContentType != "application/zip" {
		t.Errorf("ContentType = %q, want application/zip", fl.ContentType)
	}

	httpResp := h.fetch(t, fl.Url)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("fetch status = %d", httpResp.StatusCode)
	}
	if ct := httpResp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Errorf("response Content-Type = %q", ct)
	}
	if cd := httpResp.Header.Get("Content-Disposition"); !strings.Contains(cd, "logs.zip") {
		t.Errorf("Content-Disposition = %q", cd)
	}

	body, _ := io.ReadAll(httpResp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	if !mapsEqual(got, want) {
		t.Errorf("zip contents = %v, want %v", got, want)
	}
}

// include_rotated=true without LogDir → zip of files matched by LogFilePath glob.
func TestSystem_Download_RotatedZip_Glob(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.log", "current")
	writeFile(t, dir, "app.log.1", "rotated-1")
	writeFile(t, dir, "app.log.2", "rotated-2")
	// Files outside the glob must not appear in the zip.
	writeFile(t, dir, "other.txt", "ignored")

	h := newHarness(t, config.Root{
		LogFilePath: filepath.Join(dir, "app.log*"),
	})

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{IncludeRotated: true})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	if got := len(resp.Files); got != 1 {
		t.Fatalf("len(Files) = %d, want 1", got)
	}

	httpResp := h.fetch(t, resp.Files[0].Url)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("fetch status = %d", httpResp.StatusCode)
	}
	body, _ := io.ReadAll(httpResp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	want := map[string]string{
		"app.log":   "current",
		"app.log.1": "rotated-1",
		"app.log.2": "rotated-2",
	}
	if !mapsEqual(got, want) {
		t.Errorf("zip contents = %v, want %v", got, want)
	}
	if _, ok := got["other.txt"]; ok {
		t.Errorf("non-matching file other.txt leaked into zip")
	}
}

// Glob matches no files → empty Files response (no error).
func TestSystem_Download_NoMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, config.Root{
		LogFilePath: filepath.Join(dir, "*.log"),
	})

	for _, tc := range []struct {
		name string
		req  *logpb.GetDownloadLogUrlRequest
	}{
		{"single", &logpb.GetDownloadLogUrlRequest{}},
		{"rotated", &logpb.GetDownloadLogUrlRequest{IncludeRotated: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.client.GetDownloadLogUrl(context.Background(), tc.req)
			if err != nil {
				t.Fatalf("GetDownloadLogUrl: %v", err)
			}
			if got := len(resp.Files); got != 0 {
				t.Errorf("len(Files) = %d, want 0", got)
			}
		})
	}
}

// A tampered token in the URL is rejected with 401.
func TestSystem_Download_TamperedURL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.log", "contents")
	h := newHarness(t, config.Root{
		LogFilePath: filepath.Join(dir, "*.log"),
	})

	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	good := resp.Files[0].Url

	// Tamper the last character of the URL. With base64url encoding, flipping one
	// character of the token corrupts either the envelope or the signature.
	tampered := good[:len(good)-1] + altChar(good[len(good)-1])

	httpResp := h.fetch(t, tampered)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", httpResp.StatusCode)
	}
}

// An expired URL is rejected with 401. Uses testing/synctest to advance
// virtual time past the TTL without sleeping in real time.
func TestSystem_Download_ExpiredURL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "app.log", "contents")
		h := newHarness(t, config.Root{
			LogFilePath: filepath.Join(dir, "*.log"),
		})

		resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
		if err != nil {
			t.Fatalf("GetDownloadLogUrl: %v", err)
		}
		urlStr := resp.Files[0].Url

		// Cross the 15-minute TTL configured by newDownloadRouter.
		time.Sleep(16 * time.Minute)

		httpResp := h.fetch(t, urlStr)
		defer httpResp.Body.Close()
		if httpResp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", httpResp.StatusCode)
		}
	})
}

// A URL issued under config A points at files in dirA; after reapply
// switches LogDir to dirB, the old URL must no longer serve dirA's file
// (the allowedDir check applies the current config).
func TestSystem_Download_AllowedDirEnforcedAfterReload(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeFile(t, dirA, "app.log", "secret-A")
	writeFile(t, dirB, "app.log", "secret-B")

	cfgA := config.Root{
		LogFilePath: filepath.Join(dirA, "*.log"),
		LogDir:      dirA,
	}
	cfgB := config.Root{
		LogFilePath: filepath.Join(dirB, "*.log"),
		LogDir:      dirB,
	}

	h := newHarness(t, cfgA)
	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl(A): %v", err)
	}
	urlForA := resp.Files[0].Url

	// Sanity: under config A the URL works.
	if got := h.fetch(t, urlForA).StatusCode; got != http.StatusOK {
		t.Fatalf("under config A: status = %d, want 200", got)
	}

	if err := h.reapply(cfgB); err != nil {
		t.Fatalf("reapply(B): %v", err)
	}

	// The token in urlForA still points at dirA/app.log, but the System's
	// allowedDir is now dirB, so the fetch should be rejected.
	got := h.fetch(t, urlForA)
	defer got.Body.Close()
	if got.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(got.Body)
		t.Errorf("after reapply: status = 200, body=%q — expected forbidden/non-2xx", body)
	}
}

// Downloads are disabled when configuration is incomplete (LogFilePath empty):
// GetDownloadLogUrlFunc is never assigned, so the default ModelServer returns
// codes.Unimplemented.
func TestSystem_Download_Disabled(t *testing.T) {
	h := newHarness(t, config.Root{})

	_, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err == nil {
		t.Fatal("expected error from GetDownloadLogUrl, got nil")
	}
	if got := status.Code(err); got != codes.Unimplemented {
		t.Errorf("status code = %s, want Unimplemented", got)
	}
}

// The handler stays addressable across an idempotent reapply — a URL minted
// before reapply still serves successfully afterward (with the same config).
func TestSystem_Download_HandlerSurvivesReapply(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.log", "contents")
	cfg := config.Root{LogFilePath: filepath.Join(dir, "*.log")}

	h := newHarness(t, cfg)
	resp, err := h.client.GetDownloadLogUrl(context.Background(), &logpb.GetDownloadLogUrlRequest{})
	if err != nil {
		t.Fatalf("GetDownloadLogUrl: %v", err)
	}
	urlStr := resp.Files[0].Url
	if got := h.fetch(t, urlStr).StatusCode; got != http.StatusOK {
		t.Fatalf("before reapply: status = %d, want 200", got)
	}

	if err := h.reapply(cfg); err != nil {
		t.Fatalf("reapply: %v", err)
	}

	httpResp := h.fetch(t, urlStr)
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Errorf("after reapply: status = %d, want 200", httpResp.StatusCode)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// altChar returns a different printable URL-safe character than c, used by
// TestSystem_Download_TamperedURL to flip a single byte of a token.
func altChar(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}
