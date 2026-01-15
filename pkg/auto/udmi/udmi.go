package udmi

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	gen_udmipb "github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

const AutoType = "udmi"

var Factory auto.Factory = factory{}

type factory struct{}

func (_ factory) New(services auto.Services) service.Lifecycle {
	return NewUDMI(services)
}

func NewUDMI(services auto.Services) service.Lifecycle {
	logger := services.Logger.Named(AutoType)
	e := &udmiAuto{services: services}
	e.Service = service.New(
		service.MonoApply(e.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", logger)
		})),
	)
	e.services.Logger = services.Logger.Named(AutoType)
	return e
}

type udmiAuto struct {
	*service.Service[config.Root]
	services auto.Services
}

func (e *udmiAuto) applyConfig(ctx context.Context, cfg config.Root) error {
	udmiClient := gen_udmipb.NewUdmiServiceClient(e.services.Node.ClientConn())

	client, err := newMqttClient(cfg)
	if err != nil {
		return err
	}

	pubSub := &PubSub{
		Publisher:  mqttPublisher(client, 0, false),
		Subscriber: mqttSubscriber(client, 0),
	}

	connected := client.Connect()
	connected.Wait()
	if connected.Error() != nil {
		return connected.Error()
	}
	e.services.Logger.Debug("connected")

	go func() {
		<-ctx.Done()
		client.Disconnect(5000)
	}()

	var tasks namedTasks
	pullFrom := func(name string) {
		logger := e.services.Logger.With(zap.String("name", name))
		err := tasks.Run(ctx, name, tasksForSource(name, logger, udmiClient, pubSub),
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
			for { // loop in case we get errors
				select {
				case <-ctx.Done():
					return
				default:
				}
				for change := range e.services.Node.PullDevices(ctx, resource.WithReadPaths(&devicespb.Device{}, "metadata.traits")) {
					hadTrait, hasTrait := hasUDMITrait(change.OldValue), hasUDMITrait(change.NewValue)
					if hadTrait && !hasTrait {
						// remove
						err := tasks.Stop(change.Id)
						if err != nil && !errors.Is(err, ErrNotRunning) {
							e.services.Logger.Debug("error during stop", zap.String("name", change.Id), zap.Error(err))
						}
					}
					if !hadTrait && hasTrait {
						// add
						go pullFrom(change.Id)
					}
				}
			}
		}()
	}

	return nil
}

func hasUDMITrait(device *devicespb.Device) bool {
	md := device.GetMetadata()
	for _, t := range md.GetTraits() {
		if t.Name == udmipb.TraitName.String() {
			return true
		}
	}
	return false
}

type namedTasks struct {
	mu         sync.Mutex
	stopByName map[string]taskRuntime
}

var (
	ErrAlreadyRunning = errors.New("already running")
	ErrNotRunning     = errors.New("not running")
)

func (s *namedTasks) Run(ctx context.Context, name string, tasks []task.Task, opts ...task.Option) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	id := &ctx

	s.mu.Lock()
	if s.stopByName == nil {
		s.stopByName = make(map[string]taskRuntime)
	}

	_, ok := s.stopByName[name]
	if ok {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.stopByName[name] = taskRuntime{stop, id}
	s.mu.Unlock()

	defer func() {
		// cleanup
		s.mu.Lock()
		defer s.mu.Unlock()
		rt, ok := s.stopByName[name]
		if ok && rt.id == id {
			delete(s.stopByName, name)
		}
	}()

	group, ctx := errgroup.WithContext(ctx)
	for _, t := range tasks {
		t := t
		group.Go(func() error {
			return task.Run(ctx, t, opts...)
		})
	}
	return group.Wait()
}

func (s *namedTasks) Stop(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt, ok := s.stopByName[name]
	if !ok {
		return ErrNotRunning
	}
	rt.stop()
	delete(s.stopByName, name)
	return nil
}

type taskRuntime struct {
	stop func()
	id   *context.Context
}
