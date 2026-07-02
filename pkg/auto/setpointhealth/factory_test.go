package setpointhealth

import (
	"math"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/setpointhealth/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

const (
	// matches configureSetpointMonitor below
	testTolerance   = 1.5
	testDuration    = time.Hour
	testMaxDuration = 3 * time.Hour // matches configureSetpointMonitorWithBackstop("3h")
	checkID         = "setpointhealth"
)

// TestTripsAfterDuration verifies the check goes abnormal once the deviation has exceeded the
// tolerance continuously for the configured duration.
func TestTripsAfterDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		// measured 24, set point 18 -> deviation 6, well outside tolerance
		h.updateAirTemp(model, 24.0, 18.0)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL) // not yet, timer running

		// just before the deadline: still normal
		h.advance(testDuration - time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)

		// past the deadline: abnormal
		h.advance(2 * time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)
		h.assertFaultContains("fcu-1", "deviates")
	})
}

// TestRecoversWhenInTolerance verifies a tripped check returns to normal once the measured value
// comes back within tolerance of the set point.
func TestRecoversWhenInTolerance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		h.updateAirTemp(model, 24.0, 18.0)
		h.advance(testDuration + time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)

		// unit catches up: measured 18.5, set point 18 -> deviation 0.5, within tolerance
		h.updateAirTemp(model, 18.5, 18.0)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
	})
}

// TestResetBeforeFiring verifies that returning within tolerance before the deadline cancels the
// countdown, so a later in-tolerance period does not trip.
func TestResetBeforeFiring(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		h.updateAirTemp(model, 24.0, 18.0) // out of tolerance, timer starts
		h.advance(testDuration - time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)

		// back within tolerance before the deadline -> timer cancelled
		h.updateAirTemp(model, 18.5, 18.0)
		h.advance(2 * time.Minute) // past the original deadline
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
	})
}

// TestSetPointChangeRestartsCountdown verifies a set point change while still out of tolerance gives
// the unit a fresh window: it does not trip at the original deadline, but one duration after the change.
func TestSetPointChangeRestartsCountdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		h.updateAirTemp(model, 24.0, 18.0) // out of tolerance, timer starts at T0
		h.advance(testDuration - 10*time.Minute)

		// set point moved to 16 (still out of tolerance: |24-16|=8) -> restarts the window
		h.updateAirTemp(model, 24.0, 16.0)

		// original deadline passes: should NOT have tripped (window restarted)
		h.advance(11 * time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)

		// one full duration after the change: trips
		h.advance(testDuration)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)
	})
}

// TestRepeatedAdjustmentsNeverTrip verifies that churning the set point faster than the duration
// keeps restarting the window, so no fault fires while there is no stable target.
func TestRepeatedAdjustmentsNeverTrip(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		// keep moving the set point every ~half duration, always out of tolerance
		for i := 0; i < 5; i++ {
			setPoint := 18.0 - float64(i) // 18, 17, 16, 15, 14
			h.updateAirTemp(model, 24.0, setPoint)
			h.advance(testDuration / 2)
			h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
		}
	})
}

// TestSetPointChangeKeepsFault verifies a set point change after the fault is raised does not clear
// it; only a return to tolerance clears a confirmed fault.
func TestSetPointChangeKeepsFault(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		h.updateAirTemp(model, 24.0, 18.0)
		h.advance(testDuration + time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)

		// set point moved but still out of tolerance -> fault stays
		h.updateAirTemp(model, 24.0, 20.0)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)
	})
}

// TestBackstopTripsDespiteSetPointChurn verifies that, with a maxDuration backstop configured,
// churning the set point faster than duration (which keeps restarting the per-target window) still
// trips once the deviation has persisted continuously for maxDuration.
func TestBackstopTripsDespiteSetPointChurn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitorWithBackstop("3h")

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		// keep moving the set point every ~half duration, always out of tolerance. The per-target
		// window never completes, but the backstop clock keeps running from the first breach.
		elapsed := time.Duration(0)
		for i := 0; elapsed < testMaxDuration-testDuration; i++ {
			setPoint := 18.0 - float64(i) // 18, 17, 16, ...
			h.updateAirTemp(model, 24.0, setPoint)
			h.advance(testDuration / 2)
			elapsed += testDuration / 2
			h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
		}

		// once maxDuration has elapsed since the first breach, the backstop trips.
		h.advance(testMaxDuration - elapsed + time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_ABNORMAL)
		h.assertFaultContains("fcu-1", "deviates")
	})
}

// TestBackstopClearedOnRecovery verifies a return to tolerance before maxDuration clears the
// backstop clock, so a later long wait does not trip.
func TestBackstopClearedOnRecovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitorWithBackstop("3h")

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		h.updateAirTemp(model, 24.0, 18.0) // out of tolerance, backstop starts
		h.advance(testMaxDuration - time.Hour)

		// back within tolerance -> backstop cleared
		h.updateAirTemp(model, 18.5, 18.0)
		h.advance(2 * testMaxDuration)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
	})
}

// TestNoBackstopWhenUnset documents the default (backstop disabled): churning the set point faster
// than duration never trips, however long it continues.
func TestNoBackstopWhenUnset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor() // no maxDuration

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		for i := 0; i < 10; i++ {
			setPoint := 18.0 - float64(i%5) // churn well past any backstop multiple
			h.updateAirTemp(model, 24.0, setPoint)
			h.advance(testDuration / 2)
			h.assertNormality("fcu-1", healthpb.HealthCheck_NORMAL)
		}
	})
}

// TestHandlesUnsetValues verifies that a missing measured value or set point is reported as a bad
// response and never trips the check.
func TestHandlesUnsetValues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		// measured present but set point unset
		_, _ = model.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
			AmbientTemperature: &typespb.Temperature{ValueCelsius: 24.0},
		})
		synctest.Wait()
		// no complete reading yet: normality unknown, reliability bad
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMALITY_UNSPECIFIED)
		h.assertReliability("fcu-1", healthpb.HealthCheck_Reliability_BAD_RESPONSE)

		// even after a long wait it must not trip on a data gap
		h.advance(testDuration + time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMALITY_UNSPECIFIED)
	})
}

// TestHandlesNaNValues verifies a NaN reading (e.g. from a faulted device) is reported as a bad
// response rather than treated as a real out-of-tolerance value that would falsely trip the check.
func TestHandlesNaNValues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model := airtemperaturepb.NewModel()
		h.addDevice("fcu-1", model)
		h.waitForHealthCheck("fcu-1")

		// measured comes through as NaN: not a real reading
		h.updateAirTemp(model, math.NaN(), 18.0)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMALITY_UNSPECIFIED)
		h.assertReliability("fcu-1", healthpb.HealthCheck_Reliability_BAD_RESPONSE)

		// even after a long wait it must not trip on NaN garbage
		h.advance(testDuration + time.Minute)
		h.assertNormality("fcu-1", healthpb.HealthCheck_NORMALITY_UNSPECIFIED)
	})
}

// TestNumericLeaf verifies numericLeaf accepts a path ending at a numeric scalar and rejects one
// ending at a message field, so misconfigured value paths fail when the check is created.
func TestNumericLeaf(t *testing.T) {
	md := (&airtemperaturepb.AirTemperature{}).ProtoReflect().Descriptor()

	scalar, _, err := config.Value("ambientTemperature.valueCelsius").Parse(md)
	if err != nil {
		t.Fatalf("parse scalar path: %v", err)
	}
	if err := numericLeaf(scalar); err != nil {
		t.Errorf("numericLeaf(scalar) = %v, want nil", err)
	}

	message, _, err := config.Value("ambientTemperature").Parse(md)
	if err != nil {
		t.Fatalf("parse message path: %v", err)
	}
	if err := numericLeaf(message); err == nil {
		t.Errorf("numericLeaf(message) = nil, want error")
	}
}

// TestCreatesAndRemovesHealthChecks verifies checks are created and disposed with devices.
func TestCreatesAndRemovesHealthChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureSetpointMonitor()

		model1 := airtemperaturepb.NewModel()
		undo1 := h.addDevice("fcu-1", model1)
		h.waitForHealthCheck("fcu-1")

		model2 := airtemperaturepb.NewModel()
		h.addDevice("fcu-2", model2)
		h.waitForHealthCheck("fcu-2")

		undo1()
		h.waitForHealthCheckRemoval("fcu-1")
		h.assertHealthCheckExists("fcu-2")
	})
}

// testHarness provides a convenient test environment for the setpointhealth automation.
type testHarness struct {
	t      *testing.T
	node   *node.Node
	models map[string]*healthpb.Model
	mu     sync.Mutex
	auto   service.Lifecycle
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	n := node.New("test")

	h := &testHarness{
		t:      t,
		node:   n,
		models: make(map[string]*healthpb.Model),
	}

	registry := healthpb.NewRegistry(
		healthpb.WithOnNameCreate(func(name string) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.models[name] = healthpb.NewModel()
		}),
		healthpb.WithOnCheckCreate(func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
			h.mu.Lock()
			defer h.mu.Unlock()
			if model, ok := h.models[name]; ok {
				_, _ = model.CreateHealthCheck(c)
			}
			return c
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *healthpb.HealthCheck) {
			h.mu.Lock()
			defer h.mu.Unlock()
			if model, ok := h.models[name]; ok {
				_, _ = model.UpdateHealthCheck(c)
			}
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			h.mu.Lock()
			defer h.mu.Unlock()
			if model, ok := h.models[name]; ok {
				_ = model.DeleteHealthCheck(id)
			}
		}),
		healthpb.WithOnNameDelete(func(name string) {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.models, name)
		}),
	)

	devicesClient := devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devices.NewServer(n)))

	services := auto.Services{
		Logger:  zaptest.NewLogger(t),
		Node:    n,
		Devices: devicesClient,
		Health:  registry.ForOwner(checkID),
	}

	a := Factory.New(services)
	if _, err := a.Start(); err != nil {
		t.Fatalf("Failed to start automation: %v", err)
	}
	t.Cleanup(func() { _, _ = a.Stop() })

	h.auto = a
	return h
}

func (h *testHarness) configure(configJSON string) {
	h.t.Helper()
	_, err := h.auto.Configure([]byte(configJSON))
	if err != nil {
		h.t.Fatalf("Configure failed: %v", err)
	}
	synctest.Wait()
}

func (h *testHarness) configureSetpointMonitor() {
	h.t.Helper()
	h.configure(setpointMonitorConfig(""))
}

// configureSetpointMonitorWithBackstop configures the monitor with an absolute maxDuration backstop.
// maxDuration is a JSON duration string, e.g. "3h".
func (h *testHarness) configureSetpointMonitorWithBackstop(maxDuration string) {
	h.t.Helper()
	h.configure(setpointMonitorConfig(maxDuration))
}

// setpointMonitorConfig builds the monitor config JSON. When maxDuration is non-empty it adds the
// optional backstop field.
func setpointMonitorConfig(maxDuration string) string {
	maxDurationLine := ""
	if maxDuration != "" {
		maxDurationLine = `"maxDuration": "` + maxDuration + `",`
	}
	return `{
		"type": "setpointhealth",
		"name": "fcu-setpoint",
		"devices": [{
			"field": "metadata.traits",
			"matches": {
				"conditions": [{
					"field": "name",
					"stringEqual": "smartcore.traits.AirTemperature"
				}]
			}
		}],
		"source": {
			"trait": "smartcore.traits.AirTemperature",
			"measured": "ambientTemperature.valueCelsius",
			"setPoint": "temperatureSetPoint.valueCelsius"
		},
		"tolerance": 1.5,
		"duration": "1h",
		` + maxDurationLine + `
		"check": {
			"displayName": "Set point tracking"
		}
	}`
}

func (h *testHarness) addDevice(name string, model *airtemperaturepb.Model) node.Undo {
	return h.node.Announce(name,
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(model))),
		node.HasTrait(trait.AirTemperature),
	)
}

func (h *testHarness) updateAirTemp(model *airtemperaturepb.Model, measured, setPoint float64) {
	h.t.Helper()
	_, err := model.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
		AmbientTemperature: &typespb.Temperature{ValueCelsius: measured},
		TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &typespb.Temperature{ValueCelsius: setPoint},
		},
	})
	if err != nil {
		h.t.Fatalf("UpdateAirTemperature failed: %v", err)
	}
	synctest.Wait()
}

// advance moves the fake clock forward by d and lets the worker react.
func (h *testHarness) advance(d time.Duration) {
	h.t.Helper()
	time.Sleep(d)
	synctest.Wait()
}

func (h *testHarness) getCheck(name string) *healthpb.HealthCheck {
	h.t.Helper()
	h.mu.Lock()
	model, ok := h.models[name]
	h.mu.Unlock()
	if !ok {
		h.t.Fatalf("Health model for device %q not found", name)
	}
	check, err := model.GetHealthCheck(checkID)
	if err != nil {
		h.t.Fatalf("Health check for device %q not found: %v", name, err)
	}
	return check
}

func (h *testHarness) waitForHealthCheck(name string) {
	h.t.Helper()
	synctest.Wait()
	h.mu.Lock()
	model, ok := h.models[name]
	h.mu.Unlock()
	if !ok {
		h.t.Fatalf("Health model for device %q not found", name)
	}
	for _, check := range model.ListHealthChecks() {
		if check.GetId() == checkID {
			return
		}
	}
	h.t.Fatalf("Health check %q for device %q was not created", checkID, name)
}

func (h *testHarness) waitForHealthCheckRemoval(name string) {
	h.t.Helper()
	synctest.Wait()
	h.mu.Lock()
	model, ok := h.models[name]
	h.mu.Unlock()
	if !ok {
		return
	}
	for _, check := range model.ListHealthChecks() {
		if check.GetId() == checkID {
			h.t.Fatalf("Health check for device %q was not removed", name)
		}
	}
}

func (h *testHarness) assertHealthCheckExists(name string) {
	h.t.Helper()
	h.getCheck(name)
}

func (h *testHarness) assertNormality(name string, expected healthpb.HealthCheck_Normality) {
	h.t.Helper()
	synctest.Wait()
	got := h.getCheck(name).GetNormality()
	if got != expected {
		h.t.Errorf("device %q normality = %v, want %v", name, got, expected)
	}
}

func (h *testHarness) assertReliability(name string, expected healthpb.HealthCheck_Reliability_State) {
	h.t.Helper()
	synctest.Wait()
	got := h.getCheck(name).GetReliability().GetState()
	if got != expected {
		h.t.Errorf("device %q reliability = %v, want %v", name, got, expected)
	}
}

func (h *testHarness) assertFaultContains(name, substr string) {
	h.t.Helper()
	synctest.Wait()
	faults := h.getCheck(name).GetFaults().GetCurrentFaults()
	var summaries []string
	for _, f := range faults {
		summaries = append(summaries, f.GetSummaryText())
		if strings.Contains(f.GetSummaryText(), substr) {
			return
		}
	}
	h.t.Errorf("device %q faults %v do not contain %q", name, summaries, substr)
}
