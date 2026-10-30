package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

func Test_setupHealthRegistry(t *testing.T) {
	t.Run("create check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		h.createFaultCheck("dev1", "a test check")

		wantCheck := &healthpb.HealthCheck{
			Id:          healthpb.AbsID("test-owner", ""),
			DisplayName: "a test check",
			CreateTime:  timestamppb.Now(),
			Check:       &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{}},
		}

		h.assertCheckInHealthApi(t, "dev1", wantCheck)
		h.assertCheckInDevicesApi(t, "dev1", wantCheck)
	})

	t.Run("update check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "updatable check")

		check.SetFault(&healthpb.HealthCheck_Error{SummaryText: "test error"})
		h.assertCheckNormality(t, "dev1", healthpb.HealthCheck_ABNORMAL)
		h.assertCheckHasFault(t, "dev1", "test error")

		check.ClearFaults()
		h.assertCheckNormality(t, "dev1", healthpb.HealthCheck_NORMAL)
		h.assertCheckHasNoFaults(t, "dev1")
	})

	t.Run("delete check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "deletable check")

		h.assertCheckExists(t, "dev1")
		check.Dispose()
		h.assertCheckDeleted(t, "dev1")
	})

	t.Run("recreate while disposing", func(t *testing.T) {
		// healthbounds disposes and recreates a check with the same id when its device is reconfigured
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "recreated check")
		for range 50 {
			var wg sync.WaitGroup
			wg.Go(check.Dispose)
			for {
				var err error
				check, err = h.registry.ForOwner(h.owner).NewFaultCheck("dev1", &healthpb.HealthCheck{DisplayName: "recreated check"})
				if err == nil {
					break
				}
				if !errors.Is(err, healthpb.ErrAlreadyExists) {
					t.Fatalf("NewFaultCheck() error = %v", err)
				}
			}
			wg.Wait()

			h.getOnlyCheck(t, "dev1")
			if got := h.getLocalChecks(t, "dev1"); len(got) != 1 {
				t.Fatalf("expected 1 local check, got %d", len(got))
			}
			dev, err := h.devices.GetDevice("dev1")
			if err != nil {
				t.Fatalf("GetDevice() error = %v", err)
			}
			if len(dev.HealthChecks) != 1 {
				t.Fatalf("expected 1 device check, got %d", len(dev.HealthChecks))
			}
		}
		check.Dispose()
		h.assertCheckDeleted(t, "dev1")
	})

	t.Run("seed from history", func(t *testing.T) {
		const (
			deviceName   = "dev2"
			faultSummary = "historical error"
			faultDetails = "this error should be restored"
		)
		h := newHealthTestHarness(t)

		check1 := h.createFaultCheck(deviceName, "seeded check")
		check1.SetFault(&healthpb.HealthCheck_Error{
			SummaryText: faultSummary,
			DetailsText: faultDetails,
		})

		oldCheck := h.getOnlyCheck(t, deviceName)
		oldNormality := oldCheck.Normality

		check1.Dispose()
		h.assertCheckDeleted(t, deviceName)

		// Create a new check with the same ID - it should be seeded from history
		check2 := h.createFaultCheck(deviceName, "seeded check")
		defer check2.Dispose()

		h.assertCheckNormality(t, deviceName, oldNormality)
		h.assertCheckHasFaultWithDetails(t, deviceName, faultSummary, faultDetails)
	})

	t.Run("value filtering", func(t *testing.T) {
		t.Run("measured values in HealthApi", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev3", "temperature check", 25.5)
			defer check.Dispose()
			h.assertCheckHasCurrentValue(t, "dev3", 25.5)
		})

		t.Run("measured values omitted from DevicesApi", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev4", "humidity check", 60.0)
			defer check.Dispose()
			h.assertDeviceCheckHasNoCurrentValue(t, "dev4")
		})

		t.Run("update measured value", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev5", "pressure check", 100.0)
			defer check.Dispose()

			check.UpdateValue(h.ctx, healthpb.FloatValue(120.0))
			h.assertCheckHasCurrentValue(t, "dev5", 120.0)
			h.assertDeviceCheckHasNoCurrentValue(t, "dev5")
		})
	})
}

func Test_setupHealthRegistry_localChecks(t *testing.T) {
	t.Run("create update delete", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createBoundsCheck("dev1", "temperature check", 25.5)

		got := h.getLocalChecks(t, "dev1")
		if len(got) != 1 {
			t.Fatalf("expected 1 local check, got %d", len(got))
		}
		// the mirror keeps measured values
		if v := got[0].GetBounds().GetCurrentValue().GetFloatValue(); v != 25.5 {
			t.Errorf("local check current_value = %v, want 25.5", v)
		}

		check.UpdateValue(h.ctx, healthpb.FloatValue(120.0))
		got = h.getLocalChecks(t, "dev1")
		if v := got[0].GetBounds().GetCurrentValue().GetFloatValue(); v != 120.0 {
			t.Errorf("local check current_value = %v, want 120", v)
		}
		if got[0].Normality != healthpb.HealthCheck_HIGH {
			t.Errorf("local check normality = %v, want HIGH", got[0].Normality)
		}

		check.Dispose()
		if _, err := h.localChecks.GetDevice("dev1"); status.Code(err) != codes.NotFound {
			t.Errorf("expected local checks for dev1 to be removed, got err %v", err)
		}
	})

	t.Run("only one of many removed", func(t *testing.T) {
		h := newHealthTestHarness(t)
		c1 := h.createFaultCheck("dev1", "first")
		c2, err := h.registry.ForOwner("other-owner").NewFaultCheck("dev1", &healthpb.HealthCheck{DisplayName: "second"})
		if err != nil {
			t.Fatalf("NewFaultCheck() error = %v", err)
		}
		defer c2.Dispose()
		if got := h.getLocalChecks(t, "dev1"); len(got) != 2 {
			t.Fatalf("expected 2 local checks, got %d", len(got))
		}
		c1.Dispose()
		got := h.getLocalChecks(t, "dev1")
		if len(got) != 1 || got[0].DisplayName != "second" {
			t.Errorf("expected only the second check to remain, got %v", got)
		}
	})

	t.Run("seeded value", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check1 := h.createFaultCheck("dev1", "seeded check")
		check1.SetFault(&healthpb.HealthCheck_Error{SummaryText: "historical error"})
		check1.Dispose()

		check2 := h.createFaultCheck("dev1", "seeded check")
		defer check2.Dispose()
		got := h.getLocalChecks(t, "dev1")
		if len(got) != 1 {
			t.Fatalf("expected 1 local check, got %d", len(got))
		}
		if got[0].Normality != healthpb.HealthCheck_ABNORMAL {
			t.Errorf("local check normality = %v, want seeded ABNORMAL", got[0].Normality)
		}
	})

	t.Run("create_time survives updates", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "check")
		defer check.Dispose()
		created := h.getLocalChecks(t, "dev1")[0].GetCreateTime()
		if created == nil {
			t.Fatalf("expected create_time to be set")
		}

		check.SetFault(&healthpb.HealthCheck_Error{SummaryText: "one"})
		check.ClearFaults()
		check.SetFault(&healthpb.HealthCheck_Error{SummaryText: "two"})

		id := healthpb.AbsID(h.owner, "")
		if got := h.registry.GetCheck("dev1", id).GetCreateTime(); !proto.Equal(got, created) {
			t.Errorf("registry create_time = %v, want %v", got, created)
		}
		if got := h.getLocalChecks(t, "dev1")[0].GetCreateTime(); !proto.Equal(got, created) {
			t.Errorf("local create_time = %v, want %v", got, created)
		}
		dev, err := h.devices.GetDevice("dev1")
		if err != nil {
			t.Fatalf("GetDevice() error = %v", err)
		}
		if got := dev.HealthChecks[0].GetCreateTime(); !proto.Equal(got, created) {
			t.Errorf("devices create_time = %v, want %v", got, created)
		}
	})

	t.Run("excludes checks written to the device store directly", func(t *testing.T) {
		h := newHealthTestHarness(t)
		err := devicesToHealthCheckCollection(h.devices).MergeHealthChecks("remote1", &healthpb.HealthCheck{Id: "remote-check"})
		if err != nil {
			t.Fatalf("MergeHealthChecks() error = %v", err)
		}
		if got := h.localChecks.ListDevices(); len(got) != 0 {
			t.Errorf("expected no local checks, got %v", got)
		}
	})
}

func Test_serviceSystemCheck(t *testing.T) {
	type startFunc func(c *Controller, name, kind string) (*service.Map, error)
	tests := []struct {
		name     string
		start    startFunc
		wantType metadatapb.Metadata_DeviceType
	}{
		{
			name: "driver",
			start: func(c *Controller, name, kind string) (*service.Map, error) {
				return c.startDrivers([]driver.RawConfig{{BaseConfig: driver.BaseConfig{Name: name, Type: kind}, Raw: []byte(`{}`)}})
			},
			wantType: metadatapb.Metadata_SERVICE,
		},
		{
			name: "automation",
			start: func(c *Controller, name, kind string) (*service.Map, error) {
				return c.startAutomations([]auto.RawConfig{{Config: auto.Config{Name: name, Type: kind}, Raw: []byte(`{}`)}})
			},
			wantType: metadatapb.Metadata_AUTOMATION,
		},
		{
			name: "zone",
			start: func(c *Controller, name, kind string) (*service.Map, error) {
				return c.startZones([]zone.RawConfig{{Config: zone.Config{Name: name, Type: kind}, Raw: []byte(`{}`)}})
			},
			wantType: metadatapb.Metadata_ZONE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("reliable without reporting", func(t *testing.T) {
				// SCB-1515: a service that never reports stayed unspecified
				h, c, _ := newSystemCheckHarness(t)
				m, err := tt.start(c, "svc1", "test")
				if err != nil {
					t.Fatalf("start error = %v", err)
				}
				h.waitForSystemCheck(t, "svc1", healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_Reliability_RELIABLE)
				dev, err := h.devices.GetDevice("svc1")
				if err != nil {
					t.Fatalf("GetDevice() error = %v", err)
				}
				if got := dev.GetMetadata().GetDeviceType(); got != tt.wantType {
					t.Errorf("device type = %v, want %v", got, tt.wantType)
				}
				if _, err := m.Delete("svc1"); err != nil {
					t.Fatalf("Delete() error = %v", err)
				}
				h.waitForDeviceRemoved(t, "svc1")
			})

			t.Run("back after restart", func(t *testing.T) {
				// SCB-1477: the check was disposed on stop and never came back
				h, c, reports := newSystemCheckHarness(t)
				m, err := tt.start(c, "svc1", "test")
				if err != nil {
					t.Fatalf("start error = %v", err)
				}
				h.waitForSystemCheck(t, "svc1", healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_Reliability_RELIABLE)
				svc := m.Get("svc1").Service
				if _, err := svc.Stop(); err != nil {
					t.Fatalf("Stop() error = %v", err)
				}
				h.waitForDeviceRemoved(t, "svc1")
				if _, err := svc.Start(); err != nil {
					t.Fatalf("Start() error = %v", err)
				}
				h.waitForSystemCheck(t, "svc1", healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_Reliability_RELIABLE)

				(<-reports).MarkFailed(errors.New("connection lost"))
				h.waitForSystemCheck(t, "svc1", healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_Reliability_BAD_RESPONSE)
				_, _ = m.Delete("svc1")
			})

			t.Run("apply error", func(t *testing.T) {
				h, c, _ := newSystemCheckHarness(t)
				m, err := tt.start(c, "svc1", "failing")
				if err != nil {
					t.Fatalf("start error = %v", err)
				}
				h.waitForSystemCheck(t, "svc1", healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_Reliability_BAD_RESPONSE)
				_, _ = m.Delete("svc1")
				h.waitForDeviceRemoved(t, "svc1")
			})

			t.Run("unsupported kind", func(t *testing.T) {
				h, c, _ := newSystemCheckHarness(t)
				m, _ := tt.start(c, "svc1", "unknown")
				if m.Get("svc1") == nil {
					t.Fatalf("expected the record to be kept in an errored state")
				}
				if _, err := h.devices.GetDevice("svc1"); status.Code(err) != codes.NotFound {
					t.Errorf("expected no svc1 device for an unsupported kind, got err %v", err)
				}
			})
		})
	}
}

// newSystemCheckHarness returns a Controller with a "test" and a "failing" factory for each kind of service.
// Each test service sends its SystemCheck on the returned chan when it is created.
func newSystemCheckHarness(t *testing.T) (*healthTestHarness, *Controller, <-chan service.SystemCheck) {
	h := newHealthTestHarness(t)
	reports := make(chan service.SystemCheck, 1)
	newService := func(check service.SystemCheck, applyErr error) service.Lifecycle {
		if applyErr == nil {
			reports <- check
		}
		return service.New(func(context.Context, struct{}) error { return applyErr })
	}
	errApply := errors.New("apply failed")
	c := &Controller{
		SystemConfig: sysconf.Config{
			DriverFactories: map[string]driver.Factory{
				"test":    driver.FactoryFunc(func(s driver.Services) service.Lifecycle { return newService(s.SystemCheck, nil) }),
				"failing": driver.FactoryFunc(func(s driver.Services) service.Lifecycle { return newService(s.SystemCheck, errApply) }),
			},
			AutoFactories: map[string]auto.Factory{
				"test":    auto.FactoryFunc(func(s auto.Services) service.Lifecycle { return newService(s.SystemCheck, nil) }),
				"failing": auto.FactoryFunc(func(s auto.Services) service.Lifecycle { return newService(s.SystemCheck, errApply) }),
			},
			ZoneFactories: map[string]zone.Factory{
				"test":    zone.FactoryFunc(func(s zone.Services) service.Lifecycle { return newService(s.SystemCheck, nil) }),
				"failing": zone.FactoryFunc(func(s zone.Services) service.Lifecycle { return newService(s.SystemCheck, errApply) }),
			},
		},
		ControllerConfig: &immutableConfigStore{},
		Logger:           zaptest.NewLogger(t),
		Node:             h.node,
		CheckRegistry:    h.registry,
	}
	return h, c, reports
}

// waitForSystemCheck waits for name's system check to have the given normality and reliability.
func (h *healthTestHarness) waitForSystemCheck(t *testing.T, name string, want healthpb.HealthCheck_Normality, wantReliability healthpb.HealthCheck_Reliability_State) {
	t.Helper()
	var got *healthpb.HealthCheck
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		dev, err := h.devices.GetDevice(name)
		if err != nil || len(dev.HealthChecks) != 1 {
			continue
		}
		got = dev.HealthChecks[0]
		if got.GetNormality() == want && got.GetReliability().GetState() == wantReliability {
			return
		}
	}
	t.Fatalf("%s system check = %v, want %v and %v", name, got, want, wantReliability)
}

func (h *healthTestHarness) waitForDeviceRemoved(t *testing.T, name string) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if _, err := h.devices.GetDevice(name); status.Code(err) == codes.NotFound {
			return
		}
	}
	t.Fatalf("%s still present", name)
}

func Test_removeMeasuredValues(t *testing.T) {
	tests := []struct {
		name         string
		input        *healthpb.HealthCheck
		wantVal      *healthpb.HealthCheck
		wantNoChange bool
	}{
		{
			name:         "nil check",
			input:        nil,
			wantNoChange: true,
		},
		{
			name: "fault check unchanged",
			input: &healthpb.HealthCheck{
				Id:          "test-id",
				DisplayName: "test check",
				Check: &healthpb.HealthCheck_Faults_{
					Faults: &healthpb.HealthCheck_Faults{
						CurrentFaults: []*healthpb.HealthCheck_Error{
							{SummaryText: "test error"},
						},
					},
				},
			},
			wantNoChange: true,
		},
		{
			name: "bounds check without current_value unchanged",
			input: &healthpb.HealthCheck{
				Id:          "test-id",
				DisplayName: "test check",
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: &healthpb.HealthCheck_Bounds_NormalValue{
							NormalValue: healthpb.FloatValue(20.0),
						},
						DisplayUnit: "°C",
					},
				},
			},
			wantNoChange: true,
		},
		{
			name: "bounds check with current_value removed",
			input: &healthpb.HealthCheck{
				Id:          "temp-check",
				DisplayName: "Room Temperature",
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						CurrentValue: healthpb.FloatValue(21.5),
						Expected: &healthpb.HealthCheck_Bounds_NormalRange{
							NormalRange: &healthpb.HealthCheck_ValueRange{
								Low:  healthpb.FloatValue(18.0),
								High: healthpb.FloatValue(24.0),
							},
						},
						DisplayUnit: "°C",
					},
				},
			},
			wantVal: &healthpb.HealthCheck{
				Id:          "temp-check",
				DisplayName: "Room Temperature",
				Check: &healthpb.HealthCheck_Bounds_{
					Bounds: &healthpb.HealthCheck_Bounds{
						Expected: &healthpb.HealthCheck_Bounds_NormalRange{
							NormalRange: &healthpb.HealthCheck_ValueRange{
								Low:  healthpb.FloatValue(18.0),
								High: healthpb.FloatValue(24.0),
							},
						},
						DisplayUnit: "°C",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTransform := proto.Clone(tt.input)
			got := removeMeasuredValues(tt.input)
			want := tt.wantVal
			if tt.wantNoChange {
				want = tt.input
			}

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("removeMeasuredValues() mismatch (-want +got):\n%s", diff)
			}
			// make sure input was not modified
			if diff := cmp.Diff(beforeTransform, tt.input, protocmp.Transform()); diff != "" {
				t.Errorf("input was modified (-want +got):\n%s", diff)
			}
		})
	}
}

// healthTestHarness provides a convenient interface for testing health registry functionality
type healthTestHarness struct {
	ctx             context.Context
	t               *testing.T
	registry        *healthpb.Registry
	devices         *devicespb.Collection
	localChecks     *devicespb.Collection
	node            *node.Node
	healthApiClient healthpb.HealthApiClient
	owner           string
}

func newHealthTestHarness(t *testing.T) *healthTestHarness {
	ctx := t.Context()
	cfg := sysconf.Config{
		DataDir: t.TempDir(),
		Health:  &sysconf.Health{},
	}
	devices := devicespb.NewCollection()
	announcer := node.New("test-node", nodeopts.WithStore(devices))
	healthApiClient := healthpb.NewHealthApiClient(announcer.ClientConn())

	r, localChecks, dispose, err := setupHealthRegistry(ctx, cfg, devices, announcer, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("setupHealthRegistry() error = %v", err)
	}
	t.Cleanup(func() {
		if err := dispose(); err != nil {
			t.Errorf("dispose() error = %v", err)
		}
	})

	devs := devices.ListDevices()
	if len(devs) != 1 { // node itself
		t.Fatalf("expected 1 device in device store, got %d", len(devs))
	}

	return &healthTestHarness{
		ctx:             ctx,
		t:               t,
		registry:        r,
		devices:         devices,
		localChecks:     localChecks,
		node:            announcer,
		healthApiClient: healthApiClient,
		owner:           "test-owner",
	}
}

func (h *healthTestHarness) createFaultCheck(deviceName, displayName string) *healthpb.FaultCheck {
	check, err := h.registry.ForOwner(h.owner).NewFaultCheck(deviceName, &healthpb.HealthCheck{
		DisplayName: displayName,
	})
	if err != nil {
		h.t.Fatalf("NewFaultCheck() error = %v", err)
	}
	return check
}

func (h *healthTestHarness) createBoundsCheck(deviceName, displayName string, currentValue float64) *healthpb.BoundsCheck {
	check, err := h.registry.ForOwner(h.owner).NewBoundsCheck(deviceName, &healthpb.HealthCheck{
		DisplayName: displayName,
		Check: &healthpb.HealthCheck_Bounds_{
			Bounds: &healthpb.HealthCheck_Bounds{
				Expected: &healthpb.HealthCheck_Bounds_NormalRange{
					NormalRange: &healthpb.HealthCheck_ValueRange{
						Low:  healthpb.FloatValue(0.0),
						High: healthpb.FloatValue(100.0),
					},
				},
			},
		},
	})
	if err != nil {
		h.t.Fatalf("NewBoundsCheck() error = %v", err)
	}
	// Set initial value
	check.UpdateValue(h.ctx, healthpb.FloatValue(currentValue))
	return check
}

func (h *healthTestHarness) getLocalChecks(t *testing.T, deviceName string) []*healthpb.HealthCheck {
	t.Helper()
	dev, err := h.localChecks.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("localChecks.GetDevice() error = %v", err)
	}
	if dev.Name != deviceName {
		t.Errorf("local device name = %q, want %q", dev.Name, deviceName)
	}
	return dev.HealthChecks
}

func (h *healthTestHarness) getOnlyCheck(t *testing.T, deviceName string) *healthpb.HealthCheck {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &healthpb.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("ListHealthChecks() error = %v", err)
	}
	if len(apiChecks.HealthChecks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(apiChecks.HealthChecks))
	}
	return apiChecks.HealthChecks[0]
}

func (h *healthTestHarness) assertCheckInHealthApi(t *testing.T, deviceName string, want *healthpb.HealthCheck) {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &healthpb.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("ListHealthChecks() error = %v", err)
	}
	assertHealthChecks(t, apiChecks.HealthChecks, want)
}

func (h *healthTestHarness) assertCheckInDevicesApi(t *testing.T, deviceName string, want *healthpb.HealthCheck) {
	t.Helper()
	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	// Device API should omit measured values
	wantWithoutMeasured := removeMeasuredValues(want)
	assertHealthChecks(t, dev.HealthChecks, wantWithoutMeasured)
}

func (h *healthTestHarness) assertCheckNormality(t *testing.T, deviceName string, want healthpb.HealthCheck_Normality) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	if gotCheck.Normality != want {
		t.Errorf("expected normality %v, got %v", want, gotCheck.Normality)
	}

	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	if len(dev.HealthChecks) > 0 && dev.HealthChecks[0].Normality != want {
		t.Errorf("expected normality %v in device, got %v", want, dev.HealthChecks[0].Normality)
	}
}

func (h *healthTestHarness) assertCheckHasFault(t *testing.T, deviceName, faultSummary string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("expected 1 fault, got %d", len(faults))
	}
	if faults[0].SummaryText != faultSummary {
		t.Errorf("expected fault summary %q, got %q", faultSummary, faults[0].SummaryText)
	}
}

func (h *healthTestHarness) assertCheckHasFaultWithDetails(t *testing.T, deviceName, faultSummary, faultDetails string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("expected 1 fault, got %d", len(faults))
	}
	if faults[0].SummaryText != faultSummary {
		t.Errorf("expected fault summary %q, got %q", faultSummary, faults[0].SummaryText)
	}
	if faults[0].DetailsText != faultDetails {
		t.Errorf("expected fault details %q, got %q", faultDetails, faults[0].DetailsText)
	}
}

func (h *healthTestHarness) assertCheckHasNoFaults(t *testing.T, deviceName string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(faults))
	}
}

func (h *healthTestHarness) assertCheckExists(t *testing.T, deviceName string) {
	t.Helper()
	_, err := h.healthApiClient.ListHealthChecks(h.ctx, &healthpb.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("expected check to exist, but got error: %v", err)
	}
}

func (h *healthTestHarness) assertCheckDeleted(t *testing.T, deviceName string) {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &healthpb.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			t.Fatalf("unexpected error from ListHealthChecks() = %v", err)
		}
	} else if len(apiChecks.HealthChecks) != 0 {
		t.Errorf("expected 0 checks after disposal, got %d", len(apiChecks.HealthChecks))
	}

	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			t.Fatalf("GetDevice() error = %v", err)
		}
	} else {
		if dev == nil {
			t.Fatalf("GetDevice() returned nil device")
		}
		if len(dev.HealthChecks) != 0 {
			t.Errorf("expected 0 checks in device after disposal, got %d", len(dev.HealthChecks))
		}
	}
}

func (h *healthTestHarness) assertCheckHasCurrentValue(t *testing.T, deviceName string, expectedValue float64) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	bounds := gotCheck.GetBounds()
	if bounds == nil {
		t.Fatalf("expected bounds check, got %T", gotCheck.GetCheck())
	}
	currentValue := bounds.GetCurrentValue()
	if currentValue == nil {
		t.Fatalf("expected current_value to be set, got nil")
	}
	gotValue := currentValue.GetFloatValue()
	if gotValue != expectedValue {
		t.Errorf("expected current_value %v, got %v", expectedValue, gotValue)
	}
}

func (h *healthTestHarness) assertDeviceCheckHasNoCurrentValue(t *testing.T, deviceName string) {
	t.Helper()
	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	if len(dev.HealthChecks) == 0 {
		t.Fatalf("expected at least 1 check in device, got 0")
	}
	bounds := dev.HealthChecks[0].GetBounds()
	if bounds == nil {
		t.Fatalf("expected bounds check, got %T", dev.HealthChecks[0].GetCheck())
	}
	if bounds.GetCurrentValue() != nil {
		t.Errorf("expected current_value to be nil in device API, got %v", bounds.GetCurrentValue())
	}
}

func assertHealthChecks(t *testing.T, got []*healthpb.HealthCheck, want ...*healthpb.HealthCheck) {
	t.Helper()
	if diff := cmp.Diff(want, got, protocmp.Transform(), protocmp.IgnoreFields(&healthpb.HealthCheck{}, "create_time")); diff != "" {
		t.Errorf("health checks mismatch (-want +got):\n%s", diff)
	}
}
