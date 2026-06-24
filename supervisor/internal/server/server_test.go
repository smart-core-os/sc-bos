package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// testDeadline is the commit deadline used by tests. Its exact value does not matter: under synctest the
// fake clock advances it instantly once the reconcile goroutine is the only thing left waiting.
const testDeadline = 2 * time.Minute

// fakeInstaller records the methods called on it and returns scripted results. Confirmation is no longer
// the installer's job (BOS asserts it via Commit), so the fake only implements Apply and Rollback.
type fakeInstaller struct {
	mu sync.Mutex

	applyErr    error
	rollbackErr error

	// applyHook, if set, is called at the end of a successful Apply. Tests use it to inject a commit
	// during the install window (before the reconcile captures the post-restart commit generation).
	applyHook func()

	// current models the version :current resolves to (and so the version a restart recreates from).
	// previous is the rollback pointer. Tests may set current to model a host left by a prior crash.
	current  string
	previous string

	calls        []string
	appliedVer   string
	appliedBytes []byte // contents of the artefact passed to Apply
}

func (f *fakeInstaller) Apply(_ context.Context, artefactPath, version string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "Apply")
	f.appliedVer = version
	if b, err := os.ReadFile(artefactPath); err == nil {
		f.appliedBytes = b
	}
	if f.applyErr != nil {
		return f.applyErr
	}
	// Mirror PodmanInstaller.Apply: only record :previous and repoint :current when :current is not already
	// version. Re-applying an already-applied version (a resume after the tag-swap) preserves the rollback
	// pointer rather than overwriting it with version.
	if f.current != version {
		f.previous, f.current = f.current, version
	}
	if f.applyHook != nil {
		f.applyHook()
	}
	return nil
}

func (f *fakeInstaller) Rollback(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "Rollback")
	if f.rollbackErr != nil {
		return f.rollbackErr
	}
	if f.previous == "" {
		return errors.New("no previous image to roll back to")
	}
	f.current = f.previous // repoint :current back to the previous image
	return nil
}

func (f *fakeInstaller) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// assertCalls fails the test when the installer's recorded calls do not match want.
func assertCalls(t *testing.T, fake *fakeInstaller, want ...string) {
	t.Helper()
	if diff := cmp.Diff(want, fake.callLog()); diff != "" {
		t.Errorf("installer calls mismatch (-want +got):\n%s", diff)
	}
}

// artefactServer starts an HTTPS server returning the given bytes (created outside any synctest bubble)
// and returns its URL, type-prefixed sha256 checksum, and a client that trusts it.
func artefactServer(t *testing.T) (data []byte, url, sum string, client *http.Client) {
	t.Helper()
	data = make([]byte, 2048)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	client = srv.Client()
	client.Transport.(*http.Transport).DisableKeepAlives = true
	return data, srv.URL, checksum.Format(checksum.SHA256, h[:]), client
}

// zeroSum is a well-formed sha256 checksum of the wrong (all-zero) digest: it passes checksum parsing
// but never matches real content, for exercising mismatch and pre-download validation paths.
func zeroSum() string { return checksum.Format(checksum.SHA256, make([]byte, sha256.Size)) }

func newService(t *testing.T, fake *fakeInstaller, dir string, client *http.Client) *Service {
	t.Helper()
	return New(fake, dir, client, testDeadline, false, nil)
}

func testClient(t *testing.T, svc *Service) supervisorpb.SupervisorApiClient {
	t.Helper()
	conn := wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, svc)
	return supervisorpb.NewSupervisorApiClient(conn)
}

func install2(c supervisorpb.SupervisorApiClient, version, url, sum string) error {
	_, err := c.InstallUpdate(context.Background(), &supervisorpb.InstallUpdateRequest{
		Version: version, DownloadUrl: url, Checksum: sum,
	})
	return err
}

func mustInstall(t *testing.T, c supervisorpb.SupervisorApiClient, version, url, sum string) {
	t.Helper()
	if err := install2(c, version, url, sum); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, c supervisorpb.SupervisorApiClient, version string) {
	t.Helper()
	if _, err := c.Commit(context.Background(), &supervisorpb.CommitRequest{Version: version}); err != nil {
		t.Fatal(err)
	}
}

// elapseDeadline advances the synctest fake clock past the commit deadline, so a reconcile awaiting a
// Commit gives up and rolls back. time.Sleep advances the bubble clock once every goroutine is blocked;
// synctest.Wait does not, so timeout-driven paths must use this. Call inside a synctest bubble, then
// synctest.Wait to let the reconcile act on the elapsed deadline.
func elapseDeadline() {
	time.Sleep(testDeadline + time.Second)
}

func getStatus(t *testing.T, c supervisorpb.SupervisorApiClient) *supervisorpb.UpdateStatus {
	t.Helper()
	resp, err := c.GetUpdateStatus(context.Background(), &supervisorpb.GetUpdateStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	return resp.GetStatus()
}

func writeStateFile(t *testing.T, dir string, rec record) {
	t.Helper()
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Boundary validation: missing fields and invalid image tags are rejected before any work starts.
func TestInstallUpdate_InvalidRequest(t *testing.T) {
	c := testClient(t, newService(t, &fakeInstaller{}, t.TempDir(), http.DefaultClient))

	_, err := c.InstallUpdate(context.Background(), &supervisorpb.InstallUpdateRequest{Version: "v2"})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Errorf("status code = %v, want %v", got, codes.InvalidArgument)
	}

	err = install2(c, "bad/tag", "https://example.com/a.tar", zeroSum())
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Errorf("status code = %v, want %v", got, codes.InvalidArgument)
	}

	if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_IDLE {
		t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_IDLE)
	}
}

// A Commit while no update is in flight is a routine heartbeat: it is accepted and records the running
// version, and the reported status stays IDLE.
func TestCommit_Idle_Heartbeat(t *testing.T) {
	c := testClient(t, newService(t, &fakeInstaller{}, t.TempDir(), http.DefaultClient))

	commit(t, c, "v1")
	if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_IDLE {
		t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_IDLE)
	}
}

// Happy path: artefact downloads, Apply succeeds, BOS commits the new version within the deadline ->
// COMPLETED. Apply is the only installer call; the downloaded bytes reach it intact.
func TestInstallUpdate_HappyPath(t *testing.T) {
	data, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait() // reconcile downloads, applies, then parks awaiting Commit(v2)

		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_INSTALLING {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_INSTALLING)
		}

		commit(t, c, "v2")
		synctest.Wait()

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
		if got := st.GetVersion(); got != "v2" {
			t.Errorf("version = %q, want %q", got, "v2")
		}
		if got := st.GetError(); got != "" {
			t.Errorf("error = %q, want empty", got)
		}
		if st.GetFinishTime() == nil {
			t.Error("finish time should be set")
		}

		assertCalls(t, fake, "Apply")
		if got := fake.appliedVer; got != "v2" {
			t.Errorf("applied version = %q, want %q", got, "v2")
		}
		if !bytes.Equal(fake.appliedBytes, data) {
			t.Error("Apply received the downloaded artefact bytes")
		}
	})
}

// Identity: a Commit for the wrong version does not confirm the update (it stays INSTALLING); only a
// Commit for the awaited version completes it. Only the running BOS can Commit its own version, so this
// is the structural form of the "running == target" guarantee.
func TestCommit_WrongVersionDoesNotConfirm(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait() // parked awaiting Commit(v2)

		commit(t, c, "v3") // a different version: must not confirm v2
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_INSTALLING {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_INSTALLING)
		}

		commit(t, c, "v2")
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
	})
}

// The freshly-restarted BOS can Commit the target before the reconcile reaches awaitCommit (Apply
// restarts BOS, and BOS commits once, quickly). Because the commit generation is captured before Apply,
// that racing commit is newer than it and confirms the update - it must not be missed and roll back a
// healthy version. Apply always restarts, so no second commit would ever arrive to recover.
func TestCommit_RacingApplyConfirms(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	// Commit from within Apply, modelling the restarted BOS committing before the reconcile reads the
	// commit generation.
	fake.applyHook = func() {
		_, _ = c.Commit(context.Background(), &supervisorpb.CommitRequest{Version: "v2"})
	}

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait()

		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want COMPLETED (a commit racing Apply must confirm, not roll back)", got)
		}
	})
	assertCalls(t, fake, "Apply") // confirmed via the racing commit; never rolled back
}

// Once an update has settled, a stray heartbeat for a different version (no reconcile in flight) must
// not overwrite the settled committed version and change the terminal COMPLETED to FAILED.
func TestCommit_StrayHeartbeatDoesNotChangeTerminalStatus(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait()
		commit(t, c, "v2")
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Fatalf("state = %v, want COMPLETED", got)
		}

		commit(t, c, "v3") // a stray heartbeat after the goal settled
		synctest.Wait()
		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want COMPLETED to survive a stray heartbeat", got)
		}
		if got := st.GetVersion(); got != "v2" {
			t.Errorf("version = %q, want v2", got)
		}
	})
}

// A second InstallUpdate while a reconcile is in flight is rejected.
func TestInstallUpdate_RejectsConcurrent(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait() // reconcile is now parked awaiting Commit(v2); still in flight

		err := install2(c, "v3", url, sum)
		if got := status.Code(err); got != codes.FailedPrecondition {
			t.Errorf("status code = %v, want %v", got, codes.FailedPrecondition)
		}

		commit(t, c, "v2") // let the first reconcile finish
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
	})
}

// The terminal outcome is persisted under the staging dir, so a fresh Service (after a Supervisor
// restart) reports the last outcome rather than IDLE.
func TestInstallUpdate_PersistsTerminalStatus(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	c := testClient(t, newService(t, &fakeInstaller{}, dir, client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait()
		commit(t, c, "v2")
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
	})

	c2 := testClient(t, newService(t, &fakeInstaller{}, dir, client))
	st := getStatus(t, c2)
	if got := st.GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
		t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
	}
	if got := st.GetVersion(); got != "v2" {
		t.Errorf("version = %q, want %q", got, "v2")
	}
}

// The opaque deployment id supplied on InstallUpdate is echoed unchanged in GetUpdateStatus through the
// INSTALLING and COMPLETED states, and survives a reload from the state file (a Supervisor restart).
func TestInstallUpdate_EchoesDeploymentID(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	c := testClient(t, newService(t, &fakeInstaller{}, dir, client))

	synctest.Test(t, func(t *testing.T) {
		if _, err := c.InstallUpdate(context.Background(), &supervisorpb.InstallUpdateRequest{
			Version: "v2", DownloadUrl: url, Checksum: sum, DeploymentId: "42",
		}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := getStatus(t, c).GetDeploymentId(); got != "42" {
			t.Errorf("installing deployment id = %q, want %q", got, "42")
		}

		commit(t, c, "v2")
		synctest.Wait()
		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
		if got := st.GetDeploymentId(); got != "42" {
			t.Errorf("completed deployment id = %q, want %q", got, "42")
		}
	})

	// A fresh Service (Supervisor restart) loads the persisted record and still reports the id.
	c2 := testClient(t, newService(t, &fakeInstaller{}, dir, client))
	if got := getStatus(t, c2).GetDeploymentId(); got != "42" {
		t.Errorf("deployment id after reload = %q, want %q", got, "42")
	}
}

// Wait blocks until an in-flight reconcile reaches a terminal state, and honours its context deadline so
// shutdown can bound the wait.
func TestService_WaitBoundsInFlightInstall(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	svc := newService(t, fake, t.TempDir(), client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait() // reconcile parked awaiting Commit(v2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if err := svc.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait: want context.DeadlineExceeded, got %v", err)
		}
		cancel()

		commit(t, c, "v2") // let the reconcile finish
		synctest.Wait()
		if err := svc.Wait(context.Background()); err != nil {
			t.Errorf("Wait: want nil, got %v", err)
		}
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
	})
}

// Stop during the commit wait abandons the update rather than rolling it back: the goal is left
// unsettled so recovery re-drives it, instead of a healthy version being rolled back merely because
// shutdown raced its commit.
func TestStop_AbandonsAwaitForRecovery(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()

	// The Service is built inside the bubble so its lifecycle context (which the reconcile goroutine now
	// waits on) and its cancellation are bubble-durable under synctest.
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeInstaller{current: "v1", previous: "v1"}
		svc := newService(t, fake, dir, client)
		c := testClient(t, svc)

		commit(t, c, "v1") // baseline committed version
		mustInstall(t, c, "v2", url, sum)
		synctest.Wait() // applies v2, parks awaiting Commit(v2)
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_INSTALLING {
			t.Fatalf("state = %v, want INSTALLING before Stop", got)
		}

		start := time.Now()
		if err := svc.Stop(context.Background()); err != nil {
			t.Fatalf("Stop: %v", err)
		}
		// Stop must cancel the commit wait promptly, not let it run out the deadline. Under synctest a
		// wait that ignored cancellation would advance the fake clock by the full commit deadline here.
		if elapsed := time.Since(start); elapsed >= testDeadline {
			t.Errorf("Stop blocked for %v (>= commit deadline): the commit wait was not cancelled", elapsed)
		}

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_INSTALLING {
			t.Errorf("state = %v, want INSTALLING (abandoned, not settled)", got)
		}
		if st.GetFinishTime() != nil {
			t.Error("finish time set: the goal was settled instead of abandoned")
		}
		assertCalls(t, fake, "Apply") // never rolled back
	})

	// Recovery runs on a fresh Service (a cancelled lifecycle context cannot be reused): it re-drives the
	// goal left on disk and completes it once BOS commits.
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeInstaller{current: "v2", previous: "v1"} // :current already at v2 from the abandoned Apply
		svc := newService(t, fake, dir, client)
		c := testClient(t, svc)

		svc.Reconcile()
		synctest.Wait()
		commit(t, c, "v2")
		synctest.Wait()
		if got := getStatus(t, c).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want COMPLETED after recovery", got)
		}
	})
}

// Failed download: HTTP error and checksum mismatch both -> FAILED with nothing applied.
func TestInstallUpdate_FailedDownload(t *testing.T) {
	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		client := srv.Client()
		client.Transport.(*http.Transport).DisableKeepAlives = true

		fake := &fakeInstaller{}
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			mustInstall(t, c, "v2", srv.URL, zeroSum())
			synctest.Wait()

			st := getStatus(t, c)
			if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
				t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
			}
			if !strings.Contains(st.GetError(), "download") {
				t.Errorf("want download error, got %q", st.GetError())
			}
			assertCalls(t, fake)
		})
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		_, url, _, client := artefactServer(t)
		fake := &fakeInstaller{}
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			mustInstall(t, c, "v2", url, zeroSum())
			synctest.Wait()

			st := getStatus(t, c)
			if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
				t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
			}
			if !strings.Contains(st.GetError(), "checksum mismatch") {
				t.Errorf("want checksum mismatch error, got %q", st.GetError())
			}
			assertCalls(t, fake)
		})
	})
}

// The new version never commits within the deadline: the Supervisor rolls back to the previous image,
// the previous version commits on its way back up confirming the rollback, and the update is FAILED.
func TestInstallUpdate_RollbackOnNoCommit(t *testing.T) {
	t.Run("rollback succeeds", func(t *testing.T) {
		_, url, sum, client := artefactServer(t)
		fake := &fakeInstaller{current: "v1"} // the host is running v1 before the update
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			commit(t, c, "v1") // establish v1 as the running, good version (the rollback baseline)

			mustInstall(t, c, "v2", url, sum)
			elapseDeadline() // v2 never commits -> deadline -> rollback -> parks awaiting a fresh Commit(v1)
			synctest.Wait()

			commit(t, c, "v1") // the rolled-back-to version commits, confirming the rollback
			synctest.Wait()

			st := getStatus(t, c)
			if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
				t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
			}
			if got := st.GetVersion(); got != "v2" {
				t.Errorf("version = %q, want %q", got, "v2")
			}
			if st.GetError() == "" {
				t.Error("error should be set")
			}
			assertCalls(t, fake, "Apply", "Rollback")
			if got := fake.current; got != "v1" { // the host is left running the previous good version
				t.Errorf("current = %q, want %q", got, "v1")
			}
		})
	})

	t.Run("rollback also fails", func(t *testing.T) {
		_, url, sum, client := artefactServer(t)
		fake := &fakeInstaller{rollbackErr: errors.New("tag missing")}
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			commit(t, c, "v1")
			mustInstall(t, c, "v2", url, sum)
			elapseDeadline() // deadline elapses, rollback attempted and fails
			synctest.Wait()

			st := getStatus(t, c)
			if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
				t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
			}
			if !strings.Contains(st.GetError(), "rollback failed") {
				t.Errorf("want rollback failed error, got %q", st.GetError())
			}
			assertCalls(t, fake, "Apply", "Rollback")
		})
	})
}

// A first install that never commits has no previous image to roll back to: the rollback fails plainly
// and the update is FAILED.
func TestInstallUpdate_FirstInstallNoPrevious(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{} // no previous: Apply records previous="" so Rollback errors
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		mustInstall(t, c, "v1", url, sum)
		elapseDeadline() // no Commit(v1) -> deadline -> rollback finds no previous
		synctest.Wait()

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
		}
		if !strings.Contains(st.GetError(), "previous") {
			t.Errorf("want previous error, got %q", st.GetError())
		}
		assertCalls(t, fake, "Apply", "Rollback")
	})
}

// A Supervisor restart that finds an in-flight goal on disk (crashed mid-install) resumes the reconcile:
// it re-fetches and re-applies the artefact, then completes once the new version commits.
func TestReconcile_ResumesInterruptedInstall_Healthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, Checksum: sum}, Committed: "v1"})
	fake := &fakeInstaller{current: "v1", previous: "v1"}
	svc := newService(t, fake, dir, client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		svc.Reconcile()
		synctest.Wait() // re-downloads, re-applies, parks awaiting Commit(v2)

		commit(t, c, "v2")
		synctest.Wait()

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
		}
		if got := st.GetVersion(); got != "v2" {
			t.Errorf("version = %q, want %q", got, "v2")
		}
		assertCalls(t, fake, "Apply")
	})
}

// An interrupted install whose new version never becomes healthy is rolled back on resume and reported
// FAILED - the automatic recovery fired across a Supervisor crash.
func TestReconcile_ResumesInterruptedInstall_Unhealthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, Checksum: sum}, Committed: "v1"})
	fake := &fakeInstaller{current: "v1", previous: "v1"}
	svc := newService(t, fake, dir, client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		svc.Reconcile()
		elapseDeadline() // re-applies v2, no commit -> deadline -> rollback, parks awaiting Commit(v1)
		synctest.Wait()

		commit(t, c, "v1")
		synctest.Wait()

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
		}
		assertCalls(t, fake, "Apply", "Rollback")
	})
}

// A crash AFTER the tag-swap leaves the host already resolving to the new version. Resume re-applies it,
// which must be idempotent w.r.t. the rollback pointer: re-applying v2 must NOT overwrite the recorded
// previous version (v1) with v2. Otherwise, when v2 is unhealthy, the rollback returns to v2 and the
// node settles FAILED while running the bad version. With the pointer preserved, the rollback returns
// to v1.
func TestReconcile_ResumesAfterTagSwap_Unhealthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, Checksum: sum}, Committed: "v1"})
	// :current already points at v2 (the swap happened before the crash); :previous is the old good v1.
	fake := &fakeInstaller{current: "v2", previous: "v1"}
	svc := newService(t, fake, dir, client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		svc.Reconcile()
		elapseDeadline() // re-applies v2 (idempotent), no commit -> rollback to the preserved previous, v1
		synctest.Wait()

		commit(t, c, "v1")
		synctest.Wait()

		st := getStatus(t, c)
		if got := st.GetState(); got != supervisorpb.UpdateStatus_FAILED {
			t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_FAILED)
		}
		assertCalls(t, fake, "Apply", "Rollback")
		// rollback returned to the previous good version, not the bad v2, and did not overwrite the pointer.
		if got := fake.current; got != "v1" {
			t.Errorf("current = %q, want %q", got, "v1")
		}
		if got := fake.previous; got != "v1" {
			t.Errorf("previous = %q, want %q", got, "v1")
		}
	})
}

// A settled record on disk is left as-is: Reconcile does nothing and the last outcome is reported.
func TestReconcile_TerminalRecord_NoOp(t *testing.T) {
	dir := t.TempDir()
	finished := time.Unix(1000, 0).UTC()
	writeStateFile(t, dir, record{
		Target:     &target{Version: "v2"},
		Committed:  "v2",
		FinishTime: &finished,
	})
	fake := &fakeInstaller{}
	svc := newService(t, fake, dir, http.DefaultClient)

	svc.Reconcile()
	assertCalls(t, fake)
	if got := getStatus(t, testClient(t, svc)).GetState(); got != supervisorpb.UpdateStatus_COMPLETED {
		t.Errorf("state = %v, want %v", got, supervisorpb.UpdateStatus_COMPLETED)
	}
}
