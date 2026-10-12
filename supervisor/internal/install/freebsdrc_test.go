//go:build unix && !aix

package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
)

const testService = "bos"

// restartCmd is the command the installer runs to restart the service.
var restartCmd = []string{"service", testService, "restart"}

// tarEntry is one entry of a test artefact.
type tarEntry struct {
	name     string
	typ      byte
	mode     int64
	body     string
	linkname string
}

// releaseEntries returns the entries of a valid artefact for version on this host's platform.
func releaseEntries(version string) []tarEntry {
	return releaseEntriesFor(version, runtime.GOOS, runtime.GOARCH)
}

func releaseEntriesFor(version, goos, goarch string) []tarEntry {
	manifest := fmt.Sprintf(`{"schemaVersion": 1, "version": %q, "os": %q, "arch": %q, "entrypoint": "bin/sc-bos", "ui": {"ops": "ui/ops"}}`, version, goos, goarch)
	return []tarEntry{
		{name: "manifest.json", typ: tar.TypeReg, mode: 0o644, body: manifest},
		{name: "bin/", typ: tar.TypeDir, mode: 0o755},
		{name: "bin/sc-bos", typ: tar.TypeReg, mode: 0o755, body: "binary " + version},
		{name: "ui/", typ: tar.TypeDir, mode: 0o755},
		{name: "ui/ops/", typ: tar.TypeDir, mode: 0o755},
		{name: "ui/ops/index.html", typ: tar.TypeReg, mode: 0o644, body: "<html></html>"},
	}
}

// writeArtefact writes entries as a gzipped tarball and returns its path.
func writeArtefact(t *testing.T, entries []tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: e.mode, Size: int64(len(e.body)), Linkname: e.linkname}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// newTestRCInstaller returns an installer with a fake command runner, using a releases root that
// doesn't exist yet.
func newTestRCInstaller(t *testing.T, steps ...step) (*FreeBSDRCInstaller, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "releases")
	return &FreeBSDRCInstaller{
		root:              root,
		service:           testService,
		logger:            zap.NewNop(),
		run:               newFakeRunner(t, steps...),
		maxExtractedBytes: MaxArtefactBytes,
	}, root
}

// mustApply applies a valid artefact for each version in turn.
func mustApply(t *testing.T, i *FreeBSDRCInstaller, versions ...string) {
	t.Helper()
	for _, v := range versions {
		if err := i.Apply(context.Background(), writeArtefact(t, releaseEntries(v)), v); err != nil {
			t.Fatalf("Apply(%s) error = %v", v, err)
		}
	}
}

// restarts returns n restart steps.
func restarts(n int) []step {
	return slices.Repeat([]step{expect("", restartCmd...)}, n)
}

// stageRelease creates a complete release folder for version directly, without the installer.
func stageRelease(t *testing.T, root, version string) {
	t.Helper()
	i, tmpRoot := newTestRCInstaller(t, restarts(1)...)
	mustApply(t, i, version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(tmpRoot, version), filepath.Join(root, version)); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, root, name, target string) {
	t.Helper()
	if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
}

// assertLink checks root/name is a link to want, or is absent if want is empty.
func assertLink(t *testing.T, root, name, want string) {
	t.Helper()
	got, err := os.Readlink(filepath.Join(root, name))
	if want == "" {
		if !os.IsNotExist(err) {
			t.Errorf("%s = %q (err %v), want no link", name, got, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if got != want {
		t.Errorf("%s -> %q, want %q", name, got, want)
	}
}

// assertEntries checks root holds exactly the named entries.
func assertEntries(t *testing.T, root string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	slices.Sort(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("entries of releases root (-want +got):\n%s", diff)
	}
}

// assertRunsVersion checks root/current/bin/sc-bos is the binary for version.
func assertRunsVersion(t *testing.T, root, version string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, currentLink, "bin", "sc-bos"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "binary "+version {
		t.Errorf("current binary is %q, want the binary for %s", got, version)
	}
}

func TestFreeBSDRC_FirstInstall(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(1)...)
	mustApply(t, i, "v1")

	assertEntries(t, root, ".lock", "current", "v1")
	assertLink(t, root, currentLink, "v1")
	assertLink(t, root, previousLink, "")
	assertRunsVersion(t, root, "v1")
	info, err := os.Stat(filepath.Join(root, "v1", "bin", "sc-bos"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binary mode %v, want owner execute", info.Mode())
	}
}

func TestFreeBSDRC_Upgrade(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1", "v2")

	assertEntries(t, root, ".lock", "current", "previous", "v1", "v2")
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
	assertRunsVersion(t, root, "v2")
}

func TestFreeBSDRC_ReapplyCurrent(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(3)...)
	mustApply(t, i, "v1", "v2", "v2")

	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// A crash during extraction leaves a temporary folder: the next Apply removes it and extracts again.
func TestFreeBSDRC_RecoverDuringExtraction(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1")
	partial := filepath.Join(root, ".tmp-v2-ABC")
	if err := os.MkdirAll(filepath.Join(partial, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	mustApply(t, i, "v2")

	assertEntries(t, root, ".lock", "current", "previous", "v1", "v2")
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
	assertRunsVersion(t, root, "v2")
}

// A crash after the staging rename leaves a complete release folder: the next Apply reuses it without
// reading the artefact, then swaps and restarts.
func TestFreeBSDRC_RecoverAfterStaging(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1")
	stageRelease(t, root, "v2")

	missing := filepath.Join(t.TempDir(), "missing.tar.gz")
	if err := i.Apply(context.Background(), missing, "v2"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
	assertRunsVersion(t, root, "v2")
}

// A crash after previous changed, before current: the next Apply sets previous again (to the same
// value) and swaps current.
func TestFreeBSDRC_RecoverAfterPreviousChanged(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1")
	stageRelease(t, root, "v2")
	symlink(t, root, previousLink, "v1")

	mustApply(t, i, "v2")

	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
	assertRunsVersion(t, root, "v2")
}

// A crash after current changed, before the restart: the next Apply skips the swap, so previous is
// kept, and restarts.
func TestFreeBSDRC_RecoverAfterCurrentChanged(t *testing.T) {
	_, root := newTestRCInstaller(t)
	stageRelease(t, root, "v1")
	stageRelease(t, root, "v2")
	symlink(t, root, currentLink, "v2")
	symlink(t, root, previousLink, "v1")
	i := &FreeBSDRCInstaller{root: root, service: testService, logger: zap.NewNop(), run: newFakeRunner(t, restarts(1)...), maxExtractedBytes: MaxArtefactBytes}

	mustApply(t, i, "v2")

	assertEntries(t, root, ".lock", "current", "previous", "v1", "v2")
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// An invalid release folder for the target version is replaced by a fresh extraction.
func TestFreeBSDRC_ReplacesInvalidReleaseFolder(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(1)...)
	if err := os.MkdirAll(filepath.Join(root, "v1", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	mustApply(t, i, "v1")

	assertLink(t, root, currentLink, "v1")
	assertRunsVersion(t, root, "v1")
}

// Every restart has a deadline, even when the caller's context has none.
func TestFreeBSDRC_RestartHasDeadline(t *testing.T) {
	i, _ := newTestRCInstaller(t, restarts(4)...)
	replay := i.run
	i.run = func(ctx context.Context, name string, args ...string) (string, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Errorf("%s %s run without a deadline", name, strings.Join(args, " "))
		}
		return replay(ctx, name, args...)
	}

	mustApply(t, i, "v1", "v2")
	if err := i.Rollback(context.Background()); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	applyLocal(t, i, "v3", true)
}

func TestFreeBSDRC_Rollback(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(3)...)
	mustApply(t, i, "v1", "v2")

	if err := i.Rollback(context.Background()); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	assertLink(t, root, currentLink, "v1")
	assertLink(t, root, previousLink, "v1")
	assertRunsVersion(t, root, "v1")
}

func TestFreeBSDRC_RollbackNoPrevious(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(1)...)
	mustApply(t, i, "v1")

	err := i.Rollback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no previous release") {
		t.Fatalf("Rollback() error = %v, want no previous release error", err)
	}
	assertLink(t, root, currentLink, "v1")
}

// Rollback refuses to point current at an invalid previous release.
func TestFreeBSDRC_RollbackInvalidPrevious(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1", "v2")
	if err := os.Remove(filepath.Join(root, "v1", "bin", "sc-bos")); err != nil {
		t.Fatal(err)
	}

	err := i.Rollback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "previous release v1") {
		t.Fatalf("Rollback() error = %v, want previous release error", err)
	}
	assertLink(t, root, currentLink, "v2")
}

// A failed restart during Rollback is returned, and current stays pointed at previous.
func TestFreeBSDRC_RollbackRestartFails(t *testing.T) {
	i, root := newTestRCInstaller(t, append(restarts(2), expectErr(restartCmd...))...)
	mustApply(t, i, "v1", "v2")

	err := i.Rollback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "restart "+testService) {
		t.Fatalf("Rollback() error = %v, want restart error", err)
	}
	assertLink(t, root, currentLink, "v1")
	assertLink(t, root, previousLink, "v1")
}

// A bad artefact fails Apply and leaves the host as it was.
func TestFreeBSDRC_BadArtefact(t *testing.T) {
	without := func(names ...string) []tarEntry {
		return slices.DeleteFunc(releaseEntries("v2"), func(e tarEntry) bool { return slices.Contains(names, e.name) })
	}
	with := func(extra ...tarEntry) []tarEntry {
		return append(releaseEntries("v2"), extra...)
	}
	nonExec := releaseEntries("v2")
	for k := range nonExec {
		if nonExec[k].name == "bin/sc-bos" {
			nonExec[k].mode = 0o644
		}
	}
	otherOS := "freebsd"
	if runtime.GOOS == otherOS {
		otherOS = "linux"
	}
	otherArch := "amd64"
	if runtime.GOARCH == otherArch {
		otherArch = "arm64"
	}

	tests := map[string]struct {
		entries []tarEntry
		maxSize int64
		want    string
	}{
		"wrong version":     {entries: releaseEntries("v3"), want: "does not match the requested version"},
		"wrong os":          {entries: releaseEntriesFor("v2", otherOS, runtime.GOARCH), want: "does not match the host platform"},
		"wrong arch":        {entries: releaseEntriesFor("v2", runtime.GOOS, otherArch), want: "does not match the host platform"},
		"missing binary":    {entries: without("bin/sc-bos"), want: "entrypoint"},
		"binary not exec":   {entries: nonExec, want: "not an executable regular file"},
		"missing ops ui":    {entries: without("ui/ops/", "ui/ops/index.html"), want: "ops ui"},
		"missing manifest":  {entries: without("manifest.json"), want: "manifest"},
		"symlink entry":     {entries: with(tarEntry{name: "ui/ops/link", typ: tar.TypeSymlink, linkname: "/etc/passwd"}), want: "unsupported tar entry type"},
		"path escapes root": {entries: with(tarEntry{name: "../escaped", typ: tar.TypeReg, mode: 0o644, body: "x"}), want: "escapes"},
		"over size limit":   {entries: with(tarEntry{name: "big", typ: tar.TypeReg, mode: 0o644, body: strings.Repeat("x", 64<<10)}), maxSize: 32 << 10, want: "size limit"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i, root := newTestRCInstaller(t, restarts(1)...)
			mustApply(t, i, "v1")
			if tt.maxSize > 0 {
				i.maxExtractedBytes = tt.maxSize
			}

			err := i.Apply(context.Background(), writeArtefact(t, tt.entries), "v2")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Apply() error = %v, want error containing %q", err, tt.want)
			}

			assertEntries(t, root, ".lock", "current", "v1")
			assertLink(t, root, currentLink, "v1")
			if _, err := os.Lstat(filepath.Join(filepath.Dir(root), "escaped")); !os.IsNotExist(err) {
				t.Errorf("file outside the releases root: err = %v, want not exist", err)
			}
		})
	}
}

// A version that can't name a release folder is rejected before the releases root is touched.
func TestFreeBSDRC_InvalidVersion(t *testing.T) {
	tests := map[string]string{
		"":         "not a valid OCI tag",
		".":        "not a valid OCI tag",
		"..":       "not a valid OCI tag",
		"a/b":      "not a valid OCI tag",
		"current":  "reserved",
		"previous": "reserved",
	}
	for v, want := range tests {
		i, root := newTestRCInstaller(t)
		err := i.Apply(context.Background(), writeArtefact(t, releaseEntries(v)), v)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Apply(%q) error = %v, want error containing %q", v, err, want)
		}
		if _, err := os.Lstat(root); !os.IsNotExist(err) {
			t.Errorf("Apply(%q) created the releases root: err = %v", v, err)
		}
	}
}

// A link whose target isn't a release folder name in the root fails Apply (and Rollback, for previous)
// without removing or changing anything.
func TestFreeBSDRC_BadLinkTarget(t *testing.T) {
	for _, link := range []string{currentLink, previousLink} {
		for name, target := range map[string]func(root string) string{
			"absolute":       func(root string) string { return filepath.Join(root, "v1") },
			"trailing slash": func(string) string { return "v1/" },
		} {
			t.Run(link+" "+name, func(t *testing.T) {
				i, root := newTestRCInstaller(t, restarts(2)...)
				mustApply(t, i, "v1", "v2")
				bad := target(root)
				if err := os.Remove(filepath.Join(root, link)); err != nil {
					t.Fatal(err)
				}
				symlink(t, root, link, bad)

				err := i.Apply(context.Background(), writeArtefact(t, releaseEntries("v3")), "v3")
				if err == nil || !strings.Contains(err.Error(), link+" link target") {
					t.Errorf("Apply() error = %v, want %s link target error", err, link)
				}
				if link == previousLink {
					err = i.Rollback(context.Background())
					if err == nil || !strings.Contains(err.Error(), "previous link target") {
						t.Errorf("Rollback() error = %v, want previous link target error", err)
					}
				}
				assertEntries(t, root, ".lock", "current", "previous", "v1", "v2")
				want := map[string]string{currentLink: "v2", previousLink: "v1"}
				want[link] = bad
				assertLink(t, root, currentLink, want[currentLink])
				assertLink(t, root, previousLink, want[previousLink])
			})
		}
	}
}

// A current or previous that isn't a symlink fails Apply with an error naming the link.
func TestFreeBSDRC_LinkNotSymlink(t *testing.T) {
	for _, link := range []string{currentLink, previousLink} {
		t.Run(link, func(t *testing.T) {
			i, root := newTestRCInstaller(t)
			if err := os.MkdirAll(filepath.Join(root, link), 0o755); err != nil {
				t.Fatal(err)
			}

			err := i.Apply(context.Background(), writeArtefact(t, releaseEntries("v1")), "v1")
			if err == nil || !strings.Contains(err.Error(), "read "+link+" link") {
				t.Errorf("Apply() error = %v, want read %s link error", err, link)
			}
			assertEntries(t, root, ".lock", link)
		})
	}
}

// When current's folder is invalid and the new artefact is bad too, current still resolves to the old
// folder.
func TestFreeBSDRC_BadArtefactKeepsInvalidCurrent(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(1)...)
	mustApply(t, i, "v1")
	if err := os.Remove(filepath.Join(root, "v1", "bin", "sc-bos")); err != nil {
		t.Fatal(err)
	}

	err := i.Apply(context.Background(), writeArtefact(t, releaseEntries("v9")), "v1")
	if err == nil || !strings.Contains(err.Error(), "does not match the requested version") {
		t.Fatalf("Apply() error = %v, want version mismatch error", err)
	}

	assertEntries(t, root, ".lock", "current", "v1")
	assertLink(t, root, currentLink, "v1")
	if _, err := os.Stat(filepath.Join(root, currentLink, "manifest.json")); err != nil {
		t.Errorf("current no longer resolves to the old release: %v", err)
	}
}

// Pruning removes old releases, temporary folders and stray files, but never the targets of current
// or previous.
func TestFreeBSDRC_Prune(t *testing.T) {
	_, root := newTestRCInstaller(t)
	for _, v := range []string{"v0", "v1", "old"} {
		stageRelease(t, root, v)
	}
	symlink(t, root, currentLink, "v1")
	symlink(t, root, previousLink, "v0")
	symlink(t, root, ".tmp-link-current-ABC", "old")
	if err := os.MkdirAll(filepath.Join(root, ".tmp-v2-ABC", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stray"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	i := &FreeBSDRCInstaller{root: root, service: testService, logger: zap.NewNop(), run: newFakeRunner(t, restarts(1)...), maxExtractedBytes: MaxArtefactBytes}

	mustApply(t, i, "v2")

	// v0 was previous's target when pruning ran, so it stays until the next Apply.
	assertEntries(t, root, ".lock", "current", "previous", "v0", "v1", "v2")
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// A failed restart is returned, and the links stay swapped.
func TestFreeBSDRC_RestartFails(t *testing.T) {
	i, root := newTestRCInstaller(t, expect("", restartCmd...), expectErr(restartCmd...))
	mustApply(t, i, "v1")

	err := i.Apply(context.Background(), writeArtefact(t, releaseEntries("v2")), "v2")
	if err == nil || !strings.Contains(err.Error(), "restart "+testService) {
		t.Fatalf("Apply() error = %v, want restart error", err)
	}
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// applyLocal runs ApplyLocal with a valid artefact for version and fails the test on error.
func applyLocal(t *testing.T, i *FreeBSDRCInstaller, version string, restart bool) LocalInstall {
	t.Helper()
	got, err := i.ApplyLocal(context.Background(), writeArtefact(t, releaseEntries(version)), restart)
	if err != nil {
		t.Fatalf("ApplyLocal(%s) error = %v", version, err)
	}
	return got
}

// ApplyLocal takes the version from the manifest and doesn't restart unless asked.
func TestFreeBSDRC_ApplyLocal_FirstInstall(t *testing.T) {
	i, root := newTestRCInstaller(t)
	got := applyLocal(t, i, "v1", false)

	if want := (LocalInstall{Version: "v1"}); got != want {
		t.Errorf("ApplyLocal() = %+v, want %+v", got, want)
	}
	assertEntries(t, root, ".lock", "current", "v1")
	assertLink(t, root, currentLink, "v1")
	assertRunsVersion(t, root, "v1")
}

func TestFreeBSDRC_ApplyLocal_Restart(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1")
	got := applyLocal(t, i, "v2", true)

	if want := (LocalInstall{Version: "v2", OldCurrent: "v1"}); got != want {
		t.Errorf("ApplyLocal() = %+v, want %+v", got, want)
	}
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// Installing the current release again changes nothing and reports it was already current.
func TestFreeBSDRC_ApplyLocal_AlreadyCurrent(t *testing.T) {
	i, root := newTestRCInstaller(t, restarts(2)...)
	mustApply(t, i, "v1", "v2")
	got := applyLocal(t, i, "v2", false)

	if want := (LocalInstall{Version: "v2", OldCurrent: "v2"}); got != want {
		t.Errorf("ApplyLocal() = %+v, want %+v", got, want)
	}
	assertEntries(t, root, ".lock", "current", "previous", "v1", "v2")
	assertLink(t, root, currentLink, "v2")
	assertLink(t, root, previousLink, "v1")
}

// The manifest may be named with a leading "./".
func TestFreeBSDRC_ApplyLocal_DotSlashNames(t *testing.T) {
	i, root := newTestRCInstaller(t)
	entries := releaseEntries("v1")
	for j := range entries {
		entries[j].name = "./" + entries[j].name
	}
	if _, err := i.ApplyLocal(context.Background(), writeArtefact(t, entries), false); err != nil {
		t.Fatalf("ApplyLocal() error = %v", err)
	}
	assertLink(t, root, currentLink, "v1")
}

// An artefact without a usable manifest, or for another platform, leaves the host unchanged.
func TestFreeBSDRC_ApplyLocal_BadArtefact(t *testing.T) {
	otherOS := "freebsd"
	if runtime.GOOS == otherOS {
		otherOS = "linux"
	}
	badManifest := tarEntry{name: "manifest.json", typ: tar.TypeReg, mode: 0o644, body: `{"schemaVersion": 2}`}
	// A valid manifest followed by enough whitespace to exceed the limit.
	bigManifest := releaseEntries("v2")[0]
	bigManifest.body += strings.Repeat(" ", maxManifestBytes)
	tests := []struct {
		name    string
		entries []tarEntry
		wantErr string
	}{
		{"no manifest", releaseEntries("v2")[1:], "no manifest.json"},
		{"invalid manifest", append([]tarEntry{badManifest}, releaseEntries("v2")[1:]...), "schemaVersion"},
		{"large manifest", append([]tarEntry{bigManifest}, releaseEntries("v2")[1:]...), "larger than 1 MiB"},
		{"wrong platform", releaseEntriesFor("v2", otherOS, runtime.GOARCH), "platform"},
		{"version current", releaseEntries(currentLink), "reserved"},
		{"version previous", releaseEntries(previousLink), "reserved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, root := newTestRCInstaller(t, restarts(1)...)
			mustApply(t, i, "v1")
			_, err := i.ApplyLocal(context.Background(), writeArtefact(t, tt.entries), true)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ApplyLocal() error = %v, want it to contain %q", err, tt.wantErr)
			}
			assertEntries(t, root, ".lock", "current", "v1")
			assertLink(t, root, currentLink, "v1")
		})
	}
}
