package occupancy

import (
	"context"
	"math"
	"sync"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/occupancy/config"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("occupancy")
	f := &feature{
		announcer: node.NewReplaceAnnouncer(services.Node),
		devices:   services.Devices,
		clients:   services.Node,
		logger:    services.Logger,
	}
	f.Service = service.New(service.MonoApply(f.applyConfig), service.WithParser(config.ParseConfig))
	return f
})

type feature struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	devices   *zone.Devices
	clients   node.ClientConner
	logger    *zap.Logger
}

func (f *feature) applyConfig(ctx context.Context, cfg config.Root) error {
	announce := f.announcer.Replace(ctx)
	logger := f.logger

	if len(cfg.OccupancySensors) > 0 || len(cfg.EnterLeaveOccupancySensors) > 0 {
		group := &Group{logger: logger}
		conn := f.clients.ClientConn()

		if len(cfg.OccupancySensors) > 0 {
			group.client = occupancysensorpb.NewOccupancySensorApiClient(conn)
			group.names = cfg.OccupancySensors
		}
		if len(cfg.EnterLeaveOccupancySensors) > 0 {
			elServer := &enterLeave{
				model:  occupancysensorpb.NewModel(),
				client: enterleavesensorpb.NewEnterLeaveSensorApiClient(conn),
				names:  cfg.EnterLeaveOccupancySensors,
				logger: logger,
			}

			if cfg.EnterLeaveOccupancySensorSLA != nil {
				names := make(map[string]struct{})

				for _, name := range cfg.EnterLeaveOccupancySensorSLA.CantFail {
					names[name] = struct{}{}
				}

				elServer.sla = &sla{
					cantFail:                       names,
					percentageOfAcceptableFailures: cfg.EnterLeaveOccupancySensorSLA.PercentageOfAcceptableFailures,
					errs:                           &sync.Map{},
				}
			}

			group.clients = append(group.clients, occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, elServer)))
		}

		f.devices.Add(cfg.OccupancySensors...)
		f.devices.Add(cfg.EnterLeaveOccupancySensors...)
		features := []node.Feature{
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(group)),
			node.HasTrait(trait.OccupancySensor),
		}
		if len(cfg.EnterLeaveOccupancySensors) > 0 && len(cfg.OccupancySensors) == 0 {
			// Describing the zone as a people counter lets consumers such as connecttelemetry
			// trust a people count of 0 as an empty zone. A zone with any occupancy sensors
			// stays undescribed, since those may be presence detectors that report 0, and the
			// zone would report only their 0 while its enter/leave sensors are failing.
			features = append(features, node.HasServer(occupancysensorpb.RegisterOccupancySensorInfoServer, occupancysensorpb.OccupancySensorInfoServer(countingInfo{})))
		}
		announce.Announce(cfg.Name, features...)
	}

	return nil
}

// countingInfo describes a zone whose people count comes from enter/leave sensors.
type countingInfo struct {
	occupancysensorpb.UnimplementedOccupancySensorInfoServer
}

func (countingInfo) DescribeOccupancy(context.Context, *occupancysensorpb.DescribeOccupancyRequest) (*occupancysensorpb.OccupancySupport, error) {
	return &occupancysensorpb.OccupancySupport{
		ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
		// An enter/leave counter has no sensor limit on the people it can count.
		MaxPeople: math.MaxInt32,
	}, nil
}
