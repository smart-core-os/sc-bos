package hikcentral

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/ptzpb"
	"github.com/vanti-dev/sc-bos/pkg/driver"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/accesspb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/statuspb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/udmipb"
	"github.com/vanti-dev/sc-bos/pkg/node"
	"github.com/vanti-dev/sc-bos/pkg/task/service"

	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/api"
	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/config"
)

const DriverName = "hikcentral"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer: services.Node,
	}
	d.logger = services.Logger.Named(DriverName)
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ReadBytes),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", d.logger)
		}), service.RetryWithMinDelay(5*time.Second), service.RetryWithInitialDelay(5*time.Second)),
	)

	return d
}

type Driver struct {
	*service.Service[config.Root]
	logger    *zap.Logger
	announcer node.Announcer
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	// AnnounceContext only makes sense if using MonoApply, which we are in New
	announcer, undo := node.AnnounceScope(d.announcer)
	logger := d.logger.With(zap.String("host", cfg.API.Address))

	client := api.NewClient(cfg.API)
	client.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	grp, ctx := errgroup.WithContext(ctx)
	var cameras []*Camera
	for _, camera := range cfg.Cameras {
		logger := logger.With(zap.String("device", camera.Name))
		cam := NewCamera(client, logger, camera)
		announcer.Announce(camera.Name,
			node.HasMetadata(camera.Metadata),
			node.HasClient(gen.WrapMqttService(cam)),
			node.HasTrait(statuspb.TraitName, node.WithClients(gen.WrapStatusApi(cam))),
			node.HasTrait(trait.Ptz, node.WithClients(ptzpb.WrapApi(cam))),
			node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(cam))),
		)
		cameras = append(cameras, cam)
	}

	var ctrl *ANPRController
	if cfg.GrantManagement != nil || len(cfg.ANPRCameras) > 0 {
		resources := make(map[string]*resource.Value)
		ctrl = NewANPRController(client, &cfg, resources, logger)

		for _, anpr := range cfg.ANPRCameras {
			if _, ok := resources[anpr.Name]; ok {
				logger.Warn("ANPR resource already exists, skipping", zap.String("name", anpr.Name))
				continue
			}

			resources[anpr.Name] = resource.NewValue(resource.WithInitialValue(&gen.AccessAttempt{}), resource.WithNoDuplicates())

			announcer.Announce(anpr.Name,
				node.HasMetadata(anpr.Metadata),
				node.HasTrait(accesspb.TraitName, node.WithClients(gen.WrapAccessApi(ctrl))))
		}

		if cfg.GrantManagement != nil {
			announcer.Announce(cfg.GrantManagement.Name,
				node.HasMetadata(cfg.GrantManagement.Metadata),
				node.HasTrait(accesspb.TraitName, node.WithClients(gen.WrapAccessApi(ctrl))),
			)
		}
	}

	run(ctx, ctrl, cameras, cfg, grp, logger)

	go func() {
		err := grp.Wait()
		logger.Error("run error", zap.String("error", err.Error()))
		undo()
	}()
	return nil
}

func run(ctx context.Context, ctrl *ANPRController, cameras []*Camera, cfg config.Root, grp *errgroup.Group, logger *zap.Logger) {
	if cfg.Settings.InfoPoll != nil {
		grp.Go(func() error {
			t := newTickerWithCtx(ctx, cfg.Settings.InfoPoll.Duration)
			for range t {
				for _, c := range cameras {
					c.getInfo(ctx)
				}
			}
			return ctx.Err()
		})
	}

	if cfg.Settings.OccupancyPoll != nil {
		grp.Go(func() error {
			t := newTickerWithCtx(ctx, cfg.Settings.OccupancyPoll.Duration)
			for range t {
				for _, c := range cameras {
					c.getOcc(ctx)
				}
			}
			return ctx.Err()
		})
	}

	if cfg.Settings.EventsPoll != nil {
		grp.Go(func() error {
			t := newTickerWithCtx(ctx, cfg.Settings.EventsPoll.Duration)
			for range t {
				for _, c := range cameras {
					c.getEvents(ctx)
				}
			}
			return ctx.Err()
		})
	}

	if cfg.Settings.StreamPoll != nil {
		grp.Go(func() error {
			t := newTickerWithCtx(ctx, cfg.Settings.StreamPoll.Duration)
			for range t {
				for _, c := range cameras {
					c.getStream(ctx)
				}
			}
			return ctx.Err()
		})
	}

	if ctrl != nil {
		grp.Go(func() error {
			t := newTickerWithCtx(ctx, cfg.Settings.ANPREventsPoll.Or(5*time.Minute))

			for range t {
				if err := ctrl.poll(ctx); err != nil {
					logger.Error("failed to poll anpr controller", zap.Error(err))
					continue
				}
			}
			return nil
		})
	}
}

func newTickerWithCtx(ctx context.Context, dur time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1) // same buffer as time.NewTicker
	t := time.NewTicker(dur)
	go func() {
		defer func() {
			t.Stop()
			close(ch)
		}()
		for {
			select {
			case t := <-t.C:
				ch <- t
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
