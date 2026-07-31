package accesstoken

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auth/token"
)

func TestTokenServer(t *testing.T) {
	var (
		users    testMemoryVerifier
		services testMemoryVerifier
	)
	users.add(t, SecretData{TenantID: "user1", SystemRoles: []string{"admin"}}, "password123")
	users.add(t, SecretData{TenantID: "user2"}, "password456")
	services.add(t, SecretData{TenantID: "service1", SystemRoles: []string{"admin"}}, "secret1", "secret2")
	services.add(t, SecretData{
		TenantID: "service2",
		Permissions: []token.PermissionAssignment{
			LegacyZonePermission("foo"),
		},
	}, "secret3")
	services.add(t, SecretData{TenantID: "service3"}, "secret4")

	server, err := NewServer("test",
		WithLogger(zap.NewNop()),
		WithPasswordFlow(&users, 10*time.Minute),
		WithClientCredentialFlow(&services, 10*time.Minute),
	)
	if err != nil {
		t.Fatalf("NewServer %v", err)
	}
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	client := httpServer.Client()

	type testCase struct {
		grantType      string
		username       string
		password       string
		clientID       string
		clientSecret   string
		expectHTTPCode int
		expectErrCode  string
	}
	testCases := map[string]testCase{
		"unknown_grant_type": {
			grantType:      "foobar",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "unsupported_grant_type",
		},
		"client_credentials_success_1": {
			grantType:      "client_credentials",
			clientID:       "service1",
			clientSecret:   "secret1",
			expectHTTPCode: http.StatusOK,
		},
		"client_credentials_success_2": {
			grantType:      "client_credentials",
			clientID:       "service1",
			clientSecret:   "secret2",
			expectHTTPCode: http.StatusOK,
		},
		"client_credentials_zone_success": {
			grantType:      "client_credentials",
			clientID:       "service2",
			clientSecret:   "secret3",
			expectHTTPCode: http.StatusOK,
		},
		"client_credentials_invalid_secret": {
			grantType:      "client_credentials",
			clientID:       "service1",
			clientSecret:   "invalid_secret",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "invalid_grant",
		},
		"client_credentials_unknown_client": {
			grantType:      "client_credentials",
			clientID:       "unknown_service",
			clientSecret:   "dummy",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "invalid_grant",
		},
		"client_credentials_no_permissions": {
			grantType:      "client_credentials",
			clientID:       "service3",
			clientSecret:   "secret4",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "unauthorized_client",
		},
		"user_password_success": {
			grantType:      "password",
			username:       "user1",
			password:       "password123",
			expectHTTPCode: http.StatusOK,
		},
		"user_password_no_roles": {
			grantType:      "password",
			username:       "user2",
			password:       "password456",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "unauthorized_client",
		},
		"user_password_invalid_credentials": {
			grantType:      "password",
			username:       "user1",
			password:       "wrongpassword",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "invalid_grant",
		},
		"user_password_unknown_user": {
			grantType:      "password",
			username:       "unknown_user",
			password:       "dummy",
			expectHTTPCode: http.StatusBadRequest,
			expectErrCode:  "invalid_grant",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			values := url.Values{
				"grant_type": {tc.grantType},
			}
			switch tc.grantType {
			case "client_credentials":
				values.Add("client_id", tc.clientID)
				values.Add("client_secret", tc.clientSecret)
			case "password":
				values.Add("username", tc.username)
				values.Add("password", tc.password)
			}

			resp, err := client.PostForm(httpServer.URL, values)
			if err != nil {
				t.Fatalf("PostForm %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("ReadAll %v", err)
			}

			if resp.StatusCode != tc.expectHTTPCode {
				t.Errorf("expected HTTP status %d, got %d", tc.expectHTTPCode, resp.StatusCode)
			}
			if resp.StatusCode == http.StatusOK {
				var token tokenSuccessResponse
				err = json.Unmarshal(body, &token)
				if err != nil {
					t.Fatalf("Unmarshal success response %v", err)
				}
			} else {
				var tokErr tokenError
				err = json.Unmarshal(body, &tokErr)
				if err != nil {
					t.Fatalf("Unmarshal error response %v", err)
				}
				if tokErr.ErrorName != tc.expectErrCode {
					t.Errorf("expected error code %s, got %s", tc.expectErrCode, tokErr.ErrorName)
				}
			}
		})
	}
}

// A Server must accept the tokens it issues. jwt.ParseSigned rejects everything when handed an
// empty algorithm list, so while the permitted algorithms were configured separately from the key
// a caller could build a server that handed out a token over HTTP and then refused that very
// token — see SCB-1393.
func TestTokenServer_validatesOwnTokens(t *testing.T) {
	var services testMemoryVerifier
	services.add(t, SecretData{TenantID: "service1", SystemRoles: []string{"admin"}}, "secret1")

	server, err := NewServer("test",
		WithLogger(zap.NewNop()),
		WithClientCredentialFlow(&services, 10*time.Minute),
	)
	if err != nil {
		t.Fatalf("NewServer %v", err)
	}
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	resp, err := httpServer.Client().PostForm(httpServer.URL, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"service1"},
		"client_secret": {"secret1"},
	})
	if err != nil {
		t.Fatalf("PostForm %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var issued tokenSuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&issued); err != nil {
		t.Fatalf("Decode success response %v", err)
	}

	claims, err := server.TokenValidator().ValidateAccessToken(t.Context(), issued.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken rejected a token this server issued: %v", err)
	}
	if claims.Subject != "service1" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "service1")
	}
}

// The signing key decides which algorithm tokens are issued and validated with, so NewServer only
// accepts a key it can actually use: a non-empty []byte secret carrying HS256.
func TestNewServer_rejectsUnusableSigningKey(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	for name, key := range map[string]jose.SigningKey{
		"asymmetric algorithm": {Algorithm: jose.RS256, Key: secret},
		"other hmac algorithm": {Algorithm: jose.HS512, Key: secret},
		"missing algorithm":    {Key: secret},
		"wrong key type":       {Algorithm: jose.HS256, Key: "not-bytes"},
		"empty key":            {Algorithm: jose.HS256, Key: []byte{}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewServer("test", WithSigningKey(key)); err == nil {
				t.Error("NewServer accepted a signing key it cannot issue tokens with")
			}
		})
	}
}

// The generated fallback key must satisfy the same validation the caller's key is held to.
func TestNewServer_generatesUsableKeyWhenUnset(t *testing.T) {
	server, err := NewServer("test")
	if err != nil {
		t.Fatalf("NewServer %v", err)
	}
	if got := server.tokens.Key.Algorithm; got != jose.HS256 {
		t.Errorf("generated key algorithm = %q, want %q", got, jose.HS256)
	}
}

type testMemoryVerifier struct {
	MemoryVerifier
}

func (v *testMemoryVerifier) add(t *testing.T, data SecretData, secrets ...string) {
	t.Helper()
	if err := v.AddRecord(data); err != nil {
		t.Fatalf("AddRecord %v", err)
	}
	for _, secret := range secrets {
		if _, err := v.AddSecret(data.TenantID, secret); err != nil {
			t.Fatalf("AddSecret %v", err)
		}
	}
}
