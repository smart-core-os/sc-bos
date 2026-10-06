package cloud

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// TestAutoPoll_RetriesFailedCheckInsUntilSuccess verifies failed check-ins are retried within seconds,
// rather than waiting for the next poll interval, and stop once one succeeds.
func TestAutoPoll_RetriesFailedCheckInsUntilSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &scriptedClient{results: []error{errors.New("offline"), errors.New("offline"), nil}}
		conn := newRegisteredConn(t, client)

		runAutoPoll(t, conn, 4*time.Minute)

		gaps := client.gaps()
		if len(gaps) != 2 {
			t.Fatalf("gaps between check-ins = %v, want 2 retries then no more within the interval", gaps)
		}
		for _, gap := range gaps {
			if gap > 20*time.Second {
				t.Errorf("gaps between check-ins = %v, want each at most 20s", gaps)
			}
		}
	})
}

// TestAutoPoll_NoFastRetryAfterSuccess verifies that once a check-in has succeeded, a later failure waits
// for the next poll interval.
func TestAutoPoll_NoFastRetryAfterSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &scriptedClient{results: []error{nil, errors.New("offline"), nil}}
		conn := newRegisteredConn(t, client)

		runAutoPoll(t, conn, 11*time.Minute)

		gaps := client.gaps()
		want := []time.Duration{5 * time.Minute, 5 * time.Minute}
		if !slices.Equal(gaps, want) {
			t.Errorf("gaps between check-ins = %v, want %v", gaps, want)
		}
	})
}

// TestAutoPoll_RetryDelayCapsAtInterval verifies the retry delay stops growing at the poll interval.
func TestAutoPoll_RetryDelayCapsAtInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &scriptedClient{fail: errors.New("offline")}
		conn := newRegisteredConn(t, client, withPollInterval(30*time.Second))

		runAutoPoll(t, conn, 3*time.Minute)

		gaps := client.gaps()
		if len(gaps) < 5 {
			t.Fatalf("gaps between check-ins = %v, want at least 5", gaps)
		}
		for _, gap := range gaps {
			if gap > 30*time.Second {
				t.Errorf("gaps between check-ins = %v, want each at most the 30s interval", gaps)
			}
		}
	})
}

// TestAutoPoll_NoFastRetryAfterFailedInstall verifies a failed install waits for the next poll interval,
// so a brief outage at start-up doesn't use up the install attempt limit.
func TestAutoPoll_NoFastRetryAfterFailedInstall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &scriptedClient{
			offer: &LatestStream{
				Deployment: StreamDeployment{ID: "dep-1", Artefact: "config"},
				Version:    VersionProjection{ID: "ver-1", PayloadURL: "https://example.invalid/payload"},
			},
			downloadErr: errors.New("connection reset"),
		}
		conn := newRegisteredConn(t, client)

		runAutoPoll(t, conn, 4*time.Minute)

		if got := client.downloadCount(); got != 1 {
			t.Errorf("download attempts = %d, want 1 within the poll interval", got)
		}
		for _, p := range client.progressReports() {
			if p.State == ProgressFailed {
				t.Errorf("deployment reported failed: %+v", p)
			}
		}
	})
}

// TestAutoPoll_SkipsInitialDelayWhileInstalling verifies AutoPoll checks in straight away when the
// Supervisor is installing an update, instead of waiting out the start-up jitter.
func TestAutoPoll_SkipsInitialDelayWhileInstalling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sup := &fakeSupervisor{status: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_INSTALLING}}
		client := &scriptedClient{}
		conn := newRegisteredConn(t, client, withSupervisor(sup))

		start := time.Now()
		runAutoPoll(t, conn, time.Minute)

		times := client.checkInTimes()
		if len(times) == 0 {
			t.Fatal("no check-in")
		}
		if d := times[0].Sub(start); d != 0 {
			t.Errorf("first check-in after %v, want immediately", d)
		}
	})
}

// runAutoPoll runs AutoPoll on conn for d of fake time, then stops it.
func runAutoPoll(t *testing.T, conn *testConn, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		AutoPoll(ctx, conn.Conn, conn.interval, zap.NewNop())
	}()
	time.Sleep(d)
	cancel()
	<-done
}

type testConn struct {
	*Conn
	interval time.Duration
}

type testConnOption func(*testConnConfig)

type testConnConfig struct {
	interval time.Duration
	sup      *fakeSupervisor
}

func withPollInterval(d time.Duration) testConnOption {
	return func(c *testConnConfig) { c.interval = d }
}

func withSupervisor(sup *fakeSupervisor) testConnOption {
	return func(c *testConnConfig) { c.sup = sup }
}

// newRegisteredConn opens a Conn with a saved registration that checks in through client.
func newRegisteredConn(t *testing.T, client Client, opts ...testConnOption) *testConn {
	t.Helper()
	cfg := testConnConfig{interval: 5 * time.Minute}
	for _, opt := range opts {
		opt(&cfg)
	}
	regStore, depStore := newStores(t)
	if err := regStore.Save(context.Background(), testRegistration(t, "node-a")); err != nil {
		t.Fatalf("save registration: %v", err)
	}
	connOpts := []ConnOption{WithClientFactory(func(*Registration) Client { return client })}
	if cfg.sup != nil {
		installer := supervisorpb.NewSupervisorApiClient(wrap.ServerToClient(supervisorpb.SupervisorApi_ServiceDesc, cfg.sup))
		connOpts = append(connOpts, WithBinaryUpdater(NewBinaryUpdater(WithBinaryInstaller(installer))))
	}
	conn, err := OpenConn(context.Background(), regStore, depStore, "", connOpts...)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return &testConn{Conn: conn, interval: cfg.interval}
}

// scriptedClient is a Client whose check-ins return results in order, then fail (or nil once exhausted).
// It records when each check-in happened and the progress each reported.
type scriptedClient struct {
	mu          sync.Mutex
	results     []error
	fail        error         // returned once results is exhausted
	offer       *LatestStream // offered as the latest config on every check-in
	downloadErr error         // returned by DownloadPayload
	times       []time.Time
	progress    []ProgressReport
	downloads   int
}

func (c *scriptedClient) CheckIn(_ context.Context, req CheckInRequest) (CheckInResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.times = append(c.times, time.Now())
	c.progress = append(c.progress, req.Progress...)
	err := c.fail
	if len(c.results) > 0 {
		err, c.results = c.results[0], c.results[1:]
	}
	if err != nil {
		return CheckInResponse{}, err
	}
	return CheckInResponse{LatestConfig: c.offer}, nil
}

func (c *scriptedClient) progressReports() []ProgressReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.progress)
}

func (c *scriptedClient) downloadCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.downloads
}

func (c *scriptedClient) checkInTimes() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.times)
}

// gaps returns the durations between consecutive check-ins.
func (c *scriptedClient) gaps() []time.Duration {
	times := c.checkInTimes()
	var gaps []time.Duration
	for i := 1; i < len(times); i++ {
		gaps = append(gaps, times[i].Sub(times[i-1]))
	}
	return gaps
}

func (c *scriptedClient) DownloadPayload(context.Context, string) (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloads++
	return nil, c.downloadErr
}

func (c *scriptedClient) Renew(context.Context) (*Registration, error) {
	return nil, errors.New("not used")
}

func (c *scriptedClient) SetRegistration(*Registration) {}
