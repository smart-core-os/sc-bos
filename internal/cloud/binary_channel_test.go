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
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// createBinaryArtefact uploads a binary artefact (multipart) and returns its ID and version.
func createBinaryArtefact(t *testing.T, client *http.Client, baseURL, version string, payload []byte) int64 {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("version", version)
	_ = mw.WriteField("os", "linux")
	_ = mw.WriteField("arch", "arm64")
	fw, err := mw.CreateFormFile("payload", "binary.tar")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/v1/management/binary-artefacts", &body)
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
		t.Fatalf("create binary artefact: expected 201, got %d", resp.StatusCode)
	}
	var a sim.BinaryArtefact
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return a.ID
}

// createBinaryDeployment creates a PENDING binary deployment for the artefact + node and returns its ID.
func createBinaryDeployment(t *testing.T, client *http.Client, baseURL string, artefactID, nodeID int64) int64 {
	t.Helper()
	var dep sim.BinaryDeployment
	resp := doSimRequest(t, client, "POST", baseURL+"/api/v1/management/binary-deployments", map[string]any{
		"binaryArtefactId": strconv.FormatInt(artefactID, 10),
		"nodeId":           strconv.FormatInt(nodeID, 10),
	}, &dep)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create binary deployment: expected 201, got %d", resp.StatusCode)
	}
	return dep.ID
}

// newConnEnv wires a Conn (with the binary channel enabled) on top of a clientEnv. opts are forwarded
// to the BinaryUpdater; pass WithBinaryInstaller to enable installing (without it the binary channel
// treats the Supervisor as disabled and does nothing).
func newConnEnv(t *testing.T, env *clientEnv, opts ...BinaryUpdaterOption) *Conn {
	t.Helper()
	regStore := NewFileRegistrationStore(filepath.Join(t.TempDir(), "registration.json"))
	if err := regStore.Save(context.Background(), env.registration); err != nil {
		t.Fatalf("save registration: %v", err)
	}
	binaryUpdater := NewBinaryUpdater(opts...)
	conn, err := OpenConn(context.Background(), regStore, env.store, env.testServer.URL,
		WithClientFactory(func(*Registration) Client { return env.client }),
		WithBinaryUpdater(binaryUpdater),
		WithPlatform(env.platform),
	)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return conn
}

func TestBinaryChannel_NoLatestBinary(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)))

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// With no latestBinary the binary channel must not touch the Supervisor.
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 when there is no latestBinary", fake.calls())
	}
}

// TestSharedCheckIn_HandlerErrorMarksFailed is a regression test: when the transport check-in
// succeeds but applying the binary fails, the connection state must reflect the failure rather than
// report a healthy connection (which would let WaitConnected unblock on a half-failed poll).
func TestSharedCheckIn_HandlerErrorMarksFailed(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := dialFakeSupervisor(t, &fakeSupervisor{installFails: true})
	conn := newConnEnv(t, env, WithBinaryInstaller(fake))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "1.2.3", []byte("dummy-artefact-payload"))
	createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err == nil {
		t.Fatal("expected Update to return the binary install failure")
	}
	st := conn.State()
	if st.Connectivity != Failed {
		t.Errorf("Connectivity = %v, want Failed after a failed binary handler", st.Connectivity)
	}
	if st.LastError == nil {
		t.Error("LastError = nil, want the binary failure recorded")
	}
}

// TestInterlock_ConfigWinsSamePollTie exercises a single poll that carries BOTH a latestConfig and a
// latestBinary with nothing yet in flight. The single-install interlock lets only one install start; the
// same-poll tie is broken in favour of config (checked first), so the binary install is held back.
func TestInterlock_ConfigWinsSamePollTie(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)))

	// config channel: a pending config deployment.
	cvPayload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, cvPayload)
	cfgDepID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	// binary channel: a pending binary deployment.
	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "4.5.6", []byte("dummy-artefact-payload"))
	updDepID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	needReboot, err := conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// config channel won the tie: staged the deployment and asked for a reboot.
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

	// binary channel was held back: no install commanded, and the binary deployment stays pending.
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 while a config install is starting", fake.calls())
	}
	updDep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, updDepID)
	if updDep.Status != "pending" {
		t.Errorf("binary deployment status = %q, want pending (binary install deferred)", updDep.Status)
	}
}

// TestInterlock_BinaryInFlightBlocksConfig: with a binary install already in flight (the Supervisor is
// INSTALLING), a newly-offered config deployment must not be staged - it waits for the binary to settle.
func TestInterlock_BinaryInFlightBlocksConfig(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)))

	// binary in flight: an in_progress binary deployment plus a Supervisor reporting INSTALLING for it.
	updDepID := seedInProgressBinary(t, env, "9.9.9")
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_INSTALLING,
		Version:      "9.9.9",
		DeploymentId: fmt.Sprintf("b-%d", updDepID),
	})

	// a pending config deployment offered while the binary is mid-install.
	cvPayload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, cvPayload)
	cfgDepID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	needReboot, err := conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// config channel was blocked: no reboot, nothing staged, the deployment stays pending.
	if needReboot {
		t.Error("expected needReboot=false: config install must be blocked while a binary install is in flight")
	}
	if symlinkExists(env.storePath, "deployments/installing") {
		t.Error("config deployment must not be staged while a binary install is in flight")
	}
	cfgDep := getDeployment(t, env.httpClient, env.testServer.URL, cfgDepID)
	if cfgDep.Status != "pending" {
		t.Errorf("config deployment status = %q, want pending (blocked by binary install)", cfgDep.Status)
	}
}

// TestInterlock_BinaryInFlightWithoutOfferBlocksConfig: the Supervisor is INSTALLING, but the server no
// longer offers the binary deployment (e.g. its payload went missing, or it was superseded). The config
// reboot must still be blocked - the in-flight decision comes from the Supervisor, not the offer.
func TestInterlock_BinaryInFlightWithoutOfferBlocksConfig(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)))

	// Supervisor mid-install, but no binary deployment exists on the server, so no binary is offered.
	fake.setStatus(&supervisorpb.UpdateStatus{
		State:        supervisorpb.UpdateStatus_INSTALLING,
		Version:      "9.9.9",
		DeploymentId: "b-999",
	})

	// a pending config deployment offered while the binary is mid-install.
	cvPayload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, cvPayload)
	cfgDepID := createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)

	needReboot, err := conn.Update(ctx)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if needReboot {
		t.Error("expected needReboot=false: config must be blocked by an in-flight binary even with no offer")
	}
	if symlinkExists(env.storePath, "deployments/installing") {
		t.Error("config deployment must not be staged while a binary install is in flight")
	}
	cfgDep := getDeployment(t, env.httpClient, env.testServer.URL, cfgDepID)
	if cfgDep.Status != "pending" {
		t.Errorf("config deployment status = %q, want pending (blocked by in-flight binary)", cfgDep.Status)
	}
}

// TestInterlock_ConfigInFlightBlocksBinary: with a config install already staged (in flight across the
// reboot), a newly-offered binary deployment must not start - it waits for the config to settle.
func TestInterlock_ConfigInFlightBlocksBinary(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	fake := &fakeSupervisor{}
	conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)))

	// stage a config install: the first poll offers only config, which stages and marks installing.
	cvPayload := txtarToTarGZ(t, "single.txtar")
	cvID := createConfigVersion(t, env.httpClient, env.testServer.URL, env.nodeID, cvPayload)
	createPendingDeployment(t, env.httpClient, env.testServer.URL, cvID)
	if needReboot, err := conn.Update(ctx); err != nil {
		t.Fatalf("first Update: %v", err)
	} else if !needReboot {
		t.Fatal("expected config install to stage on the first poll")
	}

	// now offer a binary deployment while the config install is in flight.
	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "4.5.6", []byte("dummy-artefact-payload"))
	updDepID := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("second Update: %v", err)
	}

	// binary channel was blocked: no install commanded, and the binary deployment stays pending.
	if fake.calls() != 0 {
		t.Errorf("InstallUpdate calls = %d, want 0 while a config install is in flight", fake.calls())
	}
	updDep := getBinaryDeployment(t, env.httpClient, env.testServer.URL, updDepID)
	if updDep.Status != "pending" {
		t.Errorf("binary deployment status = %q, want pending (blocked by config install)", updDep.Status)
	}
}
