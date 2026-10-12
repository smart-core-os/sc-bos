package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"slices"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/ocitag"
	"github.com/smart-core-os/sc-bos/internal/util/tarutil"
	"github.com/smart-core-os/sc-bos/supervisor/internal/release"
)

const (
	lockName     = ".lock"
	currentLink  = "current"
	previousLink = "previous"

	// maxManifestBytes is the size limit for an artefact's manifest.
	maxManifestBytes = 1 << 20

	// restartTimeout limits how long a service restart can take. rc(8) waits for the old process to
	// exit without a limit, so a service that doesn't stop would otherwise block an install while it
	// holds the lock.
	restartTimeout = 5 * time.Minute
)

// FreeBSDRCInstaller implements Installer for hosts that run BOS as a native rc(8) service.
//
// Each release is extracted into its own folder under root. The relative symlinks root/current and
// root/previous select the release the service runs and the rollback target. Apply and Rollback hold an
// exclusive lock on root/.lock, so other processes using the same lock can't change root at the same time.
type FreeBSDRCInstaller struct {
	root    string // folder holding the release folders and links, e.g. "/usr/local/sc-bos/releases"
	service string // rc(8) service name, e.g. "sc_bos"
	logger  *zap.Logger

	// run executes an external command and returns its combined output. It is a field so tests can
	// substitute a fake that records the commands and returns prepared output; production uses execRunner.
	run commandRunner
	// maxExtractedBytes limits the extracted size of an artefact. Tests lower it.
	maxExtractedBytes int64
}

// NewFreeBSDRCInstaller returns an Installer that installs releases under root and restarts service.
func NewFreeBSDRCInstaller(root, service string, logger *zap.Logger) *FreeBSDRCInstaller {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FreeBSDRCInstaller{
		root:              root,
		service:           service,
		logger:            logger,
		run:               execRunner(logger),
		maxExtractedBytes: MaxArtefactBytes,
	}
}

// Apply installs the release artefact at artefactPath as version, points current at it, and restarts
// the service. A release folder for version that already passes validation is reused without reading
// artefactPath. When current already points at version, previous is left unchanged.
func (i *FreeBSDRCInstaller) Apply(ctx context.Context, artefactPath, version string) error {
	if err := checkVersion(version); err != nil {
		return err
	}
	root, unlock, err := i.openLocked(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	if _, err := i.install(root, artefactPath, version); err != nil {
		return err
	}
	return i.restart(ctx)
}

// LocalInstall describes what ApplyLocal did.
type LocalInstall struct {
	// Version is the installed release version, read from the artefact's manifest.
	Version string
	// OldCurrent is the version current pointed at before, or empty if there was none. It equals
	// Version if the release was already current.
	OldCurrent string
}

// ApplyLocal installs the release artefact at artefactPath as Apply does, taking the version from the
// artefact's manifest, and restarts the service only if restart is true.
//
// ctx bounds only the wait for the lock. If another process holds the lock until ctx is done,
// ApplyLocal returns an error wrapping ctx.Err() without changing any release or link. Once it holds
// the lock, it finishes regardless of ctx.
func (i *FreeBSDRCInstaller) ApplyLocal(ctx context.Context, artefactPath string, restart bool) (LocalInstall, error) {
	m, err := readArtefactManifest(artefactPath)
	if err != nil {
		return LocalInstall{}, err
	}
	if err := checkVersion(m.Version); err != nil {
		return LocalInstall{}, err
	}
	root, unlock, err := i.openLocked(ctx)
	if err != nil {
		return LocalInstall{}, err
	}
	defer unlock()

	oldCurrent, err := i.install(root, artefactPath, m.Version)
	if err != nil {
		return LocalInstall{}, err
	}
	res := LocalInstall{Version: m.Version, OldCurrent: oldCurrent}
	if restart {
		return res, i.restart(context.WithoutCancel(ctx))
	}
	return res, nil
}

// readArtefactManifest reads and validates the manifest in the release artefact at artefactPath, without
// extracting anything else.
func readArtefactManifest(artefactPath string) (release.Manifest, error) {
	f, err := os.Open(artefactPath)
	if err != nil {
		return release.Manifest{}, fmt.Errorf("open artefact: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return release.Manifest{}, fmt.Errorf("read artefact: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return release.Manifest{}, fmt.Errorf("artefact has no %s", release.ManifestName)
		}
		if err != nil {
			return release.Manifest{}, fmt.Errorf("read artefact: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && path.Clean(hdr.Name) == release.ManifestName {
			if hdr.Size > maxManifestBytes {
				return release.Manifest{}, fmt.Errorf("%s is larger than 1 MiB", release.ManifestName)
			}
			return release.ParseManifest(tr)
		}
	}
}

// Rollback points current at previous's target and restarts the service. previous is left unchanged.
// It returns an error if there is no previous release.
func (i *FreeBSDRCInstaller) Rollback(ctx context.Context) error {
	root, unlock, err := i.openLocked(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	target, ok, err := readLink(root, previousLink)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no previous release to roll back to")
	}
	if err := checkRelease(root, target, target); err != nil {
		return fmt.Errorf("previous release %s: %w", target, err)
	}
	if err := setLink(root, currentLink, target); err != nil {
		return err
	}
	if err := syncDir(root); err != nil {
		return err
	}
	return i.restart(ctx)
}

// openLocked creates the releases root if needed, opens it, and takes the exclusive lock on its lock
// file, waiting until the lock is free or ctx is done. Call unlock to release the lock and close the root.
func (i *FreeBSDRCInstaller) openLocked(ctx context.Context) (root *os.Root, unlock func(), err error) {
	if err := os.MkdirAll(i.root, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create releases root: %w", err)
	}
	root, err = os.OpenRoot(i.root)
	if err != nil {
		return nil, nil, fmt.Errorf("open releases root: %w", err)
	}
	lockFile, err := root.OpenFile(lockName, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		_ = root.Close()
		return nil, nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := lock(ctx, lockFile); err != nil {
		_ = lockFile.Close()
		_ = root.Close()
		return nil, nil, fmt.Errorf("lock releases root: %w", err)
	}
	return root, func() {
		_ = lockFile.Close() // closing the file releases the lock
		_ = root.Close()
	}, nil
}

// install prunes root, stages the release for version from artefactPath unless a valid one is already
// present, and points current at it, recording the old current as previous. It returns the version current
// pointed at before, or "" if there was none. The caller holds the lock and has checked version with
// checkVersion.
func (i *FreeBSDRCInstaller) install(root *os.Root, artefactPath, version string) (oldCurrent string, err error) {
	current, hasCurrent, err := readLink(root, currentLink)
	if err != nil {
		return "", err
	}
	previous, _, err := readLink(root, previousLink)
	if err != nil {
		return "", err
	}
	if err := i.prune(root, current, previous, version); err != nil {
		return "", err
	}
	if err := i.stage(root, artefactPath, version); err != nil {
		return "", err
	}
	if current != version {
		if hasCurrent {
			if err := setLink(root, previousLink, current); err != nil {
				return "", err
			}
		}
		if err := setLink(root, currentLink, version); err != nil {
			return "", err
		}
		i.logger.Info("switched current release", zap.String("current", version), zap.String("previous", current))
	}
	return current, syncDir(root)
}

// prune removes every entry of root other than the lock file, the links, and the folders named keep.
func (i *FreeBSDRCInstaller) prune(root *os.Root, keep ...string) error {
	dir, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open releases root: %w", err)
	}
	entries, err := dir.ReadDir(-1)
	_ = dir.Close()
	if err != nil {
		return fmt.Errorf("list releases root: %w", err)
	}
	keep = append(keep, lockName, currentLink, previousLink)
	for _, e := range entries {
		if slices.Contains(keep, e.Name()) {
			continue
		}
		i.logger.Info("removing unused release entry", zap.String("name", e.Name()))
		if err := root.RemoveAll(e.Name()); err != nil {
			return fmt.Errorf("remove %s: %w", e.Name(), err)
		}
	}
	return nil
}

// stage makes sure root/version holds a valid release for version, extracting artefactPath if needed.
// The folder appears complete or not at all, and an existing folder is only replaced once the new one
// has passed the checks.
func (i *FreeBSDRCInstaller) stage(root *os.Root, artefactPath, version string) error {
	checkErr := checkRelease(root, version, version)
	if checkErr == nil {
		i.logger.Debug("reusing installed release", zap.String("version", version))
		return nil
	}

	tmp := fmt.Sprintf(".tmp-%s-%s", version, rand.Text())
	if err := root.Mkdir(tmp, 0o755); err != nil {
		return fmt.Errorf("create staging folder: %w", err)
	}
	if err := i.extract(root, tmp, artefactPath, version); err != nil {
		return errors.Join(err, root.RemoveAll(tmp))
	}

	var old string
	if _, err := root.Lstat(version); err == nil {
		i.logger.Warn("replacing invalid release folder", zap.String("version", version), zap.Error(checkErr))
		old = fmt.Sprintf(".tmp-old-%s-%s", version, rand.Text())
		if err := root.Rename(version, old); err != nil {
			return errors.Join(fmt.Errorf("move invalid release aside: %w", err), root.RemoveAll(tmp))
		}
	}
	if err := root.Rename(tmp, version); err != nil {
		return errors.Join(fmt.Errorf("move release into place: %w", err), root.RemoveAll(tmp))
	}
	if old != "" {
		if err := root.RemoveAll(old); err != nil {
			i.logger.Warn("failed to remove invalid release folder", zap.String("name", old), zap.Error(err))
		}
	}
	return nil
}

// extract extracts the artefact at artefactPath into root/dir, checks it holds a release for version,
// and flushes the extracted files to disk.
func (i *FreeBSDRCInstaller) extract(root *os.Root, dir, artefactPath, version string) error {
	f, err := os.Open(artefactPath)
	if err != nil {
		return fmt.Errorf("open artefact: %w", err)
	}
	defer f.Close()
	dst, err := root.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open staging folder: %w", err)
	}
	defer dst.Close()
	if err := tarutil.ExtractTarGZ(f, dst, i.maxExtractedBytes); err != nil {
		return fmt.Errorf("extract artefact: %w", err)
	}
	if err := checkRelease(root, dir, version); err != nil {
		return err
	}
	return syncTree(dst)
}

// syncTree flushes every file and folder in root to disk.
func syncTree(root *os.Root) error {
	err := fs.WalkDir(root.FS(), ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		f, err := root.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		return f.Sync()
	})
	if err != nil {
		return fmt.Errorf("sync release: %w", err)
	}
	return nil
}

// checkRelease returns an error unless root/dir holds a release for version on this host's platform.
func checkRelease(root *os.Root, dir, version string) error {
	r, err := root.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer r.Close()
	return release.CheckDir(r, version, runtime.GOOS, runtime.GOARCH)
}

func (i *FreeBSDRCInstaller) restart(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, restartTimeout)
	defer cancel()
	if _, err := i.run(ctx, "service", i.service, "restart"); err != nil {
		return fmt.Errorf("restart %s: %w", i.service, err)
	}
	return nil
}

// readLink returns the target of the symlink root/name. ok is false, with no error, if the link
// doesn't exist. It returns an error if the target is not the name of a release folder in root.
func readLink(root *os.Root, name string) (target string, ok bool, err error) {
	target, err = root.Readlink(name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s link: %w", name, err)
	}
	if checkVersion(target) != nil {
		return "", false, fmt.Errorf("%s link target %q is not a release folder in the releases root", name, target)
	}
	return target, true, nil
}

// checkVersion returns an error if version can't name a release folder: it must be a valid OCI tag and
// not the name of a link.
func checkVersion(version string) error {
	if !ocitag.Valid(version) {
		return fmt.Errorf("version %q is not a valid OCI tag", version)
	}
	if version == currentLink || version == previousLink {
		return fmt.Errorf("version %q is reserved", version)
	}
	return nil
}

// setLink atomically points the symlink root/name at target, replacing any existing link.
func setLink(root *os.Root, name, target string) error {
	tmp := fmt.Sprintf(".tmp-link-%s-%s", name, rand.Text())
	if err := root.Symlink(target, tmp); err != nil {
		return fmt.Errorf("create %s link: %w", name, err)
	}
	if err := root.Rename(tmp, name); err != nil {
		return errors.Join(fmt.Errorf("replace %s link: %w", name, err), root.Remove(tmp))
	}
	return nil
}

// syncDir flushes root's directory entries to disk.
func syncDir(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open releases root: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync releases root: %w", err)
	}
	return nil
}
