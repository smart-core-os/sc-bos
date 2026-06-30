package hpd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
