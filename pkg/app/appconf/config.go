// Package appconf provides runtime configuration.
package appconf

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/multierr"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/app/files"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/slices"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Config struct {
	Name     string           `json:"name,omitempty"`
	Metadata *traits.Metadata `json:"metadata,omitempty"`
	// Includes lists other files and glob patterns for config to load.
	// Files are read in the order specified here then by filepath.Glob.
	// Drivers, Automation, and Zones are merged using the Name in a first-come, first-served nature.
	// Glob includes are expanded in the output when using LoadLocalConfig, files not found will be excluded.
	// Included files that also have includes will be processed once all includes in this config are processed.
	// Paths are resolved relative to the directory the config file is in.
	// Paths starting with `/` will be treated as absolute paths.
	Includes   []string           `json:"includes,omitempty"`
	Drivers    []driver.RawConfig `json:"drivers,omitempty"`
	Automation []auto.RawConfig   `json:"automation,omitempty"`
	Zones      []zone.RawConfig   `json:"zones,omitempty"`

	// the path to the file this config was loaded from
	FilePath string `json:"-"`
}

func (c *Config) mergeWith(other *Config) {
	switch {
	case c.Metadata == nil:
		c.Metadata = other.Metadata
	case other.Metadata != nil:
		proto.Merge(c.Metadata, other.Metadata)
	}

	if c.Name == "" {
		c.Name = other.Name
	}

	// if any driver/auto/zone has a duplicate name it is ignored in favour of the one already present

	driverNames := c.driverNamesMap()
	autoNames := c.autoNamesMap()
	zoneNames := c.zoneNamesMap()
	for _, d := range other.Drivers {
		if _, found := driverNames[d.Name]; !found {
			c.Drivers = append(c.Drivers, d)
		}
	}
	for _, a := range other.Automation {
		if _, found := autoNames[a.Name]; !found {
			c.Automation = append(c.Automation, a)
		}
	}
	for _, z := range other.Zones {
		if _, found := zoneNames[z.Name]; !found {
			c.Zones = append(c.Zones, z)
		}
	}
	// Includes are merged in a special way, we use the FilePath relative to c as the include.
	relInc, err := filepath.Rel(filepath.Dir(c.FilePath), other.FilePath)
	if err != nil {
		return
	}
	if !slices.Contains(relInc, c.Includes) {
		c.Includes = append(c.Includes, relInc)
	}
}

func (c *Config) driverNamesMap() map[string]bool {
	names := make(map[string]bool, len(c.Drivers))
	for _, d := range c.Drivers {
		names[d.Name] = true
	}
	return names
}

func (c *Config) autoNamesMap() map[string]bool {
	names := make(map[string]bool, len(c.Automation))
	for _, d := range c.Automation {
		names[d.Name] = true
	}
	return names
}

func (c *Config) zoneNamesMap() map[string]bool {
	names := make(map[string]bool, len(c.Zones))
	for _, d := range c.Zones {
		names[d.Name] = true
	}
	return names
}

func (c *Config) clone() Config {
	return Config{
		Name:       c.Name,
		Metadata:   proto.Clone(c.Metadata).(*traits.Metadata),
		Includes:   append([]string(nil), c.Includes...),
		Drivers:    append([]driver.RawConfig(nil), c.Drivers...),
		Automation: append([]auto.RawConfig(nil), c.Automation...),
		Zones:      append([]zone.RawConfig(nil), c.Zones...),
		FilePath:   c.FilePath,
	}
}

// LoadLocalConfig will load Config from a local file, as well as any included files
func LoadLocalConfig(dir, file string) (*Config, error) {
	if dir == "" {
		dir = "."
	}
	return LoadLocalConfigFS(os.DirFS(dir), ".", file)
}

// LoadLocalConfigFS will load Config from an fs.FS, as well as any included files.
func LoadLocalConfigFS(fsys fs.FS, dir, file string) (*Config, error) {
	p := path.Join(dir, file)
	conf, err := configFromFileFS(fsys, p)
	if err != nil {
		return nil, err
	}
	includes := conf.Includes
	conf.Includes = nil
	_, err = loadIncludesFS(fsys, dir, conf, includes, nil)
	return conf, err
}

// LoadIncludes will go through each include, load the configs, merge the configs, then load any further includes.
// Returns a list of all files that were loaded.
func LoadIncludes(dir string, dst *Config, includes []string) ([]string, error) {
	return loadIncludes(dir, dst, includes, nil)
}

// LoadIncludesFS is like LoadIncludes but reads from an fs.FS.
func LoadIncludesFS(fsys fs.FS, dir string, dst *Config, includes []string) ([]string, error) {
	return loadIncludesFS(fsys, dir, dst, includes, nil)
}

// loadIncludes recursively loads includes from config files and merges them
func loadIncludes(dir string, dst *Config, includes, seen []string) ([]string, error) {
	var errs error
	var configs []*Config
	// load first layer of includes
	for _, include := range includes {
		p := files.Path(dir, include)
		if slices.Contains(p, seen) {
			continue
		}
		matches, err := filepath.Glob(p)
		if err != nil || matches == nil {
			matches = []string{p}
		}
		for _, p := range matches {
			seen = append(seen, p) // track files we've seen, to avoid getting in a loop
			fsys := os.DirFS(filepath.Dir(p))
			extraConf, err := configFromFileFS(fsys, filepath.Base(p))
			if err != nil {
				errs = multierr.Append(errs, err)
			} else {
				extraConf.FilePath = p // restore full OS path
				configs = append(configs, extraConf)
				dst.mergeWith(extraConf)
			}
		}
	}
	// load all deeper includes
	for _, config := range configs {
		alsoSeen, err := loadIncludes(filepath.Dir(config.FilePath), dst, config.Includes, seen)
		if err != nil {
			seen = alsoSeen
		}
		errs = multierr.Append(errs, err)
	}
	return seen, errs
}

// loadIncludesFS recursively loads includes from config files in an fs.FS and merges them
func loadIncludesFS(fsys fs.FS, dir string, dst *Config, includes, seen []string) ([]string, error) {
	var errs error
	var configs []*Config
	for _, include := range includes {
		p := path.Join(dir, include)
		if slices.Contains(p, seen) {
			continue
		}
		matches, err := fs.Glob(fsys, p)
		if err != nil || matches == nil {
			matches = []string{p}
		}
		for _, p := range matches {
			seen = append(seen, p)
			extraConf, err := configFromFileFS(fsys, p)
			if err != nil {
				errs = multierr.Append(errs, err)
			} else {
				configs = append(configs, extraConf)
				dst.mergeWith(extraConf)
			}
		}
	}
	for _, config := range configs {
		alsoSeen, err := loadIncludesFS(fsys, path.Dir(config.FilePath), dst, config.Includes, seen)
		if err != nil {
			seen = alsoSeen
		}
		errs = multierr.Append(errs, err)
	}
	return seen, errs
}

// configFromFileFS will load Config from an fs.FS
func configFromFileFS(fsys fs.FS, name string) (*Config, error) {
	var conf Config
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file %s: %w", name, err)
	}
	err = json.Unmarshal(raw, &conf)
	if err != nil {
		return nil, fmt.Errorf("config JSON unmarshal %s: %w", name, err)
	}
	conf.FilePath = name
	return &conf, nil
}
