// Package system and sub packages add optional features to a controller.
package system

import (
	"crypto/tls"
	"net/http"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/vanti-dev/sc-bos/internal/account"
	"github.com/vanti-dev/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/vanti-dev/sc-bos/internal/util/pki"
	"github.com/vanti-dev/sc-bos/pkg/app/stores"
	"github.com/vanti-dev/sc-bos/pkg/auth/token"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/node"
	"github.com/vanti-dev/sc-bos/pkg/task/service"
)

type Services struct {
	ConfigDirs      []string
	DataDir         string
	Logger          *zap.Logger
	GRPCEndpoint    string     // host:port of this controllers grpc api
	Node            *node.Node // for advertising devices
	HealthChecks    HealthCheckCollection
	CohortManager   node.Remote
	Database        *bolthold.Store
	Stores          *stores.Stores
	Accounts        *account.Store
	HTTPMux         *http.ServeMux      // to allow systems to serve http requests
	TokenValidators *token.ValidatorSet // to allow systems to contribute towards client validation

	ReflectionServer *reflectionapi.Server // to allow systems to contribute types towards the reflection api

	// GRPCCerts allows a system to contribute a pki.Source that might be used for outbound or inbound gRPC connections.
	// These certs will be used only if no other certificate mechanism is in effect, for example if the controller is
	// enrolled in a cohort then the cohort certificates will be used,
	// if the controller has been configured to read certificates from a file then they will be used.
	// These certificates get used in preference to self signed certificates only.
	GRPCCerts       *pki.SourceSet
	PrivateKey      pki.PrivateKey // the key managed by the controller
	ClientTLSConfig *tls.Config    // for connecting to other smartcore nodes
}

type Factory interface {
	New(services Services) service.Lifecycle
}

type FactoryFunc func(services Services) service.Lifecycle

func (f FactoryFunc) New(services Services) service.Lifecycle {
	return f(services)
}

// HealthCheckCollection allows the modification of health checks for named devices.
type HealthCheckCollection interface {
	// MergeHealthChecks adds or updates checks for name based on existing check ids.
	MergeHealthChecks(name string, checks ...*gen.HealthCheck) error
	// RemoveHealthChecks removes any present ids from names checks.
	RemoveHealthChecks(name string, ids ...string) error
}
