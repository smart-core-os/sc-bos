// Package log implements the Log trait system plugin.
// When enabled, it exposes log streaming, log level management,
// log file metadata, and signed-URL log file download via the LogApi gRPC service.
package log

import (
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/log/config"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// Factory is the system.Factory for the Log trait.
var Factory factory

type factory struct{}

func (factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

// NewSystem creates and returns a new System (implements service.Lifecycle).
func NewSystem(services system.Services) *System {
	logger := services.Logger.Named("log")
	s := &System{
		name:      services.Node.Name(),
		announcer: node.NewReplaceAnnouncer(services.Node),
		services:  services,
		logger:    logger,
	}
	s.Service = service.New(
		service.MonoApply(s.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", logger)
		})),
	)
	return s
}

// System is the running Log trait system.
type System struct {
	*service.Service[config.Root]

	name      string
	announcer *node.ReplaceAnnouncer
	services  system.Services
	logger    *zap.Logger
}
