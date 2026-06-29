package sim

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

var codeRegexp = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func TestCreateEnrollmentCode(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   sid(site.ID),
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

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
		if ec.TargetSlot != "primary" {
			t.Errorf("expected default targetSlot primary, got %q", ec.TargetSlot)
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
	ts, _, apiServer := newTestServerWithStore(t)
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   sid(site.ID),
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	freshCode := func(t *testing.T) string {
		t.Helper()
		var ec EnrollmentCode
		resp := doRequest(t, client, "POST", enrollmentCodesURL(ts.URL, node.ID), nil, &ec)
		assertStatus(t, resp, http.StatusCreated)
		return ec.Code
	}

	t.Run("registers with valid code and issues a cert with CN=nodeId", func(t *testing.T) {
		resp, chain := doRegisterReq(t, client, deviceRegisterURL(ts.URL), freshCode(t), "test/path/AC-01")
		assertStatus(t, resp, http.StatusCreated)
		if ct := resp.Header.Get("Content-Type"); ct != "application/pem-certificate-chain" {
			t.Errorf("Content-Type = %q, want application/pem-certificate-chain", ct)
		}
		if len(chain) == 0 {
			t.Fatal("empty certificate chain")
		}
		// The CA sets the leaf CN to the node id, not the name from the CSR.
		if got, want := chain[0].Subject.CommonName, sid(node.ID); got != want {
			t.Errorf("leaf CN = %q, want %q", got, want)
		}
	})

	t.Run("code is single-use", func(t *testing.T) {
		code := freshCode(t)
		resp, _ := doRegisterReq(t, client, deviceRegisterURL(ts.URL), code, "test")
		assertStatus(t, resp, http.StatusCreated)

		resp, _ = doRegisterReq(t, client, deviceRegisterURL(ts.URL), code, "test")
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", deviceRegisterURL(ts.URL), strings.NewReader(""))
		req.Header.Set("Content-Type", "application/pkcs10")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		_ = resp.Body.Close()
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("unknown code is rejected", func(t *testing.T) {
		resp, _ := doRegisterReq(t, client, deviceRegisterURL(ts.URL), "ZZZZZZ", "test")
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("renew issues a fresh cert with the same CN over mTLS", func(t *testing.T) {
		// Enroll, then renew using the issued certificate.
		cert := enrollCert(t, ts, freshCode(t))
		deviceClient := mtlsClientFor(apiServer, cert)

		resp, chain := doRegisterReq(t, deviceClient, deviceRenewURL(ts.URL), "", "renewing")
		assertStatus(t, resp, http.StatusOK) // renewal returns 200 (matches SCC)
		if len(chain) == 0 {
			t.Fatal("empty renewed chain")
		}
		if got, want := chain[0].Subject.CommonName, sid(node.ID); got != want {
			t.Errorf("renewed leaf CN = %q, want %q", got, want)
		}
	})
}

// doRegisterReq builds a CSR for cn and POSTs it (base64 DER, application/pkcs10)
// to url, optionally with a bearer token. It returns the response and the parsed
// PEM certificate chain (nil on non-2xx).
func doRegisterReq(t *testing.T, client *http.Client, url, bearer, cn string) (*http.Response, []*x509.Certificate) {
	t.Helper()
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	csrDER, err := pki.CreateCSRDER(key, cn)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(base64.StdEncoding.EncodeToString(csrDER)))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/pkcs10")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	chain, err := pki.ParseCertificatesPEM(body)
	if err != nil {
		t.Fatalf("parse chain: %v", err)
	}
	return resp, chain
}

func enrollmentCodesURL(base string, nodeID int64) string {
	return fmt.Sprintf("%s/api/v1/management/nodes/%d/enrollment-codes", base, nodeID)
}

func deviceRegisterURL(base string) string { return base + "/v1/device/register" }
func deviceRenewURL(base string) string    { return base + "/v1/device/certificate/renew" }
