package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
)

func TestSites(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	t.Run("empty list", func(t *testing.T) {
		var list ListResponse[Site]
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL), nil, &list)
		assertStatus(t, resp, http.StatusOK)

		if diff := cmp.Diff(ListResponse[Site]{Items: []Site{}}, list); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	var site Site

	t.Run("create happy path", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
		assertStatus(t, resp, http.StatusCreated)

		want := Site{Name: "Test Site"}
		if diff := cmp.Diff(want, site, cmpopts.IgnoreFields(Site{}, "ID", "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if site.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if site.CreateTime.IsZero() {
			t.Error("expected non-zero CreateTime")
		}
	})

	t.Run("create invalid payload", func(t *testing.T) {
		testInvalidJSON(t, client, "POST", listSitesURL(ts.URL))
	})

	t.Run("create missing name", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{}, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("get", func(t *testing.T) {
		var gotSite Site
		resp := doRequest(t, client, "GET", siteURL(ts.URL, site.ID), nil, &gotSite)
		assertStatus(t, resp, http.StatusOK)

		want := Site{ID: gotSite.ID, Name: "Test Site", CreateTime: gotSite.CreateTime}
		if diff := cmp.Diff(want, gotSite); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("update", func(t *testing.T) {
		var updatedSite Site
		resp := doRequest(t, client, "PUT", siteURL(ts.URL, site.ID), map[string]string{"name": "Updated Site"}, &updatedSite)
		assertStatus(t, resp, http.StatusOK)

		want := Site{ID: site.ID, Name: "Updated Site", CreateTime: site.CreateTime}
		if diff := cmp.Diff(want, updatedSite); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		site = updatedSite
	})

	t.Run("list after create", func(t *testing.T) {
		var list ListResponse[Site]
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL), nil, &list)
		assertStatus(t, resp, http.StatusOK)

		if len(list.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(list.Items))
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL)+"/invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("update invalid id", func(t *testing.T) {
		testInvalidID(t, client, "PUT", listSitesURL(ts.URL)+"/%s")
	})

	t.Run("update invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "PUT", siteURL(ts.URL, site.ID))
	})

	t.Run("delete invalid id", func(t *testing.T) {
		testInvalidID(t, client, "DELETE", listSitesURL(ts.URL)+"/%s")
	})

	t.Run("delete", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", siteURL(ts.URL, site.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		// Verify it's gone
		resp = doRequest(t, client, "GET", siteURL(ts.URL, site.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("delete not found", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", listSitesURL(ts.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestSites_Pagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	testPagination(t,
		func(i int) int64 {
			var site Site
			resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": fmt.Sprintf("Site %d", i)}, &site)
			assertStatus(t, resp, http.StatusCreated)
			return site.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Site]
			resp = doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Site]
			resp = doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)
}

func TestNodes(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Create a site first
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node

	t.Run("create", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
			"hostname": "node-01",
			"siteId":   site.ID,
		}, &node)
		assertStatus(t, resp, http.StatusCreated)

		want := Node{Hostname: "node-01", SiteID: site.ID}
		if diff := cmp.Diff(want, node, cmpopts.IgnoreFields(Node{}, "ID", "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if node.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if node.CreateTime.IsZero() {
			t.Error("expected non-zero CreateTime")
		}
	})

	t.Run("create invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "POST", listNodesURL(ts.URL))
	})

	t.Run("create with invalid siteId", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listNodesURL(ts.URL), map[string]any{
			"hostname": "orphan-node",
			"siteId":   99999,
		})
	})

	t.Run("get", func(t *testing.T) {
		var gotNode Node
		resp := doRequest(t, client, "GET", nodeURL(ts.URL, node.ID), nil, &gotNode)
		assertStatus(t, resp, http.StatusOK)

		want := Node{ID: node.ID, Hostname: "node-01", SiteID: site.ID, CreateTime: node.CreateTime}
		if diff := cmp.Diff(want, gotNode, cmpopts.IgnoreFields(Node{}, "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("get invalid id", func(t *testing.T) {
		testInvalidID(t, client, "GET", listNodesURL(ts.URL)+"/%s")
	})

	t.Run("update", func(t *testing.T) {
		var updatedNode Node
		resp := doRequest(t, client, "PUT", nodeURL(ts.URL, node.ID), map[string]any{
			"hostname": "updatedNode-01-updated",
			"siteId":   site.ID,
		}, &updatedNode)
		assertStatus(t, resp, http.StatusOK)

		want := Node{ID: node.ID, Hostname: "updatedNode-01-updated", SiteID: site.ID, CreateTime: node.CreateTime}
		if diff := cmp.Diff(want, updatedNode); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		node = updatedNode
	})

	t.Run("update invalid id", func(t *testing.T) {
		testInvalidID(t, client, "PUT", listNodesURL(ts.URL)+"/%s")
	})

	t.Run("update not found", func(t *testing.T) {
		testNotFound(t, client, "PUT", listNodesURL(ts.URL)+"/%d", map[string]any{
			"hostname": "x",
			"siteId":   site.ID,
		})
	})

	t.Run("update invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "PUT", nodeURL(ts.URL, node.ID))
	})

	t.Run("update with invalid siteId", func(t *testing.T) {
		testForeignKeyError(t, client, "PUT", nodeURL(ts.URL, node.ID), map[string]any{
			"hostname": "x",
			"siteId":   99999,
		})
	})

	t.Run("list with filter", func(t *testing.T) {
		// Create another site with a node
		var site2 Site
		resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Site 2"}, &site2)
		assertStatus(t, resp, http.StatusCreated)

		resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
			"hostname": "node-02",
			"siteId":   site2.ID,
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all nodes
		var allNodes ListResponse[Node]
		resp = doRequest(t, client, "GET", listNodesURL(ts.URL), nil, &allNodes)
		assertStatus(t, resp, http.StatusOK)
		if len(allNodes.Items) != 2 {
			t.Errorf("expected 2 nodes total, got %d", len(allNodes.Items))
		}

		// Filter by first site
		var filtered ListResponse[Node]
		resp = doRequest(t, client, "GET", fmt.Sprintf("%s?siteId=%d", listNodesURL(ts.URL), site.ID), nil, &filtered)
		assertStatus(t, resp, http.StatusOK)
		if len(filtered.Items) != 1 {
			t.Errorf("expected 1 node for site %d, got %d", site.ID, len(filtered.Items))
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listNodesURL(ts.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("delete invalid id", func(t *testing.T) {
		testInvalidID(t, client, "DELETE", listNodesURL(ts.URL)+"/%s")
	})

	t.Run("delete not found", func(t *testing.T) {
		testNotFound(t, client, "DELETE", listNodesURL(ts.URL)+"/%d", nil)
	})

	t.Run("delete", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", nodeURL(ts.URL, node.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		resp = doRequest(t, client, "GET", nodeURL(ts.URL, node.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestNodes_Pagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create a site first
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	testPagination(t,
		func(i int) int64 {
			var node Node
			resp := doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
				"hostname": fmt.Sprintf("node-%d", i),
				"siteId":   site.ID,
			}, &node)
			assertStatus(t, resp, http.StatusCreated)
			return node.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Node]
			resp = doRequest(t, client, "GET", listNodesURL(ts.URL)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Node]
			resp = doRequest(t, client, "GET", listNodesURL(ts.URL)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)
}

func TestConfigVersions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site and node
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-01",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	var cv ConfigVersion
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	t.Run("create", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
			"nodeId":      node.ID,
			"description": "v1.0.0",
			"payload":     payload,
		}, &cv)
		assertStatus(t, resp, http.StatusCreated)

		want := ConfigVersion{NodeID: node.ID, Description: "v1.0.0", Payload: payload}
		if diff := cmp.Diff(want, cv, cmpopts.IgnoreFields(ConfigVersion{}, "ID", "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if cv.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if cv.CreateTime.IsZero() {
			t.Error("expected non-zero CreateTime")
		}
	})

	t.Run("create invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "POST", listConfigVersionsURL(ts.URL))
	})

	t.Run("create with invalid nodeId", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
			"nodeId":      99999,
			"description": "v1.0.0",
			"payload":     []byte("data"),
		})
	})

	t.Run("get", func(t *testing.T) {
		var getCV ConfigVersion
		resp := doRequest(t, client, "GET", configVersionURL(ts.URL, cv.ID), nil, &getCV)
		assertStatus(t, resp, http.StatusOK)

		want := ConfigVersion{ID: cv.ID, NodeID: node.ID, Description: "v1.0.0", Payload: payload}
		if diff := cmp.Diff(want, getCV, cmpopts.IgnoreFields(ConfigVersion{}, "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if getCV.CreateTime.IsZero() {
			t.Error("expected non-zero CreateTime")
		}
	})

	t.Run("get invalid id", func(t *testing.T) {
		testInvalidID(t, client, "GET", listConfigVersionsURL(ts.URL)+"/%s")
	})

	t.Run("list with filter", func(t *testing.T) {
		// Create another config version
		resp := doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
			"nodeId":      node.ID,
			"description": "v2.0.0",
			"payload":     payload,
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all
		var all ListResponse[ConfigVersion]
		resp = doRequest(t, client, "GET", listConfigVersionsURL(ts.URL), nil, &all)
		assertStatus(t, resp, http.StatusOK)
		if len(all.Items) != 2 {
			t.Errorf("expected 2 config versions, got %d", len(all.Items))
		}

		// Filter by node
		var filtered ListResponse[ConfigVersion]
		resp = doRequest(t, client, "GET", fmt.Sprintf("%s?nodeId=%d", listConfigVersionsURL(ts.URL), node.ID), nil, &filtered)
		assertStatus(t, resp, http.StatusOK)
		if len(filtered.Items) != 2 {
			t.Errorf("expected 2 config versions for node, got %d", len(filtered.Items))
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listConfigVersionsURL(ts.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("delete invalid id", func(t *testing.T) {
		testInvalidID(t, client, "DELETE", listConfigVersionsURL(ts.URL)+"/%s")
	})

	t.Run("delete not found", func(t *testing.T) {
		testNotFound(t, client, "DELETE", listConfigVersionsURL(ts.URL)+"/%d", nil)
	})

	t.Run("delete", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", configVersionURL(ts.URL, cv.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		resp = doRequest(t, client, "GET", configVersionURL(ts.URL, cv.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestConfigVersions_Pagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site and node first
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	testPagination(t,
		func(i int) int64 {
			var cv ConfigVersion
			resp := doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
				"nodeId":      node.ID,
				"description": fmt.Sprintf("v%d.0.0", i),
				"payload":     []byte(fmt.Sprintf("config-%d", i)),
			}, &cv)
			assertStatus(t, resp, http.StatusCreated)
			return cv.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[ConfigVersion]
			resp = doRequest(t, client, "GET", listConfigVersionsURL(ts.URL)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[ConfigVersion]
			resp = doRequest(t, client, "GET", listConfigVersionsURL(ts.URL)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)
}

func TestDeployments(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site, node, config version
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-01",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node.ID,
		"description": "v1.0.0",
		"payload":     []byte("config"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	var deployment Deployment

	t.Run("create", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": cv.ID,
			"status":          "PENDING",
		}, &deployment)
		assertStatus(t, resp, http.StatusCreated)

		want := Deployment{ConfigVersionID: cv.ID, Status: "PENDING", FinishedTime: nil}
		if diff := cmp.Diff(want, deployment, cmpopts.IgnoreFields(Deployment{}, "ID", "StartTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if deployment.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if deployment.StartTime.IsZero() {
			t.Error("expected non-zero StartTime")
		}
	})

	t.Run("create invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "POST", listDeploymentsURL(ts.URL))
	})

	t.Run("create with invalid configVersionId", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": 99999,
			"status":          "PENDING",
		})
	})

	t.Run("create with invalid status", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": cv.ID,
			"status":          "INVALID",
		}, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("create with empty status", func(t *testing.T) {
		resp := doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": cv.ID,
			"status":          "",
		}, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("get", func(t *testing.T) {
		var d Deployment
		resp := doRequest(t, client, "GET", deploymentURL(ts.URL, deployment.ID), nil, &d)
		assertStatus(t, resp, http.StatusOK)

		want := Deployment{ID: deployment.ID, ConfigVersionID: cv.ID, Status: "PENDING"}
		if diff := cmp.Diff(want, d, cmpopts.IgnoreFields(Deployment{}, "StartTime", "FinishedTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if d.StartTime.IsZero() {
			t.Error("expected non-zero StartTime")
		}
	})

	t.Run("get invalid id", func(t *testing.T) {
		testInvalidID(t, client, "GET", listDeploymentsURL(ts.URL)+"/%s")
	})

	t.Run("update status to completed", func(t *testing.T) {
		var d Deployment
		resp := doRequest(t, client, "PATCH", deploymentURL(ts.URL, deployment.ID), map[string]string{
			"status": "COMPLETED",
		}, &d)
		assertStatus(t, resp, http.StatusOK)

		want := Deployment{ID: deployment.ID, ConfigVersionID: cv.ID, StartTime: deployment.StartTime, Status: "COMPLETED"}
		if diff := cmp.Diff(want, d, cmpopts.IgnoreFields(Deployment{}, "StartTime", "FinishedTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if d.FinishedTime == nil || d.FinishedTime.IsZero() {
			t.Errorf("expected finishedTime to be set for COMPLETED, got %v", d.FinishedTime)
		}
	})

	t.Run("update status invalid id", func(t *testing.T) {
		testInvalidID(t, client, "PATCH", listDeploymentsURL(ts.URL)+"/%s")
	})

	t.Run("update status not found", func(t *testing.T) {
		testNotFound(t, client, "PATCH", listDeploymentsURL(ts.URL)+"/%d", map[string]string{
			"status": "COMPLETED",
		})
	})

	t.Run("update status invalid json", func(t *testing.T) {
		testInvalidJSON(t, client, "PATCH", deploymentURL(ts.URL, deployment.ID))
	})

	t.Run("update with invalid status", func(t *testing.T) {
		resp := doRequest(t, client, "PATCH", deploymentURL(ts.URL, deployment.ID), map[string]string{
			"status": "INVALID_STATUS",
		}, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("list with filters", func(t *testing.T) {
		// Create another deployment
		resp := doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": cv.ID,
			"status":          "PENDING",
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all
		var all ListResponse[Deployment]
		resp = doRequest(t, client, "GET", listDeploymentsURL(ts.URL), nil, &all)
		assertStatus(t, resp, http.StatusOK)
		if len(all.Items) != 2 {
			t.Errorf("expected 2 deployments, got %d", len(all.Items))
		}

		// Filter by configVersionId
		var byCv ListResponse[Deployment]
		resp = doRequest(t, client, "GET", fmt.Sprintf("%s?configVersionId=%d", listDeploymentsURL(ts.URL), cv.ID), nil, &byCv)
		assertStatus(t, resp, http.StatusOK)
		if len(byCv.Items) != 2 {
			t.Errorf("expected 2 deployments by configVersionId, got %d", len(byCv.Items))
		}

		// Filter by nodeId
		var byNode ListResponse[Deployment]
		resp = doRequest(t, client, "GET", fmt.Sprintf("%s?nodeId=%d", listDeploymentsURL(ts.URL), node.ID), nil, &byNode)
		assertStatus(t, resp, http.StatusOK)
		if len(byNode.Items) != 2 {
			t.Errorf("expected 2 deployments by nodeId, got %d", len(byNode.Items))
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("delete invalid id", func(t *testing.T) {
		testInvalidID(t, client, "DELETE", listDeploymentsURL(ts.URL)+"/%s")
	})

	t.Run("delete not found", func(t *testing.T) {
		testNotFound(t, client, "DELETE", listDeploymentsURL(ts.URL)+"/%d", nil)
	})

	t.Run("delete", func(t *testing.T) {
		resp := doRequest(t, client, "DELETE", deploymentURL(ts.URL, deployment.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		resp = doRequest(t, client, "GET", deploymentURL(ts.URL, deployment.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestDeployments_Pagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site, node, and config version first
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node.ID,
		"description": "v1.0.0",
		"payload":     []byte("config"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	testPagination(t,
		func(i int) int64 {
			var deployment Deployment
			resp := doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
				"configVersionId": cv.ID,
				"status":          "PENDING",
			}, &deployment)
			assertStatus(t, resp, http.StatusCreated)
			return deployment.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Deployment]
			resp = doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Deployment]
			resp = doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)
}

func TestCascadeDelete(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Create a full chain
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Cascade Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "cascade-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node.ID,
		"description": "v1.0.0",
		"payload":     []byte("data"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	var deployment Deployment
	resp = doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
		"configVersionId": cv.ID,
		"status":          "PENDING",
	}, &deployment)
	assertStatus(t, resp, http.StatusCreated)

	// Delete site
	resp = doRequest(t, client, "DELETE", siteURL(ts.URL, site.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Node should be gone too
	resp = doRequest(t, client, "GET", nodeURL(ts.URL, node.ID), nil, nil)
	assertStatus(t, resp, http.StatusNotFound)

	// Config version should be gone
	resp = doRequest(t, client, "GET", configVersionURL(ts.URL, cv.ID), nil, nil)
	assertStatus(t, resp, http.StatusNotFound)

	// Deployment should be gone
	resp = doRequest(t, client, "GET", deploymentURL(ts.URL, deployment.ID), nil, nil)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestEdgeCases(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	t.Run("invalid json body", func(t *testing.T) {
		req, err := http.NewRequest("POST", listSitesURL(ts.URL), bytes.NewBufferString("not json"))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid page token", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageToken=!!invalid!!", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("invalid page size", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageSize=abc", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("update not found", func(t *testing.T) {
		resp := doRequest(t, client, "PUT", listSitesURL(ts.URL)+"/99999", map[string]string{"name": "X"}, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("foreign key node with invalid site", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listNodesURL(ts.URL), map[string]any{
			"hostname": "orphan-node",
			"siteId":   99999,
		})
	})

	t.Run("foreign key config version with invalid node", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
			"nodeId":      99999,
			"description": "v1.0.0",
			"payload":     []byte("data"),
		})
	})

	t.Run("foreign key deployment with invalid config version", func(t *testing.T) {
		testForeignKeyError(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
			"configVersionId": 99999,
			"status":          "PENDING",
		})
	})
}

func TestFilters_EdgeCases(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site, nodes, config versions, and deployments
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node1, node2 Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-01",
		"siteId":   site.ID,
	}, &node1)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-02",
		"siteId":   site.ID,
	}, &node2)
	assertStatus(t, resp, http.StatusCreated)

	var cv1, cv2 ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node1.ID,
		"description": "v1",
		"payload":     []byte("data1"),
	}, &cv1)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node2.ID,
		"description": "v1",
		"payload":     []byte("data2"),
	}, &cv2)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
		"configVersionId": cv1.ID,
		"status":          "PENDING",
	}, nil)
	assertStatus(t, resp, http.StatusCreated)

	resp = doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
		"configVersionId": cv2.ID,
		"status":          "COMPLETED",
	}, nil)
	assertStatus(t, resp, http.StatusCreated)

	t.Run("nodes filter with invalid siteId", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listNodesURL(ts.URL)+"?siteId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("nodes filter with negative siteId should be rejected", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listNodesURL(ts.URL)+"?siteId=-1", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("nodes filter for non-existent site", func(t *testing.T) {
		var list ListResponse[Node]
		resp := doRequest(t, client, "GET", listNodesURL(ts.URL)+"?siteId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 nodes for non-existent site, got %d", len(list.Items))
		}
	})

	t.Run("nodes filter with pagination", func(t *testing.T) {
		var list ListResponse[Node]
		resp := doRequest(t, client, "GET", fmt.Sprintf("%s?siteId=%d&pageSize=1", listNodesURL(ts.URL), site.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 node on first page, got %d", len(list.Items))
		}
		if list.NextPageToken == "" {
			t.Error("expected next page token when using filter with pagination")
		}

		// Get second page with filter
		var page2 ListResponse[Node]
		resp = doRequest(t, client, "GET", fmt.Sprintf("%s?siteId=%d&pageSize=1&pageToken=%s", listNodesURL(ts.URL), site.ID, list.NextPageToken), nil, &page2)
		assertStatus(t, resp, http.StatusOK)
		if len(page2.Items) != 1 {
			t.Errorf("expected 1 node on second page, got %d", len(page2.Items))
		}
	})

	t.Run("config versions filter with invalid nodeId", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listConfigVersionsURL(ts.URL)+"?nodeId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("config versions filter for non-existent node", func(t *testing.T) {
		var list ListResponse[ConfigVersion]
		resp := doRequest(t, client, "GET", listConfigVersionsURL(ts.URL)+"?nodeId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 config versions for non-existent node, got %d", len(list.Items))
		}
	})

	t.Run("config versions filter with pagination", func(t *testing.T) {
		var list ListResponse[ConfigVersion]
		resp := doRequest(t, client, "GET", fmt.Sprintf("%s?nodeId=%d&pageSize=10", listConfigVersionsURL(ts.URL), node1.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 config version for node1, got %d", len(list.Items))
		}
		// Check it's the right config version
		if len(list.Items) > 0 && list.Items[0].ID != cv1.ID {
			t.Errorf("expected config version ID %d, got %d", cv1.ID, list.Items[0].ID)
		}
	})

	t.Run("deployments filter with invalid configVersionId", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"?configVersionId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("deployments filter with invalid nodeId", func(t *testing.T) {
		resp := doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"?nodeId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("deployments filter for non-existent configVersionId", func(t *testing.T) {
		var list ListResponse[Deployment]
		resp := doRequest(t, client, "GET", listDeploymentsURL(ts.URL)+"?configVersionId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 deployments for non-existent config version, got %d", len(list.Items))
		}
	})

	t.Run("deployments filter by nodeId with pagination", func(t *testing.T) {
		var list ListResponse[Deployment]
		resp := doRequest(t, client, "GET", fmt.Sprintf("%s?nodeId=%d&pageSize=1", listDeploymentsURL(ts.URL), node1.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 deployment for node1, got %d", len(list.Items))
		}
		// Verify it's the right deployment (belongs to cv1)
		if len(list.Items) > 0 && list.Items[0].ConfigVersionID != cv1.ID {
			t.Errorf("expected deployment for config version %d, got %d", cv1.ID, list.Items[0].ConfigVersionID)
		}
	})
}

func TestPagination_EdgeCases(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	testPagination(t,
		func(i int) int64 {
			var site Site
			resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": fmt.Sprintf("Site %d", i)}, &site)
			assertStatus(t, resp, http.StatusCreated)
			return site.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Site]
			resp = doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[Site]
			resp = doRequest(t, client, "GET", listSitesURL(ts.URL)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)

}

// Utility function that tests pagination invariants for any list endpoint. Caller provides hooks to invoke
// the endpoints.
//
// create is called with an index, and must create a new item with a unique ID. This is used for test setup.
// listDefault is called with a page token, and should invoke the list endpoint without a pageSize parameter. It returns the HTTP response, the list of IDs returned, and the next page token.
// listSize is called with a page size and page token, and should invoke the list endpoint with the given pageSize. It returns the HTTP response, the list of IDs returned, and the next page token.
func testPagination(t *testing.T,
	create func(i int) (id int64),
	listDefault func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string),
	listSize func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string),
) {
	const numSites = maxPageSize + 10
	var expectedIDs []int64
	for i := 0; i < numSites; i++ {
		expectedIDs = append(expectedIDs, create(i))
	}
	slices.Sort(expectedIDs)

	t.Run("pageSize=0 should be rejected", func(t *testing.T) {
		resp, _, _ := listSize("0", "")
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("negative pageSize should be rejected", func(t *testing.T) {
		resp, _, _ := listSize("-1", "")
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("absent pageSize should use default", func(t *testing.T) {
		resp, ids, _ := listDefault("")
		assertStatus(t, resp, http.StatusOK)
		if len(ids) != defaultPageSize {
			t.Errorf("expected default page size of %d, got %d", defaultPageSize, len(ids))
		}
	})

	t.Run("pageSize > maxPageSize should be clamped", func(t *testing.T) {
		// Request more items than maxPageSize, should get maxPageSize items back
		resp, ids, nextPageToken := listSize(strconv.Itoa(2*maxPageSize), "")
		assertStatus(t, resp, http.StatusOK)
		// Should get all 5 items since we only have 5, but limit was clamped to maxPageSize
		if len(ids) != maxPageSize {
			t.Errorf("expected %d items, got %d", maxPageSize, len(ids))
		}
		totalSeen := len(ids)
		// Next page token since we haven't seen all items yet
		if nextPageToken == "" {
			t.Error("expected next page token")
		}

		// the remaining items should fit in one page
		resp, ids, nextPageToken = listSize(strconv.Itoa(maxPageSize), nextPageToken)
		assertStatus(t, resp, http.StatusOK)
		totalSeen += len(ids)
		if totalSeen != numSites {
			t.Errorf("expected to see total %d items after second page, got %d", numSites, totalSeen)
		}
		if nextPageToken != "" {
			t.Error("expected no next page token after seeing all items")
		}

	})

	t.Run("negative page token should be rejected", func(t *testing.T) {
		resp, _, _ := listDefault("-1")
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("malformed page token should be rejected", func(t *testing.T) {
		resp, _, _ := listDefault("notanumber")
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("empty results with valid pagination", func(t *testing.T) {
		resp, ids, nextPageToken := listDefault("99999")
		assertStatus(t, resp, http.StatusOK)
		if len(ids) != 0 {
			t.Errorf("expected 0 items for page beyond end, got %d", len(ids))
		}
		if nextPageToken != "" {
			t.Errorf("expected no next page token for empty results, got %q", nextPageToken)
		}
	})

	t.Run("pageSize=1 should paginate correctly", func(t *testing.T) {
		// resp := doRequest(t, client, "GET", ts.URL+"/api/v1/management/sites?pageSize=1", nil, &page1)
		resp, ids1, nextPageToken := listSize("1", "")
		assertStatus(t, resp, http.StatusOK)
		if len(ids1) != 1 {
			t.Errorf("expected 1 item on first page, got %d", len(ids1))
		}
		if nextPageToken == "" {
			t.Error("expected next page token for pageSize=1")
		}
		if ids1[0] != expectedIDs[0] {
			t.Errorf("expected first ID %d, got %d", expectedIDs[0], ids1[0])
		}

		// Get second page
		// var page2 ListResponse[Site]
		// resp = doRequest(t, client, "GET", fmt.Sprintf("%s/api/v1/management/sites?pageSize=1&pageToken=%s", ts.URL, page1.NextPageToken), nil, &page2)
		resp, ids2, nextPageToken := listSize("1", nextPageToken)
		assertStatus(t, resp, http.StatusOK)
		if len(ids2) != 1 {
			t.Errorf("expected 1 item on second page, got %d", len(ids2))
		}
		if nextPageToken == "" {
			t.Error("expected next page token for second page")
		}
		if ids2[0] != expectedIDs[1] {
			t.Errorf("expected second ID %d, got %d", expectedIDs[1], ids2[0])
		}
	})

	t.Run("all results can be paginated through", func(t *testing.T) {
		var allIDs []int64
		pageToken := ""
		for {
			resp, ids, nextToken := listSize("10", pageToken)
			assertStatus(t, resp, http.StatusOK)
			allIDs = append(allIDs, ids...)
			if nextToken == "" {
				if len(ids) > 10 {
					t.Errorf("expected at most 10 items on last page, got %d", len(ids))
				}
				break
			} else if len(ids) != 10 {
				t.Errorf("expected 10 items on page with next page token, got %d", len(ids))
			}
			pageToken = nextToken
		}

		diff := cmp.Diff(expectedIDs, allIDs)
		if diff != "" {
			t.Errorf("expected to see all IDs after paginating through, but got mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid page token", func(t *testing.T) {
		res, _, _ := listDefault("!!invalid!!")
		assertStatus(t, res, http.StatusBadRequest)
	})

	t.Run("invalid page size", func(t *testing.T) {
		res, _, _ := listSize("abc", "")
		assertStatus(t, res, http.StatusBadRequest)
	})
}

// test setup and helpers

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := zap.NewNop()
	s := store.NewMemoryStore(logger)
	t.Cleanup(func() { _ = s.Close() })

	apiServer := NewServer(s, logger)
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	return httptest.NewServer(mux)
}

func doRequest(t *testing.T, client *http.Client, method, url string, req, res any) *http.Response {
	t.Helper()
	var reqBody *bytes.Buffer
	if req != nil {
		b, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	httpReq, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if req != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(httpReq)
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

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("expected status %d, got %d", want, resp.StatusCode)
	}
}

// URL construction helpers

func listSitesURL(base string) string { return base + "/api/v1/management/sites" }
func siteURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/sites/%d", base, id)
}
func listNodesURL(base string) string { return base + "/api/v1/management/nodes" }
func nodeURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/nodes/%d", base, id)
}
func listConfigVersionsURL(base string) string { return base + "/api/v1/management/config-versions" }
func configVersionURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/config-versions/%d", base, id)
}
func listDeploymentsURL(base string) string { return base + "/api/v1/management/deployments" }
func deploymentURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/deployments/%d", base, id)
}

// Edge case testing utilities

func testInvalidJSON(t *testing.T, client *http.Client, method, url string) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString("not json"))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	assertStatus(t, resp, http.StatusBadRequest)
}

func testInvalidID(t *testing.T, client *http.Client, method, urlTemplate string) {
	t.Helper()
	url := fmt.Sprintf(urlTemplate, "invalid")
	resp := doRequest(t, client, method, url, nil, nil)
	assertStatus(t, resp, http.StatusBadRequest)
}

func testNotFound(t *testing.T, client *http.Client, method, urlTemplate string, body any) {
	t.Helper()
	url := fmt.Sprintf(urlTemplate, 99999)
	resp := doRequest(t, client, method, url, body, nil)
	assertStatus(t, resp, http.StatusNotFound)
}

func testForeignKeyError(t *testing.T, client *http.Client, method, url string, body any) {
	t.Helper()
	var errResp ErrorResponse
	resp := doRequest(t, client, method, url, body, &errResp)
	assertStatus(t, resp, http.StatusBadRequest)
	if errResp.Error != "invalid_reference" {
		t.Errorf("expected error code 'invalid_reference', got %q", errResp.Error)
	}
}
