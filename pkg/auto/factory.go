package auto

import (
	"crypto/tls"
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

type Services struct {
	Logger          *zap.Logger
	Node            *node.Node // for advertising devices
	Devices         devicespb.DevicesApiClient
	Database        *bolthold.Store
	Stores          *stores.Stores
	GRPCServices    grpc.ServiceRegistrar // for registering non-routed services
	CohortManager   node.Remote
	ClientTLSConfig *tls.Config
	Now             func() time.Time
	Config          service.ConfigUpdater
	Health          *healthpb.Checks
}

// Factory constructs new automation instances.
type Factory interface {
	// note this is an interface, not a func type so that the controller can check for other interfaces, like GrpcApi.

	New(services Services) service.Lifecycle
}

type FactoryFunc func(services Services) service.Lifecycle

func (f FactoryFunc) New(services Services) service.Lifecycle {
	return f(services)
}
