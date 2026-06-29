package opsapi_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/internal/opsapi"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/pkg/proto/ops/cloudpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

const nodeName = "test-node"

func newTestEnv(t *testing.T, opts ...cloud.ConnOption) (*cloud.Conn, cloudpb.CloudConnectionApiClient) {
	t.Helper()
	credDir := t.TempDir()
	regStore := cloud.NewFileRegistrationStore(filepath.Join(credDir, "registration.json"))

	depRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("open dep root: %v", err)
	}
	t.Cleanup(func() { _ = depRoot.Close() })
	depStore := cloud.NewDeploymentStore(depRoot)

	conn, err := cloud.OpenConn(t.Context(), regStore, depStore, "", opts...)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	srv := opsapi.NewCloudConnectionServer(conn, nodeName, "")
	client := cloudpb.NewCloudConnectionApiClient(
		wrap.ServerToClient(cloudpb.CloudConnectionApi_ServiceDesc, srv),
	)
	return conn, client
}

// fakeRegisterServer returns an httptest.Server that signs a submitted CSR with a
// throwaway CA, issuing a leaf whose CN is nodeID — mimicking the SCC registration
// endpoint. It returns the server.
func fakeRegisterServer(t *testing.T, nodeID string) *httptest.Server {
	t.Helper()
	caKey, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	caDER, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject:               pkix.Name{CommonName: "test CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	caTLS := &tls.Certificate{Certificate: [][]byte{caDER}, PrivateKey: caKey}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		der, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			http.Error(w, "bad base64", http.StatusBadRequest)
			return
		}
		csr, err := pki.ParseCSRDER(der)
		if err != nil {
			http.Error(w, "bad csr", http.StatusBadRequest)
			return
		}
		chainPEM, err := pki.CreateCertificateChain(caTLS, &x509.Certificate{
			Subject:     pkix.Name{CommonName: nodeID},
			KeyUsage:    x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}, csr.PublicKey)
		if err != nil {
			http.Error(w, "sign failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pem-certificate-chain")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(chainPEM)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newTestRegistration builds a self-signed credential whose leaf CN is nodeID,
// for driving conn.Register directly without an HTTP round-trip.
func newTestRegistration(t *testing.T, nodeID string) *cloud.Registration {
	t.Helper()
	key, err := pki.GenerateECP256Key()
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	der, err := pki.CreateSelfSignedCertificate(&x509.Certificate{
		Subject: pkix.Name{CommonName: nodeID},
	}, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return &cloud.Registration{Key: key, Chain: []*x509.Certificate{leaf}}
}

// fakeCloudClient is a cloud.Client whose CheckIn error can be changed at runtime.
type fakeCloudClient struct {
	mu  sync.Mutex
	err error
}

func (f *fakeCloudClient) setErr(err error) {
	f.mu.Lock()
	f.err = err
	f.mu.Unlock()
}

func (f *fakeCloudClient) CheckIn(_ context.Context, _ cloud.CheckInRequest) (cloud.CheckInResponse, error) {
	f.mu.Lock()
	err := f.err
	f.mu.Unlock()
	return cloud.CheckInResponse{}, err
}

func (f *fakeCloudClient) DownloadPayload(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not used in tests")
}

func (f *fakeCloudClient) Renew(_ context.Context) (*cloud.Registration, error) {
	return nil, errors.New("not used in tests")
}

func (f *fakeCloudClient) SetRegistration(_ *cloud.Registration) {}

// withSucceedingClient replaces the real HTTP client with a fake that always
// succeeds at CheckIn. Use in tests that need registration to succeed.
func withSucceedingClient() cloud.ConnOption {
	return cloud.WithClientFactory(func(_ *cloud.Registration) cloud.Client {
		return &fakeCloudClient{}
	})
}

func assertStatusError(t *testing.T, err error, wantCode codes.Code, wantMsg string) {
	t.Helper()
	s, ok := status.FromError(err)
	if !ok {
		t.Fatalf("not a gRPC status error: %v", err)
	}
	if s.Code() != wantCode {
		t.Errorf("code = %v, want %v", s.Code(), wantCode)
	}
	if s.Message() != wantMsg {
		t.Errorf("message = %q, want %q", s.Message(), wantMsg)
	}
}

func TestGetCloudConnection_Unconfigured(t *testing.T) {
	_, client := newTestEnv(t)
	resp, err := client.GetCloudConnection(context.Background(), &cloudpb.GetCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("GetCloudConnection: %v", err)
	}
	if got := resp.CloudConnection.State; got != cloudpb.CloudConnection_UNCONFIGURED {
		t.Errorf("state = %v, want UNCONFIGURED", got)
	}
}

func TestGetCloudConnection_WrongName(t *testing.T) {
	_, client := newTestEnv(t)
	_, err := client.GetCloudConnection(context.Background(), &cloudpb.GetCloudConnectionRequest{Name: "other-node"})
	if err == nil {
		t.Fatal("expected error for wrong name")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
		t.Errorf("got code %v, want NotFound", err)
	}
}

func TestRegisterCloudConnection_EnrollmentCode(t *testing.T) {
	_, client := newTestEnv(t, withSucceedingClient())
	fakeSrv := fakeRegisterServer(t, "node-123")

	resp, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		EnrollmentCode: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode{
			Code:        "ABC123",
			RegisterUrl: fakeSrv.URL,
		},
	})
	if err != nil {
		t.Fatalf("RegisterCloudConnection: %v", err)
	}
	got := resp.CloudConnection
	if got.State == cloudpb.CloudConnection_UNCONFIGURED {
		t.Error("state should not be UNCONFIGURED after registration")
	}
	// The issued cert's CN (node id) is surfaced as NodeId.
	if got.NodeId != "node-123" {
		t.Errorf("NodeId = %q, want node-123", got.NodeId)
	}
}

func TestRegisterCloudConnection_RequiresEnrollmentCode(t *testing.T) {
	_, client := newTestEnv(t, withSucceedingClient())
	_, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
	})
	if err == nil {
		t.Fatal("expected error when enrollment_code is absent")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Errorf("got %v, want InvalidArgument", err)
	}
}

// TestRegisterCloudConnection_RejectedCode pins that a 401 from the registration
// endpoint (SCC returns a generic {error:"unauthorized"} for a bad/expired/used
// code) is surfaced as invalid_enrollment_code — the register endpoint's only
// authorisation is the code, so a 401 there can only mean the code was rejected.
func TestRegisterCloudConnection_RejectedCode(t *testing.T) {
	_, client := newTestEnv(t, withSucceedingClient())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Invalid enrollment code"}`))
	}))
	t.Cleanup(srv.Close)

	_, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		EnrollmentCode: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode{
			Code:        "BADCODE",
			RegisterUrl: srv.URL,
		},
	})
	if err == nil {
		t.Fatal("expected error for a rejected enrollment code")
	}
	assertStatusError(t, err, codes.PermissionDenied, "invalid_enrollment_code")
}

func TestUnlinkCloudConnection(t *testing.T) {
	conn, client := newTestEnv(t, withSucceedingClient())
	fakeSrv := fakeRegisterServer(t, "node-123")

	// Register first.
	_, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		EnrollmentCode: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode{
			Code:        "ABC123",
			RegisterUrl: fakeSrv.URL,
		},
	})
	if err != nil {
		t.Fatalf("RegisterCloudConnection: %v", err)
	}

	// Unlink.
	unlinkResp, err := client.UnlinkCloudConnection(context.Background(), &cloudpb.UnlinkCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("UnlinkCloudConnection: %v", err)
	}
	if unlinkResp.CloudConnection.State != cloudpb.CloudConnection_UNCONFIGURED {
		t.Errorf("state after unlink = %v, want UNCONFIGURED", unlinkResp.CloudConnection.State)
	}
	if conn.State().Connectivity != cloud.Unconfigured {
		t.Errorf("conn.State().Connectivity = %v, want Unconfigured", conn.State().Connectivity)
	}
}

func TestPullCloudConnection_InitialState(t *testing.T) {
	_, client := newTestEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.PullCloudConnection(ctx, &cloudpb.PullCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("PullCloudConnection: %v", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if len(resp.Changes) == 0 {
		t.Fatal("expected at least one change")
	}
	if resp.Changes[0].CloudConnection.State != cloudpb.CloudConnection_UNCONFIGURED {
		t.Errorf("initial state = %v, want UNCONFIGURED", resp.Changes[0].CloudConnection.State)
	}
}

func TestPullCloudConnection_UpdatesOnly(t *testing.T) {
	conn, client := newTestEnv(t, withSucceedingClient())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.PullCloudConnection(ctx, &cloudpb.PullCloudConnectionRequest{
		Name:        nodeName,
		UpdatesOnly: true,
	})
	if err != nil {
		t.Fatalf("PullCloudConnection: %v", err)
	}

	// Trigger a state change by registering a credential directly.
	if _, err = conn.Register(ctx, newTestRegistration(t, "node-123")); err != nil {
		t.Fatalf("conn.Register: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if len(resp.Changes) == 0 {
		t.Fatal("expected at least one change")
	}
	if resp.Changes[0].CloudConnection.State == cloudpb.CloudConnection_UNCONFIGURED {
		t.Error("update should not be UNCONFIGURED after Register")
	}
}

func TestTestCloudConnection_NotRegistered(t *testing.T) {
	_, client := newTestEnv(t)
	_, err := client.TestCloudConnection(context.Background(), &cloudpb.TestCloudConnectionRequest{Name: nodeName})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertStatusError(t, err, codes.FailedPrecondition, "not_registered")
}

// TestTestCloudConnection_ErrorCodes pins the sentinel strings returned for each
// category of connectivity failure on the test-connection path.
func TestTestCloudConnection_ErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		clientErr error
		wantCode  codes.Code
		wantMsg   string
	}{
		{
			name:      "invalid_client_certificate from a 401 (rejected client cert)",
			clientErr: &cloud.APIError{StatusCode: http.StatusUnauthorized},
			wantCode:  codes.PermissionDenied,
			wantMsg:   "invalid_client_certificate",
		},
		{
			name:      "server_unreachable from net.Error",
			clientErr: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			wantCode:  codes.Unavailable,
			wantMsg:   "server_unreachable",
		},
		{
			name:      "connection_failed for other errors",
			clientErr: errors.New("unexpected failure"),
			wantCode:  codes.Unavailable,
			wantMsg:   "connection_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fc := &fakeCloudClient{} // starts returning nil — registration succeeds
			conn, client := newTestEnv(t, cloud.WithClientFactory(
				func(_ *cloud.Registration) cloud.Client { return fc },
			))

			// Establish a credential so TestConn has an active updater.
			if _, err := conn.Register(ctx, newTestRegistration(t, "node-123")); err != nil {
				t.Fatalf("setup Register: %v", err)
			}

			// Now make CheckIn fail with the target error.
			fc.setErr(tc.clientErr)

			_, err := client.TestCloudConnection(ctx, &cloudpb.TestCloudConnectionRequest{Name: nodeName})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			assertStatusError(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

// TestGetCloudConnection_StateTransitions pins the CONNECTING → FAILED → CONNECTED state
// machine end-to-end through the gRPC API.
func TestGetCloudConnection_StateTransitions(t *testing.T) {
	ctx := context.Background()
	fc := &fakeCloudClient{}
	conn, client := newTestEnv(t, cloud.WithClientFactory(
		func(_ *cloud.Registration) cloud.Client { return fc },
	))

	// After Register succeeds the state should be CONNECTING.
	if _, err := conn.Register(ctx, newTestRegistration(t, "node-123")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	resp, err := client.GetCloudConnection(ctx, &cloudpb.GetCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("GetCloudConnection: %v", err)
	}
	if got := resp.CloudConnection.State; got != cloudpb.CloudConnection_CONNECTING {
		t.Errorf("state after Register = %v, want CONNECTING", got)
	}

	// After a failed TestConn the state should be FAILED.
	fc.setErr(errors.New("network error"))
	if _, err := client.TestCloudConnection(ctx, &cloudpb.TestCloudConnectionRequest{Name: nodeName}); err == nil {
		t.Fatal("expected TestCloudConnection to fail, got nil")
	}
	resp, err = client.GetCloudConnection(ctx, &cloudpb.GetCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("GetCloudConnection: %v", err)
	}
	if got := resp.CloudConnection.State; got != cloudpb.CloudConnection_FAILED {
		t.Errorf("state after failed TestConn = %v, want FAILED", got)
	}

	// After a successful TestConn the state should be CONNECTED.
	fc.setErr(nil)
	if _, err := client.TestCloudConnection(ctx, &cloudpb.TestCloudConnectionRequest{Name: nodeName}); err != nil {
		t.Fatalf("TestCloudConnection: %v", err)
	}
	resp, err = client.GetCloudConnection(ctx, &cloudpb.GetCloudConnectionRequest{Name: nodeName})
	if err != nil {
		t.Fatalf("GetCloudConnection: %v", err)
	}
	if got := resp.CloudConnection.State; got != cloudpb.CloudConnection_CONNECTED {
		t.Errorf("state after successful TestConn = %v, want CONNECTED", got)
	}
}
