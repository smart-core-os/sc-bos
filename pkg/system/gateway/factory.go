// Package gateway is a system that allows a node to proxy the APIs of other nodes.
// The gateway system is closely tied with the hub and cohort concepts, using these to discover which nodes to proxy.
//
// # Types Of Gateway Proxying
//
// There are two types of proxying that this system does: routed and node APIs
//
// ## Routed APIs
//
// For any API that uses a name to direct API calls to the correct target, the gateway system will maintain a table of
// name->target mappings and when a request comes in for a name, it will forward that request to the correct target.
// Examples of this kind of API are Smart Core traits, or other APIs that are announced as part of a node.
//
// Routed APIs are handled in a generic way, using node metadata to construct the routing table and general trait patterns
// to decide how to route the request.
// The gateway can route any API that the remote node advertises via the gRPC reflection API.
//
// A special case for routed APIs is the ServiceApi.
// This API is typically advertised using common names: "drivers", "automations", etc.
// as such we can't blindly route "drivers" to any specific node as the names collide.
// To solve this the gateway advertises them using modified names of the form `{node name}/{original name}`.
//
// ## Node APIs
//
// These typically do not include a name, instead targeting the node itself as the party to respond to the request.
// The hub is generally the primary source of these APIs though other nodes can also implement them.
// If more than one remote node advertises a node API, the gateway will route all requests to the first node it sees.
// Some examples of this kind of API are the services API, the history admin API, and the tenant API.
//
// Node APIs are handled generically using gRPC reflection to discover available APIs.
// Any node API this node implements via non-gateway mechanisms will take precedence over the gateway, for example the DevicesApi.
//
// # Cohorts with Multiple Gateways
//
// If the gateway discovers that a remote node is also a gateway then it will avoid re-announcing any routed or node APIs
// instead relying on discovery of these APIs from the original remote node.
package gateway

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/config"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const (
	Name       = "gateway"
	LegacyName = "proxy"
)

func Factory() system.Factory {
	return &factory{}
}

type factory struct{}

func (f *factory) New(services system.Services) service.Lifecycle {
	s := &System{
		self:   services.Node,
		hub:    services.CohortManager,
		checks: services.HealthChecks,
		ignore: []string{services.GRPCEndpoint}, // avoid infinite recursion
		newClient: func(address string) (*grpc.ClientConn, error) {
			return grpc.NewClient(address, grpc.WithTransportCredentials(credentials.NewTLS(services.ClientTLSConfig)))
		},
		reflection: services.ReflectionServer,
		announcer:  services.Node,
		logger:     services.Logger.Named(Name),
	}
	return service.New(service.MonoApply(s.applyConfig))
}

type System struct {
	self       *node.Node
	hub        node.Remote
	checks     system.HealthCheckCollection
	ignore     []string
	reflection *reflectionapi.Server
	announcer  node.Announcer
	logger     *zap.Logger
	// for testing
	newClient func(address string) (*grpc.ClientConn, error)
}

// applyConfig runs this system based on the given config.
// This will query the hub for nodes,
// for each node (not ignored) it will query for all devices,
// announcing metadata for each.
func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	s.logger.Debug("applying config", zap.Any("config", cfg))
	ignore := append([]string{}, s.ignore...)
	ignore = append(ignore, cfg.Ignore...)

	if cfg.HubMode == "" {
		cfg.HubMode = config.HubModeRemote
	}

	c := newCohort(ignore...)

	switch cfg.HubMode {
	case config.HubModeRemote:
		hubConn, err := s.hub.Connect(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialise client connection to the hub: %w", err)
		}
		go s.scanRemoteHub(ctx, c, hubConn)
	case config.HubModeLocal:
		hubClient := gen.NewHubApiClient(s.self.ClientConn())
		go s.scanLocalHub(ctx, c, hubClient)
	}

	go s.announceCohort(ctx, c)

	return nil
}

func (s *System) retry(ctx context.Context, name string, t task.Task, logFields ...zap.Field) error {
	attempt := 0
	logger := s.logger
	if name != "" {
		logger = logger.With(zap.String("task", name))
	}
	logger = logger.With(logFields...)
	return task.Run(ctx, func(taskCtx context.Context) (task.Next, error) {
		attempt++
		next, err := t(taskCtx)
		if next == task.ResetBackoff {
			// assume some success happened, reset err and attempts
			attempt = 1
		}

		if err == nil {
			return next, err
		}
		if ctx.Err() != nil {
			// s.logger.Debug("task aborted", zap.String("task", name), zap.Error(err), zap.Int("attempt", attempt))
			return task.StopNow, err // this doesn't matter as the task runner will not retry when ctx is done
		}

		switch {
		case attempt == 1:
			logger.Warn("task failed, will retry", zap.Error(err), zap.Int("attempt", attempt))
		case attempt == 5:
			logger.Warn("task failed, reducing logging", zap.Error(err), zap.Int("attempt", attempt))
		case attempt%10 == 0:
			logger.Debug("task failed", zap.Error(err), zap.Int("attempt", attempt))
		}

		return next, err
	}, task.WithRetry(task.RetryUnlimited), task.WithBackoff(10*time.Millisecond, 30*time.Second))
}

// poll calls t every 10 seconds until ctx is done.
// If t returns an error an exponential backoff mode will be entered scaling from 10ms to 30s.
// Logging is performed on the 1sh, 5th, and every 10th attempt that fails,
// as well as the first attempt that succeeds after an error.
func (s *System) poll(ctx context.Context, t func(context.Context) error, logFields ...zap.Field) error {
	logger := s.logger.With(logFields...)
	var lastState task.PollState
	return task.PollErr(t,
		task.WithPollInterval(10*time.Second),
		task.WithPollErrBackoff(10*time.Millisecond, 30*time.Second, 1.5),
		task.WithPollAttemptCallback(func(state task.PollState, err error) bool {
			defer func() { lastState = state }()
			if err == nil {
				// only print success if the last attempt succeeded after an error
				if lastState.NextDelay == 0 && state.SuccessesSinceError == 1 && lastState.ErrorsSinceSuccess > 0 {
					logger.Debug("poll task is now succeeding", zap.Int("attempts", lastState.ErrorsSinceSuccess))
				}
				return false
			}

			// similar logic to the retry code above
			attempt := state.ErrorsSinceSuccess
			switch {
			case attempt == 1:
				logger.Warn("poll failed, will retry", zap.Error(err), zap.Int("attempt", attempt), zap.Duration("nextAttempt", state.NextDelay))
			case attempt == 5:
				logger.Warn("poll failed, reducing logging", zap.Error(err), zap.Int("attempt", attempt), zap.Duration("nextAttempt", state.NextDelay))
			case attempt%10 == 0:
				logger.Debug("poll failed", zap.Error(err), zap.Int("attempt", attempt), zap.Duration("nextAttempt", state.NextDelay))
			}

			return false
		}),
	).Attach(ctx)
}
