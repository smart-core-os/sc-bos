package resourceuse

import (
	"context"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
)

func (s *System) applyConfig(ctx context.Context, cfg Root) error {
	model := resourceusepb.NewModel()
	modelServer := resourceusepb.NewModelServer(model)

	interval := cfg.pollInterval()

	announcer := node.NewReplaceAnnouncer(s.node)
	announce := announcer.Replace(ctx)
	announce.Announce(s.name,
		node.HasTrait(resourceusepb.TraitName, node.WithClients(gen.WrapApi(modelServer))),
	)

	go s.pollLoop(ctx, model, interval)

	return nil
}

func (s *System) pollLoop(ctx context.Context, model *resourceusepb.Model, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// collect immediately on start
	s.collect(ctx, model)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx, model)
		}
	}
}

func (s *System) collect(ctx context.Context, model *resourceusepb.Model) {
	v := &gen.ResourceUse{}

	if perCore, err := cpu.PercentWithContext(ctx, 0, true); err == nil {
		overall := average(perCore)
		core32 := make([]float32, len(perCore))
		for i, p := range perCore {
			core32[i] = float32(p)
		}
		v.Cpu = &gen.CpuUse{
			Utilization: ptr(float32(overall)),
			CorePercent: core32,
		}
	} else {
		s.logger.Warn("cpu percent", zap.Error(err))
	}

	if vmStat, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		if vmStat != nil {
			v.Memory = &gen.MemoryUse{
				Usage:       ptr(vmStat.Used),
				Limit:       ptr(vmStat.Total),
				Utilization: ptr(float32(vmStat.UsedPercent)),
			}
		}
	} else {
		s.logger.Warn("memory", zap.Error(err))
	}

	if parts, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, p := range parts {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			v.Disks = append(v.Disks, &gen.DiskUse{
				MountPoint:  p.Mountpoint,
				Usage:       ptr(usage.Used),
				FreeBytes:   ptr(usage.Free),
				Limit:       ptr(usage.Total),
				Utilization: ptr(float32(usage.UsedPercent)),
			})
		}
	} else {
		s.logger.Warn("disk partitions", zap.Error(err))
	}

	// Network connections (this process only)
	if conns, err := gopsnet.ConnectionsPidWithContext(ctx, "tcp", int32(os.Getpid())); err == nil {
		var established uint64
		for _, c := range conns {
			if c.Status == "ESTABLISHED" {
				established++
			}
		}
		v.Network = &gen.NetworkUse{
			ConnectionCount: ptr(established),
		}
	} else {
		s.logger.Warn("network connections", zap.Error(err))
	}

	if _, err := model.SetResourceUse(v); err != nil {
		s.logger.Warn("set resource use", zap.Error(err))
	}
}

func ptr[T any](v T) *T { return &v }

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
