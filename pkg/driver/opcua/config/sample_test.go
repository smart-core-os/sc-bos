package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// TestSampleJSON checks that the documented example is still valid config.
// It is easy to change the config structs and leave the sample behind, and a sample that
// doesn't parse is worse than none at all.
func TestSampleJSON(t *testing.T) {
	bs, err := os.ReadFile("sample.json")
	if err != nil {
		t.Fatal(err)
	}
	// sample.json is an app config file, the driver config is one entry in its drivers array
	var app struct {
		Drivers []json.RawMessage `json:"drivers"`
	}
	if err := json.Unmarshal(bs, &app); err != nil {
		t.Fatal(err)
	}
	if len(app.Drivers) != 1 {
		t.Fatalf("sample has %d drivers, want 1", len(app.Drivers))
	}

	cfg, err := ParseConfig(app.Drivers[0])
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Conn.Endpoint == "" {
		t.Error("conn.endpoint is empty, the sample's connection settings did not parse")
	}
	if len(cfg.Devices) == 0 {
		t.Error("no devices parsed")
	}

	// the sample documents the anonymous simulator setup, it should stay that way
	sec, err := cfg.Conn.ResolveSecurity()
	if err != nil {
		t.Fatalf("ResolveSecurity: %v", err)
	}
	if sec.TokenType != ua.UserTokenTypeAnonymous || !sec.AnonymousInsecure() {
		t.Errorf("ResolveSecurity() = %+v, want anonymous over an unsecured channel", sec)
	}
}
