package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

func listBinaryArtefactsURL(base string) string {
	return base + "/api/v1/management/binary-artefacts"
}
func binaryArtefactURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/binary-artefacts/%d", base, id)
}
func binaryArtefactPayloadURL(base string, id int64) string {
	return fmt.Sprintf("%s/api/v1/management/binary-artefacts/%d/payload", base, id)
}

// binaryArtefactEnv holds fixtures for binary-artefact tests.
type binaryArtefactEnv struct {
	testServer *httptest.Server
	client     *http.Client
	site       Site
	node       Node // os linux, arch arm64
}

func setupBinaryArtefactEnv(t *testing.T) binaryArtefactEnv {
	t.Helper()
	ts := newTestServer(t)
	t.Cleanup(ts.Close)
	client := ts.Client()

	var site Site
	resp := doRequest(t, client, "POST", listSitesURL(ts.URL), map[string]string{"name": "Site A"}, &site)
	assertStatus(t, resp, http.StatusCreated)

	var node Node
	resp = doRequest(t, client, "POST", listNodesURL(ts.URL), map[string]any{
		"hostname": "node-a",
		"siteId":   sid(site.ID),
		"os":       "linux",
		"arch":     "arm64",
	}, &node)
	assertStatus(t, resp, http.StatusCreated)

	return binaryArtefactEnv{testServer: ts, client: client, site: site, node: node}
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

	req, err := http.NewRequest("POST", listBinaryArtefactsURL(base), &body)
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

func TestBinaryArtefacts_UploadListDownloadDelete(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	payload := []byte("fake podman-save tarball contents")
	sum := sha256.Sum256(payload)

	var created BinaryArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version":     "1.0.0",
		"os":          "linux",
		"arch":        "arm64",
		"siteId":      sid(e.site.ID),
		"description": "first build",
	}, payload, &created)
	assertStatus(t, resp, http.StatusCreated)

	if created.Version != "1.0.0" || created.OS != "linux" || created.Arch != "arm64" {
		t.Errorf("created artefact version/os/arch = %s/%s/%s", created.Version, created.OS, created.Arch)
	}
	if want := checksum.Format(checksum.SHA256, sum[:]); created.Checksum != want {
		t.Errorf("checksum = %s, want %s", created.Checksum, want)
	}
	if created.Size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", created.Size, len(payload))
	}
	if created.PayloadURL == "" {
		t.Error("expected payloadUrl to be set")
	}

	// List shows it.
	var list ListResponse[BinaryArtefact]
	resp = doRequest(t, e.client, "GET", listBinaryArtefactsURL(e.testServer.URL), nil, &list)
	assertStatus(t, resp, http.StatusOK)
	if len(list.Items) != 1 {
		t.Fatalf("list: got %d, want 1", len(list.Items))
	}

	// Download payload matches byte-for-byte.
	dlResp, err := e.client.Get(binaryArtefactPayloadURL(e.testServer.URL, created.ID))
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
	resp = doRequest(t, e.client, "DELETE", binaryArtefactURL(e.testServer.URL, created.ID), nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
	resp = doRequest(t, e.client, "GET", binaryArtefactURL(e.testServer.URL, created.ID), nil, nil)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestBinaryArtefacts_GenericArtefact(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	var created BinaryArtefact
	resp := uploadArtefact(t, e.client, e.testServer.URL, map[string]string{
		"version": "2.0.0",
		"os":      "linux",
		"arch":    "arm64",
		// no siteId -> generic
	}, []byte("generic"), &created)
	assertStatus(t, resp, http.StatusCreated)
	if created.SiteID != nil {
		t.Errorf("expected generic artefact (nil siteId), got %d", *created.SiteID)
	}
}

func TestBinaryArtefacts_UploadValidation(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	cases := []struct {
		name   string
		fields map[string]string
		data   []byte
	}{
		{"missing version", map[string]string{"os": "linux", "arch": "arm64"}, []byte("x")},
		{"version not an image tag", map[string]string{"version": "1.2.3+build", "os": "linux", "arch": "arm64"}, []byte("x")},
		{"missing os", map[string]string{"version": "1.0.0", "arch": "arm64"}, []byte("x")},
		{"missing arch", map[string]string{"version": "1.0.0", "os": "linux"}, []byte("x")},
		{"invalid os", map[string]string{"version": "1.0.0", "os": "windows", "arch": "arm64"}, []byte("x")},
		{"empty payload", map[string]string{"version": "1.0.0", "os": "linux", "arch": "arm64"}, []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := uploadArtefact(t, e.client, e.testServer.URL, tc.fields, tc.data, nil)
			assertStatus(t, resp, http.StatusBadRequest)
		})
	}
}

func TestBinaryArtefacts_ListFilters(t *testing.T) {
	e := setupBinaryArtefactEnv(t)
	mk := func(os, arch, siteId string) {
		fields := map[string]string{"version": "v", "os": os, "arch": arch}
		if siteId != "" {
			fields["siteId"] = siteId
		}
		resp := uploadArtefact(t, e.client, e.testServer.URL, fields, []byte("data"), nil)
		assertStatus(t, resp, http.StatusCreated)
	}
	mk("linux", "arm64", sid(e.site.ID))
	mk("freebsd", "arm64", sid(e.site.ID))
	mk("linux", "arm64", "") // generic

	count := func(query string) int {
		var list ListResponse[BinaryArtefact]
		resp := doRequest(t, e.client, "GET", listBinaryArtefactsURL(e.testServer.URL)+query, nil, &list)
		assertStatus(t, resp, http.StatusOK)
		return len(list.Items)
	}
	if n := count(""); n != 3 {
		t.Errorf("no filter: %d want 3", n)
	}
	if n := count("?os=linux"); n != 2 {
		t.Errorf("os=linux: %d want 2", n)
	}
	if n := count(fmt.Sprintf("?siteId=%d&os=freebsd", e.site.ID)); n != 1 {
		t.Errorf("site+freebsd: %d want 1", n)
	}
}
