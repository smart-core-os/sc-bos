package sim

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var codeRegexp = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func TestCreateEnrollmentCode(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var nodeResp CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   sid(site.ID),
	}, &nodeResp)
	assertStatus(t, resp, http.StatusCreated)
	node := nodeResp.Node

	t.Run("generates a code for a valid node", func(t *testing.T) {
		var ec EnrollmentCode
		resp := doRequest(t, client, "POST", enrollmentCodesURL(ts.URL, node.ID), nil, &ec)
		assertStatus(t, resp, http.StatusCreated)

		if !codeRegexp.MatchString(ec.Code) {
			t.Errorf("expected 6-char alphanumeric code, got %q (len=%d)", ec.Code, len(ec.Code))
		}
		if ec.NodeID != node.ID {
			t.Errorf("expected nodeId %d, got %d", node.ID, ec.NodeID)
		}
		if time.Until(ec.ExpiresAt) < 14*time.Minute {
			t.Errorf("expected at least 14 minutes until expiry, got %v", time.Until(ec.ExpiresAt))
		}
	})

	t.Run("returns 404 for unknown node", func(t *testing.T) {
		resp := doRequest(t, client, "POST", enrollmentCodesURL(ts.URL, 99999), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("returns 400 for zero id", func(t *testing.T) {
		resp := doRequest(t, client, "POST", enrollmentCodesURL(ts.URL, 0), nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestDeviceRegister(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var nodeResp CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   sid(site.ID),
	}, &nodeResp)
	assertStatus(t, resp, http.StatusCreated)
	node := nodeResp.Node

	// Generate an enrollment code for use in sub-tests that need a fresh one.
	freshCode := func(t *testing.T) string {
		t.Helper()
		var ec EnrollmentCode
		resp := doRequest(t, client, "POST", enrollmentCodesURL(ts.URL, node.ID), nil, &ec)
		assertStatus(t, resp, http.StatusCreated)
		return ec.Code
	}

	t.Run("registers with valid code", func(t *testing.T) {
		var reg DeviceRegisterResponse
		resp := doRegister(t, client, ts.URL, freshCode(t), "test/path/AC-01", &reg)
		assertStatus(t, resp, http.StatusCreated)

		if reg.ClientID == "" {
			t.Error("expected non-empty client_id")
		}
		if expect := "test/path/AC-01"; reg.ClientName != expect {
			t.Errorf("expected client_name %q, got %q", expect, reg.ClientName)
		}
	})

	t.Run("code is single-use", func(t *testing.T) {
		code := freshCode(t)
		resp := doRegister(t, client, ts.URL, code, "test", nil)
		assertStatus(t, resp, http.StatusCreated)

		// Second use must be rejected
		resp = doRegister(t, client, ts.URL, code, "test", nil)
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("new secret is usable with token endpoint", func(t *testing.T) {
		var reg DeviceRegisterResponse
		resp := doRegister(t, client, ts.URL, freshCode(t), "test/path/AC-01", &reg)
		assertStatus(t, resp, http.StatusCreated)

		clientID, err := strconv.ParseInt(reg.ClientID, 10, 64)
		if err != nil {
			t.Fatalf("parse client_id: %v", err)
		}

		resp, err = client.PostForm(tokenURL(ts.URL), url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {strconv.FormatInt(clientID, 10)},
			"client_secret": {base64.StdEncoding.EncodeToString(reg.ClientSecret)},
		})
		if err != nil {
			t.Fatalf("token request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		resp := doRequest(t, client, "POST", deviceRegisterURL(ts.URL),
			map[string]string{"client_name": "test"}, nil)
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("unknown code is rejected", func(t *testing.T) {
		resp := doRegister(t, client, ts.URL, "ZZZZZZ", "test", nil)
		assertStatus(t, resp, http.StatusUnauthorized)
	})
}

// doRegister calls POST /v1/device/register with the given enrollment code as Bearer token.
func doRegister(t *testing.T, client *http.Client, base, code, clientName string, out any) *http.Response {
	t.Helper()

	body, err := json.Marshal(map[string]string{"client_name": clientName})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req, err := http.NewRequest("POST", deviceRegisterURL(base), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+code)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp
}

func enrollmentCodesURL(base string, nodeID int64) string {
	return fmt.Sprintf("%s/api/v1/management/nodes/%d/enrollment-codes", base, nodeID)
}

func deviceRegisterURL(base string) string {
	return base + "/v1/device/register"
}
