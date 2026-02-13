package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// configVersionsEnv holds test fixtures for config version tests
type configVersionsEnv struct {
	testServer    *httptest.Server
	client        *http.Client
	site          Site
	node          Node
	configVersion ConfigVersion
}

// setupConfigVersionsEnv creates a test environment with site → node → configVersion chain
func setupConfigVersionsEnv(t *testing.T) configVersionsEnv {
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
		"payload":     []byte("test-payload"),
	}, &cv)
	assertStatus(t, resp, http.StatusCreated)

	return configVersionsEnv{
		testServer:    ts,
		client:        client,
		site:          site,
		node:          node,
		configVersion: cv,
	}
}

func TestConfigVersions_Create(t *testing.T) {
	testCases := []struct {
		name       string
		reqBody    func(e configVersionsEnv) map[string]any
		expectCode int
		expect     func(e configVersionsEnv) ConfigVersion
	}{
		{
			name: "normal create",
			reqBody: func(e configVersionsEnv) map[string]any {
				return map[string]any{
					"nodeId":      e.node.ID,
					"description": "v1.0.0",
					"payload":     []byte{0xDE, 0xAD, 0xBE, 0xEF},
				}
			},
			expectCode: http.StatusCreated,
			expect: func(e configVersionsEnv) ConfigVersion {
				return ConfigVersion{
					NodeID:      e.node.ID,
					Description: "v1.0.0",
				}
			},
		},
		{
			name: "invalid nodeId",
			reqBody: func(e configVersionsEnv) map[string]any {
				return map[string]any{
					"nodeId":      99999,
					"description": "v1.0.0",
					"payload":     []byte("data"),
				}
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "empty payload",
			reqBody: func(e configVersionsEnv) map[string]any {
				return map[string]any{
					"nodeId":      e.node.ID,
					"description": "v1.0.0",
					"payload":     []byte{},
				}
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupConfigVersionsEnv(t)
			var got ConfigVersion
			resp := doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), tc.reqBody(e), &got)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				want := tc.expect(e)
				want.PayloadURL = configVersionPayloadURL(e.testServer.URL, got.ID)
				if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(ConfigVersion{}, "ID", "CreateTime")); diff != "" {
					t.Errorf("response mismatch (-want +got):\n%s", diff)
				}
				if got.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if got.CreateTime.IsZero() {
					t.Error("expected non-zero CreateTime")
				}
				if !strings.HasPrefix(got.PayloadURL, e.testServer.URL) {
					t.Errorf("expected PayloadURL to start with %s, got %s", e.testServer.URL, got.PayloadURL)
				}
			}
		})
	}

	t.Run("invalid json", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		testInvalidJSON(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL))
	})
}

func TestConfigVersions_Get(t *testing.T) {
	t.Run("get existing config version", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		var got ConfigVersion
		resp := doRequest(t, e.client, "GET", configVersionURL(e.testServer.URL, e.configVersion.ID), nil, &got)
		assertStatus(t, resp, http.StatusOK)

		want := ConfigVersion{
			ID:          e.configVersion.ID,
			NodeID:      e.node.ID,
			Description: "v1.0.0",
			PayloadURL:  configVersionPayloadURL(e.testServer.URL, e.configVersion.ID),
		}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(ConfigVersion{}, "CreateTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if got.CreateTime.IsZero() {
			t.Error("expected non-zero CreateTime")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		testInvalidID(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		resp := doRequest(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestConfigVersions_GetPayload(t *testing.T) {
	t.Run("get payload via URL", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}

		// Create config version with known payload
		var cv ConfigVersion
		resp := doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
			"nodeId":      e.node.ID,
			"description": "test",
			"payload":     payload,
		}, &cv)
		assertStatus(t, resp, http.StatusCreated)

		// Fetch payload via URL
		req, err := http.NewRequest("GET", cv.PayloadURL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		resp, err = e.client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		assertStatus(t, resp, http.StatusOK)

		// Check content type
		if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("expected Content-Type application/octet-stream, got %s", ct)
		}

		// Check content disposition
		if cd := resp.Header.Get("Content-Disposition"); cd == "" {
			t.Error("expected Content-Disposition header")
		}

		// Read and verify payload content
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		if !bytes.Equal(buf.Bytes(), payload) {
			t.Errorf("payload mismatch: expected %v, got %v", payload, buf.Bytes())
		}
	})

	t.Run("payload not found", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		resp := doRequest(t, e.client, "GET", configVersionPayloadURL(e.testServer.URL, 99999), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		testInvalidID(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL)+"/%s/payload")
	})
}

func TestConfigVersions_List(t *testing.T) {
	t.Run("list all config versions", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)

		// Create another config version
		resp := doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
			"nodeId":      e.node.ID,
			"description": "v2.0.0",
			"payload":     []byte{0xDE, 0xAD},
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// List all
		var all ListResponse[ConfigVersion]
		resp = doRequest(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL), nil, &all)
		assertStatus(t, resp, http.StatusOK)
		if len(all.Items) != 2 {
			t.Errorf("expected 2 config versions, got %d", len(all.Items))
		}
	})

	t.Run("filter by nodeId", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)

		// Create another config version
		resp := doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
			"nodeId":      e.node.ID,
			"description": "v2.0.0",
			"payload":     []byte{0xDE, 0xAD},
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		// Filter by node
		var filtered ListResponse[ConfigVersion]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?nodeId=%d", listConfigVersionsURL(e.testServer.URL), e.node.ID), nil, &filtered)
		assertStatus(t, resp, http.StatusOK)
		if len(filtered.Items) != 2 {
			t.Errorf("expected 2 config versions for node, got %d", len(filtered.Items))
		}
	})

	t.Run("filter with invalid nodeId", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		resp := doRequest(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL)+"?nodeId=invalid", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("filter for non-existent node", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		var list ListResponse[ConfigVersion]
		resp := doRequest(t, e.client, "GET", listConfigVersionsURL(e.testServer.URL)+"?nodeId=99999", nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 config versions for non-existent node, got %d", len(list.Items))
		}
	})

	t.Run("filter with pagination", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)

		// Create another config version
		resp := doRequest(t, e.client, "POST", listConfigVersionsURL(e.testServer.URL), map[string]any{
			"nodeId":      e.node.ID,
			"description": "v2.0.0",
			"payload":     []byte{0xDE, 0xAD},
		}, nil)
		assertStatus(t, resp, http.StatusCreated)

		var list ListResponse[ConfigVersion]
		resp = doRequest(t, e.client, "GET", fmt.Sprintf("%s?nodeId=%d&pageSize=10", listConfigVersionsURL(e.testServer.URL), e.node.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 2 {
			t.Errorf("expected 2 config versions for node, got %d", len(list.Items))
		}
		// Verify correct data returned
		if len(list.Items) > 0 && list.Items[0].NodeID != e.node.ID {
			t.Errorf("expected nodeId %d, got %d", e.node.ID, list.Items[0].NodeID)
		}
	})
}

func TestConfigVersions_Delete(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		resp := doRequest(t, e.client, "DELETE", configVersionURL(e.testServer.URL, e.configVersion.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		// Verify it's gone
		resp = doRequest(t, e.client, "GET", configVersionURL(e.testServer.URL, e.configVersion.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		testInvalidID(t, e.client, "DELETE", listConfigVersionsURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupConfigVersionsEnv(t)
		testNotFound(t, e.client, "DELETE", listConfigVersionsURL(e.testServer.URL)+"/%d", nil)
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
