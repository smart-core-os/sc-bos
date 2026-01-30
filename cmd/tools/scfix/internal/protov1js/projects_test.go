package protov1js

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestFindProjectDirs(t *testing.T) {
	fsys := fstest.MapFS{
		"app1/package.json":                      {Data: []byte(`{"dependencies":{"@smart-core-os/sc-bos-ui-gen":"^1.0.0"}}`)},
		"app2/package.json":                      {Data: []byte(`{"dependencies":{"some-other-package":"^1.0.0"}}`)},
		"app3/package.json":                      {Data: []byte(`{"dependencies":{"@smart-core-os/sc-bos-ui-gen":"^1.0.0"}}`)},
		"workspace/packages/ui-app/package.json": {Data: []byte(`{"dependencies":{"@smart-core-os/sc-bos-ui-gen":"^1.0.0"}}`)},
	}

	found, err := findProjectDirs(fsys)
	if err != nil {
		t.Fatalf("findProjectDirs failed: %v", err)
	}

	expected := []string{"app1", "app3", "workspace/packages/ui-app"}
	if len(found) != len(expected) {
		t.Errorf("found %d projects, want %d", len(found), len(expected))
	}

	// Check that all expected projects were found
	foundMap := make(map[string]bool)
	for _, dir := range found {
		foundMap[dir] = true
	}

	for _, expectedDir := range expected {
		if !foundMap[expectedDir] {
			t.Errorf("expected to find project at %q, but didn't", expectedDir)
		}
	}
}

func TestFindProjectDirs_RealFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	projects := []struct {
		path     string
		hasUIGen bool
	}{
		{"app1/package.json", true},
		{"app2/package.json", false},
		{"app3/package.json", true},
	}

	for _, p := range projects {
		fullPath := filepath.Join(tmpDir, p.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}

		content := `{"dependencies":{"some-other-package":"^1.0.0"}}`
		if p.hasUIGen {
			content = `{"dependencies":{"@smart-core-os/sc-bos-ui-gen":"^1.0.0"}}`
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	found, err := findProjectDirs(os.DirFS(tmpDir))
	if err != nil {
		t.Fatalf("findProjectDirs failed: %v", err)
	}

	expected := []string{"app1", "app3"}
	if len(found) != len(expected) {
		t.Errorf("found %d projects, want %d", len(found), len(expected))
	}

	for _, path := range found {
		if filepath.IsAbs(path) {
			t.Errorf("expected relative path, got absolute: %s", path)
		}
	}
}

func TestFindNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	appDir := filepath.Join(tmpDir, "workspace", "app")
	workspaceDir := filepath.Join(tmpDir, "workspace")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		setupFunc      func()
		projectDir     string
		expectedSuffix string
	}{
		{
			name: "node_modules in project dir",
			setupFunc: func() {
				uiGenPath := filepath.Join(appDir, "node_modules", "@smart-core-os", "sc-bos-ui-gen")
				os.MkdirAll(uiGenPath, 0755)
				os.WriteFile(filepath.Join(uiGenPath, "package.json"), []byte("{}"), 0644)
			},
			projectDir:     appDir,
			expectedSuffix: "workspace/app/node_modules",
		},
		{
			name: "node_modules in parent (workspace)",
			setupFunc: func() {
				os.RemoveAll(filepath.Join(appDir, "node_modules"))
				uiGenPath := filepath.Join(workspaceDir, "node_modules", "@smart-core-os", "sc-bos-ui-gen")
				os.MkdirAll(uiGenPath, 0755)
				os.WriteFile(filepath.Join(uiGenPath, "package.json"), []byte("{}"), 0644)
			},
			projectDir:     appDir,
			expectedSuffix: "workspace/node_modules",
		},
		{
			name: "node_modules in root",
			setupFunc: func() {
				os.RemoveAll(filepath.Join(appDir, "node_modules"))
				os.RemoveAll(filepath.Join(workspaceDir, "node_modules"))
				uiGenPath := filepath.Join(tmpDir, "node_modules", "@smart-core-os", "sc-bos-ui-gen")
				os.MkdirAll(uiGenPath, 0755)
				os.WriteFile(filepath.Join(uiGenPath, "package.json"), []byte("{}"), 0644)
			},
			projectDir:     appDir,
			expectedSuffix: "node_modules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			result := findNodeModules(tt.projectDir)
			expected := filepath.Join(tmpDir, tt.expectedSuffix)

			if result != expected {
				t.Errorf("findNodeModules() = %q, want %q", result, expected)
			}
		})
	}
}
