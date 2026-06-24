package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// PodmanInstaller implements Installer by shelling out to podman and systemctl.
//
// It deliberately does not use podman auto-update: that has no per-container filter and would act on
// every auto-update-labelled container on the host. Instead it drives a recreate scoped to the BOS
// unit and owns rollback itself via the :previous tag. Confirmation that the new version is good is not
// its concern: BOS asserts that over the socket via Commit, so the installer only applies and rolls back.
type PodmanInstaller struct {
	repo   string // image repository whose tags are swapped, e.g. "localhost/smartcore/bos"
	unit   string // systemd unit and container name, e.g. "bos"
	logger *zap.Logger
}

// NewPodmanInstaller returns an Installer that swaps tags on repo and restarts unit.
func NewPodmanInstaller(repo, unit string, logger *zap.Logger) *PodmanInstaller {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PodmanInstaller{
		repo:   repo,
		unit:   unit,
		logger: logger,
	}
}

func (p *PodmanInstaller) Apply(ctx context.Context, artefactPath, version string) error {
	if _, err := p.podman(ctx, "load", "-i", artefactPath); err != nil {
		return fmt.Errorf("load artefact: %w", err)
	}
	// Move :current onto version, recording the prior :current as the rollback pointer. Skip the swap when
	// :current already resolves to version - a resume that re-applies an already-applied version - so the
	// rollback pointer keeps pointing at the previous good image instead of being overwritten with version.
	// This is what makes Apply idempotent (see Installer.Apply): without it, a resume after the tag-swap
	// would lose the rollback target and a later rollback would return to the bad version.
	current, err := p.currentResolvesTo(ctx, version)
	if err != nil {
		return err
	}
	if !current {
		// Record the rollback pointer. Tolerate a missing :current on the very first install.
		if _, err := p.podman(ctx, "tag", p.tag("current"), p.tag("previous")); err != nil {
			p.logger.Debug("no current tag to record as previous (first install?)", zap.Error(err))
		}
		if _, err := p.podman(ctx, "tag", p.tag(version), p.tag("current")); err != nil {
			return fmt.Errorf("repoint current tag: %w", err)
		}
	}
	return p.restart(ctx)
}

// currentResolvesTo reports whether the :current tag already points at version's image, by comparing
// image IDs. A missing :current (first install) is reported as false, so the tag-swap runs.
func (p *PodmanInstaller) currentResolvesTo(ctx context.Context, version string) (bool, error) {
	current, err := p.podman(ctx, "image", "inspect", "--format", "{{.Id}}", p.tag("current"))
	if err != nil {
		return false, nil // no :current yet (first install)
	}
	want, err := p.podman(ctx, "image", "inspect", "--format", "{{.Id}}", p.tag(version))
	if err != nil {
		return false, fmt.Errorf("inspect %s image: %w", version, err)
	}
	return strings.TrimSpace(current) == strings.TrimSpace(want), nil
}

func (p *PodmanInstaller) Rollback(ctx context.Context) error {
	// A failed first install has no :previous to return to; report that plainly rather than letting the
	// tag command fail with a confusing "image not known" error.
	if _, err := p.podman(ctx, "image", "exists", p.tag("previous")); err != nil {
		return errors.New("no previous image to roll back to")
	}
	if _, err := p.podman(ctx, "tag", p.tag("previous"), p.tag("current")); err != nil {
		return fmt.Errorf("repoint current tag to previous: %w", err)
	}
	return p.restart(ctx)
}

func (p *PodmanInstaller) restart(ctx context.Context) error {
	if _, err := p.systemctl(ctx, "restart", p.unit); err != nil {
		return fmt.Errorf("restart %s: %w", p.unit, err)
	}
	return nil
}

// tag returns the fully-qualified image reference repo:suffix.
func (p *PodmanInstaller) tag(suffix string) string {
	return p.repo + ":" + suffix
}

func (p *PodmanInstaller) podman(ctx context.Context, args ...string) (string, error) {
	return p.run(ctx, "podman", args...)
}

func (p *PodmanInstaller) systemctl(ctx context.Context, args ...string) (string, error) {
	return p.run(ctx, "systemctl", args...)
}

func (p *PodmanInstaller) run(ctx context.Context, name string, args ...string) (string, error) {
	p.logger.Debug("exec", zap.String("cmd", name), zap.Strings("args", args))
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return string(out), nil
}
