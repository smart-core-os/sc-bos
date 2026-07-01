package sim

import (
	"fmt"
	"net/http"
	"testing"
)

func listUpdateDeploymentsURL(base string) string {
	return base + "/api/v1/management/update-deployments"
}
func updateDeploymentURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/update-deployments/%d", base, id)
}

// makeArtefact uploads an artefact and returns it.
func makeArtefact(t *testing.T, e updateArtefactEnv, platform, siteId string) UpdateArtefact {
	t.Helper()
	fields := map[string]string{"version": "1.0.0", "platform": platform}
	if siteId != "" {
		fields["siteId"] = siteId
	}
	var a UpdateArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, fields, []byte("artefact-data"), &a)
	assertStatus(t, resp, http.StatusCreated)
	return a
}

func TestUpdateDeployments_CreateMatching(t *testing.T) {
	e := setupUpdateArtefactEnv(t) // e.node is podman on e.site
	artefact := makeArtefact(t, e, "podman", sid(e.site.ID))

	var dep UpdateDeployment
	resp := doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	if dep.Status != "pending" {
		t.Errorf("status = %s, want pending", dep.Status)
	}
	if dep.NodeID != e.node.ID || dep.UpdateArtefactID != artefact.ID {
		t.Errorf("deployment refs wrong: node=%d artefact=%d", dep.NodeID, dep.UpdateArtefactID)
	}
}

func TestUpdateDeployments_GenericArtefactMatchesAnySite(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	generic := makeArtefact(t, e, "podman", "") // no site

	resp := doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(generic.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusCreated)
}

func TestUpdateDeployments_RejectsPlatformMismatch(t *testing.T) {
	e := setupUpdateArtefactEnv(t) // node is podman
	freebsdArtefact := makeArtefact(t, e, "freebsd", sid(e.site.ID))

	resp := doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(freebsdArtefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdateDeployments_RejectsSiteMismatch(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	// A second site with its own artefact; deploying it to e.node (site A) must fail.
	var otherSite Site
	resp := doRequest(t, e.client, "POST", listSitesURL(e.testServer.URL), map[string]string{"name": "Site B"}, &otherSite)
	assertStatus(t, resp, http.StatusCreated)
	otherArtefact := makeArtefact(t, e, "podman", sid(otherSite.ID))

	resp = doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(otherArtefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdateDeployments_CancelsPriorPending(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	artefact := makeArtefact(t, e, "podman", sid(e.site.ID))

	var first UpdateDeployment
	resp := doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &first)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusCreated)

	var old UpdateDeployment
	resp = doRequest(t, e.client, "GET", updateDeploymentURL(e.testServer.URL, first.ID), nil, &old)
	assertStatus(t, resp, http.StatusOK)
	if old.Status != "cancelled" {
		t.Errorf("prior pending deployment status = %s, want cancelled", old.Status)
	}
}

func TestUpdateDeployments_StatusLifecycle(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	artefact := makeArtefact(t, e, "podman", sid(e.site.ID))
	var dep UpdateDeployment
	resp := doRequest(t, e.client, "POST", listUpdateDeploymentsURL(e.testServer.URL), map[string]any{
		"updateArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	var updated UpdateDeployment
	resp = doRequest(t, e.client, "PATCH", updateDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "completed",
	}, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "completed" || updated.FinishedTime == nil {
		t.Errorf("completed: status=%s finished=%v", updated.Status, updated.FinishedTime)
	}

	resp = doRequest(t, e.client, "DELETE", updateDeploymentURL(e.testServer.URL, dep.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
}
