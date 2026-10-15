package release

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/google/go-cmp/cmp"
)

const validManifest = `{
  "schemaVersion": 1,
  "version": "v2026.10.1",
  "os": "freebsd",
  "arch": "amd64",
  "entrypoint": "bin/sc-bos",
  "ui": { "ops": "ui/ops" }
}
`

func TestParseManifest_Valid(t *testing.T) {
	got, err := ParseManifest(strings.NewReader(validManifest))
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{
		SchemaVersion: 1,
		Version:       "v2026.10.1",
		OS:            "freebsd",
		Arch:          "amd64",
		Entrypoint:    EntrypointPath,
		UI:            UI{Ops: OpsUIPath},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("manifest mismatch (-want +got):\n%s", diff)
	}
}

func TestParseManifest_ReadError(t *testing.T) {
	readErr := errors.New("read failed")
	r := io.MultiReader(strings.NewReader(validManifest), iotest.ErrReader(readErr))
	if _, err := ParseManifest(r); !errors.Is(err, readErr) {
		t.Errorf("error %v, want %v", err, readErr)
	}
}

func TestParseManifest_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string // a substring of the expected error
	}{
		{name: "malformed json", content: `{not json`, wantErr: "parse manifest"},
		{name: "trailing data", content: validManifest + `{}`, wantErr: "after the manifest"},
		{name: "unknown field", content: withField("extra", `true`), wantErr: `unknown field "extra"`},
		{name: "wrong schemaVersion", content: withField("schemaVersion", `2`), wantErr: "schemaVersion 2 is not supported"},
		{name: "missing schemaVersion", content: without("schemaVersion"), wantErr: "schemaVersion 0 is not supported"},
		{name: "invalid version", content: withField("version", `"bad/version"`), wantErr: `version "bad/version" is not a valid OCI tag`},
		{name: "missing version", content: without("version"), wantErr: `version "" is not a valid OCI tag`},
		{name: "missing os", content: without("os"), wantErr: "os must not be empty"},
		{name: "missing arch", content: without("arch"), wantErr: "arch must not be empty"},
		{name: "wrong entrypoint", content: withField("entrypoint", `"bin/other"`), wantErr: `entrypoint "bin/other"`},
		{name: "missing ui", content: without("ui"), wantErr: `ui.ops ""`},
		{name: "wrong ui.ops", content: withField("ui", `{"ops": "ui/other"}`), wantErr: `ui.ops "ui/other"`},
		{name: "unknown ui field", content: withField("ui", `{"ops": "ui/ops", "admin": "ui/admin"}`), wantErr: `unknown field "admin"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest(strings.NewReader(tt.content))
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestManifest_Check(t *testing.T) {
	m := Manifest{Version: "v2026.10.1", OS: "freebsd", Arch: "arm64"}
	tests := []struct {
		name                  string
		version, goos, goarch string
		wantErr               string // empty for no error
	}{
		{name: "match", version: "v2026.10.1", goos: "freebsd", goarch: "arm64"},
		{name: "version mismatch", version: "v2026.11.0", goos: "freebsd", goarch: "arm64", wantErr: "requested version"},
		{name: "os mismatch", version: "v2026.10.1", goos: "linux", goarch: "arm64", wantErr: "platform"},
		{name: "arch mismatch", version: "v2026.10.1", goos: "freebsd", goarch: "amd64", wantErr: "platform"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.Check(tt.version, tt.goos, tt.goarch)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("want nil error, got %v", err)
			case tt.wantErr != "" && err == nil:
				t.Error("want error, got nil")
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

// withField returns validManifest with the named top-level field set to the raw JSON value, adding it if absent.
func withField(name, value string) string {
	fields := manifestFields()
	fields[name] = value
	return joinFields(fields)
}

// without returns validManifest with the named top-level field removed.
func without(name string) string {
	fields := manifestFields()
	delete(fields, name)
	return joinFields(fields)
}

func manifestFields() map[string]string {
	return map[string]string{
		"schemaVersion": `1`,
		"version":       `"v2026.10.1"`,
		"os":            `"freebsd"`,
		"arch":          `"amd64"`,
		"entrypoint":    `"bin/sc-bos"`,
		"ui":            `{"ops": "ui/ops"}`,
	}
}

func joinFields(fields map[string]string) string {
	var parts []string
	for k, v := range fields {
		parts = append(parts, `"`+k+`": `+v)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
