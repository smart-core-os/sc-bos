package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// deploymentsEnv holds test fixtures for deployment tests
type deploymentsEnv struct {
	testServer    *httptest.Server
	client        *http.Client
	site          Site
	node          Node
	configVersion ConfigVersion
	deployment    Deployment
}

// setupDeploymentsEnv creates a complete test environment with all dependencies
func setupDeploymentsEnv(t *testing.T) deploymentsEnv {
	t.Helper()

	ts := newTestServer(t)
	t.Cleanup(func() { ts.Close() })
	client := ts.Client()

	// Create site
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	// Create node
	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	// Create config version
	var cv ConfigVersion
	resp = doRequest(t, client, "POST", listConfigVersionsURL(ts.URL), map[string]any{
		"nodeId":      node.ID,
		"description": "v1.0.0",
		"payload":     []byte("config"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	// Create a sample deployment
	var deployment Deployment
	resp = doRequest(t, client, "POST", listDeploymentsURL(ts.URL), map[string]any{
		"configVersionId": cv.ID,
		"status":          "PENDING",
	}, &deployment)
	assertStatus(t, resp, http.StatusCreated)

	return deploymentsEnv{
		testServer:    ts,
		client:        client,
		site:          site,
		node:          node,
		configVersion: cv,
		deployment:    deployment,
	}
}

func TestDeployments_Create(t *testing.T) {
	e := setupDeploymentsEnv(t)

	testCases := []struct {
		name       string
		reqBody    map[string]any
		expectCode int
		expect     Deployment
	}{
		{
			name: "normal create with PENDING",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "PENDING",
			},
			expectCode: http.StatusCreated,
			expect: Deployment{
				ConfigVersionID: e.configVersion.ID,
				Status:          "PENDING",
				FinishedTime:    nil,
			},
		},
		{
			name: "absent status defaults to PENDING",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
			},
			expectCode: http.StatusCreated,
			expect: Deployment{
				ConfigVersionID: e.configVersion.ID,
				Status:          "PENDING",
				FinishedTime:    nil,
			},
		},
		{
			name: "empty status defaults to PENDING",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "",
			},
			expectCode: http.StatusCreated,
			expect: Deployment{
				ConfigVersionID: e.configVersion.ID,
				Status:          "PENDING",
				FinishedTime:    nil,
			},
		},
		{
			name: "explicit PENDING allowed",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "PENDING",
			},
			expectCode: http.StatusCreated,
			expect: Deployment{
				ConfigVersionID: e.configVersion.ID,
				Status:          "PENDING",
				FinishedTime:    nil,
			},
		},
		{
			name: "IN_PROGRESS rejected",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "IN_PROGRESS",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "COMPLETED rejected",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "COMPLETED",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "FAILED rejected",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "FAILED",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "invalid status rejected",
			reqBody: map[string]any{
				"configVersionId": e.configVersion.ID,
				"status":          "INVALID",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "invalid configVersionId rejected",
			reqBody: map[string]any{
				"configVersionId": 99999,
				"status":          "PENDING",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing configVersionId rejected",
			reqBody: map[string]any{
				"status": "PENDING",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var d Deployment
			resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), tc.reqBody, &d)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if diff := cmp.Diff(tc.expect, d, cmpopts.IgnoreFields(Deployment{}, "ID", "StartTime")); diff != "" {
					t.Errorf("response mismatch (-want +got):\n%s", diff)
				}
				if d.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if d.StartTime.IsZero() {
					t.Error("expected non-zero StartTime")
				}
			}
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testInvalidJSON(t, e.client, "POST", listDeploymentsURL(e.testServer.URL))
	})
}

func TestDeployments_Get(t *testing.T) {
	t.Run("get existing deployment", func(t *testing.T) {
		e := setupDeploymentsEnv(t)

		var d Deployment
		resp := doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, e.deployment.ID), nil, &d)
		assertStatus(t, resp, http.StatusOK)

		want := Deployment{ID: e.deployment.ID, ConfigVersionID: e.configVersion.ID, Status: "PENDING"}
		if diff := cmp.Diff(want, d, cmpopts.IgnoreFields(Deployment{}, "StartTime", "FinishedTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if d.StartTime.IsZero() {
			t.Error("expected non-zero StartTime")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testInvalidID(t, e.client, "GET", listDeploymentsURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		resp := doRequest(t, e.client, "GET", listDeploymentsURL(e.testServer.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestDeployments_Update(t *testing.T) {
	testCases := []struct {
		name    string
		reqBody map[string]any
		expect  Deployment
	}{
		{
			name: "update to IN_PROGRESS",
			reqBody: map[string]any{
				"status": "IN_PROGRESS",
			},
			expect: Deployment{
				Status: "IN_PROGRESS",
			},
		},
		{
			name: "update to COMPLETED",
			reqBody: map[string]any{
				"status": "COMPLETED",
			},
			expect: Deployment{
				Status: "COMPLETED",
				// FinishedTime will be set by server, verified separately
			},
		},
		{
			name: "update to FAILED",
			reqBody: map[string]any{
				"status": "FAILED",
			},
			expect: Deployment{
				Status: "FAILED",
				// FinishedTime will be set by server, verified separately
			},
		},
		{
			name: "protected field ID ignored",
			reqBody: map[string]any{
				"id":     99999,
				"status": "IN_PROGRESS",
			},
			expect: Deployment{
				Status: "IN_PROGRESS",
			},
		},
		{
			name: "protected field configVersionId ignored",
			reqBody: map[string]any{
				"configVersionId": 99999,
				"status":          "IN_PROGRESS",
			},
			expect: Deployment{
				Status: "IN_PROGRESS",
			},
		},
		{
			name: "protected field startTime ignored",
			reqBody: map[string]any{
				"startTime": "2020-01-01T00:00:00Z",
				"status":    "IN_PROGRESS",
			},
			expect: Deployment{
				Status: "IN_PROGRESS",
			},
		},
		{
			name: "protected field finishedTime ignored",
			reqBody: map[string]any{
				"finishedTime": "2020-01-01T00:00:00Z",
				"status":       "IN_PROGRESS",
			},
			expect: Deployment{
				Status: "IN_PROGRESS",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupDeploymentsEnv(t)

			var d Deployment
			resp := doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, e.deployment.ID), tc.reqBody, &d)
			assertStatus(t, resp, http.StatusOK)

			// Compare using cmp.Diff, ignoring protected fields
			if diff := cmp.Diff(tc.expect, d, cmpopts.IgnoreFields(Deployment{}, "ID", "ConfigVersionID", "StartTime", "FinishedTime")); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}

			// Verify protected fields didn't change
			if d.ID != e.deployment.ID {
				t.Errorf("ID should not change: expected %d, got %d", e.deployment.ID, d.ID)
			}
			if d.ConfigVersionID != e.configVersion.ID {
				t.Errorf("ConfigVersionID should not change: expected %d, got %d", e.configVersion.ID, d.ConfigVersionID)
			}
			if !d.StartTime.Equal(e.deployment.StartTime) {
				t.Errorf("StartTime should not change: expected %v, got %v", e.deployment.StartTime, d.StartTime)
			}

			// Verify finishedTime behavior based on status
			if tc.expect.Status == "COMPLETED" || tc.expect.Status == "FAILED" {
				if d.FinishedTime == nil || d.FinishedTime.IsZero() {
					t.Errorf("expected finishedTime to be set for %s, got %v", tc.expect.Status, d.FinishedTime)
				}
			} else {
				if d.FinishedTime != nil && !d.FinishedTime.IsZero() {
					t.Errorf("expected finishedTime to be nil for %s, got %v", tc.expect.Status, d.FinishedTime)
				}
			}
		})
	}

	t.Run("invalid id", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testInvalidID(t, e.client, "PATCH", listDeploymentsURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testNotFound(t, e.client, "PATCH", listDeploymentsURL(e.testServer.URL)+"/%d", map[string]string{
			"status": "COMPLETED",
		})
	})

	t.Run("invalid json", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testInvalidJSON(t, e.client, "PATCH", deploymentURL(e.testServer.URL, e.deployment.ID))
	})

	t.Run("invalid status", func(t *testing.T) {
		e := setupDeploymentsEnv(t)

		resp := doRequest(t, e.client, "PATCH", deploymentURL(e.testServer.URL, e.deployment.ID), map[string]string{
			"status": "INVALID_STATUS",
		}, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestDeployments_List(t *testing.T) {
	t.Run("list with filters", func(t *testing.T) {
		e := setupDeploymentsEnv(t)

		// Create one additional deployment (e already has one)
		resp := doRequest(t, e.client, "POST", listDeploymentsURL(e.testServer.URL), map[string]any{
			"configVersionId": e.configVersion.ID,
			"status":          "PENDING",
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all - expect exactly 2 deployments
		var all ListResponse[Deployment]
		resp = doRequest(t, e.client, "GET", listDeploymentsURL(e.testServer.URL), nil, &all)
		assertStatus(t, resp, http.StatusOK)
		if len(all.Items) != 2 {
			t.Errorf("expected 2 deployments, got %d", len(all.Items))
		}

		// Filter by configVersionId
		var byCv ListResponse[Deployment]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?configVersionId=%d", listDeploymentsURL(e.testServer.URL), e.configVersion.ID), nil, &byCv)
		assertStatus(t, resp, http.StatusOK)
		if len(byCv.Items) != 2 {
			t.Errorf("expected 2 deployments by configVersionId, got %d", len(byCv.Items))
		}

		// Filter by nodeId
		var byNode ListResponse[Deployment]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?nodeId=%d", listDeploymentsURL(e.testServer.URL), e.node.ID), nil, &byNode)
		assertStatus(t, resp, http.StatusOK)
		if len(byNode.Items) != 2 {
			t.Errorf("expected 2 deployments by nodeId, got %d", len(byNode.Items))
		}
	})

	t.Run("filter with invalid configVersionId", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		resp := doRequest(t, e.client, "GET", listDeploymentsURL(e.testServer.URL)+"?configVersionId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("filter with invalid nodeId", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		resp := doRequest(t, e.client, "GET", listDeploymentsURL(e.testServer.URL)+"?nodeId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("filter for non-existent configVersionId", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		var list ListResponse[Deployment]
		resp := doRequest(t, e.client, "GET", listDeploymentsURL(e.testServer.URL)+"?configVersionId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 deployments for non-existent config version, got %d", len(list.Items))
		}
	})

	t.Run("filter by nodeId with pagination", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		var list ListResponse[Deployment]
		resp := doRequest(t, e.client, "GET", fmt.Sprintf("%s?nodeId=%d&pageSize=1", listDeploymentsURL(e.testServer.URL), e.node.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 deployment for node, got %d", len(list.Items))
		}
		// Verify it's the right deployment
		if len(list.Items) > 0 && list.Items[0].ConfigVersionID != e.configVersion.ID {
			t.Errorf("expected deployment for config version %d, got %d", e.configVersion.ID, list.Items[0].ConfigVersionID)
		}
	})
}

func TestDeployments_Delete(t *testing.T) {
	t.Run("delete existing deployment", func(t *testing.T) {
		e := setupDeploymentsEnv(t)

		resp := doRequest(t, e.client, "DELETE", deploymentURL(e.testServer.URL, e.deployment.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		resp = doRequest(t, e.client, "GET", deploymentURL(e.testServer.URL, e.deployment.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testInvalidID(t, e.client, "DELETE", listDeploymentsURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupDeploymentsEnv(t)
		testNotFound(t, e.client, "DELETE", listDeploymentsURL(e.testServer.URL)+"/%d", nil)
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
