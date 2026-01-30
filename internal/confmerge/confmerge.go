// Package confmerge provides a way to merge external configuration changes into a config store managed by the system.
//
// The system has two types of configuration: external and active.
// External configuration is configuration generated outside of Smart Core (for example, handwriten or tool-generated).
// Active configuration is the one actually used by the system, and is managed by Smart Core.
//
// When the external configuration changes, we need to apply those changes to the active configuration.
// However, there may be conflicting updates to the active configuration. To resolve these conflicts, we use a
// structure-aware diff and patch system based on the block package.
//
// Merge will save a copy of the most recently encountered external configuration in the Store.
// This allows Merge to detect what has changed in the external configuration since the last time it was called.
// The patches from this are then applied to the active configuration.
package confmerge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/renameio/v2/maybe"

	"github.com/smart-core-os/sc-bos/pkg/block"
)

// Store is the set of operations required for a config store, as used by Merge.
//
// DirStore is the canonical implementation of this interface.
type Store interface {
	// GetExternalConfig returns the external config from the store, or nil if there is no external config.
	GetExternalConfig() ([]byte, error)
	// SetExternalConfig replaces the external config in the store with the provided config.
	// c must not be nil.
	SetExternalConfig(c []byte) error
	// GetActiveConfig returns the active config from the store, or nil if there is no active config.
	GetActiveConfig() ([]byte, error)
	// SetActiveConfig replaces the active config in the store with the provided config.
	// c must not be nil.
	SetActiveConfig(c []byte) error
}

const (
	externalConfigFilename = "external.json"
	activeConfigFilename   = "active.json"
)

// DirStore is a Store that stores config files in a directory on disk.
type DirStore struct {
	dir string
}

func NewDirStore(dir string) *DirStore {
	return &DirStore{dir: dir}
}

func (s *DirStore) GetExternalConfig() ([]byte, error) {
	return s.read(externalConfigFilename)
}

func (s *DirStore) SetExternalConfig(c []byte) error {
	return s.write(externalConfigFilename, c)
}

func (s *DirStore) GetActiveConfig() ([]byte, error) {
	return s.read(activeConfigFilename)
}

func (s *DirStore) SetActiveConfig(c []byte) error {
	return s.write(activeConfigFilename, c)
}

func (s *DirStore) ensureDirExists() error {
	return os.MkdirAll(s.dir, 0755)
}

func (s *DirStore) read(name string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *DirStore) write(name string, data []byte) error {
	if err := s.ensureDirExists(); err != nil {
		return err
	}
	return maybe.WriteFile(filepath.Join(s.dir, name), data, 0644)
}

// Merge detects changes to external configuration, and applies those changes to the active configuration.
//
// Returns the new active configuration.
// external should be the fully loaded Config from outside the system.
// The provided external config and resolved active config are stored in the provided Store.
// The store may be empty:
//   - If the store has no saved external config, then external is compared against the zero value of T.
//   - If the store has no saved active config, the provided external config is used verbatim as the active config.
//
// Also returns a slice of the patches that were applied and saved successfully.
//
// schema describes the structure of T object, used to produce patches. This controls the granularity of patches
// that will be applied to the active config. For full detail of this, see the block package.
//
// Configs are stored as JSON, so T must marshal and unmarshal to/from JSON correctly.
func Merge[T any](external T, store Store, schema []block.Block) (result T, applied []block.Patch, err error) {
	var zero T

	// if external JSON doesn't exist, getExternalJSON returns the zero value which we we can use
	oldExternal, _, err := getExternalJSON[T](store)
	if err != nil {
		return zero, nil, err
	}

	oldActive, ok, err := getActiveJSON[T](store)
	if err != nil {
		return zero, nil, err
	}
	var newActive T
	if !ok {
		// no active config, just use the provided external config
		newActive = external
	} else {
		applied, err = block.Diff(oldExternal, external, schema)
		if err != nil {
			return zero, nil, err
		}

		newActive, err = block.ApplyPatches(oldActive, applied)
		if err != nil {
			return zero, nil, err
		}
	}

	err = setActiveJSON(store, newActive)
	if err != nil {
		return zero, nil, err
	}
	err = setExternalJSON(store, external)
	if err != nil {
		return zero, applied, err
	}
	return newActive, applied, nil
}

func getExternalJSON[T any](store Store) (ext T, ok bool, err error) {
	raw, err := store.GetExternalConfig()
	if err != nil {
		return ext, false, err
	}
	if raw == nil {
		return ext, false, nil
	}
	err = json.Unmarshal(raw, &ext)
	if err != nil {
		return ext, false, err
	}
	return ext, true, nil
}

func getActiveJSON[T any](store Store) (ext T, ok bool, err error) {
	raw, err := store.GetActiveConfig()
	if err != nil {
		return ext, false, err
	}
	if raw == nil {
		return ext, false, nil
	}
	err = json.Unmarshal(raw, &ext)
	if err != nil {
		return ext, false, err
	}
	return ext, true, nil
}

func setExternalJSON[T any](store Store, c T) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return store.SetExternalConfig(raw)
}

func setActiveJSON[T any](store Store, c T) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return store.SetActiveConfig(raw)
}
