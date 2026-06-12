// Package sim implements the "sim" driver: a physically-coupled simulated building.
//
// Where the mock driver gives every trait its own independent random automation,
// sim runs a single engine (see Building) in which occupancy drives lighting and
// FCU load, which in turn derives the building's electrical demand and metered
// energy. It is configured at the building level — floors, rooms and the device
// archetypes in each room — rather than as an explicit device list.
package sim

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/time/clock"
)

const DriverName = "sim"

var Factory driver.Factory = factory{}

type factory struct{}

func (factory) New(services driver.Services) service.Lifecycle {
	return NewDriver(services)
}

func (factory) ConfigBlocks() []block.Block {
	return config.Blocks
}

func NewDriver(services driver.Services) *Driver {
	d := &Driver{
		announcer:   services.Node,
		systemCheck: services.SystemCheck,
		clk:         clock.Real(),
		logger:      services.Logger.Named(DriverName),
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[config.Root](func() {
		d.stop()
		if d.systemCheck != nil {
			d.systemCheck.Dispose()
		}
	}))
	return d
}

type Driver struct {
	*service.Service[config.Root]

	logger      *zap.Logger
	announcer   node.Announcer
	systemCheck service.SystemCheck
	clk         clock.Clock // wall clock; swappable in tests

	mu     sync.Mutex
	cancel context.CancelFunc // cancels the running engine + health sim
	undo   []node.Undo        // reverses the current announcements
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	cfg.Normalise()
	if err := cfg.Validate(); err != nil {
		return err
	}
	days, err := cfg.WorkingHours.Weekdays()
	if err != nil {
		return err
	}
	scaler := scale.WorkingHours(cfg.WorkingHours.Start, cfg.WorkingHours.End, days)

	// Tear down any previous run before (re)building.
	d.stop()

	start := d.clk.Now()
	building, devices, err := Expand(cfg, scaler, start)
	if err != nil {
		return err
	}
	if rooms := zeroOccupancyRooms(cfg); len(rooms) > 0 {
		d.logger.Warn("rooms resolve to zero max occupancy; their occupancy-driven sensors will always report empty",
			zap.Strings("rooms", rooms))
	}

	var undos []node.Undo
	if cfg.Metadata != nil {
		undos = append(undos, d.announcer.Announce(cfg.Name, node.HasMetadata(cfg.Metadata)))
	}
	for _, dev := range devices {
		feats := append([]node.Feature{
			node.HasMetadata(dev.Metadata),
			node.HasDeviceType(metadatapb.Metadata_DEVICE),
		}, dev.Features...)
		undos = append(undos, d.announcer.Announce(dev.Name, feats...))
	}
	d.logger.Info("announced simulated building", zap.Int("devices", len(devices)))

	runCtx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.undo = undos
	d.mu.Unlock()

	go d.runEngine(runCtx, building, cfg)

	if d.systemCheck != nil {
		// Report healthy now. This also clears any FAILED state left by a previous
		// config whose health simulation has since been reconfigured away.
		d.systemCheck.MarkRunning()
		if cfg.HealthCheck != nil {
			// Normalise guarantees FaultProbability is non-nil when HealthCheck is set.
			go runHealthSimulation(runCtx, d.clk, d.systemCheck, *cfg.HealthCheck.FaultProbability)
		}
	}
	return nil
}

// runEngine advances the simulation on the wall clock, mapping wall time to
// simulated time via the configured TimeMultiplier, and publishes device state
// after each tick.
func (d *Driver) runEngine(ctx context.Context, b *Building, cfg config.Root) {
	// Contain panics to this driver rather than crashing the whole process: an
	// updater bug would otherwise take down every system in the controller.
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("simulation engine panicked; engine stopped, devices remain announced with their last published values",
				zap.Any("panic", r), zap.ByteString("stack", debug.Stack()))
			if d.systemCheck != nil {
				d.systemCheck.MarkFailed(fmt.Errorf("simulation engine panicked: %v", r))
			}
		}
	}()

	mult := cfg.TimeMultiplier
	tick := cfg.TickInterval.Duration

	anchorWall := d.clk.Now()
	anchorSim := anchorWall
	simAt := func(wall time.Time) time.Time {
		return anchorSim.Add(time.Duration(float64(wall.Sub(anchorWall)) * mult))
	}

	// Publish the initial steady state immediately so devices report sane values
	// before the first tick elapses.
	b.Publish(anchorSim)

	ticker := d.clk.Every(tick)
	defer ticker.Stop()
	last := anchorSim
	for {
		select {
		case <-ctx.Done():
			return
		case wall, ok := <-ticker.C():
			if !ok {
				return
			}
			now := simAt(wall)
			dt := now.Sub(last)
			last = now
			b.Tick(now, dt)
			b.Publish(now)
		}
	}
}

// stop cancels the running engine and reverses all announcements.
func (d *Driver) stop() {
	d.mu.Lock()
	cancel, undo := d.cancel, d.undo
	d.cancel, d.undo = nil, nil
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, u := range undo {
		u()
	}
}

// runHealthSimulation periodically randomises the driver's system-level health
// check: with probability p it reports a fault, otherwise healthy.
//
// It deliberately uses the package-level rand rather than the engine's seeded
// source (which is owned by the engine goroutine), so fault timing is not part
// of the config seed's reproducibility guarantee.
func runHealthSimulation(ctx context.Context, clk clock.Clock, check service.SystemCheck, p float64) {
	for {
		wait := 30*time.Second + time.Duration(rand.Int63n(int64(60*time.Second)))
		select {
		case <-ctx.Done():
			return
		case <-clk.After(wait):
		}
		if rand.Float64() < p {
			check.MarkFailed(fmt.Errorf("simulated connectivity failure"))
		} else {
			check.MarkRunning()
		}
	}
}
