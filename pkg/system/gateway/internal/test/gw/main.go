package gw

import (
	"context"
	"log"

	"github.com/smart-core-os/sc-bos/pkg/app"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway"
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
	gwFactory := gateway.Factory()
	systemConfig.SystemFactories = map[string]system.Factory{
		gateway.Name:       gwFactory,
		gateway.LegacyName: gwFactory,
		// todo: remove these services
		"hub":     hub.Factory(),
		"tenants": tenants.Factory,
	}

	err := sysconf.Load(&systemConfig)
	return systemConfig, err
}
