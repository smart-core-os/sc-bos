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
