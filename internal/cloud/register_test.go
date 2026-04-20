package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()

	// Shared setup: sim server + site + node + enrollment code URL helper.
	setupRegisterEnv := func(t *testing.T) (tsURL string, client *http.Client, nodeID int64) {
		t.Helper()
		env := setupClientEnv(t)
		return env.testServer.URL, env.httpClient, env.nodeID
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

	t.Run("happy path returns usable credentials", func(t *testing.T) {
		tsURL, client, nodeID := setupRegisterEnv(t)
		code := freshEnrollmentCode(t, client, tsURL, nodeID)

		reg, err := Register(ctx, code, tsURL+"/v1/device/register", "test-device",
			WithRegisterHTTPClient(client))
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		if reg.ClientID == "" {
			t.Error("ClientID is empty")
		}
		if reg.ClientSecret == "" {
			t.Error("ClientSecret is empty")
		}
		if reg.BosapiRoot == "" {
			t.Error("BosapiRoot is empty")
		}
		if reg.ClientID != strconv.FormatInt(nodeID, 10) {
			t.Errorf("ClientID = %q, want %q", reg.ClientID, strconv.FormatInt(nodeID, 10))
		}

		// Verify the returned credentials can authenticate against the server.
		httpClient := NewHTTPClient(reg, WithHTTPClient(client))
		if _, err := httpClient.CheckIn(ctx, CheckInRequest{}); err != nil {
			t.Errorf("CheckIn with registered credentials failed: %v", err)
		}
	})

	t.Run("enrollment code is single-use", func(t *testing.T) {
		tsURL, client, nodeID := setupRegisterEnv(t)
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

	t.Run("invalid enrollment code returns 401", func(t *testing.T) {
		tsURL, client, _ := setupRegisterEnv(t)

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
	})

	// Pin the API contract: the sim (and real SCC) must return the "invalid_enrollment_code"
	// error code so that IsInvalidEnrollmentCode remains the correct predicate and the UI
	// can map it to a friendly message.
	t.Run("invalid enrollment code satisfies IsInvalidEnrollmentCode", func(t *testing.T) {
		tsURL, client, _ := setupRegisterEnv(t)

		_, err := Register(ctx, "BOGUS1", tsURL+"/v1/device/register", "device",
			WithRegisterHTTPClient(client))
		if err == nil {
			t.Fatal("expected error for invalid enrollment code, got nil")
		}
		if !IsInvalidEnrollmentCode(err) {
			t.Errorf("IsInvalidEnrollmentCode = false, want true; got %T: %v", err, err)
		}
	})
}
