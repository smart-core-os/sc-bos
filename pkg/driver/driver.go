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
	// SystemCheck is a persistent driver-level health check for top-level connectivity state
	// (server reachable, licence valid, etc.). It is created by the app layer, survives
	// applyConfig retries, and is disposed automatically when the service stops.
	//
	// Pass it to service.WithSystemCheck — the framework then calls MarkFailed/MarkRunning
	// automatically based on applyConfig return value. Drivers do not need to call Dispose.
	SystemCheck service.SystemCheck
}

type Factory interface {
	New(services Services) service.Lifecycle
}
