package xovis

import (
	"context"
	"net/http"
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
