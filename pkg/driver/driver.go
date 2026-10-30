package driver

import (
	"crypto/tls"
	"net/http"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/connect"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

type Services struct {
	Logger          *zap.Logger
	Node            *node.Node  // for advertising devices
	ClientTLSConfig *tls.Config // for connecting to other smartcore nodes
	// CloudCredential is the node's Smart Core Connect identity, for mTLS to the
	// Connect API. Distinct from ClientTLSConfig, which is the cohort identity and
	// will not authenticate to Connect.
	//
	// Nil only when the host has no cloud connection; drivers must degrade rather
	// than fail when absent. When present the node may not be enrolled yet, and
	// can be enrolled, re-enrolled or unlinked at any time - see connect.Credential
	// for how long-lived connections should follow that.
	CloudCredential connect.Credential
	HTTPMux         *http.ServeMux
	Config          service.ConfigUpdater
	Database        *bolthold.Store
	Health          *healthpb.Checks
	// SystemCheck is the service's top-level health check. The framework marks it failed when the service fails
	// to start or apply its config, and running once its config has loaded. Call MarkFailed when the service
	// can't do its job for some other reason (the server is unreachable, the licence is invalid) and MarkRunning when it can again.
	// The handle stays valid across stops and starts; there is no need to Dispose it.
	// May be nil, for example when the service is hosted by a zone; nil-check before use.
	SystemCheck service.SystemCheck
}

type Factory interface {
	New(services Services) service.Lifecycle
}

type FactoryFunc func(services Services) service.Lifecycle

func (f FactoryFunc) New(services Services) service.Lifecycle {
	return f(services)
}
