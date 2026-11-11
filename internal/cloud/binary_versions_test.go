package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

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

// checkInRecorder is a Client that records each check-in request before passing it on.
type checkInRecorder struct {
	Client
	mu   sync.Mutex
	reqs []CheckInRequest
}

func (r *checkInRecorder) CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
	r.mu.Lock()
	r.reqs = append(r.reqs, req)
	r.mu.Unlock()
	return r.Client.CheckIn(ctx, req)
}

// take returns the requests recorded since the last take.
func (r *checkInRecorder) take() []CheckInRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	reqs := r.reqs
	r.reqs = nil
	return reqs
}

// newRecordedConn is newConnEnv with every check-in recorded.
func newRecordedConn(t *testing.T, env *clientEnv, opts ...BinaryUpdaterOption) (*Conn, *checkInRecorder) {
	t.Helper()
	rec := &checkInRecorder{Client: env.client}
	recEnv := *env
	recEnv.client = rec
	return newConnEnv(t, &recEnv, opts...), rec
}

func assertBinaryVersionID(t *testing.T, label string, req CheckInRequest, want ReportedID) {
	t.Helper()
	var got ReportedID
	if req.Running.Binary != nil {
		got = req.Running.Binary.VersionID
	}
	if got != want {
		t.Errorf("%s: running.binary.versionId = %+v, want %+v", label, got, want)
	}
}

// appliedReq returns the recorded check-in from reqs which reported that deployment depID was applied.
func appliedReq(t *testing.T, reqs []CheckInRequest, depID string) CheckInRequest {
	t.Helper()
	for _, req := range reqs {
		for _, p := range req.Progress {
			if p.DeploymentID == depID && p.State == ProgressApplied {
				return req
			}
		}
	}
	t.Fatalf("no check-in reports that depID=%s applied", depID)
	return CheckInRequest{}
}

// TestBinaryVersionID_DeploymentLifecycle follows one software deployment through install, the
// Supervisor's pending commit and its confirmation, checking which version id BOS reports at each step.
// It doesn't check the deployment's status, because the sim ignores the version ids BOS sends.
func TestBinaryVersionID_DeploymentLifecycle(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	versionsFile := WithBinaryVersionsFile(filepath.Join(t.TempDir(), "binary-versions.json"))
	fake := &fakeSupervisor{}
	sup := WithBinaryInstaller(dialFakeSupervisor(t, fake))

	artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "2.0.0", []byte("dummy-artefact-payload"))
	dep := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)
	depID := fmt.Sprintf("b-%d", dep)
	wantID := strconv.FormatInt(artID, 10)

	// BOS runs a hand-installed 1.0.0 and is offered 2.0.0.
	conn, rec := newRecordedConn(t, env, sup, versionsFile, WithBinaryVersion("1.0.0"))
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fake.calls() != 1 {
		t.Fatalf("InstallUpdate calls = %d, want 1", fake.calls())
	}
	assertBinaryVersionID(t, "hand-installed", rec.take()[0], ReportNoID())

	// BOS restarts into 2.0.0; the Supervisor waits for BOS to confirm it.
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_INSTALLING, Version: "2.0.0", DeploymentId: depID})
	conn, rec = newRecordedConn(t, env, sup, versionsFile, WithBinaryVersion("2.0.0"))
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	for i, req := range rec.take() {
		assertBinaryVersionID(t, fmt.Sprintf("awaiting commit, check-in %d", i), req, ReportedID{})
	}

	// The Supervisor confirms 2.0.0: BOS reports its version id, and reports it applied.
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_COMPLETED, Version: "2.0.0", DeploymentId: depID})
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reqs := rec.take()
	assertBinaryVersionID(t, "confirmed", reqs[0], ReportID(wantID))
	assertBinaryVersionID(t, "applied", appliedReq(t, reqs, depID), ReportID(wantID))

	// After another restart BOS still knows the id, though the cloud no longer offers the deployment.
	conn, rec = newRecordedConn(t, env, sup, versionsFile, WithBinaryVersion("2.0.0"))
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertBinaryVersionID(t, "after restart", rec.take()[0], ReportID(wantID))
}

// TestBinaryVersionID_Rollback: after the Supervisor rolls an update back, BOS reports the id of the
// version it is running again.
func TestBinaryVersionID_Rollback(t *testing.T) {
	ctx := context.Background()
	env := setupClientEnv(t)
	path := filepath.Join(t.TempDir(), "binary-versions.json")
	mustAdd(t, &binaryVersions{path: path}, "1.0.0", "v-1")
	fake := &fakeSupervisor{}

	depID := fmt.Sprintf("b-%d", seedInProgressBinary(t, env, "2.0.0"))
	mustAdd(t, &binaryVersions{path: path}, "2.0.0", "v-2") // as installBinary would
	fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_FAILED, Version: "2.0.0", DeploymentId: depID})

	conn, rec := newRecordedConn(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)),
		WithBinaryVersionsFile(path), WithBinaryVersion("1.0.0"))
	if _, err := conn.Update(ctx); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertBinaryVersionID(t, "rolled back", rec.take()[0], ReportID("v-1"))
}

// TestBinaryVersionID_AdoptsRunningVersion: deploying the version BOS already runs records its id
// without reinstalling, and reports it with the applied report.
func TestBinaryVersionID_AdoptsRunningVersion(t *testing.T) {
	for _, withSupervisor := range []bool{true, false} {
		t.Run(fmt.Sprintf("supervisor=%v", withSupervisor), func(t *testing.T) {
			ctx := context.Background()
			env := setupClientEnv(t)
			fake := &fakeSupervisor{}
			opts := []BinaryUpdaterOption{
				WithBinaryVersionsFile(filepath.Join(t.TempDir(), "binary-versions.json")),
				WithBinaryVersion("1.0.0"),
			}
			if withSupervisor {
				opts = append(opts, WithBinaryInstaller(dialFakeSupervisor(t, fake)))
			}
			conn, rec := newRecordedConn(t, env, opts...)

			artID := createBinaryArtefact(t, env.httpClient, env.testServer.URL, "1.0.0", []byte("dummy-artefact-payload"))
			dep := createBinaryDeployment(t, env.httpClient, env.testServer.URL, artID, env.nodeID)
			wantID := strconv.FormatInt(artID, 10)

			if _, err := conn.Update(ctx); err != nil {
				t.Fatalf("Update: %v", err)
			}
			if fake.calls() != 0 {
				t.Errorf("InstallUpdate calls = %d, want 0", fake.calls())
			}
			reqs := rec.take()
			assertBinaryVersionID(t, "before adoption", reqs[0], ReportNoID())
			assertBinaryVersionID(t, "applied", appliedReq(t, reqs, fmt.Sprintf("b-%d", dep)), ReportID(wantID))

			if _, err := conn.Update(ctx); err != nil {
				t.Fatalf("Update: %v", err)
			}
			assertBinaryVersionID(t, "next poll", rec.take()[0], ReportID(wantID))
		})
	}
}

// TestBinaryVersionID_Withheld: BOS sends no version id when it can't be sure the version it runs will
// stay, or can't read its record.
func TestBinaryVersionID_Withheld(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, fake *fakeSupervisor, path string)
	}{
		{"supervisor downloading", func(t *testing.T, fake *fakeSupervisor, path string) {
			fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_DOWNLOADING, Version: "2.0.0", DeploymentId: "b-1"})
		}},
		{"supervisor failed the running version", func(t *testing.T, fake *fakeSupervisor, path string) {
			// the Supervisor's rollback failed, leaving BOS on the version it failed
			fake.setStatus(&supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_FAILED, Version: "1.0.0", DeploymentId: "b-1"})
		}},
		{"supervisor status unknown", func(t *testing.T, fake *fakeSupervisor, path string) {
			fake.statusErr = errors.New("unavailable")
		}},
		{"record unreadable", func(t *testing.T, fake *fakeSupervisor, path string) {
			if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := setupClientEnv(t)
			path := filepath.Join(t.TempDir(), "binary-versions.json")
			mustAdd(t, &binaryVersions{path: path}, "1.0.0", "v-1")
			fake := &fakeSupervisor{}
			tt.setup(t, fake, path)

			conn, rec := newRecordedConn(t, env, WithBinaryInstaller(dialFakeSupervisor(t, fake)),
				WithBinaryVersionsFile(path), WithBinaryVersion("1.0.0"))
			_, _ = conn.Update(ctx) // an unknown Supervisor status fails the poll after the check-in
			reqs := rec.take()
			if len(reqs) == 0 {
				t.Fatal("no check-in made")
			}
			assertBinaryVersionID(t, tt.name, reqs[0], ReportedID{})
		})
	}
}
