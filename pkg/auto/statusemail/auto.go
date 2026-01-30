package statusemail

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/statusemail/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

const AutoName = "statusemail"

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
	logger = logger.With(zap.String("snmp.host", cfg.Destination.Host), zap.Int("snmp.port", cfg.Destination.Port))

	statusClient := gen_statuspb.NewStatusApiClient(a.Node.ClientConn())

	if cfg.DelayStart != nil {
		time.Sleep(cfg.DelayStart.Duration)
	}

	changes := make(chan change, 10)
	var activePullers sync.WaitGroup

	var tasks namedTasks
	pullFrom := func(source config.Source) {
		defer activePullers.Done() // let someone know we won't be sending to changes anymore

		logger := logger.With(zap.String("name", source.Name))
		err := tasks.Run(ctx, source.Name, tasksForSource(source, statusClient, changes, logger),
			task.WithRetry(task.RetryUnlimited), task.WithBackoff(time.Millisecond*100, time.Second*10))
		if errors.Is(err, ErrAlreadyRunning) {
			// cool, I guess someone else beat us to it
			return
		}
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("shut down unexpectedly", zap.Error(err))
			return
		}
	}

	// setup manually configured sources
	activePullers.Add(len(cfg.Sources))
	for _, name := range cfg.Sources {
		go pullFrom(name)
	}

	// setup discovered sources
	if cfg.DiscoverSources {
		// Force the counter to be non-zero so Wait and Add don't race.
		// See sync.WaitGroup docs for reasons, specifically the docs for Add.
		activePullers.Add(1)
		go func() {
			defer activePullers.Done()
			// we use a replacer here because it's an easy and memory efficient way to do prefix matching.
			// strings.Replacer uses a Trie internally.
			// This replacer replaces any matching prefix with a !, to check if we should ignore a name check s[0] == '!'
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
				for change := range a.Node.PullDevices(ctx, resource.WithReadPaths(&devicespb.Device{}, "metadata.traits", "metadata.appearance", "metadata.location", "metadata.membership")) {
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
							Title:     change.NewValue.GetMetadata().GetAppearance().GetTitle(),
							Floor:     change.NewValue.GetMetadata().GetLocation().GetFloor(),
							Zone:      change.NewValue.GetMetadata().GetLocation().GetZone(),
							Subsystem: change.NewValue.GetMetadata().GetMembership().GetSubsystem(),
						}
						activePullers.Add(1)
						go pullFrom(source)
					}
				}
			}
		}()
	}

	go func() {
		activePullers.Wait()
		close(changes)
	}()

	// returns when changes is closed
	go sendEmailOnChange(cfg.Destination, changes, logger)

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
