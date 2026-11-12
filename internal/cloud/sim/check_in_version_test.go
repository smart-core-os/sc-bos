package sim

import (
	"net/http"
	"testing"
)

func binaryDeploymentStatus(t *testing.T, e checkInEnv, id int64) string {
	t.Helper()
	var d BinaryDeployment
	resp := doRequest(t, e.client, "GET", binaryDeploymentURL(e.testServer.URL, id), nil, &d)
	assertStatus(t, resp, http.StatusOK)
	return d.Status
}

func checkInRunningBinary(t *testing.T, e checkInEnv, binary map[string]any) CheckInResponse {
	t.Helper()
	var got CheckInResponse
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"running": map[string]any{"binary": binary},
	}, &got)
	assertStatus(t, resp, http.StatusOK)
	return got
}

// TestCheckIn_BinaryVersionIDCompletesDeployment: a check-in reporting the deployment's target version
// id completes the deployment and offers nothing; any other id leaves it offered.
func TestCheckIn_BinaryVersionIDCompletesDeployment(t *testing.T) {
	e := setupCheckInEnv(t)
	artefact, dep := deployUpdateToCheckInNode(t, e, []byte("data"))

	got := checkInRunningBinary(t, e, map[string]any{"version": "1.0.0", "versionId": "999"})
	if got.LatestBinary == nil {
		t.Fatal("running another version: expected latestBinary")
	}

	got = checkInRunningBinary(t, e, map[string]any{"version": "2.0.0", "versionId": sid(artefact.ID)})
	if got.LatestBinary != nil {
		t.Errorf("running the target: latestBinary = %+v, want none", got.LatestBinary)
	}
	if status := binaryDeploymentStatus(t, e, dep.ID); status != statusCompleted {
		t.Errorf("running the target: status = %s, want completed", status)
	}
}

// TestCheckIn_BinaryVersionIDRecorded: the reported id is kept across check-ins. An absent id leaves it
// as it was, and null clears it.
func TestCheckIn_BinaryVersionIDRecorded(t *testing.T) {
	e := setupCheckInEnv(t)
	artefact, _ := deployUpdateToCheckInNode(t, e, []byte("data"))
	checkInRunningBinary(t, e, map[string]any{"version": "2.0.0", "versionId": sid(artefact.ID)})

	// absent: the recorded id completes a new deployment of that version
	redeploy := func() BinaryDeployment {
		t.Helper()
		var d BinaryDeployment
		resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
			"binaryArtefactId": sid(artefact.ID),
			"nodeId":           sid(e.node.ID),
		}, &d)
		assertStatus(t, resp, http.StatusCreated)
		return d
	}
	dep := redeploy()
	if got := checkInRunningBinary(t, e, map[string]any{"version": "2.0.0"}); got.LatestBinary != nil {
		t.Errorf("absent id: latestBinary = %+v, want none", got.LatestBinary)
	}
	if status := binaryDeploymentStatus(t, e, dep.ID); status != statusCompleted {
		t.Errorf("absent id: status = %s, want completed", status)
	}

	// null: the node runs no known version, so a new deployment is offered
	checkInRunningBinary(t, e, map[string]any{"version": "2.0.0", "versionId": nil})
	dep = redeploy()
	if got := checkInRunningBinary(t, e, map[string]any{"version": "2.0.0"}); got.LatestBinary == nil {
		t.Error("after null id: expected latestBinary")
	}
	if status := binaryDeploymentStatus(t, e, dep.ID); status == statusCompleted {
		t.Errorf("after null id: status = %s, want not completed", status)
	}
}

// TestCheckIn_ConfigVersionIDCompletesDeployment: config follows the same rule as binary.
func TestCheckIn_ConfigVersionIDCompletesDeployment(t *testing.T) {
	e := setupCheckInEnv(t)
	var dep ConfigDeployment
	resp := doRequest(t, e.client, "POST", listConfigDeploymentsURL(e.testServer.URL), map[string]any{
		"configVersionId": sid(e.configVersion.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	var got CheckInResponse
	resp = doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"running": map[string]any{"config": map[string]any{"versionId": sid(e.configVersion.ID)}},
	}, &got)
	assertStatus(t, resp, http.StatusOK)
	if got.LatestConfig != nil {
		t.Errorf("latestConfig = %+v, want none", got.LatestConfig)
	}
	resp = doRequest(t, e.client, "GET", configDeploymentURL(e.testServer.URL, dep.ID), nil, &dep)
	assertStatus(t, resp, http.StatusOK)
	if dep.Status != statusCompleted {
		t.Errorf("status = %s, want completed", dep.Status)
	}
}

func TestCheckIn_InvalidVersionIDRejected(t *testing.T) {
	e := setupCheckInEnv(t)
	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), map[string]any{
		"running": map[string]any{"binary": map[string]any{"versionId": 5}},
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}
