package resourceutilisation

import (
	"context"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/resourceutilisationpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/resourceutilisationpb"
)

func (s *System) applyConfig(ctx context.Context, cfg Root) error {
	model := resourceutilisationpb.NewModel()
	modelServer := resourceutilisationpb.NewModelServer(model)

	interval := cfg.pollInterval()

	announcer := node.NewReplaceAnnouncer(s.node)
	announce := announcer.Replace(ctx)
	announce.Announce(s.name,
		node.HasTrait(resourceutilisationpb.TraitName, node.WithClients(gen.WrapApi(modelServer))),
	)

	go s.pollLoop(ctx, model, interval)

	return nil
}

func (s *System) pollLoop(ctx context.Context, model *resourceutilisationpb.Model, interval time.Duration) {
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

func (s *System) collect(ctx context.Context, model *resourceutilisationpb.Model) {
	v := &gen.ResourceUtilisation{}

	// CPU
	if perCore, err := cpu.PercentWithContext(ctx, 0, true); err == nil {
		overall := average(perCore)
		core32 := make([]float32, len(perCore))
		for i, p := range perCore {
			core32[i] = float32(p)
		}
		v.Cpu = &gen.CpuUtilisation{
			PercentUtilised: ptr(float32(overall)),
			CorePercent:     core32,
		}
	} else {
		s.logger.Warn("cpu percent", zap.Error(err))
	}

	// Memory
	if vmStat, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		if vmStat != nil {
			v.Memory = &gen.MemoryUtilisation{
				UsedBytes:   ptr(vmStat.Used),
				TotalBytes:  ptr(vmStat.Total),
				PercentUsed: ptr(float32(vmStat.UsedPercent)),
			}
		}
	} else {
		s.logger.Warn("memory", zap.Error(err))
	}

	// Disks
	if parts, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, p := range parts {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			v.Disks = append(v.Disks, &gen.DiskUtilisation{
				MountPoint:  p.Mountpoint,
				UsedBytes:   ptr(usage.Used),
				FreeBytes:   ptr(usage.Free),
				TotalBytes:  ptr(usage.Total),
				PercentUsed: ptr(float32(usage.UsedPercent)),
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
		v.Network = &gen.NetworkUtilisation{
			ConnectionsEstablished: ptr(established),
		}
	} else {
		s.logger.Warn("network connections", zap.Error(err))
	}

	if _, err := model.SetResourceUtilisation(v); err != nil {
		s.logger.Warn("set resource utilisation", zap.Error(err))
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
