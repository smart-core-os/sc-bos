package hikcentral

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	gen_udmipb "github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/ptzpb"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
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
	for _, camera := range cfg.Cameras {
		logger := logger.With(zap.String("device", camera.Name))

		faultCheck, err := d.health.NewFaultCheck(camera.Name, deviceHealthCheck)
		if err != nil {
			return err
		}

		cam := NewCamera(client, logger, camera, faultCheck)
		rootAnnouncer.Announce(camera.Name,
			node.HasMetadata(camera.Metadata),
			node.HasClient(mqttpb.WrapService(cam)),
			node.HasTrait(trait.Ptz, node.WithClients(ptzpb.WrapApi(cam))),
			node.HasTrait(udmipb.TraitName, node.WithClients(gen_udmipb.WrapService(cam))),
		)
		cameras = append(cameras, cam)
	}

	run(ctx, cameras, cfg, grp)

	go func() {
		err := grp.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("unexpected error in polling goroutines", zap.Error(err))
		}
		client.httpClient.CloseIdleConnections()
	}()
	return nil
}

func run(ctx context.Context, cameras []*Camera, cfg config.Root, grp *errgroup.Group) {

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
