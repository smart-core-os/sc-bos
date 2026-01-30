package pgxpublications

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
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
	traits.UnimplementedPublicationApiServer

	logger *zap.Logger
	pool   *pgxpool.Pool
}

func (p *Server) CreatePublication(ctx context.Context, request *traits.CreatePublicationRequest) (*traits.Publication, error) {
	logger := rpcutil.ServerLogger(ctx, p.logger)
	input := request.GetPublication()

	pubID := input.GetId()
	if pubID == "" {
		return nil, status.Error(codes.InvalidArgument, "publication ID must be present")
	}
	audience := input.GetAudience().GetName()
	body := input.GetBody()
	mediaType := input.GetMediaType()

	var output *traits.Publication
	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		// Register the publication
		err := CreatePublication(ctx, tx, pubID, audience)
		if err != nil {
			return err
		}

		pubTime := time.Now().UTC()
		// Create the initial version for the publication
		version, err := CreatePublicationVersion(ctx, tx, PublicationVersion{
			PublicationID: pubID,
			PublishTime:   pubTime,
			Body:          body,
			MediaType:     mediaType,
		})
		if err != nil {
			return err
		}

		output = &traits.Publication{
			Id:          pubID,
			Version:     version,
			Body:        body,
			PublishTime: timestamppb.New(pubTime),
			MediaType:   mediaType,
		}
		if audience != "" {
			output.Audience = &traits.Publication_Audience{
				Name: audience,
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to create publication", zap.Error(err), zap.String("id", input.GetId()))
		return nil, status.Error(codes.Internal, "database error")
	}

	return output, nil
}

func (p *Server) GetPublication(ctx context.Context, request *traits.GetPublicationRequest) (*traits.Publication, error) {
	id := request.GetId()
	version := request.GetVersion()

	var output *traits.Publication
	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		output, err = GetPublication(ctx, tx, id, version)
		return err
	})
	return output, err
}

func (p *Server) UpdatePublication(ctx context.Context, request *traits.UpdatePublicationRequest) (*traits.Publication, error) {
	if request.GetUpdateMask() != nil {
		return nil, status.Error(codes.Unimplemented, "field mask support not implemented")
	}

	data := PublicationVersion{
		PublicationID: request.GetPublication().GetId(),
		PublishTime:   time.Now(),
		Body:          request.GetPublication().GetBody(),
		MediaType:     request.GetPublication().GetMediaType(),
	}
	var updated *traits.Publication

	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		if request.GetVersion() != "" {
			err = checkLatestVersion(ctx, tx, request.GetPublication().GetId(), request.GetVersion())
			if err != nil {
				return err
			}
		}

		versionID, err := CreatePublicationVersion(ctx, tx, data)
		if err != nil {
			return err
		}

		updated, err = GetPublication(ctx, tx, data.PublicationID, versionID)
		return err
	})

	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (p *Server) DeletePublication(ctx context.Context, request *traits.DeletePublicationRequest) (*traits.Publication, error) {
	var pub *traits.Publication

	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		// if a version is specified, check that it's the latest version
		// this helps clients to avoid racing each other
		if request.GetVersion() != "" {
			err = checkLatestVersion(ctx, tx, request.GetId(), request.GetVersion())
			if err != nil {
				return err
			}
		}

		// grab a copy of the data we're about to delete
		pub, err = GetPublication(ctx, tx, request.GetId(), request.GetVersion())
		if errors.Is(err, pgx.ErrNoRows) && request.GetAllowMissing() {
			// the publication doesn't exist, and the client indicates that this is acceptable
			// return an empty publication to indicate this
			pub = &traits.Publication{}
			return nil
		}
		if err != nil {
			return err
		}

		return DeletePublication(ctx, tx, request.GetId(), request.GetAllowMissing())
	})

	if err != nil {
		return nil, err
	}
	return pub, nil
}

func (p *Server) PullPublication(_ *traits.PullPublicationRequest, _ traits.PublicationApi_PullPublicationServer) error {
	return status.Error(codes.Unimplemented, "PullPublication not implemented")
}

func (p *Server) ListPublications(ctx context.Context, request *traits.ListPublicationsRequest) (*traits.ListPublicationsResponse, error) {
	limit := 50
	if request.GetPageSize() > 0 {
		limit = int(request.GetPageSize())
	}

	var (
		publications []*traits.Publication
		nextToken    string
	)
	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		publications, nextToken, err = GetPublicationsPaginated(ctx, tx, request.GetPageToken(), limit)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &traits.ListPublicationsResponse{
		Publications:  publications,
		NextPageToken: nextToken,
	}, nil
}

func (p *Server) PullPublications(_ *traits.PullPublicationsRequest, _ traits.PublicationApi_PullPublicationsServer) error {
	return status.Error(codes.Unimplemented, "PullPublications not implemented")
}

func (p *Server) AcknowledgePublication(ctx context.Context, request *traits.AcknowledgePublicationRequest) (*traits.Publication, error) {
	var accepted bool
	switch request.GetReceipt() {
	case traits.Publication_Audience_REJECTED:
		accepted = false
	case traits.Publication_Audience_ACCEPTED:
		accepted = true
		if request.GetReceiptRejectedReason() != "" {
			return nil, status.Error(codes.InvalidArgument, "cannot specify rejected_reason when ACCEPTED")
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "REJECTED or ACCEPTED must be specified")
	}

	var updated *traits.Publication
	err := p.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		err := CreatePublicationAcknowledgement(ctx, tx, request.GetId(), request.GetVersion(), time.Now(), accepted,
			request.GetReceiptRejectedReason(), request.GetAllowAcknowledged())
		if err != nil {
			return err
		}

		// get the updated publication
		updated, err = GetPublication(ctx, tx, request.GetId(), request.GetVersion())
		return err
	})

	return updated, err
}

func checkLatestVersion(ctx context.Context, tx pgx.Tx, pubID string, expectedVersionID string) error {
	latestVersion, err := GetLatestVersionID(ctx, tx, pubID)
	if errors.Is(err, pgx.ErrNoRows) {
		// publication has no versions
		return ErrExpectedVersion
	}
	if err != nil {
		// some other database error
		return err
	}
	if latestVersion != expectedVersionID {
		// publication latest version is not what we expected - concurrent updates detected
		return ErrExpectedVersion
	}

	return nil
}

var ErrExpectedVersion = status.Error(codes.FailedPrecondition, "expected latest version does not match")
