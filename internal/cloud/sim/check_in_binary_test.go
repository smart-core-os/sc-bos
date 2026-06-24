package sim

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// deployUpdateToCheckInNode uploads a linux/arm64 artefact for the env's site and creates a pending
// binary deployment targeting the env's node. Returns the artefact and the deployment.
func deployUpdateToCheckInNode(t *testing.T, e checkInEnv, payload []byte) (BinaryArtefact, BinaryDeployment) {
	t.Helper()
	var artefact BinaryArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version": "2.0.0",
		"os":      "linux",
		"arch":    "arm64",
		"siteId":  sid(e.site.ID),
	}, payload, &artefact)
	assertStatus(t, resp, http.StatusCreated)

	var dep BinaryDeployment
	resp = doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	return artefact, dep
}

func TestCheckIn_LatestUpdate(t *testing.T) {
	e := setupCheckInEnv(t)
	payload := []byte("update tarball bytes")
	artefact, dep := deployUpdateToCheckInNode(t, e, payload)

	var got CheckInResponse
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestBinary == nil {
		t.Fatal("expected latestBinary in check-in response")
	}
	if want := formatDeploymentID(artefactBinary, dep.ID); got.LatestBinary.Deployment.ID != want {
		t.Errorf("binary deployment id = %q, want %q", got.LatestBinary.Deployment.ID, want)
	}
	if got.LatestBinary.Deployment.Artefact != "binary" {
		t.Errorf("deployment artefact = %q, want binary", got.LatestBinary.Deployment.Artefact)
	}
	version := got.LatestBinary.Version
	if version.Version != "2.0.0" {
		t.Errorf("version = %s, want 2.0.0", version.Version)
	}
	if version.Checksum != artefact.Checksum || version.Checksum == "" {
		t.Errorf("version checksum = %q, want %q", version.Checksum, artefact.Checksum)
	}
	if version.PayloadURL == "" {
		t.Fatal("expected payloadUrl in latestBinary version")
	}

	// The capability-free payload URL is fetchable and returns the bytes.
	dl, err := e.client.Get(version.PayloadURL)
	if err != nil {
		t.Fatalf("download payloadUrl: %v", err)
	}
	defer func() { _ = dl.Body.Close() }()
	assertStatus(t, dl, http.StatusOK)
	body, _ := io.ReadAll(dl.Body)
	if !bytes.Equal(body, payload) {
		t.Errorf("payloadUrl bytes mismatch: got %d bytes", len(body))
	}
}

func TestCheckIn_UpdateInstallingThenCurrent(t *testing.T) {
	e := setupCheckInEnv(t)
	_, dep := deployUpdateToCheckInNode(t, e, []byte("data"))

	// Report installing -> deployment goes IN_PROGRESS.
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"progress": []map[string]any{{"deploymentId": formatDeploymentID(artefactBinary, dep.ID), "state": "installing", "attempts": 1}},
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	var d BinaryDeployment
	resp = doRequest(t, e.client, "GET", binaryDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	if d.Status != "in_progress" {
		t.Errorf("after installing report: status = %s, want in_progress", d.Status)
	}

	// Report current -> deployment COMPLETED.
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"progress": []map[string]any{{"deploymentId": formatDeploymentID(artefactBinary, dep.ID), "state": "applied"}},
	}, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = doRequest(t, e.client, "GET", binaryDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	if d.Status != "completed" {
		t.Errorf("after current report: status = %s, want completed", d.Status)
	}
}

func TestCheckIn_UpdateFailed(t *testing.T) {
	e := setupCheckInEnv(t)
	_, dep := deployUpdateToCheckInNode(t, e, []byte("data"))

	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"progress": []map[string]any{{"deploymentId": formatDeploymentID(artefactBinary, dep.ID), "state": "failed", "reason": "rolled back"}},
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	var d BinaryDeployment
	resp = doRequest(t, e.client, "GET", binaryDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	if d.Status != "failed" {
		t.Errorf("status = %s, want failed", d.Status)
	}
	if d.Reason != "rolled back" {
		t.Errorf("reason = %q, want %q", d.Reason, "rolled back")
	}
}

func TestCheckIn_ConfigAndUpdateChannelsIndependent(t *testing.T) {
	e := setupCheckInEnv(t)

	// Active config deployment for the node's config version.
	resp := doRequest(t, e.client, "POST", e.testServer.URL+"/api/v1/management/config-deployments", map[string]any{
		"configVersionId": sid(e.configVersion.ID),
		"status":          "pending",
	}, nil)
	assertStatus(t, resp, http.StatusCreated)

	// Active binary deployment for the node.
	_, _ = deployUpdateToCheckInNode(t, e, []byte("data"))

	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Error("expected latestConfig (config channel unaffected)")
	}
	if got.LatestBinary == nil {
		t.Error("expected latestBinary (binary channel)")
	}
}
