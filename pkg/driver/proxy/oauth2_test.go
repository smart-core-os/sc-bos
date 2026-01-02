package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/proxy/config"
)

func TestOAuth2Credentials_GetRequestMetadata(t *testing.T) {
	t.Run("fetches new token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected application/x-www-form-urlencoded, got %s", ct)
			}

			err := r.ParseForm()
			if err != nil {
				t.Fatalf("ParseForm error: %v", err)
			}

			if got := r.Form.Get("grant_type"); got != "client_credentials" {
				t.Errorf("grant_type = %s, want client_credentials", got)
			}
			if got := r.Form.Get("client_id"); got != "test-client" {
				t.Errorf("client_id = %s, want test-client", got)
			}
			if got := r.Form.Get("client_secret"); got != "test-secret" {
				t.Errorf("client_secret = %s, want test-secret", got)
			}

			resp := map[string]interface{}{
				"access_token": "test-token-123",
				"expires_in":   3600,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		creds := &oauth2Credentials{
			client:       server.Client(),
			url:          server.URL,
			clientId:     "test-client",
			clientSecret: "test-secret",
			reqLimit:     make(chan struct{}, 1),
		}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)
		if err != nil {
			t.Fatalf("GetRequestMetadata error: %v", err)
		}

		if got := metadata["authorization"]; got != "Bearer test-token-123" {
			t.Errorf("authorization = %s, want Bearer test-token-123", got)
		}

		// Verify token is cached
		if creds.token != "test-token-123" {
			t.Errorf("token = %s, want test-token-123", creds.token)
		}
		if !creds.expires.After(time.Now()) {
			t.Error("expires should be in the future")
		}
	})

	t.Run("reuses valid token", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			resp := map[string]interface{}{
				"access_token": "test-token",
				"expires_in":   3600,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		creds := &oauth2Credentials{
			client:       server.Client(),
			url:          server.URL,
			clientId:     "test-client",
			clientSecret: "test-secret",
			reqLimit:     make(chan struct{}, 1),
			token:        "cached-token",
			expires:      time.Now().Add(1 * time.Hour),
		}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)
		if err != nil {
			t.Fatalf("GetRequestMetadata error: %v", err)
		}

		if got := metadata["authorization"]; got != "Bearer cached-token" {
			t.Errorf("authorization = %s, want Bearer cached-token", got)
		}

		if callCount != 0 {
			t.Errorf("server called %d times, want 0", callCount)
		}
	})

	t.Run("refreshes expired token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"access_token": "new-token",
				"expires_in":   3600,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		creds := &oauth2Credentials{
			client:       server.Client(),
			url:          server.URL,
			clientId:     "test-client",
			clientSecret: "test-secret",
			reqLimit:     make(chan struct{}, 1),
			token:        "expired-token",
			expires:      time.Now().Add(-1 * time.Hour),
		}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)
		if err != nil {
			t.Fatalf("GetRequestMetadata error: %v", err)
		}

		if got := metadata["authorization"]; got != "Bearer new-token" {
			t.Errorf("authorization = %s, want Bearer new-token", got)
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid credentials"))
		}))
		defer server.Close()

		creds := &oauth2Credentials{
			client:       server.Client(),
			url:          server.URL,
			clientId:     "test-client",
			clientSecret: "wrong-secret",
			reqLimit:     make(chan struct{}, 1),
		}

		ctx := context.Background()
		_, err := creds.GetRequestMetadata(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		creds := &oauth2Credentials{
			client:       &http.Client{},
			url:          "http://localhost:9999",
			clientId:     "test-client",
			clientSecret: "test-secret",
			reqLimit:     make(chan struct{}, 1),
		}

		// Fill the reqLimit to block
		creds.reqLimit <- struct{}{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := creds.GetRequestMetadata(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != context.Canceled {
			t.Errorf("error = %v, want context.Canceled", err)
		}
	})
}

func TestOAuth2Credentials_RequireTransportSecurity(t *testing.T) {
	creds := &oauth2Credentials{}
	if !creds.RequireTransportSecurity() {
		t.Error("RequireTransportSecurity() = false, want true")
	}
}

func TestNewOAuth2Credentials(t *testing.T) {
	t.Run("reads secret from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")

		err := os.WriteFile(secretFile, []byte("my-secret\n"), 0600)
		if err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		cfg := config.OAuth2{
			TokenEndpoint:    "https://auth.example.com/token",
			ClientID:         "test-client",
			ClientSecretFile: secretFile,
		}

		creds, err := newOAuth2Credentials(cfg, &http.Client{})
		if err != nil {
			t.Fatalf("newOAuth2Credentials error: %v", err)
		}

		if creds.clientId != "test-client" {
			t.Errorf("clientId = %s, want test-client", creds.clientId)
		}
		if creds.clientSecret != "my-secret" {
			t.Errorf("clientSecret = %s, want my-secret", creds.clientSecret)
		}
		if creds.url != "https://auth.example.com/token" {
			t.Errorf("url = %s, want https://auth.example.com/token", creds.url)
		}
	})

	t.Run("handles missing secret file", func(t *testing.T) {
		cfg := config.OAuth2{
			TokenEndpoint:    "https://auth.example.com/token",
			ClientID:         "test-client",
			ClientSecretFile: "/nonexistent/secret.txt",
		}

		_, err := newOAuth2Credentials(cfg, &http.Client{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFetchNewToken(t *testing.T) {
	t.Run("successful token fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"access_token": "fetched-token",
				"expires_in":   1800,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		ctx := context.Background()
		token, expires, err := fetchNewToken(ctx, server.Client(), server.URL, "client", "secret")
		if err != nil {
			t.Fatalf("fetchNewToken error: %v", err)
		}

		if token != "fetched-token" {
			t.Errorf("token = %s, want fetched-token", token)
		}

		expectedExpiry := time.Now().Add(1800 * time.Second)
		if expires.Before(expectedExpiry.Add(-10*time.Second)) || expires.After(expectedExpiry.Add(10*time.Second)) {
			t.Errorf("expires = %v, want around %v", expires, expectedExpiry)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		}))
		defer server.Close()

		ctx := context.Background()
		_, _, err := fetchNewToken(ctx, server.Client(), server.URL, "client", "secret")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		ctx := context.Background()
		_, _, err := fetchNewToken(ctx, server.Client(), server.URL, "client", "secret")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRequestMetadata(t *testing.T) {
	metadata := requestMetadata("test-token-xyz")

	if len(metadata) != 1 {
		t.Errorf("len(metadata) = %d, want 1", len(metadata))
	}

	if got := metadata["authorization"]; got != "Bearer test-token-xyz" {
		t.Errorf("authorization = %s, want Bearer test-token-xyz", got)
	}
}
