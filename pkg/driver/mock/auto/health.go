package auto

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// HealthAuto returns a Lifecycle that simulates health check state changes for a device.
// A single "Device Status" fault check is created and randomly transitions between
// NORMAL and ABNORMAL every 30–90 seconds (15% fault probability).
func HealthAuto(model *healthpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			initial := &healthpb.HealthCheck{
				Id:              "status",
				DisplayName:     "Device Status",
				Description:     "Simulated device communication status",
				OccupantImpact:  healthpb.HealthCheck_NO_OCCUPANT_IMPACT,
				EquipmentImpact: healthpb.HealthCheck_FUNCTION,
				Normality:       healthpb.HealthCheck_NORMAL,
				NormalTime:      timestamppb.Now(),
				Check:           &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{}},
				Reliability: &healthpb.HealthCheck_Reliability{
					State:        healthpb.HealthCheck_Reliability_RELIABLE,
					ReliableTime: timestamppb.Now(),
				},
			}
			current, err := model.CreateHealthCheck(initial)
			if err != nil {
				return
			}

			timer := time.NewTimer(durationBetween(30*time.Second, 90*time.Second))
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer.Reset(durationBetween(30*time.Second, 90*time.Second))
					current = healthStateUpdate(current, randomBool(0.15))
					_, _ = model.UpdateHealthCheck(current)
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{})
	return slc
}

// healthStateUpdate clones check, applies a fault or recovery, and returns the clone.
func healthStateUpdate(check *healthpb.HealthCheck, fault bool) *healthpb.HealthCheck {
	c := proto.Clone(check).(*healthpb.HealthCheck)
	old := c.Normality
	if fault {
		c.Normality = healthpb.HealthCheck_ABNORMAL
		c.GetFaults().CurrentFaults = []*healthpb.HealthCheck_Error{
			{SummaryText: "simulated connectivity failure"},
		}
		if old == healthpb.HealthCheck_NORMAL {
			c.AbnormalTime = timestamppb.Now()
		}
	} else {
		c.Normality = healthpb.HealthCheck_NORMAL
		c.GetFaults().CurrentFaults = nil
		if old != healthpb.HealthCheck_NORMAL {
			c.NormalTime = timestamppb.Now()
		}
	}
	return c
}
