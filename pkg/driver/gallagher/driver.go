package gallagher

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const (
	DriverName                      = "gallagher"
	defaultOccupancyRefreshInterval = time.Minute * 30
)

type Driver struct {
	*service.Service[config.Root]
	announcer node.Announcer
	logger    *zap.Logger
	ticker    *time.Ticker
}

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)
	d := &Driver{
		announcer: services.Node,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithSystemCheck[config.Root](services.SystemCheck),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", logger)
		})),
	)
	d.logger = logger
	return d
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	cfg.ApplyDefaults()
	announcer, undo := node.AnnounceScope(d.announcer)
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
		return fmt.Errorf("gallagher BaseURL is not set")
	}

	bytes, err := os.ReadFile(cfg.HTTP.ApiKeyFile)
	if err != nil {
		return fmt.Errorf("error reading api key file: %w", err)
	}
	client, err := newHttpClient(cfg.HTTP.BaseURL, string(bytes), cfg.CaPath, cfg.ClientCertPath, cfg.ClientKeyPath)

	if client == nil {
		d.logger.Error("failed to create client", zap.Error(err))
		return nil
	}

	if err := d.probeServer(ctx, client); err != nil {
		return err
	}

	cc := newCardholderController(client, cfg.TopicPrefix, d.logger)
	grp.Go(func() error {
		return cc.run(ctx, cfg.RefreshCardholders, announcer, cfg.ScNamePrefix)
	})

	dc := newDoorController(client, cfg.TopicPrefix, d.logger)
	_ = dc.refreshDoors(announcer, cfg.ScNamePrefix) // make a blocking call to fetch the doors before we request the sc
	grp.Go(func() error {
		return dc.run(ctx, cfg.RefreshDoors, announcer, cfg.ScNamePrefix)
	})

	azc := newAccessZoneController(client, cc, d.logger)
	_ = azc.refreshAccessZones(announcer, cfg.ScNamePrefix) // blocking initial fetch
	grp.Go(func() error {
		return azc.run(ctx, cfg.RefreshAccessZones, announcer, cfg.ScNamePrefix)
	})

	sc := newSecurityEventController(client, d.logger, cfg.NumSecurityEvents)
	announcer.Announce(cfg.ScNamePrefix,
		node.HasServer(securityeventpb.RegisterSecurityEventApiServer, securityeventpb.SecurityEventApiServer(sc)),
		node.HasTrait(securityeventpb.TraitName),
	)
	grp.Go(func() error {
		return sc.run(ctx, cfg.RefreshAlarms)
	})

	if cfg.OccupancyCountEnabled {
		occupancyCtrl := newOccupancyEventController(client, d.logger, cfg.RefreshOccupancyInterval.Or(defaultOccupancyRefreshInterval))
		announcer.Announce(path.Join(cfg.ScNamePrefix, "occupancy"),
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancyCtrl)),
			node.HasTrait(trait.OccupancySensor),
		)
		grp.Go(func() error {
			if err := occupancyCtrl.run(ctx); err != nil {
				return err
			}
			return nil
		})
	}

	grp.Go(func() error {
		return d.udmiExport(ctx, cc)
	})

	go func() {
		err := grp.Wait()
		d.logger.Error("run error", zap.String("error", err.Error()))
		undo()
	}()
	return nil
}

// probeServer checks whether the Gallagher server is reachable and the API key is accepted.
// Returns an error if the server is unreachable or the licence is invalid, causing applyConfig
// to fail and the retry loop to re-attempt until the server is available.
func (d *Driver) probeServer(ctx context.Context, client *Client) error {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	statusCode, err := client.probe(probeCtx)
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	if statusCode == http.StatusUnauthorized {
		return fmt.Errorf("API licence not valid: server returned 401 Unauthorized — check the API key and Gallagher licence")
	}
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
