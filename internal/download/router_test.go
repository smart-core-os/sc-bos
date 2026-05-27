package download

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return NewRouter(NewHMACSigner(key), WithBaseURL("/download"))
}

func TestRouter_RoundTrip(t *testing.T) {
	rt := newTestRouter(t)
	var gotPayload []byte
	rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
		gotPayload = PayloadFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response-body"))
	})

	dlURL, _, err := rt.GenerateURL("test", []byte("payload-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dlURL, "/download/") {
		t.Fatalf("unexpected url prefix: %s", dlURL)
	}

	req := httptest.NewRequest("GET", dlURL, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200; body=%s", rec.Code, rec.Body.String())
	}
	if string(gotPayload) != "payload-bytes" {
		t.Fatalf("payload: got %q want %q", gotPayload, "payload-bytes")
	}
	if rec.Body.String() != "response-body" {
		t.Fatalf("body: got %q want %q", rec.Body.String(), "response-body")
	}
}

func TestRouter_TamperedEnvelope(t *testing.T) {
	rt := newTestRouter(t)
	rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be called")
	})

	dlURL, _, err := rt.GenerateURL("test", []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	prefix := "/download/"
	body := dlURL[len(prefix):]
	delim := strings.Index(body, ".")
	envBytes, err := base64.RawURLEncoding.DecodeString(body[:delim])
	if err != nil {
		t.Fatal(err)
	}
	envBytes[0] ^= 0xff
	tampered := prefix + base64.RawURLEncoding.EncodeToString(envBytes) + body[delim:]

	req := httptest.NewRequest("GET", tampered, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
}

func TestRouter_TamperedSignature(t *testing.T) {
	rt := newTestRouter(t)
	rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be called")
	})

	dlURL, _, err := rt.GenerateURL("test", []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	prefix := "/download/"
	body := dlURL[len(prefix):]
	delim := strings.Index(body, ".")
	sigBytes, err := base64.RawURLEncoding.DecodeString(body[delim+1:])
	if err != nil {
		t.Fatal(err)
	}
	sigBytes[0] ^= 0xff
	tampered := prefix + body[:delim+1] + base64.RawURLEncoding.EncodeToString(sigBytes)

	req := httptest.NewRequest("GET", tampered, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
}

func TestRouter_Expired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rt := newTestRouter(t)
		rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("handler should not be called")
		})

		dlURL, _, err := rt.GenerateURL("test", []byte("p"))
		if err != nil {
			t.Fatal(err)
		}

		// Cross the expiry boundary (default TTL is 5 minutes).
		time.Sleep(6 * time.Minute)

		req := httptest.NewRequest("GET", dlURL, nil)
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d want 401", rec.Code)
		}
	})
}

func TestRouter_WithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			t.Fatal(err)
		}
		rt := NewRouter(NewHMACSigner(key), WithBaseURL("/download"), WithTTL(5*time.Minute))
		rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {})

		dlURL, expiresAt, err := rt.GenerateURL("test", nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := time.Now().Add(5 * time.Minute); !expiresAt.Equal(want) {
			t.Fatalf("expiresAt: got %v want %v", expiresAt, want)
		}

		time.Sleep(5*time.Minute + time.Second)
		req := httptest.NewRequest("GET", dlURL, nil)
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("after TTL: got %d want 401", rec.Code)
		}
	})
}

func TestRouter_NotYetExpired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rt := newTestRouter(t)
		called := false
		rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		dlURL, _, err := rt.GenerateURL("test", []byte("p"))
		if err != nil {
			t.Fatal(err)
		}

		// Just shy of the default TTL.
		time.Sleep(4 * time.Minute)
		req := httptest.NewRequest("GET", dlURL, nil)
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 + called; got code=%d called=%v", rec.Code, called)
		}
	})
}

func TestRouter_MalformedTokens(t *testing.T) {
	rt := newTestRouter(t)
	cases := map[string]string{
		"no delimiter":      "/download/noDelimiterAtAll",
		"two delimiters":    "/download/two.delims.here",
		"all delimiters":    "/download/...",
		"non-base64 halves": "/download/notbase64!.notbase64!",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s: status got %d want 401", path, rec.Code)
			}
		})
	}
}

func TestRouter_UnknownType(t *testing.T) {
	rt := newTestRouter(t)
	// Sign for a type that we then don't register.
	dlURL, _, err := rt.GenerateURL("unregistered-type", []byte("p"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", dlURL, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rec.Code)
	}
}

func TestRouter_NoTokenSegment(t *testing.T) {
	rt := newTestRouter(t)

	cases := map[string]string{
		"empty token":         "/download/",
		"outside base prefix": "/elsewhere/something",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status: got %d want 404", rec.Code)
			}
		})
	}
}

func TestRouter_Streaming(t *testing.T) {
	rt := newTestRouter(t)
	rt.HandleFunc("stream", func(w http.ResponseWriter, r *http.Request) {
		for range 3 {
			_, _ = w.Write([]byte("chunk\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})

	dlURL, _, err := rt.GenerateURL("stream", nil)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(rt)
	defer srv.Close()
	resp, err := http.Get(srv.URL + dlURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "chunk\nchunk\nchunk\n" {
		t.Fatalf("body: got %q want %q", body, "chunk\nchunk\nchunk\n")
	}
}

func TestRouter_ContextCancellation(t *testing.T) {
	rt := newTestRouter(t)
	started := make(chan struct{})
	saw := make(chan struct{})
	rt.HandleFunc("cancel", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(saw)
	})

	dlURL, _, err := rt.GenerateURL("cancel", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, "GET", dlURL, nil)
	rec := httptest.NewRecorder()
	go rt.ServeHTTP(rec, req)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start within 1s")
	}
	cancel()
	select {
	case <-saw:
	case <-time.After(time.Second):
		t.Fatal("handler did not observe context cancellation within 1s")
	}
}

func TestRouter_BaseURL(t *testing.T) {
	cases := []struct {
		name       string
		base       string
		wantPrefix string // GenerateURL must return a URL with this prefix, followed by the token
	}{
		{"empty", "", "/"},
		{"path no slash", "/foo", "/foo/"},
		{"path trailing slash", "/foo/", "/foo/"},
		{"host no path", "http://example.com", "http://example.com/"},
		{"host root", "http://example.com/", "http://example.com/"},
		{"host with path", "http://example.com/foo", "http://example.com/foo/"},
		{"host path trailing slash", "http://example.com/foo/", "http://example.com/foo/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				t.Fatal(err)
			}
			rt := NewRouter(NewHMACSigner(key), WithBaseURL(tc.base))

			var gotPayload []byte
			rt.HandleFunc("test", func(w http.ResponseWriter, r *http.Request) {
				gotPayload = PayloadFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// URL generation: must have the expected prefix and then a token.
			dlURL, _, err := rt.GenerateURL("test", []byte("p"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(dlURL, tc.wantPrefix) {
				t.Fatalf("URL %q does not start with prefix %q", dlURL, tc.wantPrefix)
			}
			token := dlURL[len(tc.wantPrefix):]
			if !strings.Contains(token, ".") {
				t.Fatalf("URL %q does not end with a <envelope>.<sig> token after prefix %q", dlURL, tc.wantPrefix)
			}

			// Request routing: hand the generated URL back to the router.
			// httptest.NewRequest accepts either path-only or fully-qualified
			// URLs; for path-only it defaults Host to "example.com".
			req := httptest.NewRequest("GET", dlURL, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d want 200; body=%s", rec.Code, rec.Body.String())
			}
			if string(gotPayload) != "p" {
				t.Fatalf("payload: got %q want %q", gotPayload, "p")
			}
		})
	}
}

func TestRouter_BaseURL_RejectsOutsidePrefix(t *testing.T) {
	cases := map[string]struct {
		base    string
		badPath string // path that should not match the router's prefix
	}{
		"path base": {"/foo", "/bar/anything"},
		"host base": {"http://example.com/foo", "/bar/anything"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				t.Fatal(err)
			}
			rt := NewRouter(NewHMACSigner(key), WithBaseURL(tc.base))
			rt.HandleFunc("test", func(http.ResponseWriter, *http.Request) {
				t.Errorf("handler should not be called")
			})

			req := httptest.NewRequest("GET", tc.badPath, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status: got %d want 404", rec.Code)
			}
		})
	}
}

func TestRouter_DuplicateHandleReplaces(t *testing.T) {
	rt := newTestRouter(t)
	rt.HandleFunc("dup", func(http.ResponseWriter, *http.Request) {
		t.Errorf("first handler should have been replaced")
	})
	var called bool
	rt.HandleFunc("dup", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	dlURL, _, err := rt.GenerateURL("dup", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", dlURL, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if !called {
		t.Fatal("expected replacement handler to be invoked")
	}
}

func TestRouter_SecurityHeadersOnEveryResponse(t *testing.T) {
	rt := newTestRouter(t)
	rt.HandleFunc("ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	okURL, _, err := rt.GenerateURL("ok", nil)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"success (200)":    okURL,
		"missing token":    "/download/",
		"unknown type":     mustGenerateURL(t, rt, "unregistered"),
		"malformed token":  "/download/not-a-token",
		"tampered token":   mustTamperToken(t, okURL),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options: got %q want nosniff (status=%d)", got, rec.Code)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control: got %q want no-store (status=%d)", got, rec.Code)
			}
		})
	}
}

func mustGenerateURL(t *testing.T, rt *Router, typ string) string {
	t.Helper()
	u, _, err := rt.GenerateURL(typ, nil)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustTamperToken(t *testing.T, dlURL string) string {
	t.Helper()
	prefix := "/download/"
	body := dlURL[len(prefix):]
	delim := strings.Index(body, ".")
	sigBytes, err := base64.RawURLEncoding.DecodeString(body[delim+1:])
	if err != nil {
		t.Fatal(err)
	}
	sigBytes[0] ^= 0xff
	return prefix + body[:delim+1] + base64.RawURLEncoding.EncodeToString(sigBytes)
}

func TestRouter_TypeCannotBeForgedViaURL(t *testing.T) {
	// A token issued for type A should not be dispatchable as type B by URL
	// manipulation: the type is inside the signed envelope, not the URL.
	rt := newTestRouter(t)
	var calledFor string
	rt.HandleFunc("a", func(w http.ResponseWriter, r *http.Request) {
		calledFor = "a"
	})
	rt.HandleFunc("b", func(w http.ResponseWriter, r *http.Request) {
		calledFor = "b"
	})

	dlURL, _, err := rt.GenerateURL("a", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", dlURL, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if calledFor != "a" {
		t.Fatalf("expected handler 'a' to be invoked; got %q", calledFor)
	}
}
