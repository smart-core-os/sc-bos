package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// ErrInProgress is returned by Install when a self-update is already underway. The caller maps it to a
// benign gRPC FailedPrecondition, mirroring the BOS update flow.
var ErrInProgress = errors.New("a self-update is already in progress")

// Launcher starts the out-of-process applier. The production implementation runs the applier as a
// transient systemd unit so it survives the Supervisor restart the RPM install triggers.
type Launcher interface {
	Launch(ctx context.Context) error
}

// Updater is the in-process side of self-update: it accepts a request, records the intent, and launches
// the applier. It never installs anything itself (the applier does, out of process), so accepting a
// request is fast and the Supervisor is free to be restarted immediately afterwards.
type Updater struct {
	version     string // the Supervisor's own running version (the rollback baseline for a new update)
	store       *Store
	launcher    Launcher
	rpmStoreDir string // holds the last-known-good RPM(s), kept for offline rollback
	logger      *zap.Logger
}

// NewUpdater returns an Updater. version is the running Supervisor's version; store holds the durable
// self-update state; launcher starts the applier; rpmStoreDir holds last-known-good RPMs.
func NewUpdater(version string, store *Store, launcher Launcher, rpmStoreDir string, logger *zap.Logger) *Updater {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Updater{version: version, store: store, launcher: launcher, rpmStoreDir: rpmStoreDir, logger: logger}
}

// RPMFileName is the file name a version's RPM is stored under in the last-known-good store.
func RPMFileName(version string) string { return version + ".rpm" }

// Install records the intent to self-update to version and launches the applier. It returns the
// accepted (PhaseInstalling) state. It returns ErrInProgress if a non-terminal self-update already
// exists.
func (u *Updater) Install(ctx context.Context, version, url, sha256 string) (State, error) {
	if cur, ok, err := u.store.Load(); err != nil {
		return State{}, err
	} else if ok && !cur.Phase.Terminal() {
		return State{}, ErrInProgress
	}

	// Keep the currently-running version's RPM as the rollback target, if we have it. Absent on the very
	// first self-update (nothing was installed by us yet); rollback is then impossible and the applier
	// settles FAILED without it. The demo seeds it so rollback can be exercised.
	previousRPM := filepath.Join(u.rpmStoreDir, RPMFileName(u.version))
	if _, err := os.Stat(previousRPM); err != nil {
		previousRPM = ""
	}

	st := State{
		Target:          version,
		URL:             url,
		SHA256:          sha256,
		Previous:        u.version,
		PreviousRPMPath: previousRPM,
		Phase:           PhaseInstalling,
		StartTime:       time.Now(),
	}
	if err := u.store.Save(st); err != nil {
		return State{}, err
	}

	u.logger.Info("self-update accepted, launching applier",
		zap.String("target", version), zap.String("previous", u.version))
	if err := u.launcher.Launch(ctx); err != nil {
		// Mark the intent failed so a stale "installing" doesn't block future attempts.
		now := time.Now()
		st.Phase = PhaseFailed
		st.Error = fmt.Sprintf("launch applier: %v", err)
		st.FinishTime = &now
		_ = u.store.Save(st)
		return State{}, fmt.Errorf("launch applier: %w", err)
	}
	return st, nil
}

// ReconcileStartup is a backstop for an applier that restarted the Supervisor onto the new version but
// died before recording COMPLETED: if a non-terminal self-update names the version now running as its
// target, settle it COMPLETED. The rollback path is owned by the (long-lived) applier, and the
// "still installing on the old version" case is ambiguous, so neither is settled here. Call once at
// startup, before serving.
func (u *Updater) ReconcileStartup() {
	st, ok, err := u.store.Load()
	if err != nil || !ok || st.Phase.Terminal() {
		return
	}
	if u.version != "" && u.version == st.Target {
		now := time.Now()
		st.Phase = PhaseCompleted
		st.Error = ""
		st.FinishTime = &now
		u.logger.Info("self-update target is running; settling completed", zap.String("version", st.Target))
		if err := u.store.Save(st); err != nil {
			u.logger.Warn("settle self-update completed", zap.Error(err))
		}
	}
}

// Version returns the running Supervisor's own version.
func (u *Updater) Version() string { return u.version }

// LastUpdate returns the most recent self-update state, or false if none has been recorded.
func (u *Updater) LastUpdate() (State, bool) {
	st, ok, err := u.store.Load()
	if err != nil {
		u.logger.Warn("read self-update state", zap.Error(err))
		return State{}, false
	}
	return st, ok
}
