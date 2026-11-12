package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

func TestReportedID_JSON(t *testing.T) {
	tests := []struct {
		name string
		id   ReportedID
		want string
	}{
		{"absent", ReportedID{}, `{"version":"1.0.0"}`},
		{"empty id is absent", ReportID(""), `{"version":"1.0.0"}`},
		{"null", ReportNoID(), `{"version":"1.0.0","versionId":null}`},
		{"set", ReportID("v-1"), `{"version":"1.0.0","versionId":"v-1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := RunningArtefact{Version: "1.0.0", VersionID: tt.id}
			got, err := json.Marshal(in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBinaryVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary-versions.json")
	b := &binaryVersions{path: path}

	if _, ok, err := b.lookup("1.0.0"); err != nil || ok {
		t.Fatalf("lookup with no file = ok %v, err %v; want not found", ok, err)
	}
	mustAdd(t, b, "1.0.0", "a")
	mustAdd(t, b, "2.0.0", "b")
	mustAdd(t, b, "1.0.0", "c") // the same version deployed again under a new id

	// a new instance stands in for a restart
	b = &binaryVersions{path: path}
	assertLookup(t, b, "1.0.0", "c")
	assertLookup(t, b, "2.0.0", "b")
	if _, ok, _ := b.lookup("3.0.0"); ok {
		t.Error("lookup(3.0.0) found an entry, want none")
	}

	t.Run("discards the oldest", func(t *testing.T) {
		for i := range maxBinaryVersions {
			mustAdd(t, b, strconv.Itoa(i), strconv.Itoa(i))
		}
		if _, ok, _ := b.lookup("2.0.0"); ok {
			t.Error("lookup(2.0.0) found an entry, want it removed")
		}
		assertLookup(t, b, "0", "0")
	})

	t.Run("replaces an unreadable file", func(t *testing.T) {
		if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := b.lookup("0"); err == nil {
			t.Error("lookup on a corrupt file returned no error")
		}
		mustAdd(t, b, "4.0.0", "d")
		assertLookup(t, b, "4.0.0", "d")
	})
}

func mustAdd(t *testing.T, b *binaryVersions, version, id string) {
	t.Helper()
	if err := b.add(version, id); err != nil {
		t.Fatalf("add(%s, %s): %v", version, id, err)
	}
}

func assertLookup(t *testing.T, b *binaryVersions, version, wantID string) {
	t.Helper()
	id, ok, err := b.lookup(version)
	if err != nil || !ok || id != wantID {
		t.Errorf("lookup(%s) = %q, ok %v, err %v; want %q", version, id, ok, err, wantID)
	}
}

// assertReportedBinaryVersionID checks the binary version id the sim has recorded for the env's node.
// An empty want means none is recorded.
func assertReportedBinaryVersionID(t *testing.T, env *clientEnv, label, want string) {
	t.Helper()
	var node sim.Node
	resp := doSimRequest(t, env.httpClient, "GET",
		fmt.Sprintf("%s/api/v1/management/nodes/%d", env.testServer.URL, env.nodeID), nil, &node)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get node: expected 200, got %d", resp.StatusCode)
	}
	if node.ReportedBinaryVersionID != want {
		t.Errorf("%s: recorded binary version id = %q, want %q", label, node.ReportedBinaryVersionID, want)
	}
}

func assertDeploymentStatus(t *testing.T, env *clientEnv, label string, dep int64, want string) {
	t.Helper()
	if got := getBinaryDeployment(t, env.httpClient, env.testServer.URL, dep).Status; got != want {
		t.Errorf("%s: deployment status = %q, want %q", label, got, want)
	}
}

func mustUpdate(t *testing.T, conn *Conn) {
	t.Helper()
	if _, err := conn.Update(context.Background()); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

// TestBinaryVersionID_DeploymentLifecycle follows one software deployment through install, the
// Supervisor's pending commit and its confirmation. The cloud learns the new version's id, and completes
// the deployment, only once the Supervisor confirms it.
func TestBinaryVersionID_DeploymentLifecycle(t *testing.T) {
	env := setupClientEnv(t)
	versionsFile := WithBinaryVersionsFile(filepath.Join(t.TempDir(), "binary-versions.json"))
	fake := &fakeSupervisor{}
	sup := WithBinaryInstaller(dialFakeSupervisor(t, fake))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "2.0.0", []byte("dummy-artefact-payload"))
	dep := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)
	depID := fmt.Sprintf("b-%d", dep)
	wantID := strconv.FormatInt(artID, 10)

	// BOS runs a hand-installed 1.0.0 and is offered 2.0.0.
	mustUpdate(t, newConnEnv(t, env, sup, versionsFile, WithBinaryVersion("1.0.0")))
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1", fake.calls())
	}

	// BOS restarts into 2.0.0; the Supervisor waits for BOS to confirm it.
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_INSTALLING, Version: "2.0.0", DeploymentId: depID})
	conn := newConnEnv(t, env, sup, versionsFile, WithBinaryVersion("2.0.0"))
	mustUpdate(t, conn)
	assertReportedBinaryVersionID(t, env, "awaiting commit", "")
	assertDeploymentStatus(t, env, "awaiting commit", dep, "in_progress")

	// The Supervisor confirms 2.0.0.
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_COMPLETED, Version: "2.0.0", DeploymentId: depID})
	mustUpdate(t, conn)
	assertReportedBinaryVersionID(t, env, "confirmed", wantID)
	assertDeploymentStatus(t, env, "confirmed", dep, "completed")

	// BOS restarts again and keeps reporting the id.
	mustUpdate(t, newConnEnv(t, env, sup, versionsFile, WithBinaryVersion("2.0.0")))
	assertReportedBinaryVersionID(t, env, "after restart", wantID)
}

// TestBinaryVersionID_Rollback: after the Supervisor rolls an update back, the cloud learns the update
// failed, and records the id of the version BOS runs again.
func TestBinaryVersionID_Rollback(t *testing.T) {
	env := setupClientEnv(t)
	path := filepath.Join(t.TempDir(), "binary-versions.json")
	mustAdd(t, &binaryVersions{path: path}, "1.0.0", "v-1")
	fake := &fakeSupervisor{}

	dep := seedInProgressBinary(t, env, "2.0.0")
	mustAdd(t, &binaryVersions{path: path}, "2.0.0", "v-2") // as installBinary would
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_FAILED, Version: "2.0.0", DeploymentId: fmt.Sprintf("b-%d", dep)})

	mustUpdate(t, newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)),
		WithBinaryVersionsFile(path), WithBinaryVersion("1.0.0")))
	assertReportedBinaryVersionID(t, env, "rolled back", "v-1")
	assertDeploymentStatus(t, env, "rolled back", dep, "failed")
}

// TestBinaryVersionID_RollbackFailed: when the Supervisor's rollback fails, leaving BOS on the version it
// failed, the cloud records the deployment as failed and no longer records the earlier version's id.
func TestBinaryVersionID_RollbackFailed(t *testing.T) {
	env := setupClientEnv(t)
	path := filepath.Join(t.TempDir(), "binary-versions.json")
	versionsFile := WithBinaryVersionsFile(path)
	mustAdd(t, &binaryVersions{path: path}, "1.0.0", "v-1")
	fake := &fakeSupervisor{}
	sup := WithBinaryInstaller(dialFakeSupervisor(t, fake))

	mustUpdate(t, newConnEnv(t, env, sup, versionsFile, WithBinaryVersion("1.0.0")))
	assertReportedBinaryVersionID(t, env, "before update", "v-1")

	dep := seedInProgressBinary(t, env, "2.0.0")
	artID := getBinaryDeployment(t, env.httpClient, env.testServer.URL, dep).BinaryArtefactID
	mustAdd(t, &binaryVersions{path: path}, "2.0.0", strconv.FormatInt(artID, 10)) // as installBinary would
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_FAILED, Version: "2.0.0", DeploymentId: fmt.Sprintf("b-%d", dep)})

	mustUpdate(t, newConnEnv(t, env, sup, versionsFile, WithBinaryVersion("2.0.0")))
	assertReportedBinaryVersionID(t, env, "rollback failed", "")
	assertDeploymentStatus(t, env, "rollback failed", dep, "failed")
}

// TestBinaryVersionID_ManualInstall: after a manual install of a version BOS has no record of, the cloud
// no longer records a version id for the node.
func TestBinaryVersionID_ManualInstall(t *testing.T) {
	env := setupClientEnv(t)
	versionsFile := WithBinaryVersionsFile(filepath.Join(t.TempDir(), "binary-versions.json"))
	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "1.0.0", []byte("dummy-artefact-payload"))
	createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

	mustUpdate(t, newConnEnv(t, env, versionsFile, WithBinaryVersion("1.0.0")))
	assertReportedBinaryVersionID(t, env, "deployed", strconv.FormatInt(artID, 10))

	mustUpdate(t, newConnEnv(t, env, versionsFile, WithBinaryVersion("3.0.0")))
	assertReportedBinaryVersionID(t, env, "manually installed", "")
}

// TestBinaryVersionID_AdoptsRunningVersion: deploying the version BOS already runs completes the
// deployment without reinstalling, and the cloud records its id.
func TestBinaryVersionID_AdoptsRunningVersion(t *testing.T) {
	for _, withSupervisor := range []bool{true, false} {
		t.Run(fmt.Sprintf("supervisor=%v", withSupervisor), func(t *testing.T) {
			env := setupClientEnv(t)
			fake := &fakeSupervisor{}
			opts := []BinaryUpdaterOption{
				WithBinaryVersionsFile(filepath.Join(t.TempDir(), "binary-versions.json")),
				WithBinaryVersion("1.0.0"),
			}
			if withSupervisor {
				opts = append(opts, WithBinaryInstaller(dialFakeSupervisor(t, fake)))
			}
			artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "1.0.0", []byte("dummy-artefact-payload"))
			dep := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)

			mustUpdate(t, newConnEnv(t, env, opts...))
			if fake.calls() != 0 {
				t.Errorf("InstallUpdate calls = %d, want 0", fake.calls())
			}
			assertReportedBinaryVersionID(t, env, "adopted", strconv.FormatInt(artID, 10))
			assertDeploymentStatus(t, env, "adopted", dep, "completed")
		})
	}
}

// TestBinaryVersionID_Withheld: BOS sends no version id when it can't be sure the version it runs will
// stay, or can't read its record, so the cloud keeps the id it had.
func TestBinaryVersionID_Withheld(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, fake *fakeSupervisor, path string)
	}{
		{"supervisor downloading", func(t *testing.T, fake *fakeSupervisor, path string) {
			fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_DOWNLOADING, Version: "2.0.0", DeploymentId: "b-1"})
		}},
		{"supervisor status unknown", func(t *testing.T, fake *fakeSupervisor, path string) {
			fake.mu.Lock()
			fake.statusErr = errors.New("unavailable")
			fake.mu.Unlock()
		}},
		{"record unreadable", func(t *testing.T, fake *fakeSupervisor, path string) {
			if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupClientEnv(t)
			path := filepath.Join(t.TempDir(), "binary-versions.json")
			mustAdd(t, &binaryVersions{path: path}, "1.0.0", "v-1")
			fake := &fakeSupervisor{}
			conn := newConnEnv(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)),
				WithBinaryVersionsFile(path), WithBinaryVersion("1.0.0"))
			mustUpdate(t, conn)
			assertReportedBinaryVersionID(t, env, "before", "v-1")

			tt.setup(t, fake, path)
			_, _ = conn.Update(context.Background()) // an unknown Supervisor status fails the poll after the check-in
			assertReportedBinaryVersionID(t, env, "after", "v-1")
		})
	}
}
