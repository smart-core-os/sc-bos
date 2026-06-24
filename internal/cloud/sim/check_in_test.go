package sim

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

func checkInURL(base string) string { return base + "/v1/device/check-in" }

// assertStreamDeployment checks a check-in response's LatestStream.Deployment against the expected
// deployment id, artefact discriminator and status.
func assertStreamDeployment(t *testing.T, got StreamDeployment, wantID int64, wantArtefact, wantStatus string) {
	t.Helper()
	if want := formatDeploymentID(wantArtefact, wantID); got.ID != want {
		t.Errorf("deployment id = %q, want %q", got.ID, want)
	}
	if got.Artefact != wantArtefact {
		t.Errorf("deployment artefact = %q, want %q", got.Artefact, wantArtefact)
	}
	if got.Status != wantStatus {
		t.Errorf("deployment status = %q, want %q", got.Status, wantStatus)
	}
}

// checkInEnv holds test fixtures for check-in tests.
type checkInEnv struct {
	testServer    *httptest.Server
	store         *store.Store
	client        *http.Client // management client (trusts server, no client cert)
	deviceClient  *http.Client // presents the enrolled client certificate (mTLS)
	site          Site
	node          Node
	configVersion ConfigVersion
}

// setupCheckInEnv creates a test environment with a site, an enrolled node, and a config version.
func setupCheckInEnv(t *testing.T) checkInEnv {
	t.Helper()
	return setupCheckInEnvWithPlatform(t, "linux", "arm64")
}

// setupCheckInEnvWithPlatform is like setupCheckInEnv but creates the node with the given platform.
// Empty os and arch create a node with no known platform (set on first check-in).
func setupCheckInEnvWithPlatform(t *testing.T, os, arch string) checkInEnv {
	t.Helper()

	ts, dataStore, apiServer := newTestServerWithStore(t)
	client := ts.Client()

	// Create site
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	// Create node
	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "checkin-node",
		"siteId":   sid(site.ID),
		"os":       os,
		"arch":     arch,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	// Enroll the node and build a client that presents the issued certificate.
	var ec EnrollmentCode
	resp = doRequest(t, client, "POST", listNodeEnrollmentCodesURL(ts.URL, node.ID), nil, &ec)
	assertStatus(t, resp, http.StatusCreated)
	deviceClient := mtlsClientFor(apiServer, enrollCert(t, ts, ec.Code))

	// Create config version
	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      sid(node.ID),
		"description": "v1.0.0",
		"payload":     []byte("config-data"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	return checkInEnv{
		testServer:    ts,
		store:         dataStore,
		client:        client,
		deviceClient:  deviceClient,
		site:          site,
		node:          node,
		configVersion: cv,
	}
}

// doCheckIn performs a check-in request over the given client (which presents the
// client certificate for mTLS-authenticated endpoints) with an optional body.
func doCheckIn(t *testing.T, client *http.Client, url string, reqBody any, res any) *http.Response {
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
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
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
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	assertStreamDeployment(t, got.LatestConfig.Deployment, dep.ID, "config", dep.Status)
	if got.LatestConfig.Version.PayloadURL == "" {
		t.Error("expected non-empty payloadUrl")
	}
}

func TestCheckIn_InProgressDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and set to IN_PROGRESS
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	assertStreamDeployment(t, got.LatestConfig.Deployment, dep.ID, "config", dep.Status)
}

func TestCheckIn_ReturnsNewestActive(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create dep1, then dep2 — creating dep2 auto-cancels dep1, leaving only dep2 active
	var dep1 ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 ConfigDeployment
	resp = doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in — should return the one with higher ID
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Fatal("expected active deployment, got nil")
	}
	assertStreamDeployment(t, got.LatestConfig.Deployment, dep2.ID, "config", dep2.Status)
}

func TestCheckIn_CompletedAndFailedExcluded(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create COMPLETED deployment
	var dep1 ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep1)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep1.ID), map[string]any{
		"status": "completed",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Create FAILED deployment
	var dep2 ConfigDeployment
	resp = doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep2.ID), map[string]any{
		"status": "failed",
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Check in — should have no active deployment
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig != nil {
		t.Errorf("expected no active deployment, got ID %s", got.LatestConfig.Deployment.ID)
	}
}

func TestCheckIn_NoClientCertificate(t *testing.T) {
	e := setupCheckInEnv(t)

	// The management client presents no client certificate, so the per-endpoint
	// guard rejects the request.
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), nil, nil)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_UnrecognisedClientCertificate(t *testing.T) {
	e := setupCheckInEnv(t)

	// A self-signed certificate that does not chain to the sim's dev CA.
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject: pkix.Name{CommonName: sid(e.node.ID)},
	}, key)
	if err != nil {
		t.Fatalf("self-signed cert: %v", err)
	}
	bogus := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates:       []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		InsecureSkipVerify: true, // we are testing server-side client-cert rejection, not server verification
	}}}

	req, err := http.NewRequest("POST", checkInURL(e.testServer.URL), nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	// The self-signed certificate doesn't match the server's advertised client
	// CAs, so the TLS stack omits it; the server sees no client certificate and
	// the guard returns 401 (not a handshake rejection).
	resp, err := bogus.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestCheckIn_WithCurrentDeployment(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create and advance a deployment to COMPLETED
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "completed",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with currentDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateApplied}}}, &got)
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
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateInstalling}}}, &got)
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
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with failedDeployment
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateFailed, Reason: "disk full"}}}, &got)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now FAILED with the given reason
	var updated ConfigDeployment
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &updated)
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

	var node2 Node
	resp = doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
		"hostname": "other-node",
		"siteId":   sid(site2.ID),
	}, &node2)
	assertStatus(t, resp, http.StatusCreated)

	var cv2 ConfigVersion
	resp = doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
		"nodeId":      sid(node2.ID),
		"description": "v1",
		"payload":     []byte("data"),
	}, &cv2)
	assertStatus(t, resp, http.StatusCreated)

	var dep2 ConfigDeployment
	resp = doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(cv2.ID),
	}, &dep2)
	assertStatus(t, resp, http.StatusCreated)

	// Check in as first node with failedDeployment pointing to second node's deployment
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep2.ID), State: stateFailed, Reason: "bad"}}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_FailedDeploymentNotFound(t *testing.T) {
	e := setupCheckInEnv(t)

	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, 99999), State: stateFailed}}}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_InstallingAdvancesPendingToInProgress(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateInstalling}}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now IN_PROGRESS
	var updated ConfigDeployment
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "in_progress" {
		t.Errorf("expected deployment status IN_PROGRESS, got %s", updated.Status)
	}
}

func TestCheckIn_InstallingDoesNotAdvanceNonPending(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to FAILED
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "failed",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with installingDeployment
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateInstalling}}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment status is still COMPLETED (no transition)
	var updated ConfigDeployment
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "failed" {
		t.Errorf("expected deployment status FAILED, got %s", updated.Status)
	}
}

func TestCheckIn_CurrentAdvancesInProgressToCompleted(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create deployment and advance to IN_PROGRESS
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	resp = doRequest(t, e.client, "PATCH", configDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "in_progress",
	}, &dep)
	assertStatus(t, resp, http.StatusOK)

	// Check in with currentDeployment
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateApplied}}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now COMPLETED
	var updated ConfigDeployment
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "completed" {
		t.Errorf("expected deployment status COMPLETED, got %s", updated.Status)
	}
}

func TestCheckIn_AppliedCompletesPending(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create PENDING deployment (never reported installing)
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Report the deployment applied. Reporting the node is running the target completes it on its own,
	// even straight from pending.
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{DeploymentID: formatDeploymentID(artefactConfig, dep.ID), State: stateApplied}}}, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify deployment is now COMPLETED.
	var updated ConfigDeployment
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "completed" {
		t.Errorf("expected deployment status COMPLETED, got %s", updated.Status)
	}
}

func TestCheckIn_InstallingWithErrorAndAttempts(t *testing.T) {
	e := setupCheckInEnv(t)

	// Create a PENDING deployment
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	// Check in with installingDeployment including error and attempts
	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL),
		CheckInRequest{Progress: []ProgressReport{{
			DeploymentID: formatDeploymentID(artefactConfig, dep.ID),
			State:        stateInstalling,
			Error:        "transient: timeout",
			Attempts:     3,
		}}}, &got)
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

func TestCheckIn_UnknownDeploymentPrefixRejected(t *testing.T) {
	e := setupCheckInEnv(t)

	// A progress entry whose deployment id carries neither the config (c-) nor binary (b-) prefix
	// cannot be routed to a table and is an invalid request.
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"progress": []map[string]any{{"deploymentId": "x-1", "state": "installing"}},
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestCheckIn_ReconcilesPlatform(t *testing.T) {
	// sets up the node with {linux,arm64}
	e := setupCheckInEnv(t)

	var node Node
	resp := doRequest(t, e.client, "GET", nodeURL(e.testServer.URL, e.node.ID), nil, &node)
	assertStatus(t, resp, http.StatusOK)
	want := Platform{"linux", "arm64"}
	if node.OS != want.OS || node.Arch != want.Arch {
		t.Errorf("node platform = %s/%s, want %s/%s", node.OS, node.Arch, want.OS, want.Arch)
	}
}

func TestCheckIn_BlockPlatformChange(t *testing.T) {
	// sets up the node with {linux,arm64}
	e := setupCheckInEnv(t)

	// report a different one - should be blocked
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"running": map[string]any{"platform": map[string]any{"os": "linux", "arch": "amd64"}},
	}, nil)
	assertStatus(t, resp, http.StatusConflict)
}

func TestCheckIn_FirstReportSetsPlatform(t *testing.T) {
	// node created with no known platform
	e := setupCheckInEnvWithPlatform(t, "", "")

	// the first report records the reported platform onto the node
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"running": map[string]any{"platform": map[string]any{"os": "linux", "arch": "arm64"}},
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	var node Node
	resp = doRequest(t, e.client, "GET", nodeURL(e.testServer.URL, e.node.ID), nil, &node)
	assertStatus(t, resp, http.StatusOK)
	want := Platform{"linux", "arm64"}
	if node.OS != want.OS || node.Arch != want.Arch {
		t.Errorf("node platform = %s/%s, want %s/%s", node.OS, node.Arch, want.OS, want.Arch)
	}
}
