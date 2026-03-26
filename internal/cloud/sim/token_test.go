package sim

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestTokenIssuer_RoundTrip(t *testing.T) {
	ti, err := newTokenIssuer()
	if err != nil {
		t.Fatalf("newTokenIssuer: %v", err)
	}

	nodeID := int64(42)
	token, expiresIn, err := ti.issue(nodeID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresIn != int(defaultTokenExpiry.Seconds()) {
		t.Errorf("expiresIn = %d, want %d", expiresIn, int(defaultTokenExpiry.Seconds()))
	}

	claims, err := ti.validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.NodeID != nodeID {
		t.Errorf("nodeId = %d, want %d", claims.NodeID, nodeID)
	}
	if claims.Type != "bos_node" {
		t.Errorf("type = %q, want %q", claims.Type, "bos_node")
	}
}

func TestTokenIssuer_DifferentKeyRejectsToken(t *testing.T) {
	ti1, _ := newTokenIssuer()
	ti2, _ := newTokenIssuer()

	token, _, err := ti1.issue(1)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = ti2.validate(token)
	if err == nil {
		t.Error("expected error validating token with different key")
	}
}

func TestTokenIssuer_ExpiredToken(t *testing.T) {
	ti, _ := newTokenIssuer()
	ti.expiry = -1 * time.Second // issue already-expired tokens

	token, _, err := ti.issue(1)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = ti.validate(token)
	if err == nil {
		t.Error("expected error validating expired token")
	}
}

func tokenURL(base string) string { return base + "/v1/device/token" }

func TestHandleToken_Success(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()

	// Create a node to get valid credentials
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "s"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var created CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "n",
		"siteId":   sid(site.ID),
	}, &created)
	assertStatus(t, resp, http.StatusCreated)

	encodedSecret := base64.StdEncoding.EncodeToString(created.Secret)
	resp, err := client.PostForm(tokenURL(ts.URL), url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {strconv.FormatInt(created.ID, 10)},
		"client_secret": {encodedSecret},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusOK)

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tokenResp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if tokenResp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want %q", tokenResp.TokenType, "Bearer")
	}
	if tokenResp.ExpiresIn <= 0 {
		t.Errorf("expires_in = %d, want > 0", tokenResp.ExpiresIn)
	}
}

func TestHandleToken_WrongGrantType(t *testing.T) {
	ts := newTestServer(t)

	resp, err := ts.Client().PostForm(tokenURL(ts.URL), url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"id"},
		"client_secret": {"secret"},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestHandleToken_MissingCredentials(t *testing.T) {
	ts := newTestServer(t)

	resp, err := ts.Client().PostForm(tokenURL(ts.URL), url.Values{
		"grant_type": {"client_credentials"},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestHandleToken_WrongSecret(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "s"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var created CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "n",
		"siteId":   sid(site.ID),
	}, &created)
	assertStatus(t, resp, http.StatusCreated)

	wrongSecret := base64.StdEncoding.EncodeToString([]byte("wrong"))
	resp, err := client.PostForm(tokenURL(ts.URL), url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {strconv.FormatInt(created.ID, 10)},
		"client_secret": {wrongSecret},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestHandleToken_InvalidBase64Secret(t *testing.T) {
	ts := newTestServer(t)

	resp, err := ts.Client().PostForm(tokenURL(ts.URL), url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"1"},
		"client_secret": {"!!!not-base64!!!"},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusUnauthorized)
}
