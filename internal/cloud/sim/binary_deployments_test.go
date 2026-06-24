package sim

import (
	"fmt"
	"net/http"
	"testing"
)

func listBinaryDeploymentsURL(base string) string {
	return base + "/api/v1/management/binary-deployments"
}
func binaryDeploymentURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/binary-deployments/%d", base, id)
}

// makeArtefact uploads an artefact and returns it.
func makeArtefact(t *testing.T, e binaryArtefactEnv, os, arch, siteId string) BinaryArtefact {
	t.Helper()
	fields := map[string]string{"version": "1.0.0", "os": os, "arch": arch}
	if siteId != "" {
		fields["siteId"] = siteId
	}
	var a BinaryArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, fields, []byte("artefact-data"), &a)
	assertStatus(t, resp, http.StatusCreated)
	return a
}

func TestBinaryDeployments_CreateMatching(t *testing.T) {
	e := setupBinaryArtefactEnv(t) // e.node is linux/arm64 on e.site
	artefact := makeArtefact(t, e, "linux", "arm64", sid(e.site.ID))

	var dep BinaryDeployment
	resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)
	if dep.Status != "pending" {
		t.Errorf("status = %s, want pending", dep.Status)
	}
	if dep.NodeID != e.node.ID || dep.BinaryArtefactID != artefact.ID {
		t.Errorf("deployment refs wrong: node=%d artefact=%d", dep.NodeID, dep.BinaryArtefactID)
	}
}

func TestBinaryDeployments_GenericArtefactMatchesAnySite(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	generic := makeArtefact(t, e, "linux", "arm64", "") // no site

	resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(generic.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusCreated)
}

func TestBinaryDeployments_RejectsPlatformMismatch(t *testing.T) {
	e := setupBinaryArtefactEnv(t) // node is linux/arm64
	freebsdArtefact := makeArtefact(t, e, "freebsd", "arm64", sid(e.site.ID))

	resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(freebsdArtefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestBinaryDeployments_RejectsSiteMismatch(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	// A second site with its own artefact; deploying it to e.node (site A) must fail.
	var otherSite Site
	resp := doRequest(t, e.client, "POST", listSitesURL(e.testServer.URL), map[string]string{"name": "Site B"}, &otherSite)
	assertStatus(t, resp, http.StatusCreated)
	otherArtefact := makeArtefact(t, e, "linux", "arm64", sid(otherSite.ID))

	resp = doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(otherArtefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestBinaryDeployments_CancelsPriorPending(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	artefact := makeArtefact(t, e, "linux", "arm64", sid(e.site.ID))

	var first BinaryDeployment
	resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &first)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, nil)
	assertStatus(t, resp, http.StatusCreated)

	var old BinaryDeployment
	resp = doRequest(t, e.client, "GET", binaryDeploymentURL(e.testServer.URL, first.ID), nil, &old)
	assertStatus(t, resp, http.StatusOK)
	if old.Status != "cancelled" {
		t.Errorf("prior pending deployment status = %s, want cancelled", old.Status)
	}
}

func TestBinaryDeployments_StatusLifecycle(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	artefact := makeArtefact(t, e, "linux", "arm64", sid(e.site.ID))
	var dep BinaryDeployment
	resp := doRequest(t, e.client, "POST", listBinaryDeploymentsURL(e.testServer.URL), map[string]any{
		"binaryArtefactId": sid(artefact.ID),
		"nodeId":           sid(e.node.ID),
	}, &dep)
	assertStatus(t, resp, http.StatusCreated)

	var updated BinaryDeployment
	resp = doRequest(t, e.client, "PATCH", binaryDeploymentURL(e.testServer.URL, dep.ID), map[string]any{
		"status": "completed",
	}, &updated)
	assertStatus(t, resp, http.StatusOK)
	if updated.Status != "completed" || updated.FinishedTime == nil {
		t.Errorf("completed: status=%s finished=%v", updated.Status, updated.FinishedTime)
	}

	resp = doRequest(t, e.client, "DELETE", binaryDeploymentURL(e.testServer.URL, dep.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
}
