package app

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/pkg/app/appconf"
	"github.com/smart-core-os/sc-bos/pkg/app/files"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/task/serviceapi"
)

func loadLocalAppConfig(sysConfig sysconf.Config, logger *zap.Logger) (ConfigStore, error) {
	var externalConf appconf.Config
	filesLoaded, err := appconf.LoadIncludes("", &externalConf, sysConfig.AppConfig)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// warn that file(s) couldn't be found, but continue with default config
			logger.Warn("failed to load some config", zap.Strings("paths", sysConfig.AppConfig), zap.Error(err), zap.Strings("filesLoaded", filesLoaded))
		} else {
			return nil, err
		}
	} else {
		// successfully loaded the config
		logger.Debug("loaded external config",
			zap.Strings("paths", sysConfig.AppConfig),
			zap.Strings("includes", externalConf.Includes),
			zap.Strings("filesLoaded", filesLoaded),
		)
	}
	confStore, err := appconf.LoadStore(externalConf, appconf.Schema{
		Drivers:     sysConfig.DriverConfigBlocks(),
		Automations: sysConfig.AutoConfigBlocks(),
		Zones:       sysConfig.ZoneConfigBlocks(),
	}, files.Path(sysConfig.DataDir, configDirName), logger)
	if err != nil {
		return nil, err
	}
	return confStore, nil
}

func loadCloudAppConfig(ctx context.Context, sysConfig sysconf.Config, store *cloud.DeploymentStore, conn *cloud.Conn, logger *zap.Logger) (ConfigStore, error) {
	// first, try loading the installing config, if there is one
	// if that doesn't exist or didn't work, proceed to the active config
	installingConfig, loaded := loadCloudInstallingConfig(ctx, store, conn, logger)
	if loaded {
		return &immutableConfigStore{active: installingConfig}, nil
	}

	activeConfigFS, err := store.ActiveConfig()
	if err != nil {
		// There is an active config from the cloud, but we can't load it. Quite a big problem - the config should
		// have been validated when it was committed, so this likely indicates a code problem. Not safe to load local
		// config as that may be out of date.
		// Only thing we can do here is report the error.
		logger.Error("failed to open active config FS from deployment client - no config will be loaded", zap.Error(err))
		return nil, err
	}

	if activeConfigFS == nil {
		// we don't have any active config from the cloud.
		// probably means that cloud config has just been enabled, but never deployed.
		// keep using the local config
		logger.Warn("in cloud config mode, but no active config available - using local config instead")
		return loadLocalAppConfig(sysConfig, logger)
	}
	defer func() {
		_ = activeConfigFS.Close()
	}()

	activeConfig, err := loadConfigPackage(activeConfigFS)
	if err != nil {
		logger.Error("failed to load active config from deployment client - no config will be loaded", zap.Error(err))
		return nil, err
	}

	logger.Info("loaded active config from deployment client")
	return &immutableConfigStore{active: activeConfig}, nil
}

func loadCloudInstallingConfig(ctx context.Context, store *cloud.DeploymentStore, conn *cloud.Conn, logger *zap.Logger) (conf appconf.Config, loaded bool) {
	fail := func(msg string) {
		err := conn.FailInstall(ctx, msg)
		if err != nil {
			logger.Error("failed to report install failure to deployment server", zap.Error(err))
		}
	}

	installingFS, err := store.InstallingConfig()
	if err != nil {
		logger.Error("failed to open installing config FS", zap.Error(err))
		fail("failed to open installing config FS")
		return appconf.Config{}, false
	} else if installingFS == nil {
		// not really an error, just no installing config to load, so return with loaded = false to indicate that
		return appconf.Config{}, false
	}
	defer func() {
		_ = installingFS.Close()
	}()

	conf, err = loadConfigPackage(installingFS)
	if err != nil {
		logger.Error("failed to load installing config", zap.Error(err))
		fail("failed to load installing config")
		return appconf.Config{}, false
	}

	logger.Info("loaded installing config")
	err = conn.CommitInstall(ctx)
	if err != nil {
		logger.Error("failed to report install success to server", zap.Error(err))
	}

	return conf, true
}

func loadConfigPackage(src fs.FS) (appconf.Config, error) {
	c, err := appconf.LoadLocalConfigFS(src, "config", "app.conf.json")
	if err != nil {
		return appconf.Config{}, err
	}
	return *c, nil
}

type ConfigStore interface {
	Active() appconf.Config
	Drivers() serviceapi.Store
	Automations() serviceapi.Store
	Zones() serviceapi.Store
}

// a dummy ConfigStore implementation used when in cloud config mode, which disallows any modifications to the config
// and returns the config from the cloud
type immutableConfigStore struct {
	active appconf.Config
}

func (s *immutableConfigStore) Active() appconf.Config {
	return s.active
}

func (s *immutableConfigStore) Drivers() serviceapi.Store {
	return immutableServiceStore{}
}

func (s *immutableConfigStore) Automations() serviceapi.Store {
	return immutableServiceStore{}
}

func (s *immutableConfigStore) Zones() serviceapi.Store {
	return immutableServiceStore{}
}

type immutableServiceStore struct{}

func (i immutableServiceStore) SaveConfig(ctx context.Context, name, typ string, data []byte) error {
	return ErrImmutableConfig
}

var ErrImmutableConfig = errors.New("cannot modify config when in cloud config mode")
