package zone

import (
	"crypto/tls"
	"net/http"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

type Services struct {
	Logger          *zap.Logger
	Node            *node.Node
	Devices         *Devices
	ClientTLSConfig *tls.Config // for connecting to other smartcore nodes
	HTTPMux         *http.ServeMux
	Config          service.ConfigUpdater
	Health          *healthpb.Checks

	DriverFactories map[string]driver.Factory
}

type Factory interface {
	New(Services) service.Lifecycle
}

type FactoryFunc func(services Services) service.Lifecycle

func (f FactoryFunc) New(services Services) service.Lifecycle {
	return f(services)
}
