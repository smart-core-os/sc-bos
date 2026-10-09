package auto

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/connect"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
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
	// telemetry broker, plus the node identity. It is supplied by the node's cloud
	// connection and is nil only when the host has none; automations that need it
	// must fall back or error clearly when it is absent. When present the node may
	// not be enrolled yet, and can be enrolled, re-enrolled or unlinked at any time -
	// see connect.Credential for how long-lived connections should follow that.
	CloudCredential CloudCredentialSource
	Now             func() time.Time
	Config          service.ConfigUpdater
	Health          *healthpb.Checks
	// LocalHealthChecks streams the health checks this node evaluates itself, one device per name.
	// It never includes checks the node only re-announces, such as a gateway's cohort checks,
	// which Devices does include. Nil if unavailable.
	LocalHealthChecks HealthCheckSource
	// Auditor records writes an automation accepts from its own ingress, such as an MQTT
	// subscription, which the gRPC/HTTP audit interceptors never see.
	// It may be nil, and is a no-op when no audit log is configured.
	Auditor Auditor
}

// HealthCheckSource provides devices carrying health checks, see Services.LocalHealthChecks.
type HealthCheckSource interface {
	PullDevices(ctx context.Context, opts ...resource.ReadOption) <-chan devicespb.DevicesChange
}

// CloudCredentialSource is an alias for connect.Credential, retained so existing
// automations continue to compile. Prefer connect.Credential in new code.
type CloudCredentialSource = connect.Credential

// Auditor records writes that an automation accepts from its own ingress (e.g. MQTT).
type Auditor interface{ AuditIngress(policy.IngressEntry) }

// Factory constructs new automation instances.
type Factory interface {
	// note this is an interface, not a func type so that the controller can check for other interfaces, like GrpcApi.

	New(services Services) service.Lifecycle
}

type FactoryFunc func(services Services) service.Lifecycle

func (f FactoryFunc) New(services Services) service.Lifecycle {
	return f(services)
}
