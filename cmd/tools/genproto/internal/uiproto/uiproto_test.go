package uiproto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
)

// TestFixGeneratedFiles_NestedDirectories tests that fixGeneratedFiles fixes imports in nested directories.
func TestFixGeneratedFiles_NestedDirectories(t *testing.T) {
	tests := []struct {
		name  string
		txtar string
		wants map[string]string // file path -> expected content substring
	}{
		{
			name:  "nested JS imports",
			txtar: "nested_js_imports.txtar",
			wants: map[string]string{
				"services/api_pb.js":           `require('@smart-core-os/sc-api-grpc-web/traits/on_off_pb.js')`,
				"deep/nested/controller_pb.js": `require('@smart-core-os/sc-api-grpc-web/traits/brightness_pb.js')`,
			},
		},
		{
			name:  "nested DTS imports",
			txtar: "nested_dts_imports.txtar",
			wants: map[string]string{
				"services/api_pb.d.ts":           `from '@smart-core-os/sc-api-grpc-web/traits/on_off_pb'`,
				"deep/nested/controller_pb.d.ts": `from '@smart-core-os/sc-api-grpc-web/traits/brightness_pb'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := loadTxtar(t, tt.txtar)
			tmpDir := t.TempDir()
			extractAllFiles(t, archive, tmpDir)

			ctx := &generator.Context{
				RootDir: tmpDir,
				Config: generator.Config{
					DryRun: false,
				},
			}
			err := fixGeneratedFiles(ctx, tmpDir)
			if err != nil {
				t.Fatalf("fixGeneratedFiles() error = %v", err)
			}

			for file, wantSubstr := range tt.wants {
				t.Run(file, func(t *testing.T) {
					content, err := os.ReadFile(filepath.Join(tmpDir, file))
					if err != nil {
						t.Fatalf("reading file: %v", err)
					}

					contentStr := string(content)
					if !strings.Contains(contentStr, wantSubstr) {
						t.Errorf("file %s does not contain expected import:\n%s\n\nactual content:\n%s", file, wantSubstr, contentStr)
					}
				})
			}
		})
	}
}

func loadTxtar(t *testing.T, name string) *txtar.Archive {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading txtar file: %v", err)
	}
	return txtar.Parse(data)
}

func extractAllFiles(t *testing.T, archive *txtar.Archive, dir string) {
	t.Helper()
	for _, file := range archive.Files {
		path := filepath.Join(dir, file.Name)
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
		if err := os.WriteFile(path, file.Data, 0644); err != nil {
			t.Fatalf("writing file %s: %v", file.Name, err)
		}
	}
}
