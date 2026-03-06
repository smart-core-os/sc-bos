// Package resourceutilisation implements a system plugin that reports host-level resource stats
// (CPU, memory, disk, network connections, uptime) as a ResourceUtilisationApi trait.
package resourceutilisation

import (
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

var Factory system.Factory = factory{}

type factory struct{}

func (f factory) New(services system.Services) service.Lifecycle {
	s := &System{
		name:   services.Node.Name(),
		node:   services.Node,
		logger: services.Logger.Named("resourceutilisation"),
	}
	s.Service = service.New(
		service.MonoApply(s.applyConfig),
		service.WithRetry[Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", s.logger)
		})),
	)
	return s
}

type System struct {
	*service.Service[Root]

	name   string
	node   *node.Node
	logger *zap.Logger
}
