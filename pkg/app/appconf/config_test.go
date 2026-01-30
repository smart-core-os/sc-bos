package appconf

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/txtar"

	"github.com/smart-core-os/sc-bos/pkg/driver"
)

func ExampleConfig() {
	type ExampleDriverConfig struct {
		driver.BaseConfig
		Property string `json:"property"`
	}

	// language=json
	buf := []byte(`{
	"drivers": [
		{
			"name": "foo",
			"type": "example",
			"property": "bar"
		}
    ]
}`)
	var config Config
	err := json.Unmarshal(buf, &config)
	if err != nil {
		panic(err)
	}

	for _, d := range config.Drivers {
		switch d.Type {
		case "example":
			var exampleConfig ExampleDriverConfig
			err := json.Unmarshal(d.Raw, &exampleConfig)
			if err != nil {
				panic(err)
			}
			fmt.Printf("ExampleDriver name=%q property=%q\n", d.Name, exampleConfig.Property)
		default:
			fmt.Printf("unknown driver type %q\n", d.Type)
		}
	}
	// Output: ExampleDriver name="foo" property="bar"
}

func TestLoadLocalConfig(t *testing.T) {
	driverNamed := func(name string) driver.RawConfig {
		return driver.RawConfig{
			BaseConfig: driver.BaseConfig{Name: name},
			Raw:        json.RawMessage(fmt.Sprintf("{\"name\": \"%s\"}", name)),
		}
	}
	driverWithType := func(name, t string) driver.RawConfig {
		return driver.RawConfig{
			BaseConfig: driver.BaseConfig{Name: name, Type: t},
			Raw:        json.RawMessage(fmt.Sprintf("{\"name\": \"%s\", \"type\": \"%s\"}", name, t)),
		}
	}
	tests := []struct {
		name   string
		fs     fs.FS
		dir    string
		file   string
		config *Config
	}{
		{
			name:   "empty",
			fs:     fstest.MapFS{},
			dir:    "data",
			file:   "base.json",
			config: nil,
		},
		{
			name: "single file",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{"name": "my-config"}`),
				},
			},
			dir:    "data",
			file:   "base.json",
			config: &Config{Name: "my-config", FilePath: filepath.Join("data", "base.json")},
		},
		{
			name: "with include",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{
						"name": "my-config",
						"includes": ["part-1.json"],
						"drivers": [{"name": "driver-1"}]
					}`),
				},
				filepath.Join("data", "part-1.json"): {
					Data: []byte(`{
						"name": "ignored",
						"drivers": [{"name": "driver-part-1"}]
					}`),
				},
			},
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "my-config",
				Includes: []string{"part-1.json"},
				Drivers: []driver.RawConfig{
					driverNamed("driver-1"),
					driverNamed("driver-part-1"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "with multiple nested includes",
			fs:   readTxtarFS(t, "testdata/nested-includes.txtar"),
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name: "my-config",
				Includes: []string{
					"part-1.json",
					"part-2.json",
					"part-3.json",
					"part-1a.json",
					"part-1b.json",
					"part-2a.json",
					"part-2b.json",
					"part-3a.json",
					// "part-3b.json", // included in config, but the file doesn't exist so isn't added to the output
				},
				Drivers: []driver.RawConfig{
					driverNamed("driver-1"),
					driverNamed("driver-part-1"),
					driverNamed("driver-part-2"),
					driverNamed("driver-part-3"),
					driverNamed("driver-part-1a"),
					driverNamed("driver-part-1b"),
					driverNamed("driver-part-2a"),
					driverNamed("driver-part-2b"),
					driverNamed("driver-part-3a"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "avoids includes loop",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{
						"name": "my-config",
						"includes": ["part-1.json"],
						"drivers": [{"name": "driver-1"}]
					}`),
				},
				filepath.Join("data", "part-1.json"): {
					Data: []byte(`{
						"name": "ignored",
						"includes": ["part-1a.json"],
						"drivers": [{"name": "driver-part-1"}]
					}`),
				},
				filepath.Join("data", "part-1a.json"): {
					Data: []byte(`{
						"includes": ["part-1.json"],
						"drivers": [{"name": "driver-part-1a"}]
					}`),
				},
			},
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name: "my-config",
				Includes: []string{
					"part-1.json",
					"part-1a.json",
				},
				Drivers: []driver.RawConfig{
					driverNamed("driver-1"),
					driverNamed("driver-part-1"),
					driverNamed("driver-part-1a"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "duplicate driver ignored",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{
						"name": "my-config",
						"includes": ["part-1.json"],
						"drivers": [{"name": "driver-1", "type": "d-1"}]
					}`),
				},
				filepath.Join("data", "part-1.json"): {
					Data: []byte(`{
						"name": "ignored",
						"drivers": [{"name": "driver-1", "type": "d-2"}]
					}`),
				},
			},
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "my-config",
				Includes: []string{"part-1.json"},
				Drivers: []driver.RawConfig{
					driverWithType("driver-1", "d-1"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "first dup driver takes precendence",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{
						"name": "my-config",
						"includes": ["part-1.json","part-2.json"],
						"drivers": [{"name": "driver-1", "type": "d-1"}]
					}`),
				},
				filepath.Join("data", "part-1.json"): {
					Data: []byte(`{
						"name": "ignored",
						"drivers": [{"name": "driver-2", "type": "d-1a"}]
					}`),
				},
				filepath.Join("data", "part-2.json"): {
					Data: []byte(`{
						"name": "ignored",
						"drivers": [{"name": "driver-2", "type": "d-2"}]
					}`),
				},
			},
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "my-config",
				Includes: []string{"part-1.json", "part-2.json"},
				Drivers: []driver.RawConfig{
					driverWithType("driver-1", "d-1"),
					driverWithType("driver-2", "d-1a"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "reads directory",
			fs:   readTxtarFS(t, "testdata/dir.txtar"),
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "my-config",
				Includes: []string{"dir/1.json", "dir/2.json", "dir/more/1.json"},
				Drivers: []driver.RawConfig{
					driverWithType("base1", "base1"),
					driverWithType("dir1", "dir1"),
					driverWithType("dir2", "dir2"),
					driverWithType("dir/more1", "dir/more1"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "directory overrides",
			fs:   readTxtarFS(t, "testdata/dir-overrides.txtar"),
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "my-config",
				Includes: []string{"file1.json", "dir1/1.json", "dir2/1.json", "file2.json"},
				Drivers: []driver.RawConfig{
					driverWithType("base", "base"),
					driverWithType("file1", "file1"),
					driverWithType("dir1/1", "dir1/1"),
					driverWithType("dir2/1", "dir2/1"),
					driverWithType("file2", "file2"),
				},
				FilePath: filepath.Join("data", "base.json"),
			},
		},
		{
			name: "include non-json file",
			fs: fstest.MapFS{
				filepath.Join("data", "base.json"): {
					Data: []byte(`{"name": "base", "include": ["file.txt"]}`),
				},
				filepath.Join("data", "file.txt"): {
					Data: []byte(`this is not a json file`),
				},
			},
			dir:  "data",
			file: "base.json",
			config: &Config{
				Name:     "base",
				FilePath: filepath.Join("data", "base.json"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readFile = func(name string) ([]byte, error) {
				return fs.ReadFile(tt.fs, name)
			}
			glob = func(pattern string) (matches []string, err error) {
				return fs.Glob(tt.fs, pattern)
			}
			conf, err := LoadLocalConfig(tt.dir, tt.file)
			if tt.config != nil && conf == nil && err != nil {
				t.Errorf("expected config, got error: %s", err)
			}
			if diff := cmp.Diff(tt.config, conf); diff != "" {
				t.Errorf("wrong config: %s", diff)
			}
			glob = filepath.Glob
			readFile = os.ReadFile
		})
	}
}

func readTxtarFS(t *testing.T, file string) fs.FS {
	t.Helper()
	ar, err := txtar.ParseFile(file)
	if err != nil {
		t.Fatal(err)
	}
	f, err := txtar.FS(ar)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
