package paxton

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// recordingCheck is a service.SystemCheck that remembers the latest call.
type recordingCheck struct {
	mu      sync.Mutex
	running bool
	err     error
	calls   int
}

func (r *recordingCheck) Dispose() {}

func (r *recordingCheck) MarkRunning() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.err = true, nil
	r.calls++
}

func (r *recordingCheck) MarkFailed(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.err = false, err
	r.calls++
}

func (r *recordingCheck) last() (running bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running, r.err
}

// A failure on one endpoint must stay on the check while other endpoints succeed (SCB-1452).
func TestClient_ReportsPerEndpoint(t *testing.T) {
	var doorsOK atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/api/v1/doors", func(w http.ResponseWriter, _ *http.Request) {
		if !doorsOK.Load() {
			http.Error(w, "doors unavailable", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cli := retryablehttp.NewClient()
	cli.RetryMax = 0
	cli.Logger = nil
	check := &recordingCheck{}
	client := NewClient(cli, zap.NewNop(), config.Root{BaseUrl: srv.URL}, check)
	ctx := context.Background()

	if _, err := client.GetDoors(ctx); err == nil {
		t.Fatal("GetDoors succeeded, want an error")
	}
	if _, err := client.GetEvents(ctx, time.Now(), false); err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	running, err := check.last()
	if running || err == nil {
		t.Fatalf("check is running after the events poll succeeded, want doors still failing")
	}
	if !strings.Contains(err.Error(), "/api/v1/doors") {
		t.Errorf("fault %q does not name the doors endpoint", err)
	}
	if strings.Contains(err.Error(), "/api/v1/events") {
		t.Errorf("fault %q names the healthy events endpoint", err)
	}

	doorsOK.Store(true)
	if _, err := client.GetDoors(ctx); err != nil {
		t.Fatalf("GetDoors after recovery: %v", err)
	}
	if running, err := check.last(); !running {
		t.Errorf("check still failing after doors recovered: %v", err)
	}
}

// Roll calls are fetched once per SignalR connection and the driver carries on without them,
// so a failed fetch must not hold the check failed until the next reconnect.
func TestClient_RollCallFailureNotHeld(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/api/v1/rollcallreports", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cli := retryablehttp.NewClient()
	cli.RetryMax = 0
	cli.Logger = nil
	check := &recordingCheck{}
	client := NewClient(cli, zap.NewNop(), config.Root{BaseUrl: srv.URL}, check)

	if _, err := client.GetRollCallIDs(context.Background()); err == nil {
		t.Fatal("GetRollCallIDs succeeded, want an error")
	}
	if running, err := check.last(); !running {
		t.Errorf("check failed by a best-effort roll call fetch: %v", err)
	}
}

// With polling disabled, only ListSecurityEvents calls the events endpoint, on demand.
// Its failure goes back to the caller and must not hold the check failed until someone lists again.
func TestClient_OnDemandEventsFailureNotHeld(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad filter", http.StatusBadRequest)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cli := retryablehttp.NewClient()
	cli.RetryMax = 0
	cli.Logger = nil
	check := &recordingCheck{}
	client := NewClient(cli, zap.NewNop(), config.Root{BaseUrl: srv.URL}, check)

	if _, err := client.GetEventsOnDemand(context.Background(), time.Now(), true); err == nil {
		t.Fatal("GetEventsOnDemand succeeded, want an error")
	}
	if running, err := check.last(); !running {
		t.Errorf("check failed by an on-demand events fetch: %v", err)
	}
}

func TestClient_ReportFaultTextStable(t *testing.T) {
	check := &recordingCheck{}
	client := &Client{systemCheck: check}
	client.report(outcome{"/b", errFake("b down")}, outcome{"/a", errFake("a down")})
	_, first := check.last()
	client.report(outcome{"/a", errFake("a down")})
	_, second := check.last()
	const want = "/a: a down; /b: b down"
	if first.Error() != want || second.Error() != want {
		t.Errorf("fault text = %q then %q, want %q both times", first, second, want)
	}

	calls := check.calls
	client.report(outcome{"/a", context.Canceled})
	if check.calls != calls {
		t.Errorf("a cancelled call updated the check")
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
