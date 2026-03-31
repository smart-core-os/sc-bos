package hikcentral

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const DriverName = "hikcentral"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		health:    services.Health,
		logger:    services.Logger.Named(DriverName),
	}

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

	announcer *node.ReplaceAnnouncer
	health    *healthpb.Checks
	logger    *zap.Logger
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	rootAnnouncer := d.announcer.Replace(ctx)
	logger := d.logger.With(zap.String("host", cfg.API.Address))

	client := newClient(cfg.API)
	client.httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	grp, ctx := errgroup.WithContext(ctx)
	var cameras []*Camera
	var faultChecks []*healthpb.FaultCheck
	for _, camera := range cfg.Cameras {
		logger := logger.With(zap.String("device", camera.Name))

		faultCheck, err := d.health.NewFaultCheck(camera.Name, proto.Clone(deviceHealthCheck).(*healthpb.HealthCheck))
		if err != nil {
			return err
		}
		faultChecks = append(faultChecks, faultCheck)

		cam := NewCamera(client, logger, camera, faultCheck)
		rootAnnouncer.Announce(camera.Name,
			node.HasMetadata(camera.Metadata),
			node.HasClient(mqttpb.WrapService(cam)),
			node.HasTrait(trait.Ptz, node.WithClients(ptzpb.WrapApi(cam))),
			node.HasTrait(udmipb.TraitName, node.WithClients(udmipb.WrapService(cam))),
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

			resources[anpr.Name] = resource.NewValue(resource.WithInitialValue(&accesspb.AccessAttempt{}), resource.WithNoDuplicates())

			rootAnnouncer.Announce(anpr.Name,
				node.HasMetadata(anpr.Metadata),
				node.HasTrait(accesspb.TraitName, node.WithClients(accesspb.WrapApi(ctrl))))
		}

		if cfg.GrantManagement != nil {
			rootAnnouncer.Announce(cfg.GrantManagement.Name,
				node.HasMetadata(cfg.GrantManagement.Metadata),
				node.HasTrait(accesspb.TraitName, node.WithClients(accesspb.WrapApi(ctrl))),
			)
		}
	}

	run(ctx, ctrl, cameras, cfg, grp, logger)

	go func() {
		err := grp.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("unexpected error in polling goroutines", zap.Error(err))
		}
		for _, fc := range faultChecks {
			fc.Dispose()
		}
		client.httpClient.CloseIdleConnections()
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
