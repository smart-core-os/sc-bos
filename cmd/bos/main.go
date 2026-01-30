// Command bos provides the canonical Smart Core BOS executable.
package main

import (
	"context"
	"os"

	"github.com/smart-core-os/sc-bos/internal/driver/settings"
	"github.com/smart-core-os/sc-bos/pkg/app"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/auto/allautos"
	"github.com/smart-core-os/sc-bos/pkg/driver/alldrivers"
	"github.com/smart-core-os/sc-bos/pkg/system/allsystems"
	"github.com/smart-core-os/sc-bos/pkg/zone/allzones"
)

func main() {
	os.Exit(app.RunUntilInterrupt(run))
}

func run(ctx context.Context) error {
	systemConfig, err := loadSystemConfig()
	if err != nil {
		return err
	}

	controller, err := app.Bootstrap(ctx, systemConfig)
	if err != nil {
		return err
	}

	return controller.Run(ctx)
}

func loadSystemConfig() (sysconf.Config, error) {
	systemConfig := sysconf.Default()

	systemConfig.DriverFactories = alldrivers.Factories()
	systemConfig.DriverFactories[settings.DriverName] = settings.Factory
	systemConfig.AutoFactories = allautos.Factories()
	systemConfig.SystemFactories = allsystems.Factories()
	systemConfig.ZoneFactories = allzones.Factories()

	err := sysconf.Load(&systemConfig)
	return systemConfig, err
}
