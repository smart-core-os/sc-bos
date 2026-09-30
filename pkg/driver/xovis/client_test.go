package xovis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// Test_getLiveLogic_reliability checks each read reports the sensor's reliability,
// except one its caller abandoned, which says nothing about the sensor.
func Test_getLiveLogic_reliability(t *testing.T) {
	tests := []struct {
		name   string
		handle func(w http.ResponseWriter, r *http.Request)
		// cancel abandons the read once the sensor has received it
		cancel bool
		want   healthpb.HealthCheck_Reliability_State
	}{
		{
			name: "ok",
			handle: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"logic": {"id": 1}}`))
			},
			want: healthpb.HealthCheck_Reliability_RELIABLE,
		},
		{
			name: "server error",
			handle: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"code": 503, "message": "unavailable"}`))
			},
			want: healthpb.HealthCheck_Reliability_NO_RESPONSE,
		},
		{
			name: "wrong shape",
			handle: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"logic": "not an object"}`))
			},
			want: healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		},
		{
			name: "malformed json",
			handle: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"logic": {`))
			},
			want: healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		},
		{
			name: "caller cancels",
			handle: func(_ http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			},
			cancel: true,
			// left as the first, successful, read reported it
			want: healthpb.HealthCheck_Reliability_RELIABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// logic 1 always answers, logic 2 answers as the test says
			received := make(chan struct{}, 1)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v5/singlesensor/data/live/logics/1" {
					_, _ = w.Write([]byte(`{"logic": {"id": 1}}`))
					return
				}
				received <- struct{}{}
				tt.handle(w, r)
			}))
			defer server.Close()
			conn := newInsecureClient(strings.TrimPrefix(server.URL, "https://"), "admin", "secret")

			registry := healthpb.NewRegistry()
			fc, err := registry.ForOwner("driver:xovis").NewFaultCheck("sensors/01", proto.Clone(commsHealthCheck).(*healthpb.HealthCheck))
			if err != nil {
				t.Fatalf("NewFaultCheck: %v", err)
			}
			state := func() healthpb.HealthCheck_Reliability_State {
				return registry.GetCheck("sensors/01", "driver:xovis:"+commsHealthCheck.Id).GetReliability().GetState()
			}

			// start from a known good state, so a read that leaves it alone is visible
			if _, err := getLiveLogic(t.Context(), conn, false, 1, fc); err != nil {
				t.Fatalf("first read: %v", err)
			}
			if got := state(); got != healthpb.HealthCheck_Reliability_RELIABLE {
				t.Fatalf("after first read: reliability = %v, want %v", got, healthpb.HealthCheck_Reliability_RELIABLE)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if tt.cancel {
				go func() {
					<-received
					cancel()
				}()
			}
			_, _ = getLiveLogic(ctx, conn, false, 2, fc)
			if got := state(); got != tt.want {
				t.Errorf("reliability = %v, want %v", got, tt.want)
			}
		})
	}
}
