package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakePkg records dnf Install/Downgrade calls and returns scripted errors.
type fakePkg struct {
	installErr, downgradeErr error
	installs, downgrades     []string
}

func (f *fakePkg) Install(_ context.Context, rpmPath string) error {
	f.installs = append(f.installs, rpmPath)
	return f.installErr
}

func (f *fakePkg) Downgrade(_ context.Context, rpmPath string) error {
	f.downgrades = append(f.downgrades, rpmPath)
	return f.downgradeErr
}

// fakeConfirmer reports a version healthy iff it is in healthy.
type fakeConfirmer struct {
	healthy map[string]bool
	asked   []string
}

func (f *fakeConfirmer) Confirm(_ context.Context, want string) error {
	f.asked = append(f.asked, want)
	if f.healthy[want] {
		return nil
	}
	return errors.New("not healthy")
}

// rpmServer serves payload over HTTP and returns its URL and hex SHA-256.
func rpmServer(t *testing.T, payload []byte) (string, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	sum := sha256.Sum256(payload)
	return srv.URL, hex.EncodeToString(sum[:])
}

// newApplierEnv builds an Applier over a temp state dir with the given fakes. It returns the applier,
// its store, and the rpm store dir.
func newApplierEnv(t *testing.T, pkg PackageManager, confirm Confirmer) (*Applier, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	staging := filepath.Join(dir, "staging")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	rpmStore := filepath.Join(dir, "rpms")
	store := &Store{Path: filepath.Join(dir, "self-update.json")}
	a := NewApplier(store, staging, rpmStore, time.Minute, http.DefaultClient, true, pkg, confirm, nil)
	return a, store, rpmStore
}

func TestApplier_HappyPath(t *testing.T) {
	payload := []byte("a fake supervisor rpm")
	url, sum := rpmServer(t, payload)
	pkg := &fakePkg{}
	confirm := &fakeConfirmer{healthy: map[string]bool{"v2": true}}
	a, store, rpmStore := newApplierEnv(t, pkg, confirm)

	if err := store.Save(State{Target: "v2", URL: url, SHA256: sum, Previous: "v1", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(context.Background()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	st, _, _ := store.Load()
	if st.Phase != PhaseCompleted {
		t.Errorf("phase = %q, want completed (error: %q)", st.Phase, st.Error)
	}
	if len(pkg.installs) != 1 || len(pkg.downgrades) != 0 {
		t.Errorf("installs=%v downgrades=%v, want one install no downgrade", pkg.installs, pkg.downgrades)
	}
	// The new version's RPM is retained as the next rollback baseline.
	kept, err := os.ReadFile(filepath.Join(rpmStore, "v2.rpm"))
	if err != nil {
		t.Fatalf("read retained rpm: %v", err)
	}
	if string(kept) != string(payload) {
		t.Errorf("retained rpm contents = %q, want the downloaded payload", kept)
	}
}

func TestApplier_RollsBackWhenNewVersionUnhealthy(t *testing.T) {
	payload := []byte("a broken supervisor rpm")
	url, sum := rpmServer(t, payload)
	pkg := &fakePkg{}
	// The new version never becomes healthy; the previous one does after rollback.
	confirm := &fakeConfirmer{healthy: map[string]bool{"v1": true}}
	a, store, _ := newApplierEnv(t, pkg, confirm)

	// A previous RPM must exist for rollback.
	prevRPM := filepath.Join(t.TempDir(), "v1.rpm")
	if err := os.WriteFile(prevRPM, []byte("the good v1 rpm"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(State{Target: "v2bad", URL: url, SHA256: sum, Previous: "v1", PreviousRPMPath: prevRPM, Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(context.Background()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	st, _, _ := store.Load()
	if st.Phase != PhaseFailed {
		t.Errorf("phase = %q, want failed", st.Phase)
	}
	if len(pkg.installs) != 1 || len(pkg.downgrades) != 1 {
		t.Errorf("installs=%v downgrades=%v, want one install then one downgrade", pkg.installs, pkg.downgrades)
	}
	if pkg.downgrades[0] != prevRPM {
		t.Errorf("downgraded to %q, want previous rpm %q", pkg.downgrades[0], prevRPM)
	}
}

func TestApplier_FailsWhenUnhealthyAndNoPreviousRPM(t *testing.T) {
	payload := []byte("first ever supervisor rpm")
	url, sum := rpmServer(t, payload)
	pkg := &fakePkg{}
	confirm := &fakeConfirmer{} // nothing healthy
	a, store, _ := newApplierEnv(t, pkg, confirm)

	if err := store.Save(State{Target: "v2", URL: url, SHA256: sum, Previous: "v1", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(context.Background()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	st, _, _ := store.Load()
	if st.Phase != PhaseFailed {
		t.Errorf("phase = %q, want failed", st.Phase)
	}
	if len(pkg.downgrades) != 0 {
		t.Errorf("downgrades=%v, want none (no previous rpm)", pkg.downgrades)
	}
}

func TestApplier_InstallFailureSettlesFailed(t *testing.T) {
	payload := []byte("rpm that won't install")
	url, sum := rpmServer(t, payload)
	pkg := &fakePkg{installErr: errors.New("dnf boom")}
	confirm := &fakeConfirmer{healthy: map[string]bool{"v2": true}}
	a, store, _ := newApplierEnv(t, pkg, confirm)

	if err := store.Save(State{Target: "v2", URL: url, SHA256: sum, Previous: "v1", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(context.Background()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	st, _, _ := store.Load()
	if st.Phase != PhaseFailed {
		t.Errorf("phase = %q, want failed", st.Phase)
	}
	if len(confirm.asked) != 0 {
		t.Errorf("confirm asked %v, want none (install failed before confirm)", confirm.asked)
	}
	if len(pkg.downgrades) != 0 {
		t.Errorf("downgrades=%v, want none (install never changed the running version)", pkg.downgrades)
	}
}
