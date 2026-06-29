package cloud

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()

	// Shared setup: TLS sim + site + node, returning the CA pool for building an
	// mTLS client to verify the issued credential.
	setupRegisterEnv := func(t *testing.T) (tsURL string, client *http.Client, caPool *x509.CertPool, nodeID int64) {
		t.Helper()
		ts, apiServer, _ := newTLSSim(t)
		client = ts.Client()

		var site sim.Site
		resp := doSimRequest(t, client, "POST", ts.URL+"/api/v1/management/sites",
			map[string]string{"name": "Test Site"}, &site)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create site: expected 201, got %d", resp.StatusCode)
		}
		var node sim.Node
		resp = doSimRequest(t, client, "POST", ts.URL+"/api/v1/management/nodes", map[string]any{
			"hostname": "test-node",
			"siteId":   strconv.FormatInt(site.ID, 10),
		}, &node)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create node: expected 201, got %d", resp.StatusCode)
		}
		return ts.URL, client, apiServer.CACertPool(), node.ID
	}

	freshEnrollmentCode := func(t *testing.T, client *http.Client, tsURL string, nodeID int64) string {
		t.Helper()
		var ec sim.EnrollmentCode
		resp := doSimRequest(t, client, "POST",
			fmt.Sprintf("%s/api/v1/management/nodes/%d/enrollment-codes", tsURL, nodeID),
			nil, &ec)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create enrollment code: expected 201, got %d", resp.StatusCode)
		}
		return ec.Code
	}

	t.Run("happy path returns a usable credential", func(t *testing.T) {
		tsURL, client, caPool, nodeID := setupRegisterEnv(t)
		code := freshEnrollmentCode(t, client, tsURL, nodeID)

		cred, err := Register(ctx, code, tsURL+"/v1/device/register", "test-device",
			WithRegisterHTTPClient(client))
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		// The CA sets the leaf CN to the node id.
		if got, want := cred.NodeID(), strconv.FormatInt(nodeID, 10); got != want {
			t.Errorf("NodeID = %q, want %q", got, want)
		}
		if cred.Key == nil {
			t.Error("credential has no private key")
		}
		if len(cred.Chain) == 0 {
			t.Error("credential has no certificate chain")
		}

		// The issued credential can authenticate a check-in over mTLS.
		httpClient := NewHTTPClient(cred, tsURL, WithServerRootCAs(caPool))
		if _, err := httpClient.CheckIn(ctx, CheckInRequest{}); err != nil {
			t.Errorf("CheckIn with registered credential failed: %v", err)
		}
	})

	t.Run("enrollment code is single-use", func(t *testing.T) {
		tsURL, client, _, nodeID := setupRegisterEnv(t)
		code := freshEnrollmentCode(t, client, tsURL, nodeID)

		if _, err := Register(ctx, code, tsURL+"/v1/device/register", "device-a",
			WithRegisterHTTPClient(client)); err != nil {
			t.Fatalf("first Register: %v", err)
		}

		_, err := Register(ctx, code, tsURL+"/v1/device/register", "device-b",
			WithRegisterHTTPClient(client))
		if err == nil {
			t.Fatal("expected error on second use of enrollment code, got nil")
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
		}
	})

	// Pin the API contract: a rejected code is a generic 401 (SCC returns
	// {error:"unauthorized"}), which the register endpoint's only-auth-is-the-code
	// rule lets opsapi classify as an invalid enrollment code. Assert both the raw
	// status and IsInvalidCredentialsError, the predicate opsapi relies on and that
	// holds against both the sim and real SCC.
	t.Run("rejected enrollment code is a 401", func(t *testing.T) {
		tsURL, client, _, _ := setupRegisterEnv(t)

		_, err := Register(ctx, "BOGUS1", tsURL+"/v1/device/register", "device",
			WithRegisterHTTPClient(client))
		if err == nil {
			t.Fatal("expected error for invalid enrollment code, got nil")
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
		}
		if !IsInvalidCredentialsError(err) {
			t.Errorf("IsInvalidCredentialsError = false, want true; got %T: %v", err, err)
		}
	})
}
