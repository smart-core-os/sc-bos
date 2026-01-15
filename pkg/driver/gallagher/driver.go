package gallagher

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/securityevent"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

const (
	DriverName                      = "gallagher"
	defaultOccupancyRefreshInterval = time.Minute * 30
)

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	logger    *zap.Logger
	ticker    *time.Ticker
}

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {

	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		logger:    services.Logger.Named(DriverName),
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser[config.Root](config.ParseConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", d.logger)
		}), service.RetryWithMinDelay(10*time.Second)),
	)
	return d
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	announcer := d.announcer.Replace(ctx)
	grp, ctx := errgroup.WithContext(ctx)

	if d.ticker != nil {
		d.ticker.Stop()
	}
	d.ticker = time.NewTicker(cfg.UdmiExportInterval.Duration)

	if cfg.HTTP == nil {
		d.logger.Error("http config is not set")
		return fmt.Errorf("gallagher HTTP config is not set")
	}

	if cfg.HTTP.BaseURL == "" {
		d.logger.Error("baseURL is not set")
		return fmt.Errorf("gallagher baseURL is not set")
	}

	bytes, err := os.ReadFile(cfg.HTTP.ApiKeyFile)
	if err != nil {
		return fmt.Errorf("error reading api key file: %w", err)
	}
	client, err := newHttpClient(cfg.HTTP.BaseURL, string(bytes), cfg.CaPath, cfg.ClientCertPath, cfg.ClientKeyPath)

	if err != nil {
		d.logger.Error("failed to create client", zap.Error(err))
		return err
	}

	cc := newCardholderController(client, cfg.TopicPrefix, d.logger)
	grp.Go(func() error {
		return cc.run(ctx, cfg.RefreshCardholders, announcer, cfg.ScNamePrefix)
	})

	dc := newDoorController(client, cfg.TopicPrefix, d.logger)
	grp.Go(func() error {
		return dc.run(ctx, cfg.RefreshDoors, announcer, cfg.ScNamePrefix)
	})

	sc := newSecurityEventController(client, d.logger, cfg.NumSecurityEvents)
	announcer.Announce(cfg.ScNamePrefix, node.HasTrait(securityevent.TraitName, node.WithClients(securityeventpb.WrapApi(sc))))
	grp.Go(func() error {
		return sc.run(ctx, cfg.RefreshAlarms)
	})

	if cfg.OccupancyCountEnabled {
		occupancyCtrl := newOccupancyEventController(client, d.logger, cfg.RefreshOccupancyInterval.Or(defaultOccupancyRefreshInterval))
		announcer.Announce(path.Join(cfg.ScNamePrefix, "occupancy"), node.HasTrait(trait.OccupancySensor, node.WithClients(occupancysensorpb.WrapApi(occupancyCtrl))))
		grp.Go(func() error {
			return occupancyCtrl.run(ctx)
		})
	}

	grp.Go(func() error {
		return d.udmiExport(ctx, cc)
	})

	go func() {
		err := grp.Wait()
		if err != nil {
			d.logger.Error("gallagher driver error", zap.Error(err))
		}
		d.ticker.Stop()
	}()
	return nil
}

// run the udmi export for all the controllers. currently only cardholders are exported but might be extended to others
func (d *Driver) udmiExport(ctx context.Context, cc *CardholderController) error {
	for {
		select {
		case <-d.ticker.C:
			cc.sendUdmiMessages(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}
