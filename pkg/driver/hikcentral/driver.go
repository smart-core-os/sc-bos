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
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const DriverName = "hikcentral"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(services.Node),
		health:      services.Health,
		logger:      services.Logger.Named(DriverName),
		systemCheck: services.SystemCheck,
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

	announcer   *node.ReplaceAnnouncer
	health      *healthpb.Checks
	logger      *zap.Logger
	systemCheck service.SystemCheck
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	rootAnnouncer := d.announcer.Replace(ctx)
	logger := d.logger.With(zap.String("host", cfg.API.Address))

	client := newClient(cfg.API, d.systemCheck)
	client.httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	grp, ctx := errgroup.WithContext(ctx)
	cycles := &pollCycles{}
	var cameras []*Camera
	var faultChecks []*healthpb.FaultCheck
	for _, camera := range cfg.Cameras {
		logger := logger.With(zap.String("device", camera.Name))

		faultCheck, err := d.health.NewFaultCheck(camera.Name, proto.Clone(deviceHealthCheck).(*healthpb.HealthCheck))
		if err != nil {
			return err
		}
		faultChecks = append(faultChecks, faultCheck)

		cam := NewCamera(client, logger, camera, faultCheck, cfg.Settings, cycles)
		rootAnnouncer.Announce(camera.Name,
			node.HasMetadata(camera.Metadata),
			node.HasServer(mqttpb.RegisterMqttServiceServer, mqttpb.MqttServiceServer(cam)),
			node.HasServer(ptzpb.RegisterPtzApiServer, ptzpb.PtzApiServer(cam)),
			node.HasTrait(trait.Ptz),
			node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(cam)),
			node.HasTrait(udmipb.TraitName),
			node.HasDeviceType(metadatapb.Metadata_DEVICE),
		)
		cameras = append(cameras, cam)
	}

	run(ctx, cameras, cfg, grp, cycles)

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

func run(ctx context.Context, cameras []*Camera, cfg config.Root, grp *errgroup.Group, cycles *pollCycles) {
	for _, pl := range []struct {
		p     poll
		every *jsontypes.Duration
		visit func(*Camera, context.Context)
	}{
		{infoPoll, cfg.Settings.InfoPoll, (*Camera).getInfo},
		{occupancyPoll, cfg.Settings.OccupancyPoll, (*Camera).getOcc},
		{eventsPoll, cfg.Settings.EventsPoll, (*Camera).getEvents},
		{streamPoll, cfg.Settings.StreamPoll, (*Camera).getStream},
	} {
		if pl.every == nil {
			continue
		}
		grp.Go(func() error {
			for range newTickerWithCtx(ctx, pl.every.Duration) {
				walkCameras(ctx, cycles, pl.p, cameras, pl.visit, time.Now)
			}
			return ctx.Err()
		})
	}
}

// walkCameras runs one poll over every camera in turn, timing the walk in cycles.
func walkCameras(ctx context.Context, cycles *pollCycles, p poll, cameras []*Camera, visit func(*Camera, context.Context), now func() time.Time) {
	cycles.begin(p, now())
	for _, c := range cameras {
		visit(c, ctx)
	}
	cycles.end(p, now())
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
