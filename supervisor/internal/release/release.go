// Package release validates the release artefacts that the FreeBSD installer installs: gzipped tarballs
// holding a manifest, the BOS binary and the Ops UI. It parses the manifest and checks an extracted release.
// See supervisor/docs/release-artefact.md for the format.
package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/smart-core-os/sc-bos/internal/util/ocitag"
)

const (
	// SchemaVersion is the only manifest schema version this package accepts.
	SchemaVersion = 1
	// ManifestName is the path of the manifest within the artefact.
	ManifestName = "manifest.json"
	// EntrypointPath is the path of the BOS binary within the artefact.
	EntrypointPath = "bin/sc-bos"
	// OpsUIPath is the path of the Ops UI folder within the artefact.
	OpsUIPath = "ui/ops"
)

// Manifest describes the contents of a release artefact.
type Manifest struct {
	// SchemaVersion is the manifest format version. It must be SchemaVersion.
	SchemaVersion int `json:"schemaVersion"`
	// Version is the BOS version the artefact holds. It must be a valid OCI tag.
	Version string `json:"version"`
	// OS is the GOOS the binary was built for.
	OS string `json:"os"`
	// Arch is the GOARCH the binary was built for.
	Arch string `json:"arch"`
	// Entrypoint is the path of the BOS binary. It must be EntrypointPath.
	Entrypoint string `json:"entrypoint"`
	// UI lists the UI folders the artefact carries.
	UI UI `json:"ui"`
}

// UI lists the UI folders a release artefact carries.
type UI struct {
	// Ops is the path of the Ops UI folder. It must be OpsUIPath.
	Ops string `json:"ops"`
}

// ParseManifest decodes a manifest from r and validates it. It returns an error for malformed JSON,
// unknown fields, trailing data, a read error from r, or a field value that schema version 1 does not allow.
func ParseManifest(r io.Reader) (Manifest, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	_, err := dec.Token()
	if _, isSyntax := errors.AsType[*json.SyntaxError](err); err == nil || isSyntax {
		return Manifest{}, errors.New("parse manifest: unexpected data after the manifest object")
	}
	if !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if m.SchemaVersion != SchemaVersion {
		return Manifest{}, fmt.Errorf("invalid manifest: schemaVersion %d is not supported, want %d", m.SchemaVersion, SchemaVersion)
	}
	if !ocitag.Valid(m.Version) {
		return Manifest{}, fmt.Errorf("invalid manifest: version %q is not a valid OCI tag", m.Version)
	}
	if m.OS == "" {
		return Manifest{}, errors.New("invalid manifest: os must not be empty")
	}
	if m.Arch == "" {
		return Manifest{}, errors.New("invalid manifest: arch must not be empty")
	}
	if m.Entrypoint != EntrypointPath {
		return Manifest{}, fmt.Errorf("invalid manifest: entrypoint %q must be %q", m.Entrypoint, EntrypointPath)
	}
	if m.UI.Ops != OpsUIPath {
		return Manifest{}, fmt.Errorf("invalid manifest: ui.ops %q must be %q", m.UI.Ops, OpsUIPath)
	}
	return m, nil
}

// Check reports an error if the manifest does not match the expected version, GOOS and GOARCH.
func (m Manifest) Check(version, goos, goarch string) error {
	if m.Version != version {
		return fmt.Errorf("release version %q does not match the requested version %q", m.Version, version)
	}
	if m.OS != goos || m.Arch != goarch {
		return fmt.Errorf("release platform %s/%s does not match the host platform %s/%s", m.OS, m.Arch, goos, goarch)
	}
	return nil
}

// CheckDir returns an error unless dir holds an extracted release for version, goos and goarch: a valid
// manifest that matches them, an executable regular file at EntrypointPath, and a regular file named
// index.html in OpsUIPath.
func CheckDir(dir *os.Root, version, goos, goarch string) error {
	f, err := dir.Open(ManifestName)
	if err != nil {
		return fmt.Errorf("open manifest: %w", err)
	}
	m, err := ParseManifest(f)
	_ = f.Close()
	if err != nil {
		return err
	}
	if err := m.Check(version, goos, goarch); err != nil {
		return err
	}

	bin, err := dir.Lstat(filepath.FromSlash(EntrypointPath))
	if err != nil {
		return fmt.Errorf("entrypoint: %w", err)
	}
	if !bin.Mode().IsRegular() || bin.Mode().Perm()&0o100 == 0 {
		return fmt.Errorf("entrypoint %s is not an executable regular file", EntrypointPath)
	}

	index := path.Join(OpsUIPath, "index.html")
	info, err := dir.Lstat(filepath.FromSlash(index))
	if err != nil {
		return fmt.Errorf("ops ui: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("ops ui %s is not a regular file", index)
	}
	return nil
}
