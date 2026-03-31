package hub

import (
	"context"
	"log"

	"github.com/smart-core-os/sc-bos/pkg/app"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/healthbounds"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/hub"
	"github.com/smart-core-os/sc-bos/pkg/system/tenants"
)

func Main() {
	ctx := context.Background()

	systemConfig, err := loadSystemConfig()
	if err != nil {
		log.Fatal(err)
	}

	controller, err := app.Bootstrap(ctx, systemConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = controller.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func loadSystemConfig() (sysconf.Config, error) {
	systemConfig := sysconf.Default()
	systemConfig.SystemFactories = map[string]system.Factory{
		"hub":     hub.Factory(),
		"tenants": tenants.Factory,
	}
	systemConfig.DriverFactories = map[string]driver.Factory{
		"mock": mock.Factory,
	}
	systemConfig.AutoFactories = map[string]auto.Factory{
		healthbounds.AutoName: healthbounds.Factory,
	}

	err := sysconf.Load(&systemConfig)
	return systemConfig, err
}
