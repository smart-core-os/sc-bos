package statusalerts

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/statusalerts/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

const AutoName = "statusalerts"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &autoImpl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

type autoImpl struct {
	*service.Service[config.Root]
	auto.Services
}

func (a *autoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	logger := a.Logger
	if cfg.Destination == "" {
		cfg.Destination = a.Node.Name()
	}
	destName := cfg.Destination
	logger = logger.With(zap.String("destination", destName))

	alertAdminClient := alertpb.NewAlertAdminApiClient(a.Node.ClientConn())
	statusClient := gen_statuspb.NewStatusApiClient(a.Node.ClientConn())

	if cfg.DelayStart != nil {
		time.Sleep(cfg.DelayStart.Duration)
	}

	var tasks namedTasks
	pullFrom := func(source config.Source) {
		logger := logger.With(zap.String("name", source.Name))
		err := tasks.Run(ctx, source.Name, tasksForSource(source, destName, statusClient, alertAdminClient, logger),
			task.WithRetry(task.RetryUnlimited), task.WithBackoff(time.Millisecond*100, time.Second*10))
		if errors.Is(err, ErrAlreadyRunning) {
			// cool, I guess someone else beat us to it
			return
		}
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("shut down unexpectedly", zap.Error(err))
			return
		}
		logger.Debug("subscription stopped")
	}

	// setup manually configured sources
	for _, name := range cfg.Sources {
		go pullFrom(name)
	}

	// setup discovered sources
	if cfg.DiscoverSources {
		go func() {
			// we use a replacer here because it's an easy and memory efficient way to do prefix matching.
			// strings.Replacer uses a Trie internally.
			ignore := func() *strings.Replacer {
				replacements := make([]string, len(cfg.IgnorePrefixes)*2)
				for _, prefix := range cfg.IgnorePrefixes {
					replacements = append(replacements, prefix, "!")
				}
				return strings.NewReplacer(replacements...)
			}()

			for { // loop in case we get errors
				select {
				case <-ctx.Done():
					return
				default:
				}
				for change := range a.Node.PullDevices(ctx, resource.WithReadPaths(&devicespb.Device{}, "metadata.traits", "metadata.location", "metadata.membership")) {
					if s := ignore.Replace(change.Id); len(s) == 0 || s[0] == '!' {
						continue // ignore
					}
					hadTrait, hasTrait := hasStatusTrait(change.OldValue), hasStatusTrait(change.NewValue)
					switch {
					case hadTrait && !hasTrait: // remove
						err := tasks.Stop(change.Id)
						if err != nil && !errors.Is(err, ErrNotRunning) {
							logger.Debug("error during stop", zap.String("name", change.Id), zap.Error(err))
						}
					case !hadTrait && hasTrait: // add
						source := config.Source{
							Name:      change.Id,
							Floor:     change.NewValue.GetMetadata().GetLocation().GetFloor(),
							Zone:      change.NewValue.GetMetadata().GetLocation().GetZone(),
							Subsystem: change.NewValue.GetMetadata().GetMembership().GetSubsystem(),
						}
						go pullFrom(source)
					}
				}
			}
		}()
	}

	return nil
}

func hasStatusTrait(device *devicespb.Device) bool {
	md := device.GetMetadata()
	for _, t := range md.GetTraits() {
		if t.Name == statuspb.TraitName.String() {
			return true
		}
	}
	return false
}
