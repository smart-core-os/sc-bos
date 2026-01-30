package driver

import (
	"crypto/tls"
	"net/http"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

type Services struct {
	Logger          *zap.Logger
	Node            *node.Node  // for advertising devices
	ClientTLSConfig *tls.Config // for connecting to other smartcore nodes
	HTTPMux         *http.ServeMux
	Config          service.ConfigUpdater
	Database        *bolthold.Store
	Health          *healthpb.Checks
}

type Factory interface {
	New(services Services) service.Lifecycle
}
