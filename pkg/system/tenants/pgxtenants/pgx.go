package pgxtenants

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/internal/util/pass"
	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
	"github.com/smart-core-os/sc-bos/pkg/proto/tenantpb"
	"github.com/smart-core-os/sc-golang/pkg/masks"
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

var (
	errDatabase       = status.Error(codes.Internal, "database transaction failed")
	errTenantNotFound = status.Error(codes.NotFound, "tenant not found")
	errSecretNotFound = status.Error(codes.NotFound, "secret not found")
)

type Server struct {
	tenantpb.UnimplementedTenantApiServer
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func (s *Server) ListTenants(ctx context.Context, request *tenantpb.ListTenantsRequest) (*tenantpb.ListTenantsResponse, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)

	var tenants []*tenantpb.Tenant
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		tenants, err = ListTenants(ctx, tx)
		return
	})
	if err != nil {
		logger.Error("db.ListTenants failed", zap.Error(err))
		return nil, errDatabase
	}

	return &tenantpb.ListTenantsResponse{Tenants: tenants}, nil
}

func (s *Server) PullTenants(request *tenantpb.PullTenantsRequest, server tenantpb.TenantApi_PullTenantsServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) CreateTenant(ctx context.Context, request *tenantpb.CreateTenantRequest) (*tenantpb.Tenant, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)

	var newTenant *tenantpb.Tenant
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		newTenant, err = CreateTenant(ctx, tx, request.Tenant.Title)
		if err != nil {
			return
		}

		if zones := request.Tenant.GetZoneNames(); len(zones) > 0 {
			err = AddTenantZones(ctx, tx, newTenant.Id, zones)
		}

		return
	})
	if err != nil {
		logger.Error("tenant database transaction failed", zap.Error(err))
		return nil, errDatabase
	}

	return newTenant, nil
}

func (s *Server) GetTenant(ctx context.Context, request *tenantpb.GetTenantRequest) (*tenantpb.Tenant, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing tenant id")
	}
	if !ValidateTenantID(request.GetId()) {
		return nil, errTenantNotFound
	}
	logger := rpcutil.ServerLogger(ctx, s.logger)

	var tenant *tenantpb.Tenant
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		tenant, err = GetTenant(ctx, tx, request.GetId())
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errTenantNotFound
	} else if err != nil {
		logger.Error("db transaction failed", zap.Error(err))
		return nil, errDatabase
	}
	return tenant, nil
}

func (s *Server) UpdateTenant(ctx context.Context, request *tenantpb.UpdateTenantRequest) (*tenantpb.Tenant, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger).With(zap.String("tenant", request.GetTenant().GetId()))
	tenant := request.Tenant
	updater := masks.NewFieldUpdater(
		masks.WithUpdateMask(request.UpdateMask),
		masks.WithUpdateMaskFieldName("update_mask"),
		masks.WithWritableFields(&fieldmaskpb.FieldMask{Paths: []string{
			"title", "zone_names",
		}}),
	)
	err := updater.Validate(tenant)
	if err != nil {
		logger.Error("mask validation failed", zap.Error(err), zap.Strings("paths", request.UpdateMask.GetPaths()))
		return nil, err
	}

	err = s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		if rpcutil.MaskContains(request.UpdateMask, "title") {
			err = UpdateTenantTitle(ctx, tx, tenant.Id, tenant.Title)
			if err != nil {
				return err
			}
		}

		if rpcutil.MaskContains(request.UpdateMask, "zone_names") {
			err = ReplaceTenantZones(ctx, tx, tenant.Id, tenant.ZoneNames)
			if err != nil {
				return err
			}
		}

		tenant, err = GetTenant(ctx, tx, tenant.Id)
		return
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errTenantNotFound
	} else if err != nil {
		logger.Error("database transaction failed", zap.Error(err))
		return nil, errDatabase
	}

	return tenant, nil
}

func (s *Server) DeleteTenant(ctx context.Context, request *tenantpb.DeleteTenantRequest) (*tenantpb.DeleteTenantResponse, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger).With(zap.String("id", request.Id))

	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return DeleteTenant(ctx, tx, request.Id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errTenantNotFound
	} else if err != nil {
		logger.Error("db.DeleteTenant failed", zap.Error(err))
		return nil, errDatabase
	}

	return &tenantpb.DeleteTenantResponse{}, nil
}

func (s *Server) PullTenant(request *tenantpb.PullTenantRequest, server tenantpb.TenantApi_PullTenantServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) AddTenantZones(ctx context.Context, request *tenantpb.AddTenantZonesRequest) (*tenantpb.Tenant, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)

	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return AddTenantZones(ctx, tx, request.TenantId, request.AddZoneNames)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errTenantNotFound
	} else if err != nil {
		logger.Error("db transaction failed", zap.Error(err))
		return nil, errDatabase
	}

	return s.GetTenant(ctx, &tenantpb.GetTenantRequest{Id: request.TenantId})
}

func (s *Server) RemoveTenantZones(ctx context.Context, request *tenantpb.RemoveTenantZonesRequest) (*tenantpb.Tenant, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)

	var tenant *tenantpb.Tenant
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		err = RemoveTenantZones(ctx, tx, request.TenantId, request.RemoveZoneNames)
		if err != nil {
			return err
		}

		tenant, err = GetTenant(ctx, tx, request.TenantId)
		return
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errTenantNotFound
	} else if err != nil {
		logger.Error("db transaction failed", zap.Error(err))
		return nil, errDatabase
	}

	return tenant, nil
}

func (s *Server) ListSecrets(ctx context.Context, request *tenantpb.ListSecretsRequest) (*tenantpb.ListSecretsResponse, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger).With(
		zap.Namespace("request"),
		zap.String("filter", request.GetFilter()),
		zap.Bool("include_hash", request.GetIncludeHash()),
	)

	var tenantID string
	if request.Filter != "" {
		groups := filterTenantRegexp.FindStringSubmatch(request.Filter)
		if groups == nil {
			return nil, status.Error(codes.InvalidArgument, "invalid filter")
		}
		tenantID = groups[1]
	}

	var secrets []*tenantpb.Secret
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		secrets, err = ListTenantSecrets(ctx, tx, tenantID)
		return
	})
	if err != nil {
		logger.Error("db.ListTenantSecrets failed", zap.Error(err))
		return nil, errDatabase
	}
	// unless specifically requested, censor the hashes
	if !request.IncludeHash {
		for i := range secrets {
			secrets[i].SecretHash = nil
		}
	}

	return &tenantpb.ListSecretsResponse{Secrets: secrets}, nil
}

func (s *Server) PullSecrets(request *tenantpb.PullSecretsRequest, server tenantpb.TenantApi_PullSecretsServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) CreateSecret(ctx context.Context, request *tenantpb.CreateSecretRequest) (*tenantpb.Secret, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)
	secret := request.Secret

	var err error
	secret.Secret, err = genSecret()
	if err != nil {
		logger.Error("secret generation failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "secret generation failed")
	}
	secret.SecretHash, err = pass.Hash([]byte(secret.Secret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "secret hashing failed: %s", err.Error())
	}

	err = s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		secret, err = CreateTenantSecret(ctx, tx, secret)
		return
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "tenant not found")
	} else if err != nil {
		logger.Error("db transaction failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "database transaction failed")
	}
	return secret, nil
}

func (s *Server) VerifySecret(ctx context.Context, request *tenantpb.VerifySecretRequest) (*tenantpb.Secret, error) {
	if request.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing tenant_id")
	}
	if !ValidateTenantID(request.TenantId) {
		return nil, errTenantNotFound
	}

	var secrets []*tenantpb.Secret
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		secrets, err = ListTenantSecrets(ctx, tx, request.TenantId)
		return
	})
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "db: %s", err.Error())
	}

	for _, s := range secrets {
		if err := pass.Compare(s.SecretHash, []byte(request.Secret)); err == nil {
			s.SecretHash = nil
			return s, nil
		}
	}
	return nil, status.Error(codes.Unauthenticated, "unknown pass")
}

func (s *Server) GetSecret(ctx context.Context, request *tenantpb.GetSecretRequest) (*tenantpb.Secret, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger).With(zap.String("secret_id", request.GetId()))

	var secret *tenantpb.Secret
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) (err error) {
		secret, err = GetTenantSecret(ctx, tx, request.Id)
		return
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "secret not found")
	} else if err != nil {
		logger.Error("db.GetTenantSecret failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "database transaction failed")
	}

	return secret, nil
}

func (s *Server) UpdateSecret(ctx context.Context, request *tenantpb.UpdateSecretRequest) (*tenantpb.Secret, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) DeleteSecret(ctx context.Context, request *tenantpb.DeleteSecretRequest) (*tenantpb.DeleteSecretResponse, error) {
	logger := rpcutil.ServerLogger(ctx, s.logger)
	err := s.pool.BeginFunc(ctx, func(tx pgx.Tx) error {
		return DeleteTenantSecret(ctx, tx, request.Id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "secret not found")
	} else if err != nil {
		logger.Error("db transaction failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "database transaction failed")
	}
	return &tenantpb.DeleteSecretResponse{}, nil
}

func (s *Server) PullSecret(request *tenantpb.PullSecretRequest, server tenantpb.TenantApi_PullSecretServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) RegenerateSecret(ctx context.Context, request *tenantpb.RegenerateSecretRequest) (*tenantpb.Secret, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func genSecret() (string, error) {
	secretBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretBytes)
	if err != nil {
		return "", err
	}

	encoding := base64.URLEncoding
	encoded := make([]byte, encoding.EncodedLen(len(secretBytes)))
	encoding.Encode(encoded, secretBytes)

	return string(encoded), nil
}

var filterTenantRegexp = regexp.MustCompile(
	`^\s*tenant\.id\s*=\s*"?([0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12})"?\s*$`,
)
