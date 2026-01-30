package bolthub

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/remote"
)

func NewServer(dbPath string, logger *zap.Logger) (*Server, error) {
	// open the db. It will be created if it doesn't exist
	db, err := bolthold.Open(dbPath, 0750, nil)
	if err != nil {
		return nil, err
	}
	return NewServerFromBolthold(db, logger), nil
}

func NewServerFromBolthold(db *bolthold.Store, logger *zap.Logger) *Server {
	s := &Server{
		logger: logger,
		db:     db,
	}

	return s
}

type Server struct {
	hubpb.UnimplementedHubApiServer
	logger *zap.Logger
	db     *bolthold.Store

	dbChanges minibus.Bus[*hubpb.PullHubNodesResponse_Change]

	ManagerName   string
	ManagerAddr   string
	Authority     pki.Source  // trust authority for the cohort of smart core nodes
	TestTLSConfig *tls.Config // TLS config used when initiating test connections with a node
}

type DbEnrollment struct {
	Name        string
	Description string
	Address     string
	Cert        []byte
}

func (s *Server) Close() error {
	return s.db.Close()
}

func (s *Server) GetHubNode(ctx context.Context, req *hubpb.GetHubNodeRequest) (*hubpb.HubNode, error) {
	en := &DbEnrollment{}
	err := s.db.Get(req.GetAddress(), &en)
	if errors.Is(err, bolthold.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "no node registration with specified address")
	} else if err != nil {
		s.logger.Error("SelectEnrollment failed", zap.Error(err), zap.String("address", req.GetAddress()))
		return nil, status.Error(codes.Internal, "failed to retrieve enrollment")
	}

	return &hubpb.HubNode{
		Name:        en.Name,
		Address:     en.Address,
		Description: en.Description,
	}, nil
}

func (s *Server) EnrollHubNode(ctx context.Context, req *hubpb.EnrollHubNodeRequest) (*hubpb.HubNode, error) {
	nodeReg := req.GetNode()
	if nodeReg == nil {
		return nil, status.Error(codes.InvalidArgument, "node must be supplied")
	}
	if nodeReg.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "node.address must be supplied")
	}

	// check if the node is already enrolled
	err := s.db.Get(nodeReg.Address, &hubpb.HubNode{})
	if !errors.Is(err, bolthold.ErrNotFound) {
		return nil, status.Errorf(codes.AlreadyExists, "%s already enrolled", nodeReg.Address)
	}

	en, err := remote.Enroll(ctx, &enrollmentpb.Enrollment{
		TargetName:     nodeReg.Name,
		TargetAddress:  nodeReg.Address,
		ManagerName:    s.ManagerName,
		ManagerAddress: s.ManagerAddr,
	}, s.Authority, req.PublicCerts...)
	if err != nil {
		s.logger.Error("failed to enroll area controller", zap.Error(err),
			zap.String("target_address", nodeReg.Address))
		return nil, status.Error(codes.Unknown, "enrollment failed")
	}

	err = s.db.Insert(nodeReg.Address, &DbEnrollment{
		Name:        en.TargetName,
		Description: nodeReg.Description,
		Address:     en.TargetAddress,
		Cert:        en.Certificate,
	})
	if err != nil {
		// error storing enrollment, attempt to rollback
		delErr := s.deleteHubNode(ctx, nodeReg)
		if delErr != nil {
			// failed to rollback
			s.logger.Error("db.Insert(Enrollment) failed, failed to rollback",
				zap.NamedError("enroll", err), zap.NamedError("rollback", delErr))
			return nil, status.Errorf(codes.DataLoss, "enrollment failed, rollback failed - the system is in a corrupt state, manual intervention required")
		}
		s.logger.Warn("pool.InsertEnrollment failed", zap.Error(err))
		return nil, status.Error(codes.Aborted, "failed to save the enrollment, no changes have been made")
	}

	go s.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		NewValue:   &hubpb.HubNode{Address: en.TargetAddress, Name: en.TargetName, Description: nodeReg.Description},
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_ADD,
	})

	return nodeReg, nil

}

func (s *Server) deleteHubNode(ctx context.Context, reg *hubpb.HubNode) error {
	return remote.Forget(ctx, &enrollmentpb.Enrollment{
		TargetName:     reg.Name,
		TargetAddress:  reg.Address,
		ManagerName:    s.ManagerName,
		ManagerAddress: s.ManagerAddr,
	}, s.TestTLSConfig)
}

func (s *Server) ListHubNodes(ctx context.Context, request *hubpb.ListHubNodesRequest) (*hubpb.ListHubNodesResponse, error) {
	var dbEnrollments []DbEnrollment
	err := s.db.Find(&dbEnrollments, nil)
	if err != nil {
		s.logger.Error("db.Find failed for all dbEnrollment types", zap.Error(err))
		return nil, status.Error(codes.Unavailable, "unable to retrieve enrollments")
	}

	var registrations []*hubpb.HubNode
	for _, en := range dbEnrollments {
		registrations = append(registrations, &hubpb.HubNode{
			Name:        en.Name,
			Address:     en.Address,
			Description: en.Description,
		})
	}

	return &hubpb.ListHubNodesResponse{Nodes: registrations}, nil
}

func (s *Server) PullHubNodes(request *hubpb.PullHubNodesRequest, server hubpb.HubApi_PullHubNodesServer) error {
	// subscribe before we list from the db.
	events := s.dbChanges.Listen(server.Context())

	if !request.UpdatesOnly {
		nodes, err := s.ListHubNodes(server.Context(), &hubpb.ListHubNodesRequest{})
		if err != nil {
			return err
		}
		for _, node := range nodes.Nodes {
			err := server.Send(&hubpb.PullHubNodesResponse{Changes: []*hubpb.PullHubNodesResponse_Change{
				{
					Type:       types.ChangeType_ADD,
					ChangeTime: timestamppb.Now(),
					NewValue:   node,
				},
			}})
			if err != nil {
				return err
			}
		}
	}

	for event := range events {
		err := server.Send(&hubpb.PullHubNodesResponse{Changes: []*hubpb.PullHubNodesResponse_Change{event}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) InspectHubNode(ctx context.Context, request *hubpb.InspectHubNodeRequest) (*hubpb.HubNodeInspection, error) {
	if request.GetNode().GetAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "node.address must be supplied")
	}
	return remote.Inspect(ctx, request.Node.Address)
}

func (s *Server) RenewHubNode(ctx context.Context, request *hubpb.RenewHubNodeRequest) (*hubpb.HubNode, error) {
	reg, err := s.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		return nil, err
	}

	en, err := remote.Renew(ctx, &enrollmentpb.Enrollment{
		TargetName:     reg.Name,
		TargetAddress:  reg.Address,
		ManagerName:    s.ManagerName,
		ManagerAddress: s.ManagerAddr,
	}, s.Authority, s.TestTLSConfig)
	if err != nil {
		s.logger.Error("failed to renew area controller", zap.Error(err),
			zap.String("target_address", reg.Address))
		return nil, status.Error(codes.Unknown, "enrollment failed")
	}

	err = s.db.Update(reg.Address, DbEnrollment{
		Name:        en.TargetName,
		Description: reg.Description,
		Address:     en.TargetAddress,
		Cert:        en.Certificate,
	})
	if err != nil {
		s.logger.Error("db.Update(DbEnrollment) failed, no rollback", zap.Error(err))
		return nil, status.Errorf(codes.DataLoss, "renew failed, unable to rollback - the system is in a corrupt state, manual intervention may be required")
	}

	newNode := &hubpb.HubNode{
		Address:     en.TargetAddress,
		Name:        en.TargetName,
		Description: reg.Description,
	}
	go s.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		OldValue:   reg,
		NewValue:   newNode,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_UPDATE,
	})

	return reg, nil
}

func (s *Server) TestHubNode(ctx context.Context, request *hubpb.TestHubNodeRequest) (*hubpb.TestHubNodeResponse, error) {
	reg, err := s.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		return nil, err
	}

	logger := s.logger.With(zap.String("node_address", reg.Address))

	conn, err := grpc.NewClient(reg.Address, grpc.WithTransportCredentials(credentials.NewTLS(s.TestTLSConfig)))
	if err != nil {
		logger.Debug("failed connection", zap.Error(err))
		return nil, status.Error(codes.Unavailable, "failed connection")
	}

	client := traits.NewMetadataApiClient(conn)
	_, err = client.GetMetadata(ctx, &traits.GetMetadataRequest{})
	if err != nil {
		logger.Debug("failed api request", zap.Error(err))
		return nil, status.Error(codes.Unavailable, "failed api request")
	}

	return &hubpb.TestHubNodeResponse{}, nil
}

func (s *Server) ForgetHubNode(ctx context.Context, request *hubpb.ForgetHubNodeRequest) (*hubpb.ForgetHubNodeResponse, error) {
	reg, err := s.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		if request.AllowMissing {
			return &hubpb.ForgetHubNodeResponse{}, nil
		}
		return nil, err
	}

	var remoteDeleted bool
	err = s.deleteHubNode(ctx, reg)
	switch {
	case err == nil: // success case
		remoteDeleted = true
	case errors.Is(err, remote.ErrNotEnrolled), errors.Is(err, remote.ErrNotEnrolledWithUs):
		// continue in these cases, our state is out of sync with the node
	case errors.Is(err, remote.ErrNotTrusted):
		return nil, status.Error(codes.PermissionDenied, "hub is not trusted by node, unable to delete enrollment")
	default:
		s.logger.Warn("failed to forget area controller", zap.Error(err), zap.String("target_address", reg.Address))
		return nil, err
	}

	err = s.db.Delete(reg.Address, &DbEnrollment{})
	if err != nil {
		s.logger.Warn("db.Delete failed, no rollback", zap.Error(err), zap.Bool("remote_deleted", remoteDeleted))
		if remoteDeleted {
			// bad case, we changed the node but failed to update the database
			return nil, status.Errorf(codes.DataLoss, "the remote node has been changed but local state is corrupt, manual intervention may be required. Retrying may resolve this issue")
		}
		return nil, status.Errorf(codes.Unknown, "error removing enrollment from database, retrying may resolve this issue")
	}

	go s.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		OldValue:   reg,
		NewValue:   nil,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_REMOVE,
	})

	return &hubpb.ForgetHubNodeResponse{}, nil
}
