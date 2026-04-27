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
	// SystemCheck is a driver-level health check for top-level connectivity state
	// (server reachable, licence valid, etc.). Call MarkFailed when connectivity is lost and
	// MarkRunning when it is restored. Call Dispose in the driver's stop handler.
	// May be nil if the check could not be registered; always nil-check before use.
	SystemCheck service.SystemCheck
}

type Factory interface {
	New(services Services) service.Lifecycle
}
