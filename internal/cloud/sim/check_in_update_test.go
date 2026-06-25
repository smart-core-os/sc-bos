package sim

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// deployUpdateToCheckInNode uploads a podman artefact for the env's site and creates a pending update
// deployment targeting the env's node. Returns the artefact and the deployment.
func deployUpdateToCheckInNode(t *testing.T, e checkInEnv, payload []byte) (UpdateArtefact, UpdateDeployment) {
	t.Helper()
	var artefact UpdateArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version":  "2.0.0",
		"platform": "podman",
		"siteId":   sid(e.site.ID),
	}, payload, &artefact)
	assertStatus(t, resp, http.StatusCreated)

	var dep UpdateDeployment
	resp = doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	return artefact, dep
}

// deployKindToCheckInNode uploads an artefact of the given kind/version for the env's site and creates a
// pending deployment targeting the env's node.
func deployKindToCheckInNode(t *testing.T, e checkInEnv, kind, version string, payload []byte) (UpdateArtefact, UpdateDeployment) {
	t.Helper()
	var artefact UpdateArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version":  version,
		"platform": "podman",
		"kind":     kind,
		"siteId":   sid(e.site.ID),
	}, payload, &artefact)
	assertStatus(t, resp, http.StatusCreated)

	var dep UpdateDeployment
	resp = doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	return artefact, dep
}

// TestCheckIn_BothUpdateChannels: a node may have a BOS-image deployment and a supervisor-rpm
// deployment in flight at once (they do not conflict), and a check-in returns both channels.
func TestCheckIn_BothUpdateChannels(t *testing.T) {
	e := setupCheckInEnv(t)
	_, bosDep := deployKindToCheckInNode(t, e, ArtefactKindBOSImage, "2.0.0", []byte("bos tarball"))
	_, supDep := deployKindToCheckInNode(t, e, ArtefactKindSupervisorRPM, "9.9.9", []byte("supervisor rpm"))

	var got CheckInResponse
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestUpdate == nil || got.LatestUpdate.UpdateDeployment.ID != bosDep.ID {
		t.Errorf("latestUpdate = %+v, want bos deployment %d", got.LatestUpdate, bosDep.ID)
	}
	if got.LatestSupervisorUpdate == nil || got.LatestSupervisorUpdate.UpdateDeployment.ID != supDep.ID {
		t.Errorf("latestSupervisorUpdate = %+v, want supervisor deployment %d", got.LatestSupervisorUpdate, supDep.ID)
	}
	if got.LatestSupervisorUpdate != nil && got.LatestSupervisorUpdate.UpdateArtefact.Kind != ArtefactKindSupervisorRPM {
		t.Errorf("supervisor artefact kind = %q, want %q", got.LatestSupervisorUpdate.UpdateArtefact.Kind, ArtefactKindSupervisorRPM)
	}
}

func TestCheckIn_LatestUpdate(t *testing.T) {
	e := setupCheckInEnv(t)
	payload := []byte("update tarball bytes")
	artefact, dep := deployUpdateToCheckInNode(t, e, payload)

	var got CheckInResponse
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestUpdate == nil {
		t.Fatal("expected latestUpdate in check-in response")
	}
	if got.LatestUpdate.UpdateDeployment.ID != dep.ID {
		t.Errorf("update deployment id = %d, want %d", got.LatestUpdate.UpdateDeployment.ID, dep.ID)
	}
	ua := got.LatestUpdate.UpdateArtefact
	if ua.Version != "2.0.0" {
		t.Errorf("artefact version = %s, want 2.0.0", ua.Version)
	}
	if ua.SHA256 != artefact.SHA256 || ua.SHA256 == "" {
		t.Errorf("artefact sha256 = %q, want %q", ua.SHA256, artefact.SHA256)
	}
	if ua.PayloadURL == "" {
		t.Fatal("expected payloadUrl in latestUpdate artefact")
	}

	// The capability-free payload URL is fetchable and returns the bytes.
	dl, err := e.client.Get(ua.PayloadURL)
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
	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, map[string]any{
		"installingUpdate": map[string]any{"id": sid(dep.ID), "attempts": 1},
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	var d UpdateDeployment
	resp = doRequest(t, e.client, "GET", updateDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	if d.Status != "in_progress" {
		t.Errorf("after installing report: status = %s, want in_progress", d.Status)
	}

	// Report current -> deployment COMPLETED.
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, map[string]any{
		"currentUpdate": map[string]any{"id": sid(dep.ID)},
	}, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = doRequest(t, e.client, "GET", updateDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	if d.Status != "completed" {
		t.Errorf("after current report: status = %s, want completed", d.Status)
	}
}

func TestCheckIn_UpdateFailed(t *testing.T) {
	e := setupCheckInEnv(t)
	_, dep := deployUpdateToCheckInNode(t, e, []byte("data"))

	resp := doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, map[string]any{
		"failedUpdate": map[string]any{"id": sid(dep.ID), "reason": "rolled back"},
	}, nil)
	assertStatus(t, resp, http.StatusOK)

	var d UpdateDeployment
	resp = doRequest(t, e.client, "GET", updateDeploymentURL(e.testServer.URL, dep.ID), nil, &d)
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

	// Active update deployment for the node.
	_, _ = deployUpdateToCheckInNode(t, e, []byte("data"))

	var got CheckInResponse
	resp = doCheckIn(t, e.client, checkInURL(e.testServer.URL), e.accessToken, nil, &got)
	assertStatus(t, resp, http.StatusOK)

	if got.LatestConfig == nil {
		t.Error("expected latestConfig (config channel unaffected)")
	}
	if got.LatestUpdate == nil {
		t.Error("expected latestUpdate (update channel)")
	}
}
