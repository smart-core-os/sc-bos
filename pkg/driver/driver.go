package driver

import (
	"crypto/tls"
	"net/http"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
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
	// SystemCheck is a persistent driver-level health check created by the app layer.
	// It is registered under the driver's own name and survives applyConfig retries —
	// use it to report top-level connectivity state (server reachable, licence valid, etc.).
	// Drivers should update but never dispose this check; disposal is handled by the service framework.
	SystemCheck *healthpb.FaultCheck
}

type Factory interface {
	New(services Services) service.Lifecycle
}
