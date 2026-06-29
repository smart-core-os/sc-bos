// Package opsapi implements private APIs for the Ops UI.
package opsapi

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/pkg/proto/ops/cloudpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

// CloudConnectionServer implements CloudConnectionApiServer for this node's local cloud.Conn.
type CloudConnectionServer struct {
	cloudpb.UnimplementedCloudConnectionApiServer
	conn               CloudConnection
	nodeName           string
	defaultRegisterURL string
	apiEndpoint        string // origin (scheme://host) of defaultRegisterURL, surfaced for display
	registerOpts       []cloud.RegisterOption
}

// NewCloudConnectionServer creates a CloudConnectionServer that delegates to conn.
// nodeName is the authoritative name for this node's cloud connection. registerOpts
// are forwarded to cloud.Register (e.g. WithRegisterInsecureSkipVerify for local cloudsim).
func NewCloudConnectionServer(conn CloudConnection, nodeName string, defaultRegisterURL string, registerOpts ...cloud.RegisterOption) *CloudConnectionServer {
	return &CloudConnectionServer{
		conn:               conn,
		nodeName:           nodeName,
		defaultRegisterURL: defaultRegisterURL,
		apiEndpoint:        cloud.OriginOf(defaultRegisterURL),
		registerOpts:       registerOpts,
	}
}

func (s *CloudConnectionServer) GetCloudConnection(_ context.Context, req *cloudpb.GetCloudConnectionRequest) (*cloudpb.GetCloudConnectionResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	st := s.conn.State()
	return &cloudpb.GetCloudConnectionResponse{
		CloudConnection: connStateToProto(s.nodeName, s.apiEndpoint, st),
	}, nil
}

func (s *CloudConnectionServer) PullCloudConnection(req *cloudpb.PullCloudConnectionRequest, stream cloudpb.CloudConnectionApi_PullCloudConnectionServer) error {
	if err := s.validateName(req.Name); err != nil {
		return err
	}

	send := func(v cloud.ConnState) error {
		var ts *timestamppb.Timestamp
		if !v.ChangeTime.IsZero() {
			ts = timestamppb.New(v.ChangeTime)
		}
		return stream.Send(&cloudpb.PullCloudConnectionResponse{
			Changes: []*cloudpb.PullCloudConnectionResponse_Change{{
				Name:            s.nodeName,
				Type:            typespb.ChangeType_UPDATE,
				ChangeTime:      ts,
				CloudConnection: connStateToProto(s.nodeName, s.apiEndpoint, v),
			}},
		})
	}

	initial, ch := s.conn.PullState(stream.Context())
	if !req.UpdatesOnly {
		err := send(initial)
		if err != nil {
			return err
		}
	}
	for st := range ch {
		err := send(st)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CloudConnectionServer) RegisterCloudConnection(ctx context.Context, req *cloudpb.RegisterCloudConnectionRequest) (*cloudpb.RegisterCloudConnectionResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}

	ec := req.GetEnrollmentCode()
	if ec == nil {
		return nil, status.Error(codes.InvalidArgument, "enrollment_code is required")
	}
	if ec.GetCode() == "" {
		return nil, requiredFieldError("code")
	}

	regUrl := s.defaultRegisterURL
	if ec.GetRegisterUrl() != "" {
		regUrl = ec.GetRegisterUrl()
	}
	if regUrl == "" {
		// this can happen if the default URL is empty
		return nil, errNoDefaultRegisterURL
	}

	cred, err := cloud.Register(ctx, ec.GetCode(), regUrl, s.nodeName, s.registerOpts...)
	if err != nil {
		switch {
		case cloud.IsConnectionError(err):
			return nil, errServerUnreachable
		case cloud.IsInvalidEnrollmentCode(err), cloud.IsInvalidCredentialsError(err):
			// The registration endpoint's only authorisation is the enrollment
			// code, so a 401/403 (SCC returns a generic "unauthorized") — or an
			// explicit invalid_enrollment_code — means the code was rejected.
			return nil, errInvalidEnrollmentCode
		default:
			return nil, status.Errorf(codes.Internal, "register: %v", err)
		}
	}

	st, err := s.conn.Register(ctx, cred)
	if err != nil {
		if cloud.IsCredentialCheckError(err) {
			return nil, translateErrDefault(err, errCredentialCheckFailed)
		}
		return nil, status.Errorf(codes.Internal, "register: %v", err)
	}

	return &cloudpb.RegisterCloudConnectionResponse{
		CloudConnection: connStateToProto(s.nodeName, s.apiEndpoint, st),
	}, nil
}

func (s *CloudConnectionServer) RenewCloudConnection(ctx context.Context, req *cloudpb.RenewCloudConnectionRequest) (*cloudpb.RenewCloudConnectionResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	if err := s.conn.Renew(ctx); err != nil {
		if errors.Is(err, cloud.ErrNotRegistered) {
			return nil, errNotRegistered
		}
		switch {
		case cloud.IsInvalidCredentialsError(err):
			return nil, errInvalidClientCertificate
		case cloud.IsConnectionError(err):
			return nil, errServerUnreachable
		default:
			return nil, status.Errorf(codes.Internal, "renew: %v", err)
		}
	}
	st := s.conn.State()
	return &cloudpb.RenewCloudConnectionResponse{
		CloudConnection: connStateToProto(s.nodeName, s.apiEndpoint, st),
	}, nil
}

func (s *CloudConnectionServer) GetCloudConnectionDefaults(_ context.Context, req *cloudpb.GetCloudConnectionDefaultsRequest) (*cloudpb.GetCloudConnectionDefaultsResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	return &cloudpb.GetCloudConnectionDefaultsResponse{
		Defaults: &cloudpb.CloudConnectionDefaults{
			RegisterUrl: s.defaultRegisterURL,
		},
	}, nil
}

func (s *CloudConnectionServer) TestCloudConnection(ctx context.Context, req *cloudpb.TestCloudConnectionRequest) (*cloudpb.TestCloudConnectionResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	if err := s.conn.TestConn(ctx); err != nil {
		if errors.Is(err, cloud.ErrNotRegistered) {
			return nil, errNotRegistered
		}
		switch {
		case cloud.IsInvalidCredentialsError(err):
			return nil, errInvalidClientCertificate
		case cloud.IsConnectionError(err):
			return nil, errServerUnreachable
		default:
			return nil, errConnectionFailed
		}
	}
	return &cloudpb.TestCloudConnectionResponse{}, nil
}

func (s *CloudConnectionServer) UnlinkCloudConnection(ctx context.Context, req *cloudpb.UnlinkCloudConnectionRequest) (*cloudpb.UnlinkCloudConnectionResponse, error) {
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	if err := s.conn.Unlink(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "unlink: %v", err)
	}
	st := s.conn.State()
	return &cloudpb.UnlinkCloudConnectionResponse{
		CloudConnection: connStateToProto(s.nodeName, s.apiEndpoint, st),
	}, nil
}

func (s *CloudConnectionServer) validateName(name string) error {
	if name != "" && name != s.nodeName {
		return status.Errorf(codes.NotFound, "cloud connection %q not found", name)
	}
	return nil
}

// connStateToProto renders a ConnState for the API. defaultAPIEndpoint is shown
// until the node is enrolled; once it is, the endpoint the certificate was
// actually issued against is reported (which may differ from the configured
// default when a per-enrolment register-URL override was used).
func connStateToProto(name, defaultAPIEndpoint string, st cloud.ConnState) *cloudpb.CloudConnection {
	pb := &cloudpb.CloudConnection{
		Name:        name,
		State:       cloudStateToProto(st.Connectivity),
		ApiEndpoint: defaultAPIEndpoint,
	}
	if st.LastError != nil {
		pb.LastError = status.Convert(translateErr(st.LastError)).Message()
	}
	if st.Registration != nil {
		pb.NodeId = st.Registration.NodeID()
		if st.Registration.APIEndpoint != "" {
			pb.ApiEndpoint = st.Registration.APIEndpoint
		}
		if leaf := st.Registration.Leaf(); leaf != nil {
			pb.CertificateIssuedTime = timestamppb.New(leaf.NotBefore)
			pb.CertificateExpiryTime = timestamppb.New(leaf.NotAfter)
			pb.NextRenewalTime = timestamppb.New(cloud.RenewAt(leaf))
		}
	}
	if !st.LastCheckInTime.IsZero() {
		pb.LastCheckInTime = timestamppb.New(st.LastCheckInTime)
	}
	return pb
}

func cloudStateToProto(s cloud.Connectivity) cloudpb.CloudConnection_State {
	switch s {
	case cloud.Unconfigured:
		return cloudpb.CloudConnection_UNCONFIGURED
	case cloud.Connecting:
		return cloudpb.CloudConnection_CONNECTING
	case cloud.Connected:
		return cloudpb.CloudConnection_CONNECTED
	case cloud.Failed:
		return cloudpb.CloudConnection_FAILED
	default:
		return cloudpb.CloudConnection_STATE_UNSPECIFIED
	}
}

func requiredFieldError(fields ...string) error {
	switch len(fields) {
	case 0:
		return status.Error(codes.InvalidArgument, "missing required fields")
	case 1:
		return status.Errorf(codes.InvalidArgument, "missing required field %s", fields[0])
	default:
		return status.Errorf(codes.InvalidArgument, "missing required fields %s", strings.Join(fields, ", "))
	}
}

var (
	errInvalidEnrollmentCode    = status.Error(codes.PermissionDenied, "invalid_enrollment_code")
	errNoDefaultRegisterURL     = status.Error(codes.InvalidArgument, "register_url not supplied and no default configured")
	errInvalidClientCertificate = status.Error(codes.PermissionDenied, "invalid_client_certificate")
	errServerUnreachable        = status.Error(codes.Unavailable, "server_unreachable")
	errCredentialCheckFailed    = status.Error(codes.PermissionDenied, "credential_check_failed")
	errConnectionFailed         = status.Error(codes.Unavailable, "connection_failed")
	errNotRegistered            = status.Error(codes.FailedPrecondition, "not_registered")
)

func translateErr(err error) error {
	return translateErrDefault(err, err)
}

func translateErrDefault(err, defaultErr error) error {
	switch {
	case cloud.IsInvalidEnrollmentCode(err):
		return errInvalidEnrollmentCode
	case cloud.IsInvalidCredentialsError(err):
		return errInvalidClientCertificate
	case cloud.IsConnectionError(err):
		return errServerUnreachable
	case errors.Is(err, cloud.ErrNotRegistered):
		return errNotRegistered
	default:
		return defaultErr
	}
}

type CloudConnection interface {
	State() cloud.ConnState
	PullState(context.Context) (initial cloud.ConnState, changes <-chan cloud.ConnState)
	Register(ctx context.Context, reg *cloud.Registration) (cloud.ConnState, error)
	Renew(ctx context.Context) error
	Unlink(ctx context.Context) error
	TestConn(ctx context.Context) error
}
