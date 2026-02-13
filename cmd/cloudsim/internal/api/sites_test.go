package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sitesEnv holds test fixtures for site tests
type sitesEnv struct {
	testServer *httptest.Server
	client     *http.Client
	site       Site
}

// setupSitesEnv creates a test environment with one pre-created site
func setupSitesEnv(t *testing.T) sitesEnv {
	t.Helper()

	ts := newTestServer(t)
	t.Cleanup(func() { ts.Close() })
	client := ts.Client()

	// Create site
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	return sitesEnv{
		testServer: ts,
		client:     client,
		site:       site,
	}
}

func TestSites_Create(t *testing.T) {
	testCases := []struct {
		name       string
		reqBody    map[string]any
		expectCode int
		expect     Site
	}{
		{
			name:       "normal create",
			reqBody:    map[string]any{"name": "Test Site"},
			expectCode: http.StatusCreated,
			expect:     Site{Name: "Test Site"},
		},
		{
			name:       "missing name",
			reqBody:    map[string]any{},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupSitesEnv(t)
			var got Site
			resp := doRequest(t, e.client, "POST", listSitesURL(e.testServer.URL), tc.reqBody, &got)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if diff := cmp.Diff(tc.expect, got, cmpopts.IgnoreFields(Site{}, "ID", "CreateTime")); diff != "" {
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

	t.Run("invalid json", func(t *testing.T) {
		e := setupSitesEnv(t)
		testInvalidJSON(t, e.client, "POST", listSitesURL(e.testServer.URL))
	})
}

func TestSites_Get(t *testing.T) {
	t.Run("get existing site", func(t *testing.T) {
		e := setupSitesEnv(t)
		var got Site
		resp := doRequest(t, e.client, "GET", siteURL(e.testServer.URL, e.site.ID), nil, &got)
		assertStatus(t, resp, http.StatusOK)

		want := Site{ID: got.ID, Name: "Test Site", CreateTime: got.CreateTime}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupSitesEnv(t)
		testInvalidID(t, e.client, "GET", listSitesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupSitesEnv(t)
		resp := doRequest(t, e.client, "GET", listSitesURL(e.testServer.URL)+"/99999", nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})
}

func TestSites_Update(t *testing.T) {
	testCases := []struct {
		name       string
		reqBody    map[string]any
		expectCode int
		expect     Site
	}{
		{
			name:       "normal update",
			reqBody:    map[string]any{"name": "Updated Site"},
			expectCode: http.StatusOK,
			expect:     Site{Name: "Updated Site"},
		},
		{
			name:       "empty name rejected",
			reqBody:    map[string]any{"name": ""},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupSitesEnv(t)
			var got Site
			resp := doRequest(t, e.client, "PUT", siteURL(e.testServer.URL, e.site.ID), tc.reqBody, &got)
			assertStatus(t, resp, tc.expectCode)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				want := Site{ID: e.site.ID, Name: tc.expect.Name, CreateTime: e.site.CreateTime}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("invalid id", func(t *testing.T) {
		e := setupSitesEnv(t)
		testInvalidID(t, e.client, "PUT", listSitesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupSitesEnv(t)
		testNotFound(t, e.client, "PUT", listSitesURL(e.testServer.URL)+"/%d", map[string]string{"name": "X"})
	})

	t.Run("invalid json", func(t *testing.T) {
		e := setupSitesEnv(t)
		testInvalidJSON(t, e.client, "PUT", siteURL(e.testServer.URL, e.site.ID))
	})
}

func TestSites_List(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		ts := newTestServer(t)
		defer ts.Close()
		client := ts.Client()

		var list ListResponse[Site]
		resp := doRequest(t, client, "GET", listSitesURL(ts.URL), nil, &list)
		assertStatus(t, resp, http.StatusOK)

		if diff := cmp.Diff(ListResponse[Site]{Items: []Site{}}, list); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("list after create", func(t *testing.T) {
		e := setupSitesEnv(t)
		var list ListResponse[Site]
		resp := doRequest(t, e.client, "GET", listSitesURL(e.testServer.URL), nil, &list)
		assertStatus(t, resp, http.StatusOK)

		if len(list.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(list.Items))
		}
	})
}

func TestSites_Delete(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		e := setupSitesEnv(t)
		resp := doRequest(t, e.client, "DELETE", siteURL(e.testServer.URL, e.site.ID), nil, nil)
		assertStatus(t, resp, http.StatusNoContent)

		// Verify it's gone
		resp = doRequest(t, e.client, "GET", siteURL(e.testServer.URL, e.site.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupSitesEnv(t)
		testInvalidID(t, e.client, "DELETE", listSitesURL(e.testServer.URL)+"/%s")
	})

	t.Run("not found", func(t *testing.T) {
		e := setupSitesEnv(t)
		resp := doRequest(t, e.client, "DELETE", listSitesURL(e.testServer.URL)+"/99999", nil, nil)
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
