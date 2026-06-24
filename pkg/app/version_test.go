package app

import "testing"

func TestEffectiveVersion_override(t *testing.T) {
	t.Setenv("BOS_VERSION_OVERRIDE", "v1.2.3-test")
	got := EffectiveVersion()
	if got != "v1.2.3-test" {
		t.Errorf("EffectiveVersion() = %q, want %q", got, "v1.2.3-test")
	}
}

func TestEffectiveVersion_buildInfo(t *testing.T) {
	// Unset any override so we exercise the build-info path.
	t.Setenv("BOS_VERSION_OVERRIDE", "")
	got := EffectiveVersion()
	// Under `go test`, build info is available but the main module version is typically "(devel)".
	// We only assert that the result equals what the build info reports (or empty when absent),
	// not a specific value, to keep the test portable across build modes.
	var want string
	if Version.BuildInfo != nil {
		want = Version.BuildInfo.Main.Version
	}
	if got != want {
		t.Errorf("EffectiveVersion() = %q, want %q", got, want)
	}
}
