//go:build unix

package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDir(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(t *testing.T, dir string)
		wantErr string // a substring of the expected error, or "" for no error
	}{
		{name: "valid"},
		{
			name:    "missing manifest",
			modify:  func(t *testing.T, dir string) { mustRemove(t, filepath.Join(dir, ManifestName)) },
			wantErr: "open manifest",
		},
		{
			name:    "invalid manifest",
			modify:  func(t *testing.T, dir string) { mustWrite(t, filepath.Join(dir, ManifestName), "{}", 0o644) },
			wantErr: "invalid manifest",
		},
		{
			name: "wrong platform",
			modify: func(t *testing.T, dir string) {
				mustWrite(t, filepath.Join(dir, ManifestName), withField("arch", `"arm64"`), 0o644)
			},
			wantErr: "does not match the host platform",
		},
		{
			name:    "missing entrypoint",
			modify:  func(t *testing.T, dir string) { mustRemove(t, filepath.Join(dir, "bin", "sc-bos")) },
			wantErr: "entrypoint",
		},
		{
			name: "entrypoint not executable",
			modify: func(t *testing.T, dir string) {
				if err := os.Chmod(filepath.Join(dir, "bin", "sc-bos"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "not an executable regular file",
		},
		{
			name: "entrypoint is a symlink",
			modify: func(t *testing.T, dir string) {
				mustRemove(t, filepath.Join(dir, "bin", "sc-bos"))
				mustWrite(t, filepath.Join(dir, "bin", "real"), "binary", 0o755)
				if err := os.Symlink("real", filepath.Join(dir, "bin", "sc-bos")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "not an executable regular file",
		},
		{
			name:    "missing ops ui index",
			modify:  func(t *testing.T, dir string) { mustRemove(t, filepath.Join(dir, "ui", "ops", "index.html")) },
			wantErr: "ops ui",
		},
		{
			name: "ops ui index is a folder",
			modify: func(t *testing.T, dir string) {
				mustRemove(t, filepath.Join(dir, "ui", "ops", "index.html"))
				if err := os.Mkdir(filepath.Join(dir, "ui", "ops", "index.html"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "not a regular file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, ManifestName), validManifest, 0o644)
			mustWrite(t, filepath.Join(dir, "bin", "sc-bos"), "binary", 0o755)
			mustWrite(t, filepath.Join(dir, "ui", "ops", "index.html"), "<html></html>", 0o644)
			if tt.modify != nil {
				tt.modify(t, dir)
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = CheckDir(root, "v2026.10.1", "freebsd", "amd64")
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("CheckDir() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("CheckDir() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func mustWrite(t *testing.T, name, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	// WriteFile only applies perm when it creates the file.
	if err := os.Chmod(name, perm); err != nil {
		t.Fatal(err)
	}
}

func mustRemove(t *testing.T, name string) {
	t.Helper()
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
}
