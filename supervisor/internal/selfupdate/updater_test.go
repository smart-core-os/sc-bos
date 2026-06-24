package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeLauncher records whether the applier was launched and can fail.
type fakeLauncher struct {
	launched int
	err      error
}

func (f *fakeLauncher) Launch(context.Context) error {
	f.launched++
	return f.err
}

func newUpdaterEnv(t *testing.T, version string) (*Updater, *Store, *fakeLauncher, string) {
	t.Helper()
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "self-update.json")}
	rpmStore := filepath.Join(dir, "rpms")
	launcher := &fakeLauncher{}
	return NewUpdater(version, store, launcher, rpmStore, nil), store, launcher, rpmStore
}

func TestUpdater_InstallRecordsIntentAndLaunches(t *testing.T) {
	u, store, launcher, rpmStore := newUpdaterEnv(t, "v1")

	// Seed the running version's RPM so it is recorded as the rollback baseline.
	if err := os.MkdirAll(rpmStore, 0o700); err != nil {
		t.Fatal(err)
	}
	prev := filepath.Join(rpmStore, "v1.rpm")
	if err := os.WriteFile(prev, []byte("v1"), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := u.Install(context.Background(), "v2", "http://x/v2.rpm", "abc")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if st.Phase != PhaseInstalling || st.Target != "v2" || st.Previous != "v1" {
		t.Errorf("state = %+v, want installing v2 from v1", st)
	}
	if st.PreviousRPMPath != prev {
		t.Errorf("previous rpm = %q, want %q", st.PreviousRPMPath, prev)
	}
	if launcher.launched != 1 {
		t.Errorf("launched = %d, want 1", launcher.launched)
	}
	// Intent is persisted before the applier runs.
	if saved, ok, _ := store.Load(); !ok || saved.Target != "v2" {
		t.Errorf("persisted state = %+v ok=%v, want target v2", saved, ok)
	}
}

func TestUpdater_InstallNoPreviousRPM(t *testing.T) {
	u, _, _, _ := newUpdaterEnv(t, "v1") // rpm store empty -> no rollback baseline

	st, err := u.Install(context.Background(), "v2", "http://x/v2.rpm", "abc")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if st.PreviousRPMPath != "" {
		t.Errorf("previous rpm = %q, want empty (none kept)", st.PreviousRPMPath)
	}
}

func TestUpdater_InstallRejectedWhenInProgress(t *testing.T) {
	u, store, launcher, _ := newUpdaterEnv(t, "v1")
	if err := store.Save(State{Target: "v2", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}

	_, err := u.Install(context.Background(), "v3", "http://x/v3.rpm", "abc")
	if !errors.Is(err, ErrInProgress) {
		t.Fatalf("err = %v, want ErrInProgress", err)
	}
	if launcher.launched != 0 {
		t.Errorf("launched = %d, want 0 (rejected)", launcher.launched)
	}
}

func TestUpdater_InstallSupersedesSettledUpdate(t *testing.T) {
	u, store, launcher, _ := newUpdaterEnv(t, "v2")
	if err := store.Save(State{Target: "v2", Phase: PhaseCompleted}); err != nil {
		t.Fatal(err)
	}

	if _, err := u.Install(context.Background(), "v3", "http://x/v3.rpm", "abc"); err != nil {
		t.Fatalf("Install over a settled update: %v", err)
	}
	if launcher.launched != 1 {
		t.Errorf("launched = %d, want 1", launcher.launched)
	}
}

func TestUpdater_ReconcileStartupSettlesCompletedWhenTargetRunning(t *testing.T) {
	u, store, _, _ := newUpdaterEnv(t, "v2") // running version == target
	if err := store.Save(State{Target: "v2", Previous: "v1", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}

	u.ReconcileStartup()

	st, _, _ := store.Load()
	if st.Phase != PhaseCompleted {
		t.Errorf("phase = %q, want completed", st.Phase)
	}
	if st.FinishTime == nil {
		t.Error("finish time not set")
	}
}

func TestUpdater_ReconcileStartupLeavesAmbiguousInFlight(t *testing.T) {
	u, store, _, _ := newUpdaterEnv(t, "v1") // still on the previous version: ambiguous, leave it
	if err := store.Save(State{Target: "v2", Previous: "v1", Phase: PhaseInstalling}); err != nil {
		t.Fatal(err)
	}

	u.ReconcileStartup()

	st, _, _ := store.Load()
	if st.Phase != PhaseInstalling {
		t.Errorf("phase = %q, want still installing (applier owns the rollback path)", st.Phase)
	}
}
