package cloud

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

func TestRenewAt(t *testing.T) {
	nb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	leaf := &x509.Certificate{NotBefore: nb, NotAfter: nb.Add(90 * 24 * time.Hour)}
	got := RenewAt(leaf)
	want := nb.Add(60 * 24 * time.Hour) // two-thirds of a 90-day lifetime
	if !got.Equal(want) {
		t.Errorf("RenewAt = %v, want %v", got, want)
	}
}

func TestUntilRenew(t *testing.T) {
	nb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	na := nb.Add(90 * 24 * time.Hour)
	leaf := &x509.Certificate{NotBefore: nb, NotAfter: na}
	renewAt := nb.Add(60 * 24 * time.Hour)

	t.Run("before the renewal point returns the remaining time", func(t *testing.T) {
		now := nb.Add(30 * 24 * time.Hour)
		if d := untilRenew(leaf, now); d != renewAt.Sub(now) {
			t.Errorf("untilRenew = %v, want %v", d, renewAt.Sub(now))
		}
	})
	t.Run("at the renewal point is due (zero)", func(t *testing.T) {
		if d := untilRenew(leaf, renewAt); d != 0 {
			t.Errorf("untilRenew = %v, want 0", d)
		}
	})
	t.Run("past the renewal point is due (zero)", func(t *testing.T) {
		if d := untilRenew(leaf, renewAt.Add(time.Hour)); d != 0 {
			t.Errorf("untilRenew = %v, want 0", d)
		}
	})
	t.Run("after expiry returns negative (do not renew)", func(t *testing.T) {
		if d := untilRenew(leaf, na.Add(time.Hour)); d >= 0 {
			t.Errorf("untilRenew = %v, want negative", d)
		}
	})
}

// TestConnRenew_AgainstSim exercises the full renewal path end to end: a node
// enrolled against the mTLS sim renews over its authenticated connection, and the
// new certificate is persisted, hot-swapped, and validated by a check-in.
func TestConnRenew_AgainstSim(t *testing.T) {
	ctx := context.Background()
	ts, apiServer, _ := newTLSSim(t)
	mgmt := ts.Client()

	var site sim.Site
	if resp := doSimRequest(t, mgmt, "POST", ts.URL+"/api/v1/management/sites",
		map[string]string{"name": "Test Site"}, &site); resp.StatusCode != 201 {
		t.Fatalf("create site: got %d", resp.StatusCode)
	}
	var node sim.Node
	if resp := doSimRequest(t, mgmt, "POST", ts.URL+"/api/v1/management/nodes",
		map[string]any{"hostname": "test-node", "siteId": strconv.FormatInt(site.ID, 10)}, &node); resp.StatusCode != 201 {
		t.Fatalf("create node: got %d", resp.StatusCode)
	}
	var ec sim.EnrollmentCode
	if resp := doSimRequest(t, mgmt, "POST",
		fmt.Sprintf("%s/api/v1/management/nodes/%d/enrollment-codes", ts.URL, node.ID), nil, &ec); resp.StatusCode != 201 {
		t.Fatalf("create enrollment code: got %d", resp.StatusCode)
	}
	cred, err := Register(ctx, ec.Code, ts.URL+"/v1/device/register", "test-node", WithRegisterHTTPClient(mgmt))
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	credStore, depStore := newStores(t)
	conn, err := OpenConn(ctx, credStore, depStore, ts.URL,
		WithClientOptions(WithServerRootCAs(apiServer.CACertPool())))
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	if _, err := conn.Register(ctx, cred, ""); err != nil {
		t.Fatalf("conn.Register: %v", err)
	}

	before := conn.State().Credential.Leaf().SerialNumber

	if err := conn.Renew(ctx); err != nil {
		t.Fatalf("Renew: %v", err)
	}

	st := conn.State()
	if st.Credential.Leaf().SerialNumber.Cmp(before) == 0 {
		t.Error("expected a fresh certificate serial after renew")
	}
	if st.Credential.NodeID() != cred.NodeID() {
		t.Errorf("node id changed across renew: %q -> %q", cred.NodeID(), st.Credential.NodeID())
	}
	if st.Connectivity != Connected {
		t.Errorf("connectivity = %v, want Connected after a renew + check-in", st.Connectivity)
	}
	if st.LastError != nil {
		t.Errorf("LastError = %v, want nil", st.LastError)
	}

	// The renewed certificate must be the one on disk, so a restart keeps it.
	reloaded, ok, err := credStore.Load(ctx)
	if err != nil || !ok {
		t.Fatalf("reload credential: ok=%v err=%v", ok, err)
	}
	if reloaded.Leaf().SerialNumber.Cmp(st.Credential.Leaf().SerialNumber) != 0 {
		t.Error("persisted credential is not the renewed certificate")
	}
}

// TestConnRenew_RefreshesFailedState pins that a successful renew performed while
// the connection is Failed clears the stale error and returns to Connected,
// rather than leaving the old Failed status behind.
func TestConnRenew_RefreshesFailedState(t *testing.T) {
	ctx := context.Background()
	fake := &fakeRenewClient{}
	conn := newConnWithFake(t, fake)

	if _, err := conn.Register(ctx, testCredential(t, "node-x"), ""); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Drive the connection into Failed via a failing check-in.
	fake.setCheckErr(errors.New("check-in boom"))
	_ = conn.TestConn(ctx)
	if got := conn.State().Connectivity; got != Failed {
		t.Fatalf("setup: connectivity = %v, want Failed", got)
	}

	// Renew succeeds and the subsequent check-in recovers.
	fake.setCheckErr(nil)
	fake.setNext(testCredential(t, "node-x"))
	if err := conn.Renew(ctx); err != nil {
		t.Fatalf("Renew: %v", err)
	}

	st := conn.State()
	if st.Connectivity != Connected {
		t.Errorf("connectivity = %v, want Connected", st.Connectivity)
	}
	if st.LastError != nil {
		t.Errorf("LastError = %v, want nil after successful renew", st.LastError)
	}
}

// TestAutoRenew_FiresWhenDueAndReArms verifies AutoRenew renews a certificate that
// is already past its renewal point, then re-arms to the freshly-issued
// certificate's renewal point instead of renewing again in a loop.
func TestAutoRenew_FiresWhenDueAndReArms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := &fakeRenewClient{}
	conn := newConnWithFake(t, fake)

	now := time.Now()
	// Already two-thirds through its life → due for renewal immediately.
	due := datedCredential(t, "node-y", now.Add(-100*time.Hour), now.Add(20*time.Hour))
	if _, err := conn.Register(ctx, due, ""); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// The renewed certificate is freshly issued, so its renewal point is far off.
	fresh := datedCredential(t, "node-y", now, now.Add(120*time.Hour))
	fake.setNext(fresh)

	go AutoRenew(ctx, conn, zap.NewNop())

	waitFor(t, 2*time.Second, func() bool {
		leaf := conn.State().Credential.Leaf()
		return leaf.SerialNumber.Cmp(fresh.Leaf().SerialNumber) == 0
	})

	// Give the loop a moment; it must re-arm to the fresh cert and not renew again.
	time.Sleep(200 * time.Millisecond)
	if n := fake.renewCount(); n != 1 {
		t.Errorf("Renew called %d times, want exactly 1 (loop should re-arm, not spin)", n)
	}
}

// --- test helpers ---

func newStores(t *testing.T) (CredentialStore, *DeploymentStore) {
	t.Helper()
	keyDir := t.TempDir()
	credStore := NewFileCredentialStore(
		filepath.Join(keyDir, "cloud.key.pem"),
		filepath.Join(keyDir, "cloud.cert.pem"),
		zap.NewNop(),
	)
	depRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("open dep root: %v", err)
	}
	t.Cleanup(func() { _ = depRoot.Close() })
	return credStore, NewDeploymentStore(depRoot)
}

func newConnWithFake(t *testing.T, fake Client) *Conn {
	t.Helper()
	credStore, depStore := newStores(t)
	conn, err := OpenConn(context.Background(), credStore, depStore, "",
		WithClientFactory(func(*Credential) Client { return fake }))
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	return conn
}

// datedCredential builds a self-signed credential whose leaf CN is nodeID and
// whose validity window is exactly [notBefore, notAfter], for exercising the
// renewal timer arithmetic.
func datedCredential(t *testing.T, nodeID string, notBefore, notAfter time.Time) *Credential {
	t.Helper()
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("GenerateECP256Key: %v", err)
	}
	der, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject:   pkix.Name{CommonName: nodeID},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}, key)
	if err != nil {
		t.Fatalf("CreateSelfSignedCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	cred, err := newCredential(key, []*x509.Certificate{leaf})
	if err != nil {
		t.Fatalf("newCredential: %v", err)
	}
	return cred
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

// fakeRenewClient is a cloud.Client whose check-in error and next renewed
// credential are settable, for testing Conn.Renew/AutoRenew without a server.
type fakeRenewClient struct {
	mu       sync.Mutex
	checkErr error
	next     *Credential
	renews   int
}

func (f *fakeRenewClient) setCheckErr(err error) {
	f.mu.Lock()
	f.checkErr = err
	f.mu.Unlock()
}

func (f *fakeRenewClient) setNext(cred *Credential) {
	f.mu.Lock()
	f.next = cred
	f.mu.Unlock()
}

func (f *fakeRenewClient) renewCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.renews
}

func (f *fakeRenewClient) CheckIn(context.Context, CheckInRequest) (CheckInResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return CheckInResponse{}, f.checkErr
}

func (f *fakeRenewClient) DownloadPayload(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}

func (f *fakeRenewClient) Renew(context.Context) (*Credential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.renews++
	return f.next, nil
}

func (f *fakeRenewClient) SetCredential(*Credential) {}
