package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
)

// createUpdateArtefact uploads an update artefact (multipart) and returns its ID and version.
func createUpdateArtefact(t *testing.T, client *http.Client, baseURL, version string, payload []byte) int64 {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("version", version)
	_ = mw.WriteField("platform", "podman")
	fw, err := mw.CreateFormFile("payload", "update.tar")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/v1/management/update-artefacts", &body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create update artefact: expected 201, got %d", resp.StatusCode)
	}
	var a sim.UpdateArtefact
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return a.ID
}

// createUpdateDeployment creates a PENDING update deployment for the artefact + node and returns its ID.
func createUpdateDeployment(t *testing.T, client *http.Client, baseURL string, artefactID, nodeID int64) int64 {
	t.Helper()
	var dep sim.UpdateDeployment
	resp := doSimRequest(t, client, "POST", baseURL+"/api/v1/management/update-deployments", map[string]any{
		"updateArtefactId": strconv.FormatInt(artefactID, 10),
		"nodeId":           strconv.FormatInt(nodeID, 10),
	}, &dep)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create update deployment: expected 201, got %d", resp.StatusCode)
	}
	return dep.ID
}

// newConnEnv wires a Conn (with the update channel enabled) on top of a clientEnv. opts are forwarded
// to the SoftwareUpdater; pass WithUpdateInstaller to enable installing (without it the update channel
// treats the Supervisor as disabled and does nothing).
func newConnEnv(t *testing.T, env *clientEnv, opts ...SoftwareUpdaterOption) *Conn {
	t.Helper()
	regStore := NewFileRegistrationStore(filepath.Join(t.TempDir(), "registration.json"))
	if err := regStore.Save(context.Background(), env.registration); err != nil {
		t.Fatalf("save registration: %v", err)
	}
	softwareUpdater := NewSoftwareUpdater(opts...)
	conn, err := OpenConn(context.Background(), regStore, env.store,
		WithClientFactory(func(Registration) Client { return env.client }),
		WithSoftwareUpdater(softwareUpdater),
	)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return conn
}

func TestUpdateChannel_NoLatestUpdate(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithUpdateInstaller(dialFakeSupervisor(t, fake)))

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// With no latestUpdate the update channel must not touch the Supervisor.
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 when there is no latestUpdate", fake.calls())
	}
}

// TestSharedCheckIn_HandlerErrorMarksFailed is a regression test: when the transport check-in
// succeeds but applying the update fails, the connection state must reflect the failure rather than
// report a healthy connection (which would let WaitConnected unblock on a half-failed poll).
func TestSharedCheckIn_HandlerErrorMarksFailed(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := dialFakeSupervisor(t, &fakeSupervisor{installFails: true})
	conn := newConnEnv(t, env, WithUpdateInstaller(fake))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "1.2.3", []byte("dummy-artefact-payload"))
	createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err == nil {
		t.Fatal("expected Update to return the update install failure")
	}
	st := conn.State()
	if st.Connectivity != Failed {
		t.Errorf("Connectivity = %v, want Failed after a failed update handler", st.Connectivity)
	}
	if st.LastError == nil {
		t.Error("LastError = nil, want the update failure recorded")
	}
}

// TestSharedCheckIn_BothChannels exercises a single poll that carries BOTH a latestConfig and a
// latestUpdate, confirming both handlers act from one check-in.
func TestSharedCheckIn_BothChannels(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithUpdateInstaller(dialFakeSupervisor(t, fake)))

	// config channel: a pending config deployment.
	cvPayload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, cvPayload)
	cfgDepID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	// update channel: a pending update deployment.
	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "4.5.6", []byte("dummy-artefact-payload"))
	updDepID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	needReboot, err := conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// config channel acted: staged the deployment and asked for a reboot.
	if !needReboot {
		t.Error("expected needReboot=true from config channel")
	}
	if !symlinkExists(env.storePath, "deployments/installing") {
		t.Error("expected config deployment staged as installing")
	}
	cfgDep := getDeployment(t, env.httpClient, env.testServer.URL, cfgDepID)
	if cfgDep.Status != "in_progress" {
		t.Errorf("config deployment status = %q, want in_progress", cfgDep.Status)
	}

	// update channel acted: commanded the Supervisor with the update deployment's id and reported it
	// installing, moving the update deployment to in_progress.
	if got := fake.lastInstall().GetDeploymentId(); got != fmt.Sprintf("%d", updDepID) {
		t.Errorf("InstallUpdate deployment id = %q, want %q", got, fmt.Sprintf("%d", updDepID))
	}
	updDep := getUpdateDeployment(t, env.httpClient, env.testServer.URL, updDepID)
	if updDep.Status != "in_progress" {
		t.Errorf("update deployment status = %q, want in_progress", updDep.Status)
	}
}
