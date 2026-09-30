package hpd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// TestPoller_process_reliability checks how the poller classifies sensor responses. A 200 with
// an empty body ({}) — which the base returns when it is connected but the sensor module is
// missing — and unparseable bodies are reported as bad responses rather than treated as healthy.
func TestPoller_process_reliability(t *testing.T) {
	const owner = "driver:steinel-hpd"
	const deviceName = "sensors/01"
	checkID := healthpb.AbsID(owner, "commsCheck")

	tests := map[string]struct {
		body      string
		wantState healthpb.HealthCheck_Reliability_State
		wantCode  string // expected LastError code, empty for a reliable response
	}{
		"empty body":     {body: `{}`, wantState: healthpb.HealthCheck_Reliability_BAD_RESPONSE, wantCode: SensorMissing},
		"wrong type":     {body: `{"SensorName": "HPD2", "Temperature": "warm"}`, wantState: healthpb.HealthCheck_Reliability_BAD_RESPONSE, wantCode: BadResponse},
		"invalid json":   {body: `not json`, wantState: healthpb.HealthCheck_Reliability_BAD_RESPONSE, wantCode: BadResponse},
		"healthy sensor": {body: `{"SensorName": "HPD2", "Temperature": 20.2}`, wantState: healthpb.HealthCheck_Reliability_RELIABLE},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			host := strings.TrimPrefix(server.URL, "https://")

			registry := healthpb.NewRegistry()
			faultCheck, err := registry.ForOwner(owner).NewFaultCheck(deviceName, commsHealthCheck())
			if err != nil {
				t.Fatalf("NewFaultCheck: %v", err)
			}
			defer faultCheck.Dispose()

			client := newInsecureClient(host, "")
			p := newPoller(client, time.Minute, zap.NewNop(), faultCheck)
			p.process(context.Background())

			got := registry.GetCheck(deviceName, checkID)
			if got == nil {
				t.Fatalf("GetCheck(%q, %q) = nil", deviceName, checkID)
			}
			if state := got.GetReliability().GetState(); state != tc.wantState {
				t.Errorf("reliability state = %v, want %v", state, tc.wantState)
			}
			if code := got.GetReliability().GetLastError().GetCode().GetCode(); code != tc.wantCode {
				t.Errorf("reliability error code = %q, want %q", code, tc.wantCode)
			}
		})
	}
}

// TestPoller_process_waitHonoursCtx checks a process waiting on another, stalled,
// read gives up when its own ctx is done instead of queueing behind it.
func TestPoller_process_waitHonoursCtx(t *testing.T) {
	stalled := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(stalled)
		<-release
		_, _ = w.Write([]byte(`{"SensorName": "HPD2"}`))
	}))
	defer server.Close()

	faultCheck, err := healthpb.NewRegistry().ForOwner("driver:steinel-hpd").NewFaultCheck("sensors/01", commsHealthCheck())
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	defer faultCheck.Dispose()
	p := newPoller(newInsecureClient(strings.TrimPrefix(server.URL, "https://"), ""), time.Minute, zap.NewNop(), faultCheck)

	first := make(chan struct{})
	go func() {
		defer close(first)
		_ = p.process(context.Background())
	}()
	// let the stalled read finish, so server.Close doesn't wait on it
	defer func() {
		close(release)
		<-first
	}()
	<-stalled

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.process(ctx) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("process = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("process waited on the stalled read past its own deadline")
	}
}

// TestPoller_process_afterDispose checks the fault check can be disposed while a
// process, such as one from GetExportMessage, is still running, and that the process
// finishing afterwards doesn't update it.
func TestPoller_process_afterDispose(t *testing.T) {
	const deviceName = "sensors/01"
	checkID := healthpb.AbsID("driver:steinel-hpd", "commsCheck")
	stalled := make(chan struct{})
	released := make(chan struct{})
	release := sync.OnceFunc(func() { close(released) })
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(stalled)
		<-released
		// no sensor data, which process reports on the fault check
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	defer release() // so a failure doesn't leave server.Close waiting on the handler

	var disposed atomic.Bool
	registry := healthpb.NewRegistry(healthpb.WithOnCheckUpdate(func(_ string, c *healthpb.HealthCheck) {
		if disposed.Load() {
			t.Errorf("fault check updated after Dispose: %v", c)
		}
	}))
	faultCheck, err := registry.ForOwner("driver:steinel-hpd").NewFaultCheck(deviceName, commsHealthCheck())
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	p := newPoller(newInsecureClient(strings.TrimPrefix(server.URL, "https://"), ""), time.Minute, zap.NewNop(), faultCheck)

	processed := make(chan error, 1)
	go func() { processed <- p.process(context.Background()) }()
	<-stalled

	faultCheck.Dispose()
	disposed.Store(true)
	if registry.GetCheck(deviceName, checkID) != nil {
		t.Fatal("fault check still registered after Dispose")
	}

	release()
	if err := <-processed; !errors.Is(err, errSensorMissing) {
		t.Errorf("process = %v, want %v", err, errSensorMissing)
	}
}
