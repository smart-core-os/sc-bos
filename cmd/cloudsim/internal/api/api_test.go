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
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
)

// TestCascadeDelete verifies that deleting a site cascades to nodes, config versions, and deployments
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

// TestPagination_EdgeCases tests pagination edge cases using sites as the test subject
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

// testPagination tests pagination invariants for any list endpoint. Caller provides hooks to invoke
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

// Test setup and helper functions

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
func configVersionPayloadURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/config-versions/%d/payload", base, id)
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
