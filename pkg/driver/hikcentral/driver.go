package hikcentral

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/ptzpb"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
)

const DriverName = "hikcentral"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
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
	logger    *zap.Logger
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	rootAnnouncer := d.announcer.Replace(ctx)
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
		rootAnnouncer.Announce(camera.Name,
			node.HasMetadata(camera.Metadata),
			node.HasMetadata(camera.Metadata),
			node.HasClient(gen.WrapMqttService(cam)),
			node.HasTrait(statuspb.TraitName, node.WithClients(gen.WrapStatusApi(cam))),
			node.HasTrait(trait.Ptz, node.WithClients(ptzpb.WrapApi(cam))),
			node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(cam))),
		)
		cameras = append(cameras, cam)
	}

	run(ctx, cameras, cfg, grp)

	go func() {
		err := grp.Wait()
		logger.Error("run error", zap.String("error", err.Error()))
		client.HTTPClient.CloseIdleConnections()
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
