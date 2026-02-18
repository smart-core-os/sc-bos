package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// nodesEnv holds test fixtures for node tests
type nodesEnv struct {
	testServer *httptest.Server
	client     *http.Client
	site       Site
	node       Node
}

// setupNodesEnv creates a test environment with a site and one pre-created node
func setupNodesEnv(t *testing.T) nodesEnv {
	t.Helper()

	ts := newTestServer(t)
	t.Cleanup(func() { ts.Close() })
	client := ts.Client()

	// Create site (node dependency)
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

	return nodesEnv{
		testServer: ts,
		client:     client,
		site:       site,
		node:       node,
	}
}

func TestNodes_Create(t *testing.T) {
	testCases := []struct {
		name       string
		setupEnv   func(t *testing.T) nodesEnv
		reqBody    func(e nodesEnv) map[string]any
		expectCode int
		expect     func(e nodesEnv) Node
	}{
		{
			name:     "normal create",
			setupEnv: setupNodesEnv,
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "node-01",
					"siteId":   e.site.ID,
				}
			},
			expectCode: http.StatusCreated,
			expect: func(e nodesEnv) Node {
				return Node{Hostname: "node-01", SiteID: e.site.ID}
			},
		},
		{
			name:     "invalid siteId",
			setupEnv: setupNodesEnv,
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "orphan-node",
					"siteId":   99999,
				}
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:     "empty hostname",
			setupEnv: setupNodesEnv,
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "",
					"siteId":   e.site.ID,
				}
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.setupEnv(t)
			var got CreateNodeResponse
			resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), tc.reqBody(e), &got)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				want := tc.expect(e)
				if diff := cmp.Diff(want, got.Node, cmpopts.IgnoreFields(Node{}, "ID", "CreateTime")); diff != "" {
					t.Errorf("response mismatch (-want +got):\n%s", diff)
				}
				if got.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if got.CreateTime.IsZero() {
					t.Error("expected non-zero CreateTime")
				}
			}
		})
	}

	t.Run("returns secret", func(t *testing.T) {
		e := setupNodesEnv(t)
		var got CreateNodeResponse
		resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "secret-node",
			"siteId":   e.site.ID,
		}, &got)
		assertStatus(t, resp, http.StatusCreated)
		if len(got.Secret) != 32 {
			t.Errorf("expected 32-byte secret, got %d bytes", len(got.Secret))
		}

		// A second create should return a different secret
		var got2 CreateNodeResponse
		resp = doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "secret-node-2",
			"siteId":   e.site.ID,
		}, &got2)
		assertStatus(t, resp, http.StatusCreated)
		if bytes.Equal(got.Secret, got2.Secret) {
			t.Error("expected different secrets for different nodes")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		e := setupNodesEnv(t)
		testInvalidJSON(t, e.client, "POST", listNodesURL(e.testServer.URL))
	})
}

func TestNodes_Get(t *testing.T) {
	t.Run("get existing node", func(t *testing.T) {
		e := setupNodesEnv(t)
		var got Node
		resp := doRequest(t, e.client, "GET", nodeURL(e.testServer.URL, e.node.ID), nil, &got)
		assertStatus(t, resp, http.StatusOK)

		want := Node{ID: e.node.ID, Hostname: "test-node", SiteID: e.site.ID, CreateTime: e.node.CreateTime}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Node{}, "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupNodesEnv(t)
		testInvalidID(t, e.client, "GET", listNodesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("does not return secret", func(t *testing.T) {
		e := setupNodesEnv(t)
		var raw map[string]any
		resp := doRequest(t, e.client, "GET", nodeURL(e.testServer.URL, e.node.ID), nil, &raw)
		assertStatus(t, resp, http.StatusOK)
		if _, ok := raw["secret"]; ok {
			t.Error("GET node response should not contain secret")
		}
	})
}

func TestNodes_Update(t *testing.T) {
	testCases := []struct {
		name       string
		reqBody    func(e nodesEnv) map[string]any
		expectCode int
		expect     func(e nodesEnv) Node
	}{
		{
			name: "normal update",
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "updated-node",
					"siteId":   e.site.ID,
				}
			},
			expectCode: http.StatusOK,
			expect: func(e nodesEnv) Node {
				return Node{Hostname: "updated-node", SiteID: e.site.ID}
			},
		},
		{
			name: "invalid siteId",
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "x",
					"siteId":   99999,
				}
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "empty hostname",
			reqBody: func(e nodesEnv) map[string]any {
				return map[string]any{
					"hostname": "",
					"siteId":   e.site.ID,
				}
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupNodesEnv(t)
			var got Node
			resp := doRequest(t, e.client, "PUT", nodeURL(e.testServer.URL, e.node.ID), tc.reqBody(e), &got)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				want := tc.expect(e)
				want.ID = e.node.ID
				want.CreateTime = e.node.CreateTime
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("invalid id", func(t *testing.T) {
		e := setupNodesEnv(t)
		testInvalidID(t, e.client, "PUT", listNodesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupNodesEnv(t)
		testNotFound(t, e.client, "PUT", listNodesURL(e.testServer.URL)+"/%d", map[string]any{
			"hostname": "x",
			"siteId":   e.site.ID,
		})
	})

	t.Run("invalid json", func(t *testing.T) {
		e := setupNodesEnv(t)
		testInvalidJSON(t, e.client, "PUT", nodeURL(e.testServer.URL, e.node.ID))
	})
}

func TestNodes_List(t *testing.T) {
	t.Run("list all nodes", func(t *testing.T) {
		e := setupNodesEnv(t)

		// Create second node
		resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "node-02",
			"siteId":   e.site.ID,
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		var all ListResponse[Node]
		resp = doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL), nil, &all)
		assertStatus(t, resp, http.StatusOK)
		if len(all.Items) != 2 {
			t.Errorf("expected 2 nodes, got %d", len(all.Items))
		}
	})

	t.Run("filter by siteId", func(t *testing.T) {
		e := setupNodesEnv(t)

		// Create another site with a node
		var site2 Site
		resp := doRequest(t, e.client, "POST", listSitesURL(e.testServer.URL), map[string]string{"name": "Site 2"}, &site2)
		assertStatus(t, resp, http.StatusCreated)

		resp = doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "node-02",
			"siteId":   site2.ID,
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all nodes
		var allNodes ListResponse[Node]
		resp = doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL), nil, &allNodes)
		assertStatus(t, resp, http.StatusOK)
		if len(allNodes.Items) != 2 {
			t.Errorf("expected 2 nodes total, got %d", len(allNodes.Items))
		}

		// Filter by first site
		var filtered ListResponse[Node]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?siteId=%d", listNodesURL(e.testServer.URL), e.site.ID), nil, &filtered)
		assertStatus(t, resp, http.StatusOK)
		if len(filtered.Items) != 1 {
			t.Errorf("expected 1 node for site %d, got %d", e.site.ID, len(filtered.Items))
		}
	})

	t.Run("filter with invalid siteId", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL)+"?siteId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("filter with negative siteId", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL)+"?siteId=-1", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("filter for non-existent site", func(t *testing.T) {
		e := setupNodesEnv(t)
		var list ListResponse[Node]
		resp := doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL)+"?siteId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 nodes for non-existent site, got %d", len(list.Items))
		}
	})

	t.Run("filter with pagination", func(t *testing.T) {
		e := setupNodesEnv(t)

		// Create another node for pagination test
		resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "node-02",
			"siteId":   e.site.ID,
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// Test pagination with filter
		var list ListResponse[Node]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?siteId=%d&pageSize=1", listNodesURL(e.testServer.URL), e.site.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 node on first page, got %d", len(list.Items))
		}
		if list.NextPageToken == "" {
			t.Error("expected next page token when using filter with pagination")
		}

		// Get second page with filter
		var page2 ListResponse[Node]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?siteId=%d&pageSize=1&pageToken=%s", listNodesURL(e.testServer.URL), e.site.ID, list.NextPageToken), nil, &page2)
		assertStatus(t, resp, http.StatusOK)
		if len(page2.Items) != 1 {
			t.Errorf("expected 1 node on second page, got %d", len(page2.Items))
		}
	})

	t.Run("does not return secret", func(t *testing.T) {
		e := setupNodesEnv(t)
		var res ListResponse[map[string]any]
		resp := doRequest(t, e.client, "GET", listNodesURL(e.testServer.URL), nil, &res)
		assertStatus(t, resp, http.StatusOK)

		for i, item := range res.Items {
			if _, ok := item["secret"]; ok {
				t.Errorf("item %d in list response should not contain secret", i)
			}
		}
	})
}

func TestNodes_Delete(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "DELETE", nodeURL(e.testServer.URL, e.node.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		// Verify it's gone
		resp = doRequest(t, e.client, "GET", nodeURL(e.testServer.URL, e.node.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupNodesEnv(t)
		testInvalidID(t, e.client, "DELETE", listNodesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupNodesEnv(t)
		testNotFound(t, e.client, "DELETE", listNodesURL(e.testServer.URL)+"/%d", nil)
	})
}

func TestNodes_RotateSecret(t *testing.T) {
	t.Run("rotate secret", func(t *testing.T) {
		e := setupNodesEnv(t)

		// Get original secret from create
		var created CreateNodeResponse
		resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "rotate-node",
			"siteId":   e.site.ID,
		}, &created)
		assertStatus(t, resp, http.StatusCreated)

		// Rotate secret
		var rotated CreateNodeResponse
		resp = doRequest(t, e.client, "POST", rotateNodeSecretURL(e.testServer.URL, created.ID), nil, &rotated)
		assertStatus(t, resp, http.StatusOK)

		if len(rotated.Secret) != 32 {
			t.Errorf("expected 32-byte secret, got %d bytes", len(rotated.Secret))
		}
		if bytes.Equal(created.Secret, rotated.Secret) {
			t.Error("rotated secret should differ from original")
		}
		if rotated.ID != created.ID {
			t.Errorf("expected node ID %d, got %d", created.ID, rotated.ID)
		}
		if rotated.Hostname != created.Hostname {
			t.Errorf("expected hostname %q, got %q", created.Hostname, rotated.Hostname)
		}
	})

	t.Run("not found", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "POST", rotateNodeSecretURL(e.testServer.URL, 99999), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupNodesEnv(t)
		resp := doRequest(t, e.client, "POST", e.testServer.URL+"/api/v1/management/nodes/invalid/rotate-secret", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

func rotateNodeSecretURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/nodes/%d/rotate-secret", base, id)
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
