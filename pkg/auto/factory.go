package auto

import (
	"crypto/tls"
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
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
	// CloudCredential provides the node's Connect leaf certificate for mTLS to the
	// telemetry broker, plus the node identity. It is nil until the cloud leaf
	// credential is wired (PR #890); automations that need it must fall back or
	// error clearly when it is absent.
	CloudCredential CloudCredentialSource
	Now             func() time.Time
	Config          service.ConfigUpdater
	Health          *healthpb.Checks
}

// CloudCredentialSource exposes the node's current Connect leaf certificate and
// identity for authenticating to the Connect telemetry (Event Grid MQTT) broker.
// It is expected to be satisfied by the cloud connection once the leaf credential
// lands (PR #890); GetClientCertificate must reflect credential renewals live so
// callers can install it directly as tls.Config.GetClientCertificate.
type CloudCredentialSource interface {
	GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error)
	// NodeID returns the SCC node id (the leaf Subject CN), stable across renewals.
	NodeID() string
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
