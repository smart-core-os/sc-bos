package pgxhub

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/remote"
)

//go:embed schema.sql
var schemaSql string

func SetupDB(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, schemaSql)
		return err
	})
}

func NewServer(ctx context.Context, connStr string) (*Server, error) {
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect %w", err)
	}

	return NewServerFromPool(ctx, pool)
}

func NewServerFromPool(ctx context.Context, pool *pgxpool.Pool, opts ...Option) (*Server, error) {
	err := SetupDB(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("setup %w", err)
	}

	s := &Server{
		pool: pool,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

type Server struct {
	hubpb.UnimplementedHubApiServer
	logger *zap.Logger
	pool   *pgxpool.Pool

	dbChanges minibus.Bus[*hubpb.PullHubNodesResponse_Change]

	ManagerName   string
	ManagerAddr   string
	Authority     pki.Source  // trust authority for the cohort of smart core nodes
	TestTLSConfig *tls.Config // TLS config used when initiating test connections with a node
}

func (n *Server) GetHubNode(ctx context.Context, request *hubpb.GetHubNodeRequest) (*hubpb.HubNode, error) {
	logger := rpcutil.ServerLogger(ctx, n.logger)
	var dbEnrollment Enrollment
	err := n.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		dbEnrollment, err = SelectEnrollment(ctx, tx, request.GetAddress())
		return
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "no node registration with specified address")
	} else if err != nil {
		logger.Error("SelectEnrollment failed", zap.Error(err), zap.String("address", request.GetAddress()))
		return nil, status.Error(codes.Internal, "failed to retrieve enrollment")
	}

	return &hubpb.HubNode{
		Name:        dbEnrollment.Name,
		Address:     dbEnrollment.Address,
		Description: dbEnrollment.Description,
	}, nil
}

func (n *Server) EnrollHubNode(ctx context.Context, request *hubpb.EnrollHubNodeRequest) (*hubpb.HubNode, error) {
	logger := rpcutil.ServerLogger(ctx, n.logger)
	nodeReg := request.GetNode()
	if nodeReg == nil {
		return nil, status.Error(codes.InvalidArgument, "node must be supplied")
	}
	if nodeReg.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "node.address must be supplied")
	}

	// check if the node is already enrolled
	err := n.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		_, err := SelectEnrollment(ctx, tx, nodeReg.Address)
		return err
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.AlreadyExists, "%s already enrolled", nodeReg.Address)
	}

	en, err := remote.Enroll(ctx, &enrollmentpb.Enrollment{
		TargetName:     nodeReg.Name,
		TargetAddress:  nodeReg.Address,
		ManagerName:    n.ManagerName,
		ManagerAddress: n.ManagerAddr,
	}, n.Authority, request.PublicCerts...)
	if err != nil {
		logger.Error("failed to enroll area controller", zap.Error(err),
			zap.String("target_address", nodeReg.Address))
		return nil, status.Error(codes.Unknown, "enrollment failed")
	}

	err = n.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return InsertEnrollment(ctx, tx, Enrollment{
			Name:        en.TargetName,
			Description: nodeReg.Description,
			Address:     en.TargetAddress,
			Cert:        en.Certificate,
		})
	})
	if err != nil {
		delErr := n.deleteHubNode(ctx, nodeReg)
		if delErr != nil {
			// failed to rollback!
			logger.Error("pool.InsertEnrollment failed, failed to rollback",
				zap.NamedError("enroll", err), zap.NamedError("rollback", delErr))
			return nil, status.Errorf(codes.DataLoss, "enrollment failed, rollback failed - the system is in a corrupt state, manual intervention required")
		}
		logger.Warn("pool.InsertEnrollment failed", zap.Error(err))
		return nil, status.Error(codes.Aborted, "failed to save the enrollment, no changes have been made")
	}

	go n.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		NewValue:   &hubpb.HubNode{Address: en.TargetAddress, Name: en.TargetName, Description: nodeReg.Description},
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_ADD,
	})

	return nodeReg, nil
}

func (n *Server) deleteHubNode(ctx context.Context, reg *hubpb.HubNode) error {
	return remote.Forget(ctx, &enrollmentpb.Enrollment{
		TargetName:     reg.Name,
		TargetAddress:  reg.Address,
		ManagerName:    n.ManagerName,
		ManagerAddress: n.ManagerAddr,
	}, n.TestTLSConfig)
}

func (n *Server) ListHubNodes(ctx context.Context, request *hubpb.ListHubNodesRequest) (*hubpb.ListHubNodesResponse, error) {
	logger := rpcutil.ServerLogger(ctx, n.logger)
	var dbEnrollments []Enrollment
	err := n.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		dbEnrollments, err = SelectEnrollments(ctx, tx)
		return
	})
	if err != nil {
		logger.Error("pool.SelectEnrollments failed", zap.Error(err))
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

func (n *Server) PullHubNodes(request *hubpb.PullHubNodesRequest, server hubpb.HubApi_PullHubNodesServer) error {
	// subscribe before we list from the db.
	// There's still a race here as db access isn't guarded by the same locks as dbChanges is, sorry future dev
	events := n.dbChanges.Listen(server.Context())

	if !request.UpdatesOnly {
		nodes, err := n.ListHubNodes(server.Context(), &hubpb.ListHubNodesRequest{})
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

func (n *Server) InspectHubNode(ctx context.Context, request *hubpb.InspectHubNodeRequest) (*hubpb.HubNodeInspection, error) {
	if request.GetNode().GetAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "node.address must be supplied")
	}
	return remote.Inspect(ctx, request.Node.Address)
}

func (n *Server) RenewHubNode(ctx context.Context, request *hubpb.RenewHubNodeRequest) (*hubpb.HubNode, error) {
	logger := rpcutil.ServerLogger(ctx, n.logger)
	reg, err := n.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		return nil, err
	}

	en, err := remote.Renew(ctx, &enrollmentpb.Enrollment{
		TargetName:     reg.Name,
		TargetAddress:  reg.Address,
		ManagerName:    n.ManagerName,
		ManagerAddress: n.ManagerAddr,
	}, n.Authority, n.TestTLSConfig)
	if err != nil {
		logger.Error("failed to renew area controller", zap.Error(err),
			zap.String("target_address", reg.Address))
		return nil, status.Error(codes.Unknown, "enrollment failed")
	}

	err = n.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return UpdateEnrollment(ctx, tx, Enrollment{
			Name:        en.TargetName,
			Description: reg.Description,
			Address:     en.TargetAddress,
			Cert:        en.Certificate,
		})
	})
	if err != nil {
		// failed to rollback!
		logger.Error("pool.UpdateEnrollment failed, no rollback",
			zap.Error(err))
		return nil, status.Errorf(codes.DataLoss, "renew failed, unable to rollback - the system is in a corrupt state, manual intervention may be required")
	}

	newNode := &hubpb.HubNode{
		Address:     en.TargetAddress,
		Name:        en.TargetName,
		Description: reg.Description,
	}
	go n.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		OldValue:   reg,
		NewValue:   newNode,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_UPDATE,
	})

	return reg, nil
}

func (n *Server) TestHubNode(ctx context.Context, request *hubpb.TestHubNodeRequest) (*hubpb.TestHubNodeResponse, error) {
	reg, err := n.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		return nil, err
	}

	logger := n.logger.With(zap.String("node_address", reg.Address))

	conn, err := grpc.NewClient(reg.Address, grpc.WithTransportCredentials(credentials.NewTLS(n.TestTLSConfig)))
	if err != nil {
		logger.Debug("failed connection", zap.Error(err))
		return nil, status.Error(codes.Unavailable, "failed connection")
	}

	client := traits.NewMetadataApiClient(conn)
	_, err = client.GetMetadata(ctx, &traits.GetMetadataRequest{})
	if err != nil {
		logger.Debug("failed api request", zap.Error(err))
		return nil, status.Errorf(codes.Unavailable, "failed api request: %v", err)
	}

	return &hubpb.TestHubNodeResponse{}, nil
}

func (n *Server) ForgetHubNode(ctx context.Context, request *hubpb.ForgetHubNodeRequest) (*hubpb.ForgetHubNodeResponse, error) {
	reg, err := n.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: request.GetAddress()})
	if err != nil {
		if request.AllowMissing {
			return &hubpb.ForgetHubNodeResponse{}, nil
		}
		return nil, err
	}
	logger := rpcutil.ServerLogger(ctx, n.logger).With(zap.String("node_address", reg.Address))

	var remoteDeleted bool
	err = n.deleteHubNode(ctx, reg)
	switch {
	case err == nil: // success case
		remoteDeleted = true
	case errors.Is(err, remote.ErrNotEnrolled), errors.Is(err, remote.ErrNotEnrolledWithUs):
		// continue in these cases, our state is out of sync with the node
	case errors.Is(err, remote.ErrNotTrusted):
		return nil, status.Error(codes.PermissionDenied, "hub is not trusted by node, unable to delete enrollment")
	default:
		logger.Warn("failed to forget area controller", zap.Error(err),
			zap.String("target_address", reg.Address))
		return nil, err
	}

	err = n.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return DeleteEnrollment(ctx, tx, reg.Address)
	})
	if err != nil {
		// failed to rollback!
		logger.Warn("pool.DeleteEnrollment failed, no rollback", zap.Error(err), zap.Bool("remote_deleted", remoteDeleted))
		if remoteDeleted {
			// bad case, we changed the node but failed to update the database
			return nil, status.Errorf(codes.DataLoss, "the remote node has been changed but local state is corrupt, manual intervention may be required. Retrying may resolve this issue")
		}
		return nil, status.Errorf(codes.Unknown, "error removing enrollment from database, retrying may resolve this issue")
	}

	go n.dbChanges.Send(context.Background(), &hubpb.PullHubNodesResponse_Change{
		OldValue:   reg,
		NewValue:   nil,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_REMOVE,
	})

	return &hubpb.ForgetHubNodeResponse{}, nil
}
