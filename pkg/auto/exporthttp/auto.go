// Package exporthttp provides a simple and flexible way to automate
// the process of POSTing recordings to a REST API endpoint.
// It supports scheduled or regular posting of recordings
//
// This package is designed to be extensible, enabling future
// support for other CMS platforms or REST APIs. It abstracts
// the complexities of API interactions
package exporthttp

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/config"
	exportHttp "github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/http"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/job"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const AutoName = "exporthttp"

var Factory auto.Factory = factory{}

type factory struct{}

type autoImpl struct {
	*service.Service[config.Root]
	auto.Services
}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &autoImpl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

func (a *autoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	logger := a.Logger.Named(cfg.Name)

	jobs := job.FromConfig(cfg, a.Database, AutoName, cfg.Name, a.Node, logger)

	if len(jobs) < 1 {
		return nil
	}

	var client *exportHttp.Client

	switch cfg.Auth.Type {
	case config.AuthenticationBearer:
		client = exportHttp.New(exportHttp.WithAuthorizationBearer(cfg.Auth.Token), exportHttp.WithLogger(cfg.Logs, logger))
	default:
		return fmt.Errorf("authentication type %s not supported", cfg.Auth.Type)
	}

	go func() {
		if err := job.ExecuteAll(ctx, client.Post, jobs...); err != nil {
			logger.Error("exporthttp automation execution failed", zap.Error(err))
			return
		}

		logger.Info("exporthttp automation execution stopped")
	}()

	return nil
}
