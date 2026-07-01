package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseConfig_legacySingleDevice(t *testing.T) {
	data := `{
		"name": "steinel-hpd",
		"type": "steinel-hpd",
		"ipAddress": "10.0.0.1",
		"passwordFile": ` + quote(writePasswordFile(t, "secret")) + `,
		"udmiTopicPrefix": "topic/prefix",
		"metadata": {"membership": {"subsystem": "sensors"}}
	}`
	root, err := ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(root.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(root.Devices))
	}
	dev := root.Devices[0]
	if dev.Name != "steinel-hpd" {
		t.Errorf("device name = %q, want %q", dev.Name, "steinel-hpd")
	}
	if dev.IpAddress != "10.0.0.1" {
		t.Errorf("device ipAddress = %q, want %q", dev.IpAddress, "10.0.0.1")
	}
	if dev.ResolvedPassword() != "secret" {
		t.Errorf("device password = %q, want %q", dev.ResolvedPassword(), "secret")
	}
	if dev.UDMITopicPrefix != "topic/prefix" {
		t.Errorf("device udmiTopicPrefix = %q, want %q", dev.UDMITopicPrefix, "topic/prefix")
	}
	if dev.Metadata.GetMembership().GetSubsystem() != "sensors" {
		t.Errorf("device metadata subsystem = %q, want %q", dev.Metadata.GetMembership().GetSubsystem(), "sensors")
	}
	if got, want := dev.PollInterval.Duration, DefaultPollInterval; got != want {
		t.Errorf("device pollInterval = %v, want %v", got, want)
	}
}

func TestParseConfig_multiDevice(t *testing.T) {
	data := `{
		"name": "steinel-hpd",
		"type": "steinel-hpd",
		"passwordFile": ` + quote(writePasswordFile(t, "shared")) + `,
		"pollInterval": "30s",
		"devices": [
			{"name": "sensors/01", "ipAddress": "10.0.0.1", "udmiTopicPrefix": "site/01", "metadata": {"membership": {"subsystem": "sensors"}}},
			{"name": "sensors/02", "ipAddress": "10.0.0.2", "passwordFile": ` + quote(writePasswordFile(t, "override")) + `, "pollInterval": "10s", "udmiTopicPrefix": "site/02"}
		]
	}`
	root, err := ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(root.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(root.Devices))
	}
	d0, d1 := root.Devices[0], root.Devices[1]
	if d0.ResolvedPassword() != "shared" {
		t.Errorf("devices[0] password = %q, want shared default %q", d0.ResolvedPassword(), "shared")
	}
	if got, want := d0.PollInterval.Duration, 30*time.Second; got != want {
		t.Errorf("devices[0] pollInterval = %v, want root default %v", got, want)
	}
	if d0.UDMITopicPrefix != "site/01" {
		t.Errorf("devices[0] udmiTopicPrefix = %q, want %q", d0.UDMITopicPrefix, "site/01")
	}
	if d0.Metadata.GetMembership().GetSubsystem() != "sensors" {
		t.Errorf("devices[0] metadata subsystem = %q, want %q", d0.Metadata.GetMembership().GetSubsystem(), "sensors")
	}
	if d1.ResolvedPassword() != "override" {
		t.Errorf("devices[1] password = %q, want %q", d1.ResolvedPassword(), "override")
	}
	if got, want := d1.PollInterval.Duration, 10*time.Second; got != want {
		t.Errorf("devices[1] pollInterval = %v, want %v", got, want)
	}
	if d1.UDMITopicPrefix != "site/02" {
		t.Errorf("devices[1] udmiTopicPrefix = %q, want %q", d1.UDMITopicPrefix, "site/02")
	}
}

func TestParseConfig_multiDeviceDefaults(t *testing.T) {
	// no pollInterval or udmiTopicPrefix anywhere: devices poll at the default rate,
	// and multiple devices without a udmi topic prefix are allowed
	data := `{
		"name": "steinel-hpd",
		"passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
		"devices": [
			{"name": "sensors/01", "ipAddress": "10.0.0.1"},
			{"name": "sensors/02", "ipAddress": "10.0.0.2"}
		]
	}`
	root, err := ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	for i, dev := range root.Devices {
		if got, want := dev.PollInterval.Duration, DefaultPollInterval; got != want {
			t.Errorf("devices[%d] pollInterval = %v, want %v", i, got, want)
		}
		if dev.UDMITopicPrefix != "" {
			t.Errorf("devices[%d] udmiTopicPrefix = %q, want empty", i, dev.UDMITopicPrefix)
		}
	}
}

func TestParseConfig_deviceModel(t *testing.T) {
	data := `{
		"name": "steinel-hpd",
		"passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
		"devices": [
			{"name": "sensors/hpd", "ipAddress": "10.0.0.1", "model": "hpd"},
			{"name": "sensors/multi", "ipAddress": "10.0.0.2", "model": "multisensor"},
			{"name": "sensors/default", "ipAddress": "10.0.0.3"}
		]
	}`
	root, err := ParseConfig([]byte(data))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	// the hpd has no air quality module
	if hpd := root.Devices[0]; hpd.HasAirQuality() {
		t.Errorf("devices[0] model %q: HasAirQuality() = true, want false", hpd.Model)
	}
	// the multisensor and the default (no model) both have air quality
	if multi := root.Devices[1]; !multi.HasAirQuality() {
		t.Errorf("devices[1] model %q: HasAirQuality() = false, want true", multi.Model)
	}
	if def := root.Devices[2]; !def.HasAirQuality() {
		t.Errorf("devices[2] model %q: HasAirQuality() = false, want true", def.Model)
	}
}

func TestParseConfig_errors(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name:    "no devices or ipAddress",
			data:    `{"name": "steinel-hpd"}`,
			wantErr: "one of ipAddress or devices is required",
		},
		{
			name:    "both devices and ipAddress",
			data:    `{"name": "x", "ipAddress": "10.0.0.1", "devices": [{"name": "a", "ipAddress": "10.0.0.2"}]}`,
			wantErr: "mutually exclusive",
		},
		{
			name:    "device missing name",
			data:    `{"name": "x", "devices": [{"ipAddress": "10.0.0.1"}]}`,
			wantErr: "name is required",
		},
		{
			name:    "device missing ipAddress",
			data:    `{"name": "x", "devices": [{"name": "a"}]}`,
			wantErr: "ipAddress is required",
		},
		{
			name: "duplicate device names",
			data: `{"name": "x", "passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
				"devices": [{"name": "a", "ipAddress": "10.0.0.1"}, {"name": "a", "ipAddress": "10.0.0.2"}]}`,
			wantErr: "duplicate name",
		},
		{
			name:    "no passwordFile",
			data:    `{"name": "x", "devices": [{"name": "a", "ipAddress": "10.0.0.1"}]}`,
			wantErr: "passwordFile is required",
		},
		{
			name:    "root plaintext password",
			data:    `{"name": "x", "ipAddress": "10.0.0.1", "password": "p"}`,
			wantErr: "plaintext passwords in config are not supported",
		},
		{
			// rejected even when a valid passwordFile is also configured
			name: "device plaintext password",
			data: `{"name": "x", "passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
				"devices": [{"name": "a", "ipAddress": "10.0.0.1", "password": "p"}]}`,
			wantErr: "plaintext passwords in config are not supported",
		},
		{
			name:    "empty password file",
			data:    `{"name": "x", "passwordFile": ` + quote(writePasswordFile(t, "")) + `, "devices": [{"name": "a", "ipAddress": "10.0.0.1"}]}`,
			wantErr: "is empty",
		},
		{
			name: "root metadata with devices",
			data: `{"name": "x", "metadata": {"membership": {"subsystem": "sensors"}},
				"devices": [{"name": "a", "ipAddress": "10.0.0.1"}]}`,
			wantErr: "legacy single-device options",
		},
		{
			name:    "root udmiTopicPrefix with devices",
			data:    `{"name": "x", "udmiTopicPrefix": "t", "devices": [{"name": "a", "ipAddress": "10.0.0.1"}]}`,
			wantErr: "legacy single-device options",
		},
		{
			name: "unknown model",
			data: `{"name": "x", "passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
				"devices": [{"name": "a", "ipAddress": "10.0.0.1", "model": "not-a-model"}]}`,
			wantErr: "unknown model",
		},
		{
			name: "duplicate udmiTopicPrefix",
			data: `{"name": "x", "passwordFile": ` + quote(writePasswordFile(t, "p")) + `,
				"devices": [{"name": "a", "ipAddress": "10.0.0.1", "udmiTopicPrefix": "t"}, {"name": "b", "ipAddress": "10.0.0.2", "udmiTopicPrefix": "t"}]}`,
			wantErr: "duplicate udmiTopicPrefix",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tt.data))
			if err == nil {
				t.Fatalf("ParseConfig: expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseConfig error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func writePasswordFile(t *testing.T, password string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(file, []byte(password+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
