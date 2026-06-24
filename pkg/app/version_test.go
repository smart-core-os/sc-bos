package app

import "testing"

func TestEffectiveVersion_override(t *testing.T) {
	t.Setenv("BOS_VERSION_OVERRIDE", "v1.2.3-test")
	got := EffectiveVersion()
	if got != "v1.2.3-test" {
		t.Errorf("EffectiveVersion() = %q, want %q", got, "v1.2.3-test")
	}
}

func TestEffectiveVersion_buildVersion(t *testing.T) {
	// With no override, a link-time stamp is reported as-is.
	t.Setenv("BOS_VERSION_OVERRIDE", "")
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })
	buildVersion = "v2025.1.2"
	if got := EffectiveVersion(); got != "v2025.1.2" {
		t.Errorf("EffectiveVersion() = %q, want %q", got, "v2025.1.2")
	}
}

func TestEffectiveVersion_overrideBeatsBuildVersion(t *testing.T) {
	// The override wins over a link-time stamp, so a developer can force a version.
	t.Setenv("BOS_VERSION_OVERRIDE", "v1.2.3-test")
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })
	buildVersion = "v2025.1.2"
	if got := EffectiveVersion(); got != "v1.2.3-test" {
		t.Errorf("EffectiveVersion() = %q, want %q", got, "v1.2.3-test")
	}
}

func TestEffectiveVersion_noStamp(t *testing.T) {
	// With neither an override nor a link-time stamp, EffectiveVersion is "" - it must not fall back to
	// the main module version (which is "(devel)" for an unstamped build).
	t.Setenv("BOS_VERSION_OVERRIDE", "")
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })
	buildVersion = ""
	if got := EffectiveVersion(); got != "" {
		t.Errorf("EffectiveVersion() = %q, want empty", got)
	}
}
