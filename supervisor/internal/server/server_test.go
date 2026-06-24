package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// artefactServer starts an HTTPS server returning the given bytes (created outside any synctest bubble)
// and returns its URL, hex sha256, and a client that trusts it.
func artefactServer(t *testing.T) (data []byte, url, sum string, client *http.Client) {
	t.Helper()
	data = make([]byte, 2048)
	_, err := rand.Read(data)
	require.NoError(t, err)
	h := sha256.Sum256(data)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	client = srv.Client()
	client.Transport.(*http.Transport).DisableKeepAlives = true
	return data, srv.URL, hex.EncodeToString(h[:]), client
}

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
		Version: version, DownloadUrl: url, Sha256: sum,
	})
	return err
}

func commit(t *testing.T, c supervisorpb.SupervisorApiClient, version string) {
	t.Helper()
	_, err := c.Commit(context.Background(), &supervisorpb.CommitRequest{Version: version})
	require.NoError(t, err)
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
	require.NoError(t, err)
	return resp.GetStatus()
}

func writeStateFile(t *testing.T, dir string, rec record) {
	t.Helper()
	data, err := json.Marshal(rec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), data, 0o600))
}

// Boundary validation: missing fields and invalid image tags are rejected before any work starts.
func TestInstallUpdate_InvalidRequest(t *testing.T) {
	c := testClient(t, newService(t, &fakeInstaller{}, t.TempDir(), http.DefaultClient))

	_, err := c.InstallUpdate(context.Background(), &supervisorpb.InstallUpdateRequest{Version: "v2"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	err = install2(c, "bad/tag", "https://example.com/a.tar", hex.EncodeToString(make([]byte, sha256.Size)))
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	assert.Equal(t, supervisorpb.UpdateStatus_IDLE, getStatus(t, c).GetState())
}

// A Commit while no update is in flight is a routine heartbeat: it is accepted and records the running
// version, and the reported status stays IDLE.
func TestCommit_Idle_Heartbeat(t *testing.T) {
	c := testClient(t, newService(t, &fakeInstaller{}, t.TempDir(), http.DefaultClient))

	commit(t, c, "v1")
	assert.Equal(t, supervisorpb.UpdateStatus_IDLE, getStatus(t, c).GetState())
}

// Happy path: artefact downloads, Apply succeeds, BOS commits the new version within the deadline ->
// COMPLETED. Apply is the only installer call; the downloaded bytes reach it intact.
func TestInstallUpdate_HappyPath(t *testing.T) {
	data, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		require.NoError(t, install2(c, "v2", url, sum))
		synctest.Wait() // reconcile downloads, applies, then parks awaiting Commit(v2)

		assert.Equal(t, supervisorpb.UpdateStatus_INSTALLING, getStatus(t, c).GetState())

		commit(t, c, "v2")
		synctest.Wait()

		st := getStatus(t, c)
		assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, st.GetState())
		assert.Equal(t, "v2", st.GetVersion())
		assert.Empty(t, st.GetError())
		assert.NotNil(t, st.GetFinishTime())

		assert.Equal(t, []string{"Apply"}, fake.callLog())
		assert.Equal(t, "v2", fake.appliedVer)
		assert.True(t, bytes.Equal(fake.appliedBytes, data), "Apply received the downloaded artefact bytes")
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
		require.NoError(t, install2(c, "v2", url, sum))
		synctest.Wait() // parked awaiting Commit(v2)

		commit(t, c, "v3") // a different version: must not confirm v2
		synctest.Wait()
		assert.Equal(t, supervisorpb.UpdateStatus_INSTALLING, getStatus(t, c).GetState())

		commit(t, c, "v2")
		synctest.Wait()
		assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, getStatus(t, c).GetState())
	})
}

// A second InstallUpdate while a reconcile is in flight is rejected.
func TestInstallUpdate_RejectsConcurrent(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	c := testClient(t, newService(t, fake, t.TempDir(), client))

	synctest.Test(t, func(t *testing.T) {
		require.NoError(t, install2(c, "v2", url, sum))
		synctest.Wait() // reconcile is now parked awaiting Commit(v2); still in flight

		err := install2(c, "v3", url, sum)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))

		commit(t, c, "v2") // let the first reconcile finish
		synctest.Wait()
		assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, getStatus(t, c).GetState())
	})
}

// The terminal outcome is persisted under the staging dir, so a fresh Service (after a Supervisor
// restart) reports the last outcome rather than IDLE.
func TestInstallUpdate_PersistsTerminalStatus(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	c := testClient(t, newService(t, &fakeInstaller{}, dir, client))

	synctest.Test(t, func(t *testing.T) {
		require.NoError(t, install2(c, "v2", url, sum))
		synctest.Wait()
		commit(t, c, "v2")
		synctest.Wait()
		require.Equal(t, supervisorpb.UpdateStatus_COMPLETED, getStatus(t, c).GetState())
	})

	c2 := testClient(t, newService(t, &fakeInstaller{}, dir, client))
	st := getStatus(t, c2)
	assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, st.GetState())
	assert.Equal(t, "v2", st.GetVersion())
}

// Wait blocks until an in-flight reconcile reaches a terminal state, and honours its context deadline so
// shutdown can bound the wait.
func TestService_WaitBoundsInFlightInstall(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	fake := &fakeInstaller{}
	svc := newService(t, fake, t.TempDir(), client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		require.NoError(t, install2(c, "v2", url, sum))
		synctest.Wait() // reconcile parked awaiting Commit(v2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		assert.ErrorIs(t, svc.Wait(ctx), context.DeadlineExceeded)
		cancel()

		commit(t, c, "v2") // let the reconcile finish
		synctest.Wait()
		assert.NoError(t, svc.Wait(context.Background()))
		assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, getStatus(t, c).GetState())
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
			require.NoError(t, install2(c, "v2", srv.URL, hex.EncodeToString(make([]byte, sha256.Size))))
			synctest.Wait()

			st := getStatus(t, c)
			assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
			assert.Contains(t, st.GetError(), "download")
			assert.Empty(t, fake.callLog())
		})
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		_, url, _, client := artefactServer(t)
		fake := &fakeInstaller{}
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, install2(c, "v2", url, hex.EncodeToString(make([]byte, sha256.Size))))
			synctest.Wait()

			st := getStatus(t, c)
			assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
			assert.Contains(t, st.GetError(), "checksum mismatch")
			assert.Empty(t, fake.callLog())
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

			require.NoError(t, install2(c, "v2", url, sum))
			elapseDeadline() // v2 never commits -> deadline -> rollback -> parks awaiting a fresh Commit(v1)
			synctest.Wait()

			commit(t, c, "v1") // the rolled-back-to version commits, confirming the rollback
			synctest.Wait()

			st := getStatus(t, c)
			assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
			assert.Equal(t, "v2", st.GetVersion())
			assert.NotEmpty(t, st.GetError())
			assert.Equal(t, []string{"Apply", "Rollback"}, fake.callLog())
			assert.Equal(t, "v1", fake.current, "the host is left running the previous good version")
		})
	})

	t.Run("rollback also fails", func(t *testing.T) {
		_, url, sum, client := artefactServer(t)
		fake := &fakeInstaller{rollbackErr: errors.New("tag missing")}
		c := testClient(t, newService(t, fake, t.TempDir(), client))

		synctest.Test(t, func(t *testing.T) {
			commit(t, c, "v1")
			require.NoError(t, install2(c, "v2", url, sum))
			elapseDeadline() // deadline elapses, rollback attempted and fails
			synctest.Wait()

			st := getStatus(t, c)
			assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
			assert.Contains(t, st.GetError(), "rollback failed")
			assert.Equal(t, []string{"Apply", "Rollback"}, fake.callLog())
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
		require.NoError(t, install2(c, "v1", url, sum))
		elapseDeadline() // no Commit(v1) -> deadline -> rollback finds no previous
		synctest.Wait()

		st := getStatus(t, c)
		assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
		assert.Contains(t, st.GetError(), "previous")
		assert.Equal(t, []string{"Apply", "Rollback"}, fake.callLog())
	})
}

// A Supervisor restart that finds an in-flight goal on disk (crashed mid-install) resumes the reconcile:
// it re-fetches and re-applies the artefact, then completes once the new version commits.
func TestReconcile_ResumesInterruptedInstall_Healthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, SHA256: sum}, Committed: "v1"})
	fake := &fakeInstaller{current: "v1", previous: "v1"}
	svc := newService(t, fake, dir, client)
	c := testClient(t, svc)

	synctest.Test(t, func(t *testing.T) {
		svc.Reconcile()
		synctest.Wait() // re-downloads, re-applies, parks awaiting Commit(v2)

		commit(t, c, "v2")
		synctest.Wait()

		st := getStatus(t, c)
		assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, st.GetState())
		assert.Equal(t, "v2", st.GetVersion())
		assert.Equal(t, []string{"Apply"}, fake.callLog())
	})
}

// An interrupted install whose new version never becomes healthy is rolled back on resume and reported
// FAILED - the automatic recovery fired across a Supervisor crash.
func TestReconcile_ResumesInterruptedInstall_Unhealthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, SHA256: sum}, Committed: "v1"})
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
		assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
		assert.Equal(t, []string{"Apply", "Rollback"}, fake.callLog())
	})
}

// A crash AFTER the tag-swap leaves the host already resolving to the new version. Resume re-applies it,
// which must be idempotent w.r.t. the rollback pointer: re-applying v2 must NOT overwrite the recorded
// previous version (v1) with v2. Otherwise, when v2 is unhealthy, the rollback returns to v2 and the
// node settles FAILED while running the bad version (violating I3/I4). With the pointer preserved, the
// rollback returns to v1.
func TestReconcile_ResumesAfterTagSwap_Unhealthy(t *testing.T) {
	_, url, sum, client := artefactServer(t)
	dir := t.TempDir()
	writeStateFile(t, dir, record{Target: &target{Version: "v2", URL: url, SHA256: sum}, Committed: "v1"})
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
		assert.Equal(t, supervisorpb.UpdateStatus_FAILED, st.GetState())
		assert.Equal(t, []string{"Apply", "Rollback"}, fake.callLog())
		assert.Equal(t, "v1", fake.current, "rollback returned to the previous good version, not the bad v2")
		assert.Equal(t, "v1", fake.previous, "the rollback pointer was not overwritten with v2")
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
	assert.Empty(t, fake.callLog())
	assert.Equal(t, supervisorpb.UpdateStatus_COMPLETED, getStatus(t, testClient(t, svc)).GetState())
}
