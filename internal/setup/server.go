package setup

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/setuppb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// Server implements setuppb.SetupApiServer for the node.
type Server struct {
	setuppb.UnimplementedSetupApiServer

	codes    *Codes
	nodeName string
	version  string
}

func NewServer(codes *Codes, nodeName, version string) *Server {
	return &Server{codes: codes, nodeName: nodeName, version: version}
}

func (s *Server) GetSetupState(ctx context.Context, req *setuppb.GetSetupStateRequest) (*setuppb.SetupState, error) {
	claimed, err := s.codes.Claimed(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read setup state")
	}
	res := &setuppb.SetupState{
		Claimed:  claimed,
		NodeName: s.nodeName,
		Version:  s.version,
	}
	filter := masks.NewResponseFilter(masks.WithFieldMask(req.GetReadMask()))
	return filter.FilterClone(res).(*setuppb.SetupState), nil
}

func (s *Server) CompleteSetup(ctx context.Context, req *setuppb.CompleteSetupRequest) (*setuppb.CompleteSetupResponse, error) {
	if err := s.codes.Check(ctx, req.GetSetupCode()); err != nil {
		return nil, checkStatus(err)
	}
	// todo(SCB-1493): create the first admin, name the node, refresh its certificate and restart.
	return nil, status.Error(codes.Unimplemented, "completing setup is not implemented yet")
}

func checkStatus(err error) error {
	switch {
	case errors.Is(err, ErrWrongCode):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrTooManyAttempts):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, ErrNotPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "failed to check setup code")
	}
}
