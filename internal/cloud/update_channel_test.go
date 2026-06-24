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

// newConnEnv wires a Conn (with the update channel enabled) on top of a clientEnv, returning the Conn
// and its UpdateStore so tests can assert persisted intent. opts are forwarded to the SoftwareUpdater;
// pass WithUpdateInstaller to enable installing (without it the update channel treats the
// Supervisor as disabled and persists no intent).
func newConnEnv(t *testing.T, env *clientEnv, opts ...SoftwareUpdaterOption) (*Conn, UpdateStore) {
	t.Helper()
	regStore := NewFileRegistrationStore(filepath.Join(t.TempDir(), "registration.json"))
	if err := regStore.Save(context.Background(), env.registration); err != nil {
		t.Fatalf("save registration: %v", err)
	}
	updateStore := NewFileUpdateStore(filepath.Join(t.TempDir(), "update.json"))
	softwareUpdater := NewSoftwareUpdater(updateStore, opts...)
	conn, err := OpenConn(context.Background(), regStore, env.store,
		WithClientFactory(func(Registration) Client { return env.client }),
		WithSoftwareUpdater(softwareUpdater),
	)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return conn, updateStore
}

func TestUpdateChannel_PersistsIntent(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := dialFakeSupervisor(t, &fakeSupervisor{})
	conn, updateStore := newConnEnv(t, env, WithUpdateInstaller(fake))

	artID := createUpdateArtefact(t, env.httpClient, env.testServer.URL, "1.2.3", []byte("dummy-artefact-payload"))
	depID := createUpdateDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}

	state, ok, err := updateStore.Load(ctx)
	if err != nil {
		t.Fatalf("load update state: %v", err)
	}
	if !ok {
		t.Fatal("expected persisted update state after latestUpdate")
	}
	if state.DeploymentID != fmt.Sprintf("%d", depID) {
		t.Errorf("DeploymentID = %q, want %q", state.DeploymentID, fmt.Sprintf("%d", depID))
	}
	if state.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", state.Version, "1.2.3")
	}
	if state.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}

	// A repeat poll with the same deployment id must not overwrite the record.
	first := state
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("second Update: %v", err)
	}
	state2, _, err := updateStore.Load(ctx)
	if err != nil {
		t.Fatalf("load update state again: %v", err)
	}
	if !state2.StartTime.Equal(first.StartTime) {
		t.Errorf("expected StartTime unchanged on repeat poll; got %v want %v", state2.StartTime, first.StartTime)
	}
}

func TestUpdateChannel_NoLatestUpdate(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	conn, updateStore := newConnEnv(t, env)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok, err := updateStore.Load(ctx); err != nil || ok {
		t.Errorf("expected no persisted update state; ok=%v err=%v", ok, err)
	}
}

// TestSharedCheckIn_HandlerErrorMarksFailed is a regression test: when the transport check-in
// succeeds but applying the update fails, the connection state must reflect the failure rather than
// report a healthy connection (which would let WaitConnected unblock on a half-failed poll).
func TestSharedCheckIn_HandlerErrorMarksFailed(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := dialFakeSupervisor(t, &fakeSupervisor{installFails: true})
	conn, _ := newConnEnv(t, env, WithUpdateInstaller(fake))

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
	fake := dialFakeSupervisor(t, &fakeSupervisor{})
	conn, updateStore := newConnEnv(t, env, WithUpdateInstaller(fake))

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

	// update channel acted: persisted the in-flight intent.
	state, ok, err := updateStore.Load(ctx)
	if err != nil {
		t.Fatalf("load update state: %v", err)
	}
	if !ok {
		t.Fatal("expected persisted update state from update channel")
	}
	if state.DeploymentID != fmt.Sprintf("%d", updDepID) {
		t.Errorf("DeploymentID = %q, want %q", state.DeploymentID, fmt.Sprintf("%d", updDepID))
	}
}
