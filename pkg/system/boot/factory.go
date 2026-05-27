// Package boot implements the Boot trait for the sc-bos process itself.
// When enabled, it records the process boot time and handles reboot requests by
// calling os.Exit(0), relying on the process supervisor (e.g. systemd) to restart it.
package boot

import (
	"context"
	"os"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bootpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// bootTime is captured at package init to record the process start time as accurately as possible.
var bootTime = time.Now()

var Factory factory

type factory struct{}

func (factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

// config is empty because no configuration is required for this system plugin.
type config struct{}

// System implements the Boot trait for this sc-bos process.
type System struct {
	*service.Service[config]

	nodeName      string
	announcer     *node.ReplaceAnnouncer
	logger        *zap.Logger
	dataDir       string
	requestReboot func()
}

func NewSystem(services system.Services) *System {
	s := &System{
		nodeName:      services.Node.Name(),
		announcer:     node.NewReplaceAnnouncer(services.Node),
		logger:        services.Logger.Named("boot"),
		dataDir:       services.DataDir,
		requestReboot: services.RequestReboot,
	}
	s.Service = service.New(service.MonoApply(s.applyConfig))
	return s
}

func (s *System) applyConfig(ctx context.Context, _ config) error {
	announcer := s.announcer.Replace(ctx)

	// Load persisted state from before the last restart.
	var lastReason string
	var lastActor *actorpb.Actor
	if st, err := ReadStateFile(s.dataDir); err == nil {
		if !st.CleanExit {
			// File exists but clean-exit flag was never set — previous run crashed.
			lastReason = "unexpected process exit"
		} else {
			lastReason = st.Reason
			if len(st.Actor) > 0 {
				lastActor = &actorpb.Actor{}
				if err := protojson.Unmarshal(st.Actor, lastActor); err != nil {
					s.logger.Warn("failed to unmarshal persisted actor", zap.Error(err))
					lastActor = nil
				}
			}
		}
	}

	// Mark this run as in-progress; if we crash the next startup will see CleanExit=false.
	if err := WriteStateFile(s.dataDir, RebootState{}); err != nil {
		s.logger.Warn("failed to write reboot state", zap.Error(err))
	}

	model := bootpb.NewModel(resource.WithInitialValue(&bootpb.BootState{
		BootTime:         timestamppb.New(bootTime),
		LastRebootReason: lastReason,
		LastRebootActor:  lastActor,
	}))

	server := bootpb.NewModelServer(model)
	server.OnReboot = s.onReboot

	// In-memory store for history within the current session.
	// If the operator configures a `history` auto with trait=smartcore.bos.Boot,
	// that provides persistent cross-restart history backed by a durable store.
	store := memstore.New()
	histServer := historypb.NewBootServer(store)

	announcer.Announce(s.nodeName,
		node.HasServer(bootpb.RegisterBootApiServer, bootpb.BootApiServer(server)),
		node.HasServer(bootpb.RegisterBootHistoryServer, bootpb.BootHistoryServer(histServer)),
		node.HasTrait(bootpb.TraitName),
	)

	// Block until shutdown. The clean-exit state file write is handled by
	// Controller.Run's defer, which covers both graceful shutdown and deployment
	// restarts. onReboot handles the Boot RPC path (writing reason + actor).
	<-ctx.Done()
	return nil
}

// onReboot is called by the ModelServer when a Reboot RPC is received.
// It records the event then schedules a clean process exit.
func (s *System) onReboot(_ context.Context, req *bootpb.RebootRequest) error {
	st := RebootState{Reason: req.Reason, CleanExit: true}
	if req.Actor != nil {
		actorJSON, err := protojson.Marshal(req.Actor)
		if err != nil {
			s.logger.Warn("failed to marshal reboot actor", zap.Error(err))
		} else {
			st.Actor = actorJSON
		}
	}

	// Persist reason and actor so they survive the restart.
	if err := WriteStateFile(s.dataDir, st); err != nil {
		s.logger.Warn("failed to persist reboot state", zap.Error(err))
	}

	s.logger.Info("reboot requested",
		zap.String("reason", req.Reason),
		zap.String("actor", req.GetActor().GetDisplayName()),
	)

	if req.Force {
		go func() {
			// Give the gRPC response time to be flushed to the client before
			// we kill the connection by exiting. 100ms is generous for a local
			// write; callers should treat a dropped connection as success anyway.
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	} else {
		s.requestReboot()
	}

	return nil
}

