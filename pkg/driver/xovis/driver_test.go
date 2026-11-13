package xovis

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/xovis/config"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// TestDriver_healthChecks checks every device gets its own comms check with the same id, and that applying the
// config again replaces the checks rather than nesting their ids or clashing with the previous apply's.
func TestDriver_healthChecks(t *testing.T) {
	var mu sync.Mutex
	live := map[string][]string{} // device name -> check ids
	reg := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
			mu.Lock()
			defer mu.Unlock()
			live[name] = append(live[name], c.GetId())
			return nil
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			mu.Lock()
			defer mu.Unlock()
			var ids []string
			for _, got := range live[name] {
				if got != id {
					ids = append(ids, got)
				}
			}
			if len(ids) == 0 {
				delete(live, name)
			} else {
				live[name] = ids
			}
		}),
	)
	liveChecks := func() map[string][]string {
		mu.Lock()
		defer mu.Unlock()
		out := make(map[string][]string, len(live))
		for k, v := range live {
			out[k] = append([]string(nil), v...)
		}
		return out
	}

	cfg, err := config.ParseConfig([]byte(`{
		"name": "xovis",
		"host": "127.0.0.1:1",
		"username": "admin",
		"password": "secret",
		"devices": [
			{"name": "sensors/01", "occupancy": {"id": 1}},
			{"name": "sensors/02", "occupancy": {"id": 1}},
			{"name": "sensors/03", "enterLeave": {"id": 2}}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(node.New("test")),
		health:      reg.ForOwner("driver:xovis"),
		httpMux:     http.NewServeMux(),
		logger:      zap.NewNop(),
		pushDataBus: &minibus.Bus[PushData]{},
	}
	want := map[string][]string{
		"sensors/01": {"driver:xovis:commsCheck"},
		"sensors/02": {"driver:xovis:commsCheck"},
		"sensors/03": {"driver:xovis:commsCheck"},
	}

	ctx1, stop1 := context.WithCancel(t.Context())
	defer stop1()
	if err := d.applyConfig(ctx1, cfg); err != nil {
		t.Fatalf("first applyConfig: %v", err)
	}
	if diff := cmp.Diff(want, liveChecks()); diff != "" {
		t.Errorf("checks after first apply (-want +got):\n%s", diff)
	}

	// The first apply's ctx is deliberately left running, so only applyConfig itself can make room for the new checks.
	ctx2, stop2 := context.WithCancel(t.Context())
	defer stop2()
	if err := d.applyConfig(ctx2, cfg); err != nil {
		t.Fatalf("second applyConfig: %v", err)
	}
	if diff := cmp.Diff(want, liveChecks()); diff != "" {
		t.Errorf("checks after second apply (-want +got):\n%s", diff)
	}

	// stopping the driver removes its checks
	stop1()
	stop2()
	deadline := time.Now().Add(5 * time.Second)
	for len(liveChecks()) > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("checks still registered after the driver stopped: %v", liveChecks())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestDriver_reapplyWebhook checks applying the config again leaves the data push webhook working, rather than
// registering it on the shared mux a second time, which panics.
func TestDriver_reapplyWebhook(t *testing.T) {
	newDriver := func() *Driver {
		return &Driver{
			announcer:   node.NewReplaceAnnouncer(node.New("test")),
			health:      healthpb.NewRegistry().ForOwner("driver:xovis"),
			httpMux:     http.NewServeMux(),
			logger:      zap.NewNop(),
			pushDataBus: &minibus.Bus[PushData]{},
		}
	}
	parse := func(dataPush string) config.Root {
		t.Helper()
		cfg, err := config.ParseConfig([]byte(`{
			"name": "xovis", "host": "127.0.0.1:1", "username": "admin", "password": "secret",
			"dataPush": ` + dataPush + `,
			"devices": [{"name": "sensors/01", "occupancy": {"id": 1}}]
		}`))
		if err != nil {
			t.Fatalf("ParseConfig: %v", err)
		}
		return cfg
	}
	apply := func(d *Driver, cfg config.Root) {
		t.Helper()
		ctx, stop := context.WithCancel(t.Context())
		t.Cleanup(stop)
		if err := d.applyConfig(ctx, cfg); err != nil {
			t.Fatalf("applyConfig: %v", err)
		}
	}
	// pushReaches posts to the shared mux and reports whether the push reached the driver's bus
	pushReaches := func(d *Driver) bool {
		t.Helper()
		ctx, stop := context.WithTimeout(t.Context(), 5*time.Second)
		defer stop()
		pushes := d.pushDataBus.Listen(ctx)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/xovis/push", strings.NewReader(`{"logics_data": {"logics": []}}`))
		req.Header.Set("Content-Type", "application/json")
		go d.httpMux.ServeHTTP(httptest.NewRecorder(), req)
		select {
		case <-pushes:
			return true
		case <-ctx.Done():
			return false
		}
	}

	t.Run("same config", func(t *testing.T) {
		d := newDriver()
		cfg := parse(`{"webhookPath": "/xovis/push"}`)
		apply(d, cfg)
		apply(d, cfg)
		if !pushReaches(d) {
			t.Error("webhook stopped delivering after the config was applied again")
		}
	})

	t.Run("dedicated port, then the shared mux", func(t *testing.T) {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen: %v", err)
		}
		port := lis.Addr().(*net.TCPAddr).Port
		_ = lis.Close()

		d := newDriver()
		apply(d, parse(fmt.Sprintf(`{"webhookPath": "/xovis/push", "httpPort": %d}`, port)))
		apply(d, parse(`{"webhookPath": "/xovis/push"}`))
		if !pushReaches(d) {
			t.Error("webhook not served on the shared mux after moving off the dedicated port")
		}
	})
}
