package sim

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func checkInURL(base string) string { return base + "/v1/device/check-in" }

// checkInEnv holds test fixtures for check-in tests.
type checkInEnv struct {
	testServer    *httptest.Server
	client        *http.Client
	site          Site
	node          Node
	secret        []byte // raw secret returned from create node
	accessToken   string // JWT access token obtained from /v1/device/token
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
		"siteId":   sid(site.ID),
	}, &created)
	assertStatus(t, resp, http.StatusCreated)

	// Obtain an access token via the token endpoint
	accessToken := obtainAccessToken(t, client, ts.URL, created.ID, created.Secret)

	// Create config version
	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      sid(created.ID),
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
		accessToken:   accessToken,
		configVersion: cv,
	}
}

// obtainAccessToken exchanges client credentials for a JWT access token via the token endpoint.
func obtainAccessToken(t *testing.T, client *http.Client, baseURL string, nodeID int64, secret []byte) string {
	t.Helper()
	encodedSecret := base64.StdEncoding.EncodeToString(secret)
	resp, err := client.PostForm(baseURL+"/v1/device/token", url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {sid(nodeID)},
		"client_secret": {encodedSecret},
	})
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token endpoint returned %d", resp.StatusCode)
	}
	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	return tokenResp.AccessToken
}

// doCheckIn performs a check-in request with a JWT access token and optional request body.
func doCheckIn(t *testing.T, client *http.Client, url string, accessToken string, reqBody any, res any) *http.Response {
	t.Helper()
	var body io.Reader
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
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
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
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.CheckIn.NodeID != e.node.ID {
		t.Errorf("expected nodeId %d, got %d", e.node.ID, got.CheckIn.NodeID)
	}
	if got.CheckIn.CheckInTime.IsZero() {
		t.Error("expected non-zero check-in time")
	}
	if got.LatestConfig != nil {
		t.Errorf("expected no latestConfig, got %+v", got.LatestConfig)
	}

	// verify check-in was recorded in the store via management API
	var list ListResponse[NodeCheckIn]
	resp = doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 check-in, got %d", len(list.Items))
	}
}

func TestCheckIn_PendingDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
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
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
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
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in — should return the one with higher ID
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
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
		"configVersionId": sid(e.configVersion.ID),
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep1.ID), map[string]any{
		"status": "completed",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Create FAILED deployment
	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep2.ID), map[string]any{
		"status": "failed",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Check in — should have no active deployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig != nil {
		t.Errorf("expected no active deployment, got ID %d", got.LatestConfig.Deployment.ID)
	}
}

func TestCheckIn_MissingAuthHeader(t *testing.T) {
	e := setupCheckInEnv(t)

	// POST with no Authorization header — empty access token
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), "", nil, nil)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_InvalidBearerToken(t *testing.T) {
	e := setupCheckInEnv(t)

	// Issue a valid-looking JWT signed with a different key
	otherIssuer, err := newTokenIssuer()
	if err != nil {
		t.Fatalf("newTokenIssuer: %v", err)
	}
	token, _, err := otherIssuer.issue(e.node.ID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), token, nil, nil)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_MalformedBase64(t *testing.T) {
	e := setupCheckInEnv(t)

	req, err := http.NewRequest("POST", checkInURL(e.testServer.URL), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer !!!not-a-valid-token!!!")
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
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "completed",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with currentDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{CurrentDeployment: &CheckInDeploymentRef{ID: dep.ID}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// Verify the check-in was recorded with the currentDeploymentId via management API
	var list ListResponse[NodeCheckIn]
	resp = doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) == 0 {
		t.Fatal("expected at least 1 check-in")
	}
	latest := list.Items[len(list.Items)-1]
	if latest.CurrentDeploymentID == nil || *latest.CurrentDeploymentID != dep.ID {
		t.Errorf("stored currentDeploymentId: expected %d, got %v", dep.ID, latest.CurrentDeploymentID)
	}
}

func TestCheckIn_WithInstallingDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create a PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{InstallingDeployment: &CheckInInstallingDeployment{ID: dep.ID}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// Verify the check-in was recorded with the installingDeploymentId via management API
	var list ListResponse[NodeCheckIn]
	resp = doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) == 0 {
		t.Fatal("expected at least 1 check-in")
	}
	latest := list.Items[len(list.Items)-1]
	if latest.InstallingDeploymentID == nil || *latest.InstallingDeploymentID != dep.ID {
		t.Errorf("stored installingDeploymentId: expected %d, got %v", dep.ID, latest.InstallingDeploymentID)
	}
}

func TestCheckIn_WithFailedDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to IN_PROGRESS
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with failedDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: dep.ID, Reason: "disk full"}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now FAILED with the given reason
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "failed" {
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
		"siteId":   sid(site2.ID),
	}, &created2)
	assertStatus(t, resp, http.StatusCreated)

	var cv2 ConfigVersion
	resp = doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
		"nodeId":      sid(created2.ID),
		"description": "v1",
		"payload":     []byte("data"),
	}, &cv2)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 Deployment
	resp = doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(cv2.ID),
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in as first node with failedDeployment pointing to second node's deployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: dep2.ID, Reason: "bad"}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_FailedDeploymentNotFound(t *testing.T) {
	e := setupCheckInEnv(t)

	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{FailedDeployment: &CheckInFailedDeployment{ID: 99999}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_InstallingAdvancesPendingToInProgress(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{InstallingDeployment: &CheckInInstallingDeployment{ID: dep.ID}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now IN_PROGRESS
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "in_progress" {
		t.Errorf("expected deployment status IN_PROGRESS, got %s", updated.Status)
	}
}

func TestCheckIn_InstallingDoesNotAdvanceNonPending(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to FAILED
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "failed",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with installingDeployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{InstallingDeployment: &CheckInInstallingDeployment{ID: dep.ID}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment status is still COMPLETED (no transition)
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "failed" {
		t.Errorf("expected deployment status FAILED, got %s", updated.Status)
	}
}

func TestCheckIn_CurrentAdvancesInProgressToCompleted(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to IN_PROGRESS
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with currentDeployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{CurrentDeployment: &CheckInDeploymentRef{ID: dep.ID}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now COMPLETED
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "completed" {
		t.Errorf("expected deployment status COMPLETED, got %s", updated.Status)
	}
}

func TestCheckIn_CurrentDoesNotAdvanceNonInProgress(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment (do not advance)
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with currentDeployment
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{CurrentDeployment: &CheckInDeploymentRef{ID: dep.ID}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment status is still PENDING (no transition)
	var updated Deployment
	resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "pending" {
		t.Errorf("expected deployment status PENDING, got %s", updated.Status)
	}
}

func TestCheckIn_InstallingWithErrorAndAttempts(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create a PENDING deployment
	var dep Deployment
	resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment including error and attempts
	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken,
		CheckInRequest{InstallingDeployment: &CheckInInstallingDeployment{
			ID:       dep.ID,
			Error:    "transient: timeout",
			Attempts: 3,
		}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// The check-in response only contains an ack — verify the details were stored via management API
	var list ListResponse[NodeCheckIn]
	resp = doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) == 0 {
		t.Fatal("expected at least 1 check-in")
	}
	stored := list.Items[len(list.Items)-1]

	attempts := int64(3)
	wantCheckIn := NodeCheckIn{
		NodeID:                       e.node.ID,
		InstallingDeploymentID:       &dep.ID,
		InstallingDeploymentError:    "transient: timeout",
		InstallingDeploymentAttempts: &attempts,
	}
	if diff := cmp.Diff(wantCheckIn, stored, cmpopts.IgnoreFields(NodeCheckIn{}, "ID", "CheckInTime")); diff != "" {
		t.Errorf("stored check-in mismatch (-want +got):\n%s", diff)
	}
}
