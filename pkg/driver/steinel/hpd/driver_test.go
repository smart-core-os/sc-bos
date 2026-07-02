package hpd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/steinel/hpd/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// TestDriver_applyConfig_reapply checks that each device registers a health check under its own
// un-compounded id, and that re-applying config — like service.MonoApply does — doesn't lose
// devices to the asynchronous release of the previous apply's health checks.
func TestDriver_applyConfig_reapply(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Temperature": 20.2, "Humidity": 50}`))
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "https://")

	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data := `{
		"name": "steinel-hpd",
		"passwordFile": "` + strings.ReplaceAll(passwordFile, `\`, `\\`) + `",
		"devices": [
			{"name": "sensors/01", "ipAddress": "` + host + `"},
			{"name": "sensors/02", "ipAddress": "` + host + `"}
		]
	}`
	cfg, err := config.ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	const owner = "driver:steinel-hpd"
	registry := healthpb.NewRegistry()
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(node.New("test")),
		health:    registry.ForOwner(owner),
		logger:    zap.NewNop(),
	}

	checkID := healthpb.AbsID(owner, "commsCheck")
	wantChecks := func(t *testing.T) {
		t.Helper()
		for _, name := range []string{"sensors/01", "sensors/02"} {
			if got := registry.GetCheck(name, checkID); got == nil {
				t.Errorf("GetCheck(%q, %q) = nil, want a registered check", name, checkID)
			}
		}
	}

	ctx1, stop1 := context.WithCancel(context.Background())
	if err := d.applyConfig(ctx1, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	wantChecks(t)

	// MonoApply cancels the previous context then immediately re-applies
	ctx2, stop2 := context.WithCancel(context.Background())
	stop1()
	if err := d.applyConfig(ctx2, cfg); err != nil {
		t.Fatalf("applyConfig (reapply): %v", err)
	}
	wantChecks(t)

	stop2()
	d.devicesStopped()
}

// TestDriver_applyConfig_deviceModel checks that an hpd device does not announce the air
// quality trait, while a multisensor (and a device with no model) does.
func TestDriver_applyConfig_deviceModel(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"SensorName": "steinel", "Temperature": 20.2, "Humidity": 50}`))
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "https://")

	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data := `{
		"name": "steinel-hpd",
		"passwordFile": "` + strings.ReplaceAll(passwordFile, `\`, `\\`) + `",
		"devices": [
			{"name": "sensors/hpd", "ipAddress": "` + host + `", "model": "hpd"},
			{"name": "sensors/multi", "ipAddress": "` + host + `", "model": "multisensor"},
			{"name": "sensors/default", "ipAddress": "` + host + `"}
		]
	}`
	cfg, err := config.ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	n := node.New("test")
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(n),
		health:    healthpb.NewRegistry().ForOwner("driver:steinel-hpd"),
		logger:    zap.NewNop(),
	}

	ctx, stop := context.WithCancel(context.Background())
	if err := d.applyConfig(ctx, cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	defer func() {
		stop()
		d.devicesStopped()
	}()

	deviceTraits := func(t *testing.T, name string) map[string]bool {
		t.Helper()
		dev, err := n.GetDevice(name)
		if err != nil {
			t.Fatalf("GetDevice(%q): %v", name, err)
		}
		traits := make(map[string]bool)
		for _, tm := range dev.GetMetadata().GetTraits() {
			traits[tm.GetName()] = true
		}
		return traits
	}

	const airQuality = "smartcore.traits.AirQualitySensor"
	// every model has these regardless of air quality support
	commonTraits := []string{
		"smartcore.traits.AirTemperature",
		"smartcore.traits.BrightnessSensor",
		"smartcore.traits.OccupancySensor",
		"smartcore.bos.SoundSensor",
	}

	hpd := deviceTraits(t, "sensors/hpd")
	if hpd[airQuality] {
		t.Errorf("sensors/hpd announced %s, want it omitted; traits = %v", airQuality, hpd)
	}
	for _, want := range commonTraits {
		if !hpd[want] {
			t.Errorf("sensors/hpd missing trait %q; traits = %v", want, hpd)
		}
	}

	for _, name := range []string{"sensors/multi", "sensors/default"} {
		got := deviceTraits(t, name)
		if !got[airQuality] {
			t.Errorf("%s missing %s, want it announced; traits = %v", name, airQuality, got)
		}
		for _, want := range commonTraits {
			if !got[want] {
				t.Errorf("%s missing trait %q; traits = %v", name, want, got)
			}
		}
	}
}
