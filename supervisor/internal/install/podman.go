package install

import (
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

	// run executes an external command and returns its combined output. It is a field so tests can
	// substitute a fake that records the commands and returns prepared output; production uses execRun.
	run commandRunner
}

// commandRunner executes name with args and returns its combined stdout+stderr. A command that runs but
// exits non-zero yields an *exitCodeError carrying the exit code.
type commandRunner func(ctx context.Context, name string, args ...string) (string, error)

// NewPodmanInstaller returns an Installer that swaps tags on repo and restarts unit.
func NewPodmanInstaller(repo, unit string, logger *zap.Logger) *PodmanInstaller {
	if logger == nil {
		logger = zap.NewNop()
	}
	p := &PodmanInstaller{
		repo:   repo,
		unit:   unit,
		logger: logger,
	}
	p.run = p.execRun
	return p
}

func (p *PodmanInstaller) Apply(ctx context.Context, artefactPath, version string) error {
	if _, err := p.podman(ctx, "load", "-i", artefactPath); err != nil {
		return fmt.Errorf("load artefact: %w", err)
	}
	// Move :current onto version, recording the prior :current as the rollback pointer. Skip the swap when
	// version is already :current - can happen when resuming an interrupted operation - so the rollback
	// pointer keeps pointing at the previous good image instead of being overwritten with version. This is
	// required for idempotence: without it, a resume after the tag-swap would lose the rollback target and a later
	// rollback would return to the bad version.
	equal, err := p.compareTags(ctx, version, "current")
	if err != nil {
		return fmt.Errorf("compare version against current tag: %w", err)
	}
	if !equal {
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

// compareTags reports whether tags a and b resolve to the same image, by comparing image IDs. A tag that
// does not exist - e.g. a missing :current on the first install - never matches and is not an error. An
// error is returned only when resolving a tag that does exist fails, so a transient lookup failure is not
// silently taken as "not equal" - which would let Apply overwrite the rollback pointer with a bad image.
func (p *PodmanInstaller) compareTags(ctx context.Context, a, b string) (bool, error) {
	idA, existsA, err := p.imageID(ctx, a)
	if err != nil {
		return false, err
	}
	idB, existsB, err := p.imageID(ctx, b)
	if err != nil {
		return false, err
	}
	if !existsA || !existsB {
		return false, nil
	}
	return idA == idB, nil
}

// imageID returns the image ID that tag resolves to. exists is false, with no error, when the tag is
// absent (e.g. a missing :current on the first install); err is set only when the lookup of a present tag
// fails, letting callers tell "not present" apart from a genuine failure.
func (p *PodmanInstaller) imageID(ctx context.Context, tag string) (id string, exists bool, err error) {
	exists, err = p.imageExists(ctx, tag)
	if err != nil || !exists {
		return "", exists, err
	}
	out, err := p.podman(ctx, "image", "inspect", "--format", "{{.Id}}", p.tag(tag))
	if err != nil {
		return "", false, fmt.Errorf("inspect image %s: %w", tag, err)
	}
	return strings.TrimSpace(out), true, nil
}

// imageExists reports whether tag is present in local storage
func (p *PodmanInstaller) imageExists(ctx context.Context, tag string) (bool, error) {
	_, err := p.podman(ctx, "image", "exists", p.tag(tag))
	if err == nil {
		return true, nil
	}
	// exit code 1 means images does not exist; other exit codes (e.g. 125) mean there was an error accessing image info
	if codeErr, ok := errors.AsType[*exitCodeError](err); ok && codeErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check image %s exists: %w", tag, err)
}

func (p *PodmanInstaller) Rollback(ctx context.Context) error {
	// A failed first install has no :previous to return to; report that plainly rather than letting the
	// tag command fail with a confusing "image not known" error.
	previous, err := p.imageExists(ctx, "previous")
	if err != nil {
		return fmt.Errorf("check previous image: %w", err)
	}
	if !previous {
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

func (p *PodmanInstaller) execRun(ctx context.Context, name string, args ...string) (string, error) {
	p.logger.Debug("exec", zap.String("cmd", name), zap.Strings("args", args))
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	wrapped := fmt.Errorf("%s %s %w", name, strings.Join(args, " "), err)
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return string(out), &exitCodeError{code: exitErr.ExitCode(), err: wrapped}
	}
	return string(out), wrapped
}

type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }
func (e *exitCodeError) ExitCode() int { return e.code }
