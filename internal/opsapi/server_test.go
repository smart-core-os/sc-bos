package opsapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/internal/opsapi"
	"github.com/smart-core-os/sc-bos/pkg/proto/ops/cloudpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

const nodeName = "test-node"

func newTestEnv(t *testing.T, opts ...cloud.ConnOption) (*cloud.Conn, cloudpb.CloudConnectionApiClient) {
	t.Helper()
	regDir := t.TempDir()
	regStore := cloud.NewFileRegistrationStore(regDir + "/registration.json")

	depRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("open dep root: %v", err)
	}
	t.Cleanup(func() { _ = depRoot.Close() })
	depStore := cloud.NewDeploymentStore(depRoot)

	conn, err := cloud.OpenConn(t.Context(), regStore, depStore, opts...)
	if err != nil {
		t.Fatalf("OpenConn: %v", err)
	}
	srv := opsapi.NewCloudConnectionServer(conn, nodeName, "")
	client := cloudpb.NewCloudConnectionApiClient(
		wrap.ServerToClient(cloudpb.CloudConnectionApi_ServiceDesc, srv),
	)
	return conn, client
}

// fakeRegisterServer returns an httptest.Server that accepts any POST and responds
// with a Registration JSON payload.
func fakeRegisterServer(t *testing.T) (*httptest.Server, cloud.Registration) {
	t.Helper()
	reg := cloud.Registration{
		ClientID:     "client-id-test",
		ClientSecret: "c2VjcmV0", // base64("secret")
		BosapiRoot:   "http://127.0.0.1:9999",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reg)
	}))
	t.Cleanup(srv.Close)
	return srv, reg
}

// fakeCloudClient is a cloud.Client whose CheckIn error can be changed at runtime.
// Start with err = nil to simulate a healthy connection, then set err to inject failures.
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

// withSucceedingClient returns a ConnOption that replaces the real HTTP client with a
// fake that always returns success from CheckIn. Use in tests that need registration to
// succeed but do not care about real server interaction.
func withSucceedingClient() cloud.ConnOption {
	return cloud.WithClientFactory(func(_ cloud.Registration) cloud.Client {
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
	fakeSrv, wantReg := fakeRegisterServer(t)

	resp, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		Method: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode_{
			EnrollmentCode: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode{
				Code:        "ABC123",
				RegisterUrl: fakeSrv.URL,
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterCloudConnection: %v", err)
	}
	got := resp.CloudConnection
	if got.State == cloudpb.CloudConnection_UNCONFIGURED {
		t.Error("state should not be UNCONFIGURED after registration")
	}
	if got.ClientId != wantReg.ClientID {
		t.Errorf("ClientId = %q, want %q", got.ClientId, wantReg.ClientID)
	}
	if got.BosapiRoot != wantReg.BosapiRoot {
		t.Errorf("BosapiRoot = %q, want %q", got.BosapiRoot, wantReg.BosapiRoot)
	}
}

func TestRegisterCloudConnection_Manual(t *testing.T) {
	_, client := newTestEnv(t, withSucceedingClient())
	wantReg := cloud.Registration{
		ClientID:     "manual-client-id",
		ClientSecret: "manual-secret",
		BosapiRoot:   "https://bosapi.example.com",
	}

	resp, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		Method: &cloudpb.RegisterCloudConnectionRequest_Manual{
			Manual: &cloudpb.RegisterCloudConnectionRequest_ManualCredentials{
				ClientId:     wantReg.ClientID,
				ClientSecret: wantReg.ClientSecret,
				BosapiRoot:   wantReg.BosapiRoot,
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterCloudConnection: %v", err)
	}
	got := resp.CloudConnection
	want := &cloudpb.CloudConnection{
		Name:       nodeName,
		State:      cloudpb.CloudConnection_CONNECTING,
		ClientId:   wantReg.ClientID,
		BosapiRoot: wantReg.BosapiRoot,
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

// TestRegisterCloudConnection_Manual_ErrorCodes pins the sentinel strings returned for
// each category of credential-check failure on the manual registration path.
func TestRegisterCloudConnection_Manual_ErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		clientErr error
		wantCode  codes.Code
		wantMsg   string
	}{
		{
			name:      "invalid_client_credentials from oauth2.RetrieveError",
			clientErr: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}},
			wantCode:  codes.PermissionDenied,
			wantMsg:   "invalid_client_credentials",
		},
		{
			name:      "server_unreachable from net.Error",
			clientErr: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			wantCode:  codes.Unavailable,
			wantMsg:   "server_unreachable",
		},
		{
			name:      "credential_check_failed for other errors",
			clientErr: errors.New("unexpected server error"),
			wantCode:  codes.PermissionDenied,
			wantMsg:   "credential_check_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeCloudClient{err: tc.clientErr}
			_, client := newTestEnv(t, cloud.WithClientFactory(
				func(_ cloud.Registration) cloud.Client { return fc },
			))

			_, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
				Name: nodeName,
				Method: &cloudpb.RegisterCloudConnectionRequest_Manual{
					Manual: &cloudpb.RegisterCloudConnectionRequest_ManualCredentials{
						ClientId:     "id",
						ClientSecret: "secret",
						BosapiRoot:   "http://example.com",
					},
				},
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			assertStatusError(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestUnlinkCloudConnection(t *testing.T) {
	conn, client := newTestEnv(t, withSucceedingClient())
	fakeSrv, _ := fakeRegisterServer(t)

	// Register first.
	_, err := client.RegisterCloudConnection(context.Background(), &cloudpb.RegisterCloudConnectionRequest{
		Name: nodeName,
		Method: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode_{
			EnrollmentCode: &cloudpb.RegisterCloudConnectionRequest_EnrollmentCode{
				Code:        "ABC123",
				RegisterUrl: fakeSrv.URL,
			},
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
		t.Errorf("conn.State().State = %v, want StateUnconfigured", conn.State().Connectivity)
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
	fakeSrv, _ := fakeRegisterServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.PullCloudConnection(ctx, &cloudpb.PullCloudConnectionRequest{
		Name:        nodeName,
		UpdatesOnly: true,
	})
	if err != nil {
		t.Fatalf("PullCloudConnection: %v", err)
	}

	// Trigger a state change: register via the fake HTTP server, then persist via conn.
	reg, err := cloud.Register(ctx, "CODE", fakeSrv.URL, "test-node")
	if err != nil {
		t.Fatalf("cloud.Register: %v", err)
	}
	if _, err = conn.Register(ctx, reg); err != nil {
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
			name:      "invalid_client_credentials from oauth2.RetrieveError",
			clientErr: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}},
			wantCode:  codes.PermissionDenied,
			wantMsg:   "invalid_client_credentials",
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
				func(_ cloud.Registration) cloud.Client { return fc },
			))

			// Establish a registration so TestConn has an active updater.
			if _, err := conn.Register(ctx, cloud.Registration{
				ClientID: "id", ClientSecret: "secret", BosapiRoot: "http://example.com",
			}); err != nil {
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
		func(_ cloud.Registration) cloud.Client { return fc },
	))

	// After Register succeeds the state should be CONNECTING (credentials present,
	// but no successful AutoPoll round-trip yet).
	if _, err := conn.Register(ctx, cloud.Registration{
		ClientID: "id", ClientSecret: "secret", BosapiRoot: "http://example.com",
	}); err != nil {
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
