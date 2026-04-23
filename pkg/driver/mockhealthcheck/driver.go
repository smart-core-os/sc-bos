// Package mockhealthcheck provides a mock driver that creates health checks and randomly
// cycles their state between healthy, degraded, and fault to support UI testing.
package mockhealthcheck

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const DriverName = "mock-health"

// Factory is the driver factory registered with alldrivers.
var Factory driver.Factory = factory{}

type factory struct{}

func (factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		health: services.Health,
		logger: services.Logger.Named(DriverName),
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[Root](func() {
		d.clean()
		if services.SystemCheck != nil {
			services.SystemCheck.Dispose()
		}
	}))
	return d
}

// Root is the top-level config for the mock-health driver.
type Root struct {
	driver.BaseConfig
	Checks []Check `json:"checks,omitempty"`
}

// Check defines a single simulated health check.
type Check struct {
	// Name is the SC-BOS device name under which the health check is registered.
	Name string `json:"name"`
	// DisplayName is the human-readable label shown in the UI.
	// Defaults to Name if empty.
	DisplayName string `json:"displayName,omitempty"`
	// Description explains what is being checked.
	Description string `json:"description,omitempty"`
	// FaultProbability is the probability (0.0–1.0) that each simulation tick results in
	// a fault or unreliable state. The remaining probability results in a healthy state.
	// Defaults to 0.15.
	FaultProbability float64 `json:"faultProbability,omitempty"`
}

// Driver creates and randomises health checks defined in its config.
type Driver struct {
	*service.Service[Root]
	health *healthpb.Checks
	logger *zap.Logger
	checks []*healthpb.FaultCheck // current active checks; protected by sequential applyConfig calls
}

// applyConfig is called each time the driver config is applied or updated.
func (d *Driver) applyConfig(ctx context.Context, cfg Root) error {
	// Dispose checks from any previous config application before creating new ones.
	// Goroutines from the previous ctx invocation will naturally stop via ctx.Done().
	d.clean()

	for _, check := range cfg.Checks {
		p := check.FaultProbability
		if p <= 0 {
			p = 0.15
		}

		displayName := check.DisplayName
		if displayName == "" {
			displayName = check.Name
		}

		fc, err := d.health.NewFaultCheck(check.Name, &healthpb.HealthCheck{
			Id:          "simulated",
			DisplayName: displayName,
			Description: check.Description,
		})
		if err != nil {
			d.logger.Warn("failed to create health check", zap.String("name", check.Name), zap.Error(err))
			continue
		}
		d.checks = append(d.checks, fc)

		fc.ClearFaults() // start healthy
		go runSimulation(ctx, fc, p)
	}

	return nil
}

// clean disposes all active per-device health checks. Called on config re-apply and driver stop.
func (d *Driver) clean() {
	for _, fc := range d.checks {
		fc.Dispose()
	}
	d.checks = nil
}

// runSimulation periodically randomises the health check state.
// With probability p, the check transitions to a fault or unreliable state; otherwise it is healthy.
// The unreliable sub-state occurs with probability p*0.3 (≈30% of all bad ticks).
func runSimulation(ctx context.Context, fc *healthpb.FaultCheck, p float64) {
	timer := time.NewTimer(durationBetween(30*time.Second, 90*time.Second))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(durationBetween(30*time.Second, 90*time.Second))

		r := rand.Float64()
		switch {
		case r < p*0.3:
			// Simulate a connection/transport failure
			fc.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
				State: healthpb.HealthCheck_Reliability_CONN_TRANSIENT_FAILURE,
				LastError: &healthpb.HealthCheck_Error{
					SummaryText: "Simulated connection failure",
					Code: &healthpb.HealthCheck_Error_Code{
						System: "MOCK",
						Code:   "SIMULATED_UNRELIABLE",
					},
				},
			})
		case r < p:
			// Simulate a fault (check is reachable but reports a problem)
			fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
				SummaryText: "Simulated fault condition",
				Code: &healthpb.HealthCheck_Error_Code{
					System: "MOCK",
					Code:   "SIMULATED_FAULT",
				},
			})
		default:
			// Healthy: clear all faults and mark reliable
			fc.ClearFaults()
		}
	}
}

func durationBetween(min, max time.Duration) time.Duration {
	return time.Duration(rand.Intn(int(max-min))) + min
}
