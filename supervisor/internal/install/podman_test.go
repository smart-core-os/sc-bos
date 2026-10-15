package install

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

const (
	testRepo = "localhost/smartcore/bos"
	testUnit = "sc-bos"
)

func tagRef(suffix string) string { return testRepo + ":" + suffix }

// exists is the command that probes whether a tag is present.
func exists(suffix string) []string {
	return []string{"podman", "image", "exists", tagRef(suffix)}
}

// inspect is the command that resolves a tag to its image ID.
func inspect(suffix string) []string {
	return []string{"podman", "image", "inspect", "--format", "{{.Id}}", tagRef(suffix)}
}

func newTestInstaller(t *testing.T, steps ...step) *PodmanInstaller {
	t.Helper()
	return &PodmanInstaller{repo: testRepo, unit: testUnit, logger: zap.NewNop(), run: newFakeRunner(t, steps...)}
}

// A first install has no :current image: Apply loads the artefact, records :current as :previous (a no-op
// here, tolerated), repoints :current at the new version, and restarts the unit.
func TestApply_FirstInstall(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expect("", exists("v2")...),
		expect("sha-v2", inspect("v2")...),
		expectExit(1, exists("current")...),                               // absent on first install
		expectErr("podman", "tag", tagRef("current"), tagRef("previous")), // no :current to record, tolerated
		expect("", "podman", "tag", tagRef("v2"), tagRef("current")),
		expect("", "systemctl", "restart", testUnit),
	)

	if err := p.Apply(context.Background(), "/art.tar", "v2"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

// An upgrade where :current differs from the new version records the prior :current as the rollback
// pointer, then swaps :current onto the new version.
func TestApply_Upgrade(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expect("", exists("v2")...),
		expect("sha-v2", inspect("v2")...),
		expect("", exists("current")...),
		expect("sha-v1", inspect("current")...), // a different image
		expect("", "podman", "tag", tagRef("current"), tagRef("previous")),
		expect("", "podman", "tag", tagRef("v2"), tagRef("current")),
		expect("", "systemctl", "restart", testUnit),
	)

	if err := p.Apply(context.Background(), "/art.tar", "v2"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

// When :current already resolves to the new version (a resume after the tag-swap), Apply skips the swap
// so the rollback pointer is preserved, and only restarts.
func TestApply_AlreadyCurrentSkipsSwap(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expect("", exists("v2")...),
		expect("sha-v2", inspect("v2")...),
		expect("", exists("current")...),
		expect("sha-v2", inspect("current")...), // same image
		expect("", "systemctl", "restart", testUnit),
	)

	if err := p.Apply(context.Background(), "/art.tar", "v2"); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
}

// A failed artefact load aborts Apply before any tag is touched.
func TestApply_LoadFails(t *testing.T) {
	p := newTestInstaller(t,
		expectErr("podman", "load", "-i", "/art.tar"),
	)

	err := p.Apply(context.Background(), "/art.tar", "v2")
	if err == nil || !strings.Contains(err.Error(), "load artefact") {
		t.Fatalf("Apply() error = %v, want load artefact error", err)
	}
}

// A failure repointing :current onto the new version is surfaced.
func TestApply_RepointFails(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expect("", exists("v2")...),
		expect("sha-v2", inspect("v2")...),
		expect("", exists("current")...),
		expect("sha-v1", inspect("current")...),
		expect("", "podman", "tag", tagRef("current"), tagRef("previous")),
		expectErr("podman", "tag", tagRef("v2"), tagRef("current")),
	)

	err := p.Apply(context.Background(), "/art.tar", "v2")
	if err == nil || !strings.Contains(err.Error(), "repoint current tag") {
		t.Fatalf("Apply() error = %v, want repoint current tag error", err)
	}
}

// A tag lookup that fails on a present image aborts Apply before any tag is touched, rather than being
// taken as "not equal" and overwriting the rollback pointer with the running (bad) image.
func TestApply_TagLookupFails(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expect("", exists("v2")...),
		expectErr(inspect("v2")...), // present, but inspect fails transiently
	)

	err := p.Apply(context.Background(), "/art.tar", "v2")
	if err == nil || !strings.Contains(err.Error(), "compare version against current tag") {
		t.Fatalf("Apply() error = %v, want compare error", err)
	}
}

// A storage error (exit 125, not the "absent" 1) probing existence aborts Apply, rather than being taken
// as "absent" and letting the swap overwrite the rollback pointer.
func TestApply_ExistsStorageError(t *testing.T) {
	p := newTestInstaller(t,
		expect("", "podman", "load", "-i", "/art.tar"),
		expectExit(125, exists("v2")...),
	)

	err := p.Apply(context.Background(), "/art.tar", "v2")
	if err == nil || !strings.Contains(err.Error(), "compare version against current tag") {
		t.Fatalf("Apply() error = %v, want compare error", err)
	}
}

// Rollback repoints :current back at :previous and restarts.
func TestRollback_Success(t *testing.T) {
	p := newTestInstaller(t,
		expect("", exists("previous")...),
		expect("", "podman", "tag", tagRef("previous"), tagRef("current")),
		expect("", "systemctl", "restart", testUnit),
	)

	if err := p.Rollback(context.Background()); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
}

// A failed first install has no :previous: Rollback reports that plainly without touching any tag.
func TestRollback_NoPrevious(t *testing.T) {
	p := newTestInstaller(t,
		expectExit(1, exists("previous")...), // absent
	)

	err := p.Rollback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no previous image") {
		t.Fatalf("Rollback() error = %v, want no previous image error", err)
	}
}

// A storage error (exit 125) probing :previous is surfaced, not misreported as "no previous image".
func TestRollback_StorageError(t *testing.T) {
	p := newTestInstaller(t,
		expectExit(125, exists("previous")...),
	)

	err := p.Rollback(context.Background())
	if err == nil || !strings.Contains(err.Error(), "check previous image") {
		t.Fatalf("Rollback() error = %v, want check previous image error", err)
	}
}
