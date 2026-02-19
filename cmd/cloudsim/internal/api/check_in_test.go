package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func checkInURL(base string) string { return base + "/api/v1/check-in" }

// checkInEnv holds test fixtures for check-in tests.
type checkInEnv struct {
	testServer    *httptest.Server
	client        *http.Client
	site          Site
	node          Node
	secret        []byte // raw secret returned from create node
	configVersion ConfigVersion
}

// setupCheckInEnv creates a test environment with a site, node, and config version.
func setupCheckInEnv(t *testing.T) checkInEnv {
	t.Helper()

	ts := newTestServer(t)
	client := ts.Client()

	// Create site
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	// Create node (capture secret)
	var created CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "checkin-node",
		"siteId":   site.ID,
	}, &created)
	assertStatus(t, resp, http.StatusCreated)

	// Create config version
	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      created.ID,
		"description": "v1.0.0",
		"payload":     []byte("config-data"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	return checkInEnv{
		testServer:    ts,
		client:        client,
		site:          site,
		node:          created.Node,
		secret:        created.Secret,
		configVersion: cv,
	}
}

// doCheckIn performs a check-in request with the given bearer secret and optional request body.
func doCheckIn(t *testing.T, client *http.Client, url string, secret []byte, reqBody any, res any) *http.Response {
	t.Helper()
	var body *bytes.Buffer
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		body = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if secret != nil {
		token := base64.URLEncoding.EncodeToString(secret)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if res != nil {
		if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	}
	return resp
}

func TestCheckIn_NoDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	var got CheckInResponse
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	expect := CheckInResponse{
		CheckIn: NodeCheckIn{
			NodeID: e.node.ID,
		},
		LatestConfig: nil,
	}
	diff := cmp.Diff(expect, got,
		cmpopts.IgnoreFields(NodeCheckIn{}, "ID", "CheckInTime"))
	if diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
	if got.CheckIn.CheckInTime.IsZero() {
		t.Error("expected non-zero check-in time")
	}

	// verify check-in was recorded in the store
	// List check-ins via management API
	var list ListResponse[NodeCheckIn]
	resp = doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
	assertStatus(t, resp, http.StatusOK)

	if len(list.Items) != 1 {
		t.Fatalf("expected 1 check-in, got %d", len(list.Items))
	}
	if diff := cmp.Diff(got.CheckIn, list.Items[0]); diff != "" {
		t.Errorf("check-in mismatch (-want +got):\n%s", diff)
	}
}

func TestCheckIn_PendingDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
		"status":          "PENDING",
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	if diff := cmp.Diff(dep, got.LatestConfig.Deployment); diff != "" {
		t.Errorf("deployment mismatch (-want +got):\n%s", diff)
	}
	if got.LatestConfig.ConfigVersion.PayloadURL == "" {
		t.Error("expected non-empty payloadUrl")
	}
}

func TestCheckIn_InProgressDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and set to IN_PROGRESS
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "IN_PROGRESS",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	if diff := cmp.Diff(dep, got.LatestConfig.Deployment); diff != "" {
		t.Errorf("deployment mismatch (-want +got):\n%s", diff)
	}
}

func TestCheckIn_ReturnsNewestActive(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create dep1, then dep2 — creating dep2 auto-cancels dep1, leaving only dep2 active
	var dep1 Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
		"status":          "PENDING",
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
		"status":          "PENDING",
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in — should return the one with higher ID
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	if diff := cmp.Diff(dep2, got.LatestConfig.Deployment); diff != "" {
		t.Errorf("deployment mismatch (-want +got):\n%s", diff)
	}
}

func TestCheckIn_CompletedAndFailedExcluded(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create COMPLETED deployment
	var dep1 Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep1.ID), map[string]any{
		"status": "COMPLETED",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Create FAILED deployment
	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep2.ID), map[string]any{
		"status": "FAILED",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Check in — should have no active deployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig != nil {
		t.Errorf("expected no active deployment, got ID %d", got.LatestConfig.Deployment.ID)
	}
}

func TestCheckIn_MissingAuthHeader(t *testing.T) {
	e := setupCheckInEnv(t)

	// POST with no Authorization head
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), nil, nil, nil)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_InvalidBearerToken(t *testing.T) {
	e := setupCheckInEnv(t)

	// Use a wrong secret
	wrongSecret := bytes.Repeat([]byte{0xFF}, 32)
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), wrongSecret, nil, nil)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_MalformedBase64(t *testing.T) {
	e := setupCheckInEnv(t)

	req, err := http.NewRequest("POST", checkInURL(e.testServer.URL), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "!!!not-base64!!!"))
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_WithCurrentDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create and advance a deployment to COMPLETED
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "COMPLETED",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with currentDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret,
		CheckInRequest{CurrentDeployment: &CheckInDeploymentRef{ID: dep.ID}}, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.CheckIn.CurrentDeploymentID == nil || *got.CheckIn.CurrentDeploymentID != dep.ID {
		t.Errorf("expected currentDeploymentId %d, got %v", dep.ID, got.CheckIn.CurrentDeploymentID)
	}

	// Verify field persisted via management GET
	var stored NodeCheckIn
	resp = doRequest(t, e.client, "GET",
		fmt.Sprintf("%s/%d", listNodeCheckInsURL(e.testServer.URL, e.node.ID), got.CheckIn.ID),
		nil, &stored)
	assertStatus(t, resp, http.StatusOK)
	if stored.CurrentDeploymentID == nil || *stored.CurrentDeploymentID != dep.ID {
		t.Errorf("stored currentDeploymentId: expected %d, got %v", dep.ID, stored.CurrentDeploymentID)
	}
}

func TestCheckIn_WithInstallingDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create a PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret,
		CheckInRequest{InstallingDeployment: &CheckInDeploymentRef{ID: dep.ID}}, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.CheckIn.InstallingDeploymentID == nil || *got.CheckIn.InstallingDeploymentID != dep.ID {
		t.Errorf("expected installingDeploymentId %d, got %v", dep.ID, got.CheckIn.InstallingDeploymentID)
	}
}

func TestCheckIn_WithFailedDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to IN_PROGRESS
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": e.configVersion.ID,
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "IN_PROGRESS",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with failedDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: dep.ID, Reason: "disk full"}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now FAILED with the given reason
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "FAILED" {
		t.Errorf("expected deployment status FAILED, got %s", updated.Status)
	}
	if updated.Reason != "disk full" {
		t.Errorf("expected reason %q, got %q", "disk full", updated.Reason)
	}
}

func TestCheckIn_FailedDeploymentWrongNode(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create a second node with its own config version and deployment
	var site2 Site
	resp := doRequest(t, e.client, "POST", listSitesURL(e.testServer.URL),
		map[string]string{"name": "Site 2"}, &site2)
	assertStatus(t, resp, http.StatusCreated)

	var created2 CreateNodeResponse
	resp = doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
		"hostname": "other-node",
		"siteId":   site2.ID,
	}, &created2)
	assertStatus(t, resp, http.StatusCreated)

	var cv2 ConfigVersion
	resp = doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
		"nodeId":      created2.ID,
		"description": "v1",
		"payload":     []byte("data"),
	}, &cv2)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": cv2.ID,
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in as first node with failedDeployment pointing to second node's deployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: dep2.ID, Reason: "bad"}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_FailedDeploymentNotFound(t *testing.T) {
	e := setupCheckInEnv(t)

	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.secret,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: 99999}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}
