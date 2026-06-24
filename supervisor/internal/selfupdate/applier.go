package selfupdate

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/supervisor/internal/install"
)

// PackageManager installs and downgrades the Supervisor RPM. The production implementation shells out
// to dnf; installing a new RPM (or downgrading to a previous one) triggers the package's
// %systemd_postun_with_restart, which restarts the Supervisor onto that binary.
type PackageManager interface {
	Install(ctx context.Context, rpmPath string) error
	Downgrade(ctx context.Context, rpmPath string) error
}

// Confirmer blocks until the running Supervisor reports version want and is healthy, or ctx is done. The
// production implementation dials the Supervisor socket and polls GetSupervisorInfo; "healthy" means the
// socket serves and reports the wanted version, i.e. the new binary actually came up.
type Confirmer interface {
	Confirm(ctx context.Context, want string) error
}

// Applier runs the risky self-update window out-of-process: download + verify the RPM, install it, wait
// for the new version to come up healthy, and roll back to the last-known-good RPM if it does not.
type Applier struct {
	store       *Store
	stagingDir  string
	rpmStoreDir string
	deadline    time.Duration // how long to wait for a (re)installed version to come up healthy
	httpClient  *http.Client
	allowInsec  bool
	pkg         PackageManager
	confirm     Confirmer
	logger      *zap.Logger
}

// NewApplier returns an Applier. deadline bounds how long to wait for an installed version to confirm
// healthy before rolling back.
func NewApplier(store *Store, stagingDir, rpmStoreDir string, deadline time.Duration, httpClient *http.Client, allowInsecure bool, pkg PackageManager, confirm Confirmer, logger *zap.Logger) *Applier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Applier{
		store: store, stagingDir: stagingDir, rpmStoreDir: rpmStoreDir, deadline: deadline,
		httpClient: httpClient, allowInsec: allowInsecure, pkg: pkg, confirm: confirm, logger: logger,
	}
}

// Apply carries out the self-update recorded in the state file. It is the entry point of the applier
// process. It always settles the state to a terminal phase (COMPLETED or FAILED) unless it cannot read
// the state at all.
func (a *Applier) Apply(ctx context.Context) error {
	st, ok, err := a.store.Load()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no self-update state to apply")
	}
	if st.Phase.Terminal() {
		a.logger.Info("self-update already settled, nothing to apply", zap.String("phase", string(st.Phase)))
		return nil
	}

	// Download + verify the RPM, then give it a .rpm name so dnf recognises it as a local package.
	raw, err := install.DownloadAndVerify(ctx, a.httpClient, st.URL, st.SHA256, a.stagingDir, install.MaxArtefactBytes, a.allowInsec)
	if err != nil {
		return a.settleFailed(st, fmt.Sprintf("download: %v", err))
	}
	rpmPath := filepath.Join(a.stagingDir, RPMFileName(st.Target))
	if err := os.Rename(raw, rpmPath); err != nil {
		_ = os.Remove(raw)
		return a.settleFailed(st, fmt.Sprintf("stage rpm: %v", err))
	}
	defer func() { _ = os.Remove(rpmPath) }()

	a.logger.Info("installing supervisor rpm", zap.String("target", st.Target))
	if err := a.pkg.Install(ctx, rpmPath); err != nil {
		// The install transaction failed; the old version is still in place, so there is nothing to roll
		// back - settle failed.
		return a.settleFailed(st, fmt.Sprintf("install: %v", err))
	}

	if err := a.confirmWithin(ctx, st.Target); err == nil {
		// The new version is up and healthy. Keep its RPM as the new last-known-good for the next update's
		// rollback, and drop older ones.
		if err := a.promote(rpmPath, st.Target); err != nil {
			a.logger.Warn("retain last-known-good rpm", zap.Error(err))
		}
		return a.settleCompleted(st)
	}

	a.logger.Warn("new supervisor did not come up healthy, rolling back",
		zap.String("target", st.Target), zap.String("previous", st.Previous))
	return a.rollback(ctx, st)
}

// rollback reinstalls the previous RPM and confirms it comes back up, then settles FAILED.
func (a *Applier) rollback(ctx context.Context, st State) error {
	st.Phase = PhaseRollingBack
	if err := a.store.Save(st); err != nil {
		a.logger.Warn("persist rolling-back state", zap.Error(err))
	}
	if st.PreviousRPMPath == "" {
		return a.settleFailed(st, fmt.Sprintf("%s did not come up and no previous rpm to roll back to", st.Target))
	}
	if err := a.pkg.Downgrade(ctx, st.PreviousRPMPath); err != nil {
		return a.settleFailed(st, fmt.Sprintf("%s did not come up; rollback failed: %v", st.Target, err))
	}
	if err := a.confirmWithin(ctx, st.Previous); err != nil {
		return a.settleFailed(st, fmt.Sprintf("%s did not come up; rolled back but %s did not return: %v", st.Target, st.Previous, err))
	}
	return a.settleFailed(st, fmt.Sprintf("%s did not come up healthy; rolled back to %s", st.Target, st.Previous))
}

// confirmWithin waits up to the deadline for the running Supervisor to report want.
func (a *Applier) confirmWithin(ctx context.Context, want string) error {
	ctx, cancel := context.WithTimeout(ctx, a.deadline)
	defer cancel()
	return a.confirm.Confirm(ctx, want)
}

// promote copies the just-installed RPM into the last-known-good store as the rollback baseline for the
// next update, and removes any other RPMs there.
func (a *Applier) promote(rpmPath, version string) error {
	if err := os.MkdirAll(a.rpmStoreDir, 0o700); err != nil {
		return err
	}
	data, err := os.ReadFile(rpmPath)
	if err != nil {
		return err
	}
	dest := filepath.Join(a.rpmStoreDir, RPMFileName(version))
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return err
	}
	entries, err := os.ReadDir(a.rpmStoreDir)
	if err != nil {
		return err
	}
	keep := RPMFileName(version)
	for _, e := range entries {
		if e.Name() != keep {
			_ = os.Remove(filepath.Join(a.rpmStoreDir, e.Name()))
		}
	}
	return nil
}

func (a *Applier) settleCompleted(st State) error {
	now := time.Now()
	st.Phase = PhaseCompleted
	st.Error = ""
	st.FinishTime = &now
	a.logger.Info("self-update completed", zap.String("version", st.Target))
	return a.store.Save(st)
}

func (a *Applier) settleFailed(st State, reason string) error {
	now := time.Now()
	st.Phase = PhaseFailed
	st.Error = reason
	st.FinishTime = &now
	a.logger.Warn("self-update failed", zap.String("target", st.Target), zap.String("reason", reason))
	return a.store.Save(st)
}
