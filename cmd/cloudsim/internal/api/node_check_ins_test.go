package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// nodeCheckInsEnv holds test fixtures for node check-in tests
type nodeCheckInsEnv struct {
	testServer *httptest.Server
	client     *http.Client
	store      *store.Store
	site       Site
	node       Node
	checkIn    NodeCheckIn
}

// setupNodeCheckInsEnv creates a test environment with a site, node, and one pre-created check-in
func setupNodeCheckInsEnv(t *testing.T) nodeCheckInsEnv {
	t.Helper()

	ts, s := newTestServerWithStore(t)
	client := ts.Client()

	// Create site and node via HTTP
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	// Create check-in directly via store (no POST endpoint)
	var checkIn queries.NodeCheckIn
	err := s.Write(t.Context(), func(tx *store.Tx) error {
		var err error
		checkIn, err = tx.CreateNodeCheckIn(t.Context(), queries.CreateNodeCheckInParams{NodeID: node.ID})
		return err
	})
	if err != nil {
		t.Fatalf("failed to create check-in: %v", err)
	}

	return nodeCheckInsEnv{
		testServer: ts,
		client:     client,
		store:      s,
		site:       site,
		node:       node,
		checkIn:    toNodeCheckIn(checkIn),
	}
}

func TestNodeCheckIns_Get(t *testing.T) {
	t.Run("get existing check-in", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)
		var got NodeCheckIn
		resp := doRequest(t, e.client, "GET", nodeCheckInURL(e.testServer.URL, e.node.ID, e.checkIn.ID), nil, &got)
		assertStatus(t, resp, http.StatusOK)

		if diff := cmp.Diff(e.checkIn, got, cmpopts.IgnoreFields(NodeCheckIn{}, "CheckInTime")); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("not found", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)
		resp := doRequest(t, e.client, "GET", nodeCheckInURL(e.testServer.URL, e.node.ID, 99999), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)
		resp := doRequest(t, e.client, "GET",
			fmt.Sprintf("%s/api/v1/management/nodes/%d/check-ins/invalid", e.testServer.URL, e.node.ID), nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("wrong nodeId returns 404", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)

		// Create a second node
		var node2 Node
		resp := doRequest(t, e.client, "POST", listNodesURL(e.testServer.URL), map[string]any{
			"hostname": "other-node",
			"siteId":   e.site.ID,
		}, &node2)
		assertStatus(t, resp, http.StatusCreated)

		// Try to get check-in under wrong node
		resp = doRequest(t, e.client, "GET", nodeCheckInURL(e.testServer.URL, node2.ID, e.checkIn.ID), nil, nil)
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("invalid nodeId", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)
		resp := doRequest(t, e.client, "GET",
			fmt.Sprintf("%s/api/v1/management/nodes/invalid/check-ins/%d", e.testServer.URL, e.checkIn.ID), nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestNodeCheckIns_List(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		ts, _ := newTestServerWithStore(t)
		client := ts.Client()

		// Create site and node but no check-ins
		var site Site
		resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
			map[string]string{"name": "Test Site"}, &site)
		assertStatus(t, resp, http.StatusCreated)

		var node Node
		resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
			"hostname": "test-node",
			"siteId":   site.ID,
		}, &node)
		assertStatus(t, resp, http.StatusCreated)

		var list ListResponse[NodeCheckIn]
		resp = doRequest(t, client, "GET", listNodeCheckInsURL(ts.URL, node.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 0 {
			t.Errorf("expected 0 check-ins, got %d", len(list.Items))
		}
	})

	t.Run("list after create", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)

		var list ListResponse[NodeCheckIn]
		resp := doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 1 {
			t.Errorf("expected 1 check-in, got %d", len(list.Items))
		}
	})

	t.Run("multiple check-ins", func(t *testing.T) {
		e := setupNodeCheckInsEnv(t)

		// Create more check-ins via store
		err := e.store.Write(t.Context(), func(tx *store.Tx) error {
			for i := 0; i < 3; i++ {
				if _, err := tx.CreateNodeCheckIn(t.Context(), queries.CreateNodeCheckInParams{NodeID: e.node.ID}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to create check-ins: %v", err)
		}

		var list ListResponse[NodeCheckIn]
		resp := doRequest(t, e.client, "GET", listNodeCheckInsURL(e.testServer.URL, e.node.ID), nil, &list)
		assertStatus(t, resp, http.StatusOK)
		if len(list.Items) != 4 { // 1 from setup + 3 new
			t.Errorf("expected 4 check-ins, got %d", len(list.Items))
		}
	})

	t.Run("invalid nodeId", func(t *testing.T) {
		ts := newTestServer(t)
		defer ts.Close()
		resp := doRequest(t, ts.Client(), "GET",
			ts.URL+"/api/v1/management/nodes/invalid/check-ins", nil, nil)
		assertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestNodeCheckIns_Pagination(t *testing.T) {
	ts, s := newTestServerWithStore(t)
	defer ts.Close()
	client := ts.Client()

	// Setup: create site and node
	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL),
		map[string]string{"name": "Test Site"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "test-node",
		"siteId":   site.ID,
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	testPagination(t,
		func(i int) int64 {
			var checkIn queries.NodeCheckIn
			err := s.Write(t.Context(), func(tx *store.Tx) error {
				var err error
				checkIn, err = tx.CreateNodeCheckIn(t.Context(), queries.CreateNodeCheckInParams{NodeID: node.ID})
				return err
			})
			if err != nil {
				t.Fatalf("failed to create check-in %d: %v", i, err)
			}
			return checkIn.ID
		},
		func(pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[NodeCheckIn]
			resp = doRequest(t, client, "GET", listNodeCheckInsURL(ts.URL, node.ID)+"?pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
		func(pageSize string, pageToken string) (resp *http.Response, ids []int64, nextPageToken string) {
			var list ListResponse[NodeCheckIn]
			resp = doRequest(t, client, "GET", listNodeCheckInsURL(ts.URL, node.ID)+"?pageSize="+pageSize+"&pageToken="+pageToken, nil, &list)
			for _, item := range list.Items {
				ids = append(ids, item.ID)
			}
			return resp, ids, list.NextPageToken
		},
	)
}

func TestNodeCheckIns_CascadeDelete(t *testing.T) {
	e := setupNodeCheckInsEnv(t)

	// Delete the site - should cascade delete node and check-in
	resp := doRequest(t, e.client, "DELETE", siteURL(e.testServer.URL, e.site.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Verify check-in is gone via store
	err := e.store.Read(t.Context(), func(tx *store.Tx) error {
		_, err := tx.GetNodeCheckIn(t.Context(), e.checkIn.ID)
		if err == nil {
			t.Error("expected check-in to be deleted, but it still exists")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to verify cascade delete: %v", err)
	}
}
