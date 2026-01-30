package appconf

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/renameio/v2/maybe"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/smart-core-os/sc-bos/internal/confmerge"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Store struct {
	logger  *zap.Logger
	backing confmerge.Store

	m      sync.Mutex
	active Config
}

func LoadStore(external Config, schema Schema, storeDir string, logger *zap.Logger) (*Store, error) {
	store := confmerge.NewDirStore(storeDir)
	active, patches, err := confmerge.Merge(external, store, schema.Blocks())
	err = multierr.Append(err, saveConfigPatches(patches, filepath.Join(storeDir, patchDirName), logger))
	if err != nil {
		return nil, err
	}

	return &Store{
		logger:  logger,
		backing: store,
		active:  active,
	}, nil
}

func (s *Store) Active() Config {
	s.m.Lock()
	defer s.m.Unlock()
	return s.active.clone()
}

func (s *Store) Drivers() *DriverStore {
	return &DriverStore{store: s}
}

func (s *Store) Automations() *AutomationStore {
	return &AutomationStore{store: s}
}

func (s *Store) Zones() *ZoneStore {
	return &ZoneStore{store: s}
}

func (s *Store) save(updated Config) error {
	encoded, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}
	err = s.backing.SetActiveConfig(encoded)
	if err != nil {
		return err
	}
	s.active = updated
	return nil
}

type serviceConfigOps[T any] struct {
	getMetadata func(T) (name, typ string)
	update      func(existing T, typ string, data []byte) T
}

func updateServiceConfig[T any](services []T, name, typ string, data []byte, ops serviceConfigOps[T]) ([]T, error) {
	if name == "" {
		return services, errors.New("name is required")
	}
	idx := slices.IndexFunc(services, func(s T) bool {
		n, _ := ops.getMetadata(s)
		return n == name
	})
	var serviceCfg T
	if idx < 0 {
		// add new service
		if typ == "" {
			return services, errors.New("type is required to add a new service")
		}
		serviceCfg = ops.update(serviceCfg, typ, data)
		services = append(services, serviceCfg)
	} else {
		serviceCfg = services[idx]
		if typ != "" {
			_, existingType := ops.getMetadata(serviceCfg)
			if typ != existingType {
				return services, errors.New("type cannot be changed")
			}
		}
		// when updating an existing service, use the existing type if not provided
		if typ == "" {
			_, typ = ops.getMetadata(serviceCfg)
		}
		serviceCfg = ops.update(serviceCfg, typ, data)
		services[idx] = serviceCfg
	}

	// test that the updated config marshalls successfully - we don't want to store a bad config
	_, err := json.Marshal(serviceCfg)
	if err != nil {
		return services, err
	}

	return services, nil
}

type DriverStore struct {
	store *Store
}

func (ds *DriverStore) SaveConfig(_ context.Context, name string, typ string, data []byte) error {
	ds.store.m.Lock()
	defer ds.store.m.Unlock()

	// insert the updated driver into a copy of the config
	updated := ds.store.active.clone()
	var err error
	updated.Drivers, err = updateServiceConfig(updated.Drivers, name, typ, data, serviceConfigOps[driver.RawConfig]{
		getMetadata: func(d driver.RawConfig) (string, string) {
			return d.Name, d.Type
		},
		update: func(cfg driver.RawConfig, typ string, data []byte) driver.RawConfig {
			cfg.Type = typ
			cfg.Raw = data
			return cfg
		},
	})
	if err != nil {
		return err
	}

	return ds.store.save(updated)
}

type AutomationStore struct {
	store *Store
}

func (as *AutomationStore) SaveConfig(_ context.Context, name string, typ string, data []byte) error {
	as.store.m.Lock()
	defer as.store.m.Unlock()

	// insert the updated driver into a copy of the config
	updated := as.store.active.clone()
	var err error
	updated.Automation, err = updateServiceConfig(updated.Automation, name, typ, data, serviceConfigOps[auto.RawConfig]{
		getMetadata: func(a auto.RawConfig) (string, string) {
			return a.Name, a.Type
		},
		update: func(cfg auto.RawConfig, typ string, data []byte) auto.RawConfig {
			cfg.Type = typ
			cfg.Raw = data
			return cfg
		},
	})
	if err != nil {
		return err
	}

	return as.store.save(updated)
}

type ZoneStore struct {
	store *Store
}

func (zs *ZoneStore) SaveConfig(_ context.Context, name string, typ string, data []byte) error {
	zs.store.m.Lock()
	defer zs.store.m.Unlock()

	// insert the updated driver into a copy of the config
	updated := zs.store.active.clone()
	var err error
	updated.Zones, err = updateServiceConfig(updated.Zones, name, typ, data, serviceConfigOps[zone.RawConfig]{
		getMetadata: func(z zone.RawConfig) (string, string) {
			return z.Name, z.Type
		},
		update: func(cfg zone.RawConfig, typ string, data []byte) zone.RawConfig {
			cfg.Type = typ
			cfg.Raw = data
			return cfg
		},
	})
	if err != nil {
		return err
	}

	return zs.store.save(updated)
}

const patchDirName = "patches"

func saveConfigPatches(patches []block.Patch, dir string, logger *zap.Logger) error {
	if len(patches) == 0 {
		return nil
	}

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	filename := filepath.Join(dir, "patch-"+time.Now().UTC().Format("20060102-150405")+".json")
	raw, err := json.MarshalIndent(patches, "", "  ")
	if err != nil {
		return err
	}
	err = maybe.WriteFile(filename, raw, 0644)
	if err != nil {
		return err
	}

	logger.Info("applied patches based on external config",
		zap.Int("count", len(patches)),
		zap.String("patchLogFile", filename),
	)
	return nil
}

type Schema struct {
	Drivers     map[string][]block.Block
	Automations map[string][]block.Block
	Zones       map[string][]block.Block
}

func (s Schema) Blocks() []block.Block {
	defaultBlocks := []block.Block{
		{
			Path: []string{"disabled"},
		},
	}

	return []block.Block{
		{
			Path:         []string{"drivers"},
			Key:          "name",
			TypeKey:      "type",
			BlocksByType: s.Drivers,
			Blocks:       defaultBlocks,
		},
		{
			Path:         []string{"automation"},
			Key:          "name",
			TypeKey:      "type",
			BlocksByType: s.Automations,
			Blocks:       defaultBlocks,
		},
		{
			Path:         []string{"zones"},
			Key:          "name",
			TypeKey:      "type",
			BlocksByType: s.Zones,
			Blocks:       defaultBlocks,
		},
		{
			Path: []string{"metadata"},
		},
	}
}
