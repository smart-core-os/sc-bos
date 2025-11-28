package helvarnet

import (
	"context"
	"net"
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/lightpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

const (
	DriverName = "helvarnet"
)

var Factory driver.Factory = factory{}

type factory struct{}

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	logger    *zap.Logger
	clients   map[string]*tcpClient
	database  *bolthold.Store
	health    *healthpb.Checks
}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)

	d := &Driver{
		logger:    logger,
		announcer: node.NewReplaceAnnouncer(services.Node),
		database:  services.Database,
		health:    services.Health,
	}

	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser[config.Root](config.ParseConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logCtx service.RetryContext) {
			logCtx.LogTo("applyConfig", logger)
		}), service.RetryWithMinDelay(10*time.Second)),
	)

	return d
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	rootAnnouncer := d.announcer.Replace(ctx)
	grp, ctx := errgroup.WithContext(ctx)
	d.clients = make(map[string]*tcpClient)
	var faultChecks []*healthpb.FaultCheck

	for _, l := range cfg.LightingGroups {
		if _, ok := d.clients[l.IpAddress]; !ok {
			tcpAddr, err := net.ResolveTCPAddr("tcp", l.IpAddress+*cfg.Port)
			if err != nil {
				return err
			}
			d.clients[l.IpAddress] = newTcpClient(tcpAddr, d.logger, &cfg)
		}

		if l.GroupNumber == nil {
			d.logger.Warn("lighting group without group number", zap.String("name", l.Name))
			continue
		}
		lightingGroup := newLightingGroup(d.clients[l.IpAddress], d.logger, l, *l.GroupNumber)
		// try to get the last scene on a restart of the area controller
		lightingGroup.getLastScene()
		err := lightingGroup.getSceneNames()
		if err != nil {
			d.logger.Error("getSceneNames error", zap.String("error", err.Error()))
		}
		rootAnnouncer.Announce(l.Name,
			node.HasTrait(trait.Light,
				node.WithClients(lightpb.WrapApi(lightingGroup)),
				node.WithClients(lightpb.WrapInfo(lightingGroup))),
			node.HasMetadata(l.Meta))
	}

	for _, l := range cfg.Lights {
		if _, ok := d.clients[l.IpAddress]; !ok {
			tcpAddr, err := net.ResolveTCPAddr("tcp", l.IpAddress+*cfg.Port)
			if err != nil {
				return err
			}
			d.clients[l.IpAddress] = newTcpClient(tcpAddr, d.logger, &cfg)
		}

		lum := newLight(d.clients[l.IpAddress], d.logger, l, d.database, false)

		faultCheck, err := d.health.NewFaultCheck(l.Name, getDeviceHealthCheck())
		if err != nil {
			d.logger.Error("failed to create health check", zap.String("device", l.Name), zap.Error(err))
			return err
		}
		faultChecks = append(faultChecks, faultCheck)

		rootAnnouncer.Announce(l.Name,
			node.HasTrait(trait.Light,
				node.WithClients(lightpb.WrapApi(lum))),
			node.HasTrait(udmipb.TraitName,
				node.WithClients(gen.WrapUdmiService(lum))),
			node.HasMetadata(l.Meta))
		grp.Go(func() error {
			return lum.queryDevice(ctx, cfg.RefreshStatus.Duration, faultCheck)
		})
	}

	for _, pir := range cfg.Pirs {
		if _, ok := d.clients[pir.IpAddress]; !ok {
			tcpAddr, err := net.ResolveTCPAddr("tcp", pir.IpAddress+*cfg.Port)
			if err != nil {
				return err
			}
			d.clients[pir.IpAddress] = newTcpClient(tcpAddr, d.logger, &cfg)
		}
		p := newPir(d.clients[pir.IpAddress], d.logger, pir)
		rootAnnouncer.Announce(pir.Name,
			node.HasTrait(trait.OccupancySensor,
				node.WithClients(occupancysensorpb.WrapApi(p))),
			node.HasTrait(udmipb.TraitName,
				node.WithClients(gen.WrapUdmiService(p))),
			node.HasMetadata(pir.Meta))
		grp.Go(func() error {
			return p.runUpdateState(ctx, cfg.RefreshOccupancy.Duration)
		})
	}

	for _, em := range cfg.EmergencyLights {
		if _, ok := d.clients[em.IpAddress]; !ok {
			tcpAddr, err := net.ResolveTCPAddr("tcp", em.IpAddress+*cfg.Port)
			if err != nil {
				return err
			}
			d.clients[em.IpAddress] = newTcpClient(tcpAddr, d.logger, &cfg)
		}
		emergencyLight := newLight(d.clients[em.IpAddress], d.logger, em, d.database, true)
		err := emergencyLight.loadTestResults()
		if err != nil {
			d.logger.Error("loadTestResults error", zap.Error(err))
		}

		faultCheck, err := d.health.NewFaultCheck(em.Name, getDeviceHealthCheck())
		if err != nil {
			d.logger.Error("failed to create health check", zap.String("device", em.Name), zap.Error(err))
			return err
		}
		faultChecks = append(faultChecks, faultCheck)

		rootAnnouncer.Announce(em.Name,
			node.HasTrait(trait.Light,
				node.WithClients(lightpb.WrapApi(emergencyLight))),
			node.HasTrait(emergencylightpb.TraitName,
				node.WithClients(gen.WrapEmergencyLightApi(emergencyLight))),
			node.HasTrait(udmipb.TraitName,
				node.WithClients(gen.WrapUdmiService(emergencyLight))),
			node.HasMetadata(em.Meta))
		grp.Go(func() error {
			return emergencyLight.queryDevice(ctx, cfg.RefreshStatus.Duration, faultCheck)
		})
	}

	go func() {
		err := grp.Wait()
		for _, client := range d.clients {
			if client.conn != nil {
				client.close()
			}
		}
		for _, fc := range faultChecks {
			fc.Dispose()
		}
		if err != nil {
			d.logger.Error("run error", zap.String("error", err.Error()))
		}
	}()
	return nil
}

// this health check monitors the device to check if it is online, communicating properly and if it is reporting a fault itself
// via the status register in the device.
func getDeviceHealthCheck() *gen.HealthCheck {
	return &gen.HealthCheck{
		Id:              "deviceStatusCheck",
		DisplayName:     "Device Status Check",
		Description:     "Checks the status from the device itself and also if communication is healthy",
		OccupantImpact:  gen.HealthCheck_COMFORT,
		EquipmentImpact: gen.HealthCheck_FUNCTION,
	}
}
