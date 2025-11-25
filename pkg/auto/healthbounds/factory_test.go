package healthbounds

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

// TestTracksDeviceValues verifies that the automation extracts values
// from device traits and updates health checks.
func TestTracksDeviceValues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureAirTempMonitor()

		airTempModel := airtemperaturepb.NewModel()
		h.addAirTempDevice("room-1", airTempModel)
		h.waitForHealthCheck("room-1")

		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 21.5},
		})

		h.assertHealthCheckValue("room-1", 21.5)
	})
}

// TestCreatesAndRemovesHealthChecks verifies that when devices are
// added or removed, health checks are created or disposed accordingly.
func TestCreatesAndRemovesHealthChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureAirTempMonitor()

		h.assertNoHealthChecks()

		airTempModel1 := airtemperaturepb.NewModel()
		undo1 := h.addAirTempDevice("room-1", airTempModel1)
		h.waitForHealthCheck("room-1")

		airTempModel2 := airtemperaturepb.NewModel()
		undo2 := h.addAirTempDevice("room-2", airTempModel2)
		h.waitForHealthCheck("room-2")

		h.assertHealthCheckExists("room-1")
		h.assertHealthCheckExists("room-2")

		undo1()
		h.waitForHealthCheckRemoval("room-1")

		h.assertHealthCheckDoesNotExist("room-1")
		h.assertHealthCheckExists("room-2")

		undo2()
		h.waitForHealthCheckRemoval("room-2")

		h.assertNoHealthChecks()
	})
}

// TestCreatesHealthCheckPerDevice verifies that each device gets
// its own independent health check.
func TestCreatesHealthCheckPerDevice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureAirTempMonitor()

		airTempModel1 := airtemperaturepb.NewModel()
		h.addAirTempDevice("room-1", airTempModel1)
		airTempModel2 := airtemperaturepb.NewModel()
		h.addAirTempDevice("room-2", airTempModel2)

		h.waitForHealthCheck("room-1")
		h.waitForHealthCheck("room-2")

		_, _ = airTempModel1.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 20.0},
		})
		_, _ = airTempModel2.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 22.0},
		})

		h.assertHealthCheckValue("room-1", 20.0)
		h.assertHealthCheckValue("room-2", 22.0)
	})
}

// TestUpdatesNormality verifies that the automation updates the normality
// when values change in and out of normal ranges.
func TestUpdatesNormality(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureAirTempMonitor()

		airTempModel := airtemperaturepb.NewModel()
		h.addAirTempDevice("room-1", airTempModel)
		h.waitForHealthCheck("room-1")

		// Normal value (within 15-25 range)
		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 20.0},
		})
		h.assertHealthCheckNormality("room-1", gen.HealthCheck_NORMAL)

		// Low value (below 15)
		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 10.0},
		})
		h.assertHealthCheckNormality("room-1", gen.HealthCheck_LOW)

		// High value (above 25)
		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 30.0},
		})
		h.assertHealthCheckNormality("room-1", gen.HealthCheck_HIGH)

		// Back to normal
		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 22.0},
		})
		h.assertHealthCheckNormality("room-1", gen.HealthCheck_NORMAL)
	})
}

// TestHealthCheckProperties verifies that the created health check
// matches the configured properties.
func TestHealthCheckProperties(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTestHarness(t)
		h.configureAirTempMonitor()

		airTempModel := airtemperaturepb.NewModel()
		h.addAirTempDevice("room-1", airTempModel)
		h.waitForHealthCheck("room-1")

		_, _ = airTempModel.UpdateAirTemperature(&traits.AirTemperature{
			AmbientTemperature: &types.Temperature{ValueCelsius: 20.0},
		})

		want := &gen.HealthCheck{
			Id:          "healthbounds",
			DisplayName: "Ambient Temperature",
			Check: &gen.HealthCheck_Bounds_{
				Bounds: &gen.HealthCheck_Bounds{
					CurrentValue: &gen.HealthCheck_Value{
						Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 20.0},
					},
					Expected: &gen.HealthCheck_Bounds_NormalRange{
						NormalRange: &gen.HealthCheck_ValueRange{
							Low:  &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 15.0}},
							High: &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 25.0}},
						},
					},
				},
			},
			Normality: gen.HealthCheck_NORMAL,
		}

		h.assertHealthCheck("room-1", want)
	})
}

// testHarness provides a convenient test environment for healthbounds automation.
type testHarness struct {
	t      *testing.T
	node   *node.Node
	models map[string]*healthpb.Model // health models per device
	mu     sync.Mutex                 // protects models map
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
		healthpb.WithOnCheckCreate(func(name string, c *gen.HealthCheck) *gen.HealthCheck {
			h.mu.Lock()
			defer h.mu.Unlock()
			if model, ok := h.models[name]; ok {
				_, _ = model.CreateHealthCheck(c)
			}
			return c
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen.HealthCheck) {
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

	devicesClient := gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devices.NewServer(n)))

	services := auto.Services{
		Logger:  zaptest.NewLogger(t),
		Node:    n,
		Devices: devicesClient,
		Health:  registry.ForOwner("healthbounds"),
	}

	a := Factory.New(services)

	if _, err := a.Start(); err != nil {
		t.Fatalf("Failed to start automation: %v", err)
	}

	t.Cleanup(func() {
		_, _ = a.Stop()
	})

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

func (h *testHarness) configureAirTempMonitor() {
	h.t.Helper()
	h.configure(`{
		"type": "healthbounds",
		"name": "temp-monitor",
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
			"value": "ambientTemperature.valueCelsius"
		},
		"check": {
			"displayName": "Ambient Temperature",
			"bounds": {
				"normalRange": {
					"low": {"floatValue": 15.0},
					"high": {"floatValue": 25.0}
				}
			}
		}
	}`)
}

func (h *testHarness) addAirTempDevice(name string, model *airtemperaturepb.Model) node.Undo {
	return h.node.Announce(name,
		node.HasTrait(trait.AirTemperature, node.WithClients(airtemperaturepb.WrapApi(airtemperaturepb.NewModelServer(model)))),
	)
}

func (h *testHarness) waitForHealthCheck(deviceName string) {
	h.t.Helper()
	synctest.Wait()

	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		h.t.Fatalf("Health model for device %q not found", deviceName)
	}

	checks := model.ListHealthChecks()
	for _, check := range checks {
		if check.GetId() == "healthbounds" {
			return
		}
	}

	var foundIDs []string
	for _, check := range checks {
		foundIDs = append(foundIDs, check.GetId())
	}
	h.t.Fatalf("Health check for device %q (expected ID %q) was not created. Found %d checks: %v",
		deviceName, "healthbounds", len(foundIDs), foundIDs)
}

func (h *testHarness) waitForHealthCheckRemoval(deviceName string) {
	h.t.Helper()
	synctest.Wait()

	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		return
	}

	checks := model.ListHealthChecks()
	for _, check := range checks {
		if check.GetId() == "healthbounds" {
			h.t.Fatalf("Health check for device %q was not removed", deviceName)
		}
	}
}

func (h *testHarness) assertHealthCheckValue(deviceName string, expected float64) {
	h.t.Helper()
	synctest.Wait()

	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		h.t.Fatalf("Health model for device %q not found", deviceName)
	}
	check, err := model.GetHealthCheck("healthbounds")
	if err != nil {
		h.t.Fatalf("Health check for device %q not found: %v", deviceName, err)
	}
	got := check.GetBounds().GetCurrentValue().GetFloatValue()
	if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
		h.t.Errorf("Health check value mismatch (-want +got):\n%s", diff)
	}
}

func (h *testHarness) assertHealthCheckNormality(deviceName string, expected gen.HealthCheck_Normality) {
	h.t.Helper()
	synctest.Wait()

	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		h.t.Fatalf("Health model for device %q not found", deviceName)
	}
	check, err := model.GetHealthCheck("healthbounds")
	if err != nil {
		h.t.Fatalf("Health check for device %q not found: %v", deviceName, err)
	}
	got := check.GetNormality()
	if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
		h.t.Errorf("Health check normality mismatch (-want +got):\n%s", diff)
	}
}

func (h *testHarness) assertHealthCheck(deviceName string, expected *gen.HealthCheck) {
	h.t.Helper()
	synctest.Wait()

	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		h.t.Fatalf("Health model for device %q not found", deviceName)
	}
	check, err := model.GetHealthCheck("healthbounds")
	if err != nil {
		h.t.Fatalf("Health check for device %q not found: %v", deviceName, err)
	}
	if diff := cmp.Diff(expected, check, protocmp.Transform(), protocmp.IgnoreFields(&gen.HealthCheck{}, "create_time", "normal_time", "abnormal_time", "reliability")); diff != "" {
		h.t.Errorf("Health check mismatch (-want +got):\n%s", diff)
	}
}

func (h *testHarness) assertHealthCheckExists(deviceName string) {
	h.t.Helper()
	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		h.t.Fatalf("Health model for device %q not found", deviceName)
	}
	_, err := model.GetHealthCheck("healthbounds")
	if err != nil {
		h.t.Fatalf("Health check for device %q should exist but doesn't: %v", deviceName, err)
	}
}

func (h *testHarness) assertHealthCheckDoesNotExist(deviceName string) {
	h.t.Helper()
	h.mu.Lock()
	model, ok := h.models[deviceName]
	h.mu.Unlock()

	if !ok {
		return
	}
	checks := model.ListHealthChecks()
	for _, check := range checks {
		if check.GetId() == "healthbounds" {
			h.t.Fatalf("Health check for device %q should not exist but does", deviceName)
		}
	}
}

func (h *testHarness) assertNoHealthChecks() {
	h.t.Helper()
	synctest.Wait()
	h.mu.Lock()
	numModels := len(h.models)
	var deviceNames []string
	for name := range h.models {
		deviceNames = append(deviceNames, name)
	}
	h.mu.Unlock()

	if numModels > 0 {
		h.t.Fatalf("Expected no health models, but found models for %d devices: %v", numModels, deviceNames)
	}
}

// TestConfigValidation tests that invalid configurations are rejected.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid config",
			config: `{
				"type": "healthbounds",
				"name": "test",
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
					"trait": "smartcore.traits.AirTemperature"
				},
				"check": {
					"displayName": "Test",
					"bounds": {
						"normalRange": {
							"low": {"floatValue": 10.0},
							"high": {"floatValue": 30.0}
						}
					}
				}
			}`,
			wantError: false,
		},
		{
			name:      "invalid json",
			config:    `{invalid json`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarness(t)

			_, err := h.auto.Configure([]byte(tt.config))
			if tt.wantError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
