package authn

import (
	"context"
	"errors"
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/auth/accesstoken"
	"github.com/smart-core-os/sc-bos/pkg/proto/tenantpb"
	"github.com/smart-core-os/sc-bos/pkg/system/authn/config"
)

func (s *System) systemTenantVerifier(cfg config.Root) (accesstoken.Verifier, error) {
	// System account credentials are verified and exchanged to an access token using the
	// OAuth2 Client Credentials flow.
	// There are multiple actors who can verify credentials, these are setup here.
	// Only access tokens that were produced by this verification will be validated.

	// verify system accounts using file provided identities
	fileVerifier, err := loadFileVerifier(cfg.System.FileAccounts, s.configDirs, "tenants.json")
	if err != nil {
		// note, NotFound errors are already handled by loadFileVerifier
		return nil, fmt.Errorf("system %w", err)
	}

	// verify system accounts using the local systems/tenants package, via TenantApi
	nodeVerifier := accesstoken.NeverVerify(errors.New("tenant system verification not enabled"))
	if cfg.System.TenantAccounts {
		client := tenantpb.NewTenantApiClient(s.clienter.ClientConn())
		nodeVerifier = accesstoken.VerifierFunc(func(ctx context.Context, id, secret string) (accesstoken.SecretData, error) {
			return accesstoken.RemoteVerify(ctx, id, secret, client)
		})
	}

	// verify system accounts using the cohort manager, via TenantApi
	cohortVerifier := accesstoken.NeverVerify(errors.New("cohort verification not enabled"))
	if cfg.System.CohortAccounts {
		cohortVerifier = accesstoken.VerifierFunc(func(ctx context.Context, id, secret string) (data accesstoken.SecretData, err error) {
			conn, err := s.cohortManager.Connect(ctx)
			if err != nil {
				return data, err
			}
			if conn == nil {
				return data, errors.New("cohort manager not available")
			}
			return accesstoken.RemoteVerify(ctx, id, secret, tenantpb.NewTenantApiClient(conn))
		})
	}

	verifier := accesstoken.FirstSuccessfulVerifier{
		fileVerifier,
		nodeVerifier,
		cohortVerifier,
	}
	return verifier, nil
}
