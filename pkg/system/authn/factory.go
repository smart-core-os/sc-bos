package authn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/cors"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/internal/auth/accesstoken"
	"github.com/smart-core-os/sc-bos/internal/auth/keycloak"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/authn/config"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const TokenEndpointPath = "/oauth2/token"

func Factory() system.Factory {
	return &factory{
		server: &nextOrNotFound{},
	}
}

type factory struct {
	handleOnce sync.Once // ensures we only mux.Handle once - otherwise it would panic
	server     *nextOrNotFound
}

func (f *factory) New(services system.Services) service.Lifecycle {
	f.handleOnce.Do(func() {
		services.HTTPMux.Handle(TokenEndpointPath, f.server)
	})
	s := &System{
		server:        f.server,
		configDirs:    services.ConfigDirs,
		clienter:      services.Node,
		cohortManager: services.CohortManager,
		logger:        services.Logger.Named("authn"),
		validators:    services.TokenValidators,
		accounts:      services.Accounts,
	}
	s.Service = service.New(service.MonoApply(s.applyConfig),
		service.WithParser(config.ReadConfig),
		service.WithOnStop[config.Root](func() {
			s.Clear()
		}),
	)
	return s
}

type System struct {
	*service.Service[config.Root]
	server *nextOrNotFound

	configDirs    []string
	clienter      node.ClientConner
	cohortManager node.Remote
	logger        *zap.Logger
	accounts      *account.Store // may be nil

	validators      *token.ValidatorSet
	addedValidators []token.Validator // the validators we setup in applyConfig, used to remove them again
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	// other cleanup is done as part of service.WithOnStop in New
	s.deleteValidators()

	var serveTokenEndpoint bool
	tokenServerOpts := []accesstoken.ServerOption{
		accesstoken.WithLogger(s.logger.Named("server")),
		accesstoken.WithPermittedSignatureAlgorithms(keycloak.DefaultPermittedSignatureAlgorithms),
	}

	if cfg.System != nil {
		validity := cfg.System.Validity.Or(15 * time.Minute)
		tenantVerifier, err := s.systemTenantVerifier(cfg)
		if err != nil {
			return err
		}
		serveTokenEndpoint = true
		tokenServerOpts = append(tokenServerOpts, accesstoken.WithClientCredentialFlow(tenantVerifier, validity))

		if cfg.System.LocalAccounts && s.accounts != nil {
			verifier := newLocalServiceVerifier(s.accounts)
			tokenServerOpts = append(tokenServerOpts, accesstoken.WithClientCredentialFlow(verifier, validity))
		}

		s.logger.Debug("using system tenant verifier", zap.Duration("validity", validity))
	}

	if cfg.User != nil {
		// User accounts that are verified by external authorization servers don't need us to
		// host the oauth/token endpoint, so don't

		validity := cfg.User.Validity.Or(24 * time.Hour)

		var localAccountsAvailable bool
		if cfg.User.LocalAccounts && s.accounts != nil {
			localAccountsAvailable = true
			serveTokenEndpoint = true
			verifier := newLocalUserVerifier(s.accounts)
			tokenServerOpts = append(tokenServerOpts, accesstoken.WithPasswordFlow(verifier, validity))
			s.logger.Debug("using local user database verifier", zap.Duration("validity", validity))
		}

		// Verify user credentials via the OAuth2 Password Flow using a local file containing accounts.
		// Validate access tokens that were generated via this flow.
		if cfg.User.FileAccounts != nil {
			identities, err := loadFileIdentities(cfg.User.FileAccounts, s.configDirs, "users.json")
			if err != nil {
				return fmt.Errorf("user %w", err)
			}
			s.logger.Debug("loaded user accounts from file", zap.Int("count", len(identities)))

			// if we are importing accounts, they will all be available through the local accounts verifier,
			// so no need to add the static verifier as well
			if cfg.User.ImportFileAccounts && localAccountsAvailable {
				s.logger.Debug("importing user accounts from file into database", zap.Int("count", len(identities)))
				err = importIdentities(ctx, s.accounts, identities, s.logger.Named("import"))
			} else {
				s.logger.Debug("using static file verifier for user accounts")
				fileVerifier, err := newStaticVerifier(identities)
				if err != nil {
					return fmt.Errorf("user %w", err)
				}
				tokenServerOpts = append(tokenServerOpts, accesstoken.WithPasswordFlow(fileVerifier, validity))
				serveTokenEndpoint = true
			}
		}

		// Validate access tokens against a remote keycloak server.
		// This is done against the keys returned by the file the OIDC JWKS URI property, which Keycloak supports.
		if cfg.User.Keycloak != nil {
			authConfig := *cfg.User.Keycloak // copy
			if authConfig.ClientID == "" {
				authConfig.ClientID = "sc-api"
			}
			validator := keycloak.NewOIDCTokenValidator(authConfig)

			s.addedValidators = append(s.addedValidators, validator)
			s.validators.Append(validator)
			s.logger.Debug("using keycloak OIDC token validator")
		}

	}

	if serveTokenEndpoint {
		server, err := accesstoken.NewServer("authn", tokenServerOpts...)
		if err != nil {
			return fmt.Errorf("new token server: %w", err)
		}
		validator := server.TokenValidator()

		// validate any access tokens this server issued
		s.addedValidators = append(s.addedValidators, validator)
		s.validators.Append(validator)
		// serve the handler
		s.server.Next(cors.Default().Handler(server))
	}

	return nil
}

func (s *System) Clear() {
	s.server.Clear()
	s.deleteValidators()
}

func (s *System) deleteValidators() {
	vs := s.addedValidators
	s.addedValidators = nil
	for _, v := range vs {
		s.validators.Delete(v)
	}
}
