package paxton

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// fakeSystemCheck records the last state reported to it.
type fakeSystemCheck struct {
	mu      sync.Mutex
	failed  error
	running int
}

func (f *fakeSystemCheck) Dispose() {}

func (f *fakeSystemCheck) MarkRunning() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = nil
	f.running++
}

func (f *fakeSystemCheck) MarkFailed(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = err
}

func (f *fakeSystemCheck) lastFailure() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failed
}

// recordedRequest is what a token handler saw.
type recordedRequest struct {
	method      string
	path        string
	contentType string
	body        []byte
}

// setupTokenClient returns a Client pointed at a test server. handler serves every request
// other than the auth token request; the requests it receives are recorded in order.
func setupTokenClient(t *testing.T, handler http.HandlerFunc) (*Client, *fakeSystemCheck, *[]recordedRequest) {
	t.Helper()

	var (
		mu       sync.Mutex
		requests []recordedRequest
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuthResponse{
			AccessToken:    "test-token",
			ExpiryDatetime: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		requests = append(requests, recordedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			body:        body,
		})
		mu.Unlock()
		r.Body = io.NopCloser(bytes.NewReader(body))
		handler(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	check := &fakeSystemCheck{}
	c := NewClientFromConfig(config.Root{BaseUrl: srv.URL}, zap.NewNop(), check)
	c.cli.RetryMax = 0
	c.cli.Logger = nil

	return c, check, &requests
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestClient_AddUserToken(t *testing.T) {
	c, _, requests := setupTokenClient(t, func(w http.ResponseWriter, r *http.Request) {
		var in UserToken
		_ = json.NewDecoder(r.Body).Decode(&in)
		in.ID = 42
		writeJSON(w, http.StatusCreated, in)
	})

	got, err := c.AddUserToken(context.Background(), 7, UserToken{
		ID:         99, // ignored
		TokenType:  TokenTypeProxCard,
		TokenValue: "12345678",
	})
	require.NoError(t, err)
	assert.Equal(t, UserToken{ID: 42, TokenType: TokenTypeProxCard, TokenValue: "12345678"}, got)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, http.MethodPost, req.method)
	assert.Equal(t, "/api/v1/users/7/tokens", req.path)
	assert.Equal(t, "application/json", req.contentType)
	assert.JSONEq(t, `{"TokenType":"ProxCard","TokenValue":"12345678","IsLost":false}`, string(req.body))
}

func TestClient_UpdateUserToken(t *testing.T) {
	c, _, requests := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.UpdateUserToken(context.Background(), 7, 42, UserToken{
		TokenType:  TokenTypeKeyfob,
		TokenValue: "abc",
		IsLost:     true,
	})
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, http.MethodPut, req.method)
	assert.Equal(t, "/api/v1/users/7/tokens/42", req.path)
	assert.Equal(t, "application/json", req.contentType)
	assert.JSONEq(t, `{"Id":42,"TokenType":"Keyfob","TokenValue":"abc","IsLost":true}`, string(req.body))
}

func TestClient_GetUserTokens(t *testing.T) {
	// Net2 sends camelCase names; decoding must still match the PascalCase tags.
	c, _, requests := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"id":1,"tokenType":"ProxCard","tokenValue":"111","isLost":false},
			{"id":2,"tokenType":"VehicleNumberPlate","tokenValue":"AB12CDE","isLost":true}
		]`)
	})

	got, err := c.GetUserTokens(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, []UserToken{
		{ID: 1, TokenType: TokenTypeProxCard, TokenValue: "111"},
		{ID: 2, TokenType: TokenTypeVehicleNumberPlate, TokenValue: "AB12CDE", IsLost: true},
	}, got)

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodGet, (*requests)[0].method)
	assert.Equal(t, "/api/v1/users/7/tokens", (*requests)[0].path)
}

func TestClient_GetUserToken(t *testing.T) {
	c, _, requests := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, UserToken{ID: 42, TokenType: TokenTypeWatchprox, TokenValue: "w"})
	})

	got, err := c.GetUserToken(context.Background(), 7, 42)
	require.NoError(t, err)
	assert.Equal(t, UserToken{ID: 42, TokenType: TokenTypeWatchprox, TokenValue: "w"}, got)

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodGet, (*requests)[0].method)
	assert.Equal(t, "/api/v1/users/7/tokens/42", (*requests)[0].path)
}

func TestClient_DeleteUserToken(t *testing.T) {
	c, _, requests := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	require.NoError(t, c.DeleteUserToken(context.Background(), 7, 42))

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodDelete, (*requests)[0].method)
	assert.Equal(t, "/api/v1/users/7/tokens/42", (*requests)[0].path)
	assert.Empty(t, (*requests)[0].body)
}

// Token requests are made for callers, so neither their failures nor their successes reach the system check.
func TestClient_TokenRequestsLeaveSystemCheck(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c, check, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "User not found", status)
			})
			pollErr := errors.New("events poll failed")
			check.MarkFailed(pollErr)

			_, err := c.GetUserTokens(context.Background(), 404)
			var statusErr *StatusError
			require.ErrorAs(t, err, &statusErr, "a %d should be a StatusError, 5xx included", status)
			assert.Equal(t, status, statusErr.StatusCode)
			assert.Contains(t, statusErr.Body, "User not found")
			assert.Equal(t, pollErr, check.lastFailure())
		})
	}

	t.Run("success", func(t *testing.T) {
		c, check, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []UserToken{})
		})
		pollErr := errors.New("events poll failed")
		check.MarkFailed(pollErr)

		_, err := c.GetUserTokens(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, pollErr, check.lastFailure(), "a successful token request mustn't clear a poll's fault")
	})
}

// The polls go through Do, which marks the system check failed for any failure, 400 and 404 included.
func TestClient_DoMarksSystemCheck(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c, check, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", status)
			})

			_, err := c.GetUsers(context.Background())
			var statusErr *StatusError
			require.ErrorAs(t, err, &statusErr)
			assert.Equal(t, status, statusErr.StatusCode)
			assert.Equal(t, err, check.lastFailure())
		})
	}
}

func TestClient_GetUserTokens_emptyBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c, _, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})
			got, err := c.GetUserTokens(context.Background(), 7)
			require.NoError(t, err)
			assert.Empty(t, got)

			// a single token can't be empty
			_, err = c.GetUserToken(context.Background(), 7, 1)
			assert.Error(t, err)
		})
	}
}

// A caller with a deadline mustn't queue behind a token request that Net2 never answers.
func TestClient_authStall(t *testing.T) {
	release := make(chan struct{})
	var authCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", func(w http.ResponseWriter, r *http.Request) {
		authCalls.Add(1)
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	check := &fakeSystemCheck{}
	c := NewClientFromConfig(config.Root{BaseUrl: srv.URL}, zap.NewNop(), check)
	c.cli.RetryMax = 0
	c.cli.Logger = nil
	c.authTimeout = 500 * time.Millisecond

	// a poll, with no deadline, starts refreshing the token and holds the lock
	pollDone := make(chan error, 1)
	go func() {
		_, err := c.GetUsers(context.Background())
		pollDone <- err
	}()
	require.Eventually(t, func() bool { return authCalls.Load() == 1 }, time.Second, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.GetUserTokens(ctx, 7)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 400*time.Millisecond, "should give up at its own deadline, not wait for the token request")

	// the stalled token request gives up by itself, and the poll reports it
	select {
	case err := <-pollDone:
		assert.ErrorIs(t, err, errAuthTimeout)
	case <-time.After(5 * time.Second):
		t.Fatal("token request never timed out")
	}
	assert.ErrorIs(t, check.lastFailure(), errAuthTimeout)
}

func TestClient_TokenValidation(t *testing.T) {
	var calls atomic.Int32
	c, _, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	})

	tests := []struct {
		name  string
		token UserToken
	}{
		{name: "empty value", token: UserToken{TokenType: TokenTypeProxCard}},
		{name: "empty type", token: UserToken{TokenValue: "1"}},
		{name: "unknown type", token: UserToken{TokenType: "AppleWallet", TokenValue: "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.AddUserToken(context.Background(), 7, tt.token)
			assert.Error(t, err)
			assert.Error(t, c.UpdateUserToken(context.Background(), 7, 42, tt.token))
		})
	}
	assert.Zero(t, calls.Load())
}

func TestStatusError_Error(t *testing.T) {
	err := error(&StatusError{StatusCode: 404, Body: "nope"})
	assert.EqualError(t, err, "unexpected status 404: nope")
	assert.True(t, errors.As(err, new(*StatusError)))
}
