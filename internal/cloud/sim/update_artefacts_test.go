package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func listUpdateArtefactsURL(base string) string {
	return base + "/api/v1/management/update-artefacts"
}
func updateArtefactURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/update-artefacts/%d", base, id)
}
func updateArtefactPayloadURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/update-artefacts/%d/payload", base, id)
}

// updateArtefactEnv holds fixtures for update-artefact tests.
type updateArtefactEnv struct {
	testServer *httptest.Server
	client     *http.Client
	site       Site
	node       Node // platform podman
}

func setupUpdateArtefactEnv(t *testing.T) updateArtefactEnv {
	t.Helper()
	ts := newTestServer(t)
	t.Cleanup(ts.Close)
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Site A"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node CreateNodeResponse
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-a",
		"siteId":   sid(site.ID),
		"platform": "podman",
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	return updateArtefactEnv{testServer: ts, client: client, site: site, node: node.Node}
}

// uploadArtefact POSTs a multipart create-artefact request: a "payload" file part plus form fields.
func uploadArtefact(t *testing.T, client *http.Client, base string, fields map[string]string, payload []byte, res any) *http.Response {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	fw, err := mw.CreateFormFile("payload", "artefact.tar")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req, err := http.NewRequest("POST", listUpdateArtefactsURL(base), &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if res != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp
}

func TestUpdateArtefacts_UploadListDownloadDelete(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	payload := []byte("fake podman-save tarball contents")
	sum := sha256.Sum256(payload)

	var created UpdateArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version":     "1.0.0",
		"platform":    "podman",
		"siteId":      sid(e.site.ID),
		"description": "first build",
	}, payload, &created)
	assertStatus(t, resp, http.StatusCreated)

	if created.Version != "1.0.0" || created.Platform != "podman" {
		t.Errorf("created artefact version/platform = %s/%s", created.Version, created.Platform)
	}
	if created.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("sha256 = %s, want %s", created.SHA256, hex.EncodeToString(sum[:]))
	}
	if created.Size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", created.Size, len(payload))
	}
	if created.PayloadURL == "" {
		t.Error("expected payloadUrl to be set")
	}

	// List shows it.
	var list ListResponse[UpdateArtefact]
	resp = doRequest(t, e.client, "GET", listUpdateArtefactsURL(e.testServer.URL), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) != 1 {
		t.Fatalf("list: got %d, want 1", len(list.Items))
	}

	// Download payload matches byte-for-byte.
	dlResp, err := e.client.Get(updateArtefactPayloadURL(e.testServer.URL, created.ID))
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer func() { _ = dlResp.Body.Close() }()
	assertStatus(t, dlResp, http.StatusOK)
	got, _ := io.ReadAll(dlResp.Body)
	if !bytes.Equal(got, payload) {
		t.Errorf("downloaded payload mismatch: got %d bytes", len(got))
	}

	// Delete then 404.
	resp = doRequest(t, e.client, "DELETE", updateArtefactURL(e.testServer.URL, created.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
	resp = doRequest(t, e.client, "GET", updateArtefactURL(e.testServer.URL, created.ID), nil, nil)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestUpdateArtefacts_GenericArtefact(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	var created UpdateArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version":  "2.0.0",
		"platform": "podman",
		// no siteId -> generic
	}, []byte("generic"), &created)
	assertStatus(t, resp, http.StatusCreated)
	if created.SiteID != nil {
		t.Errorf("expected generic artefact (nil siteId), got %d", *created.SiteID)
	}
}

func TestUpdateArtefacts_UploadValidation(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	cases := []struct {
		name   string
		fields map[string]string
		data   []byte
	}{
		{"missing version", map[string]string{"platform": "podman"}, []byte("x")},
		{"missing platform", map[string]string{"version": "1.0.0"}, []byte("x")},
		{"invalid platform", map[string]string{"version": "1.0.0", "platform": "windows"}, []byte("x")},
		{"empty payload", map[string]string{"version": "1.0.0", "platform": "podman"}, []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := uploadArtefact(t, e.client, e.testServer.URL, tc.fields, tc.data, nil)
			assertStatus(t, resp, http.StatusBadRequest)
		})
	}
}

func TestUpdateArtefacts_ListFilters(t *testing.T) {
	e := setupUpdateArtefactEnv(t)
	mk := func(platform, siteId string) {
		fields := map[string]string{"version": "v", "platform": platform}
		if siteId != "" {
			fields["siteId"] = siteId
		}
		resp := uploadArtefact(t, e.client, e.testServer.URL, fields, []byte("data"), nil)
		assertStatus(t, resp, http.StatusCreated)
	}
	mk("podman", sid(e.site.ID))
	mk("freebsd", sid(e.site.ID))
	mk("podman", "") // generic

	count := func(query string) int {
		var list ListResponse[UpdateArtefact]
		resp := doRequest(t, e.client, "GET", listUpdateArtefactsURL(e.testServer.URL)+query, nil, &list)
		assertStatus(t, resp, http.StatusOK)
		return len(list.Items)
	}
	if n := count(""); n != 3 {
		t.Errorf("no filter: %d want 3", n)
	}
	if n := count("?platform=podman"); n != 2 {
		t.Errorf("platform=podman: %d want 2", n)
	}
	if n := count(fmt.Sprintf("?siteId=%d&platform=freebsd", e.site.ID)); n != 1 {
		t.Errorf("site+freebsd: %d want 1", n)
	}
}
