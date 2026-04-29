package resourceuse

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
)

func shouldHideDisk(p disk.PartitionStat, hiddenFsTypes map[string]struct{}, hiddenMountPrefixes []string) bool {
	if _, hidden := hiddenFsTypes[p.Fstype]; hidden {
		return true
	}
	for _, prefix := range hiddenMountPrefixes {
		if p.Mountpoint == prefix || strings.HasPrefix(p.Mountpoint, prefix+"/") {
			return true
		}
	}
	return false
}

func (s *System) applyConfig(ctx context.Context, cfg Root) error {
	model := resourceusepb.NewModel()
	modelServer := resourceusepb.NewModelServer(model)

	interval := cfg.pollInterval()

	announcer := node.NewReplaceAnnouncer(s.node)
	announce := announcer.Replace(ctx)
	announce.Announce(s.name,
		node.HasServer(resourceusepb.RegisterResourceUseApiServer, resourceusepb.ResourceUseApiServer(modelServer)),
		node.HasTrait(resourceusepb.TraitName),
	)

	hiddenFsTypes := cfg.effectiveHiddenFsTypes()
	hiddenMountPrefixes := cfg.effectiveHiddenMountPrefixes()
	go s.pollLoop(ctx, model, interval, hiddenFsTypes, hiddenMountPrefixes)

	return nil
}

func (s *System) pollLoop(ctx context.Context, model *resourceusepb.Model, interval time.Duration, hiddenFsTypes map[string]struct{}, hiddenMountPrefixes []string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// collect immediately on start
	s.collect(ctx, model, hiddenFsTypes, hiddenMountPrefixes)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx, model, hiddenFsTypes, hiddenMountPrefixes)
		}
	}
}

func (s *System) collect(ctx context.Context, model *resourceusepb.Model, hiddenFsTypes map[string]struct{}, hiddenMountPrefixes []string) {
	v := &resourceusepb.ResourceUse{}

	// Note: With interval 0, on the very first invocation there's no prior sample, so the initial value may be inaccurate
	if perCore, err := cpu.PercentWithContext(ctx, 0, true); err == nil {
		overall := average(perCore)
		core32 := make([]float32, len(perCore))
		for i, p := range perCore {
			core32[i] = float32(p)
		}
		v.Cpu = &resourceusepb.CpuUse{
			Utilization: new(float32(overall)),
			CorePercent: core32,
		}
	} else {
		s.logger.Warn("cpu percent", zap.Error(err))
	}

	if vmStat, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		if vmStat != nil {
			v.Memory = &resourceusepb.MemoryUse{
				Usage:       new(vmStat.Used),
				Limit:       new(vmStat.Total),
				Utilization: new(float32(vmStat.UsedPercent)),
			}
		}
	} else {
		s.logger.Warn("memory", zap.Error(err))
	}

	if parts, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, p := range parts {
			if shouldHideDisk(p, hiddenFsTypes, hiddenMountPrefixes) {
				continue
			}
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			v.Disks = append(v.Disks, &resourceusepb.DiskUse{
				MountPoint:  p.Mountpoint,
				Usage:       new(usage.Used),
				FreeBytes:   new(usage.Free),
				Limit:       new(usage.Total),
				Utilization: new(float32(usage.UsedPercent)),
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
		v.Network = &resourceusepb.NetworkUse{
			ConnectionCount: new(established),
		}
	} else {
		s.logger.Warn("network connections", zap.Error(err))
	}

	if _, err := model.SetResourceUse(v); err != nil {
		s.logger.Warn("set resource use", zap.Error(err))
	}
}

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
