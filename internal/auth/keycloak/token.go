package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-jose/go-jose/v4/jwt"

	jose_utils "github.com/smart-core-os/sc-bos/internal/util/jose"
	"github.com/smart-core-os/sc-bos/pkg/auth"
	"github.com/smart-core-os/sc-bos/pkg/auth/jwks"
	"github.com/smart-core-os/sc-bos/pkg/auth/oidc"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
)

// accessTokenPayload describes the claims present in a token issued by a Keycloak Authorization Server.
type accessTokenPayload struct {
	jwt.Claims
	Roles          []string                  `json:"roles"`
	Scopes         auth.JWTScopes            `json:"scope"`
	ResourceAccess map[string]resourceAccess `json:"resource_access"`
}

func (a *accessTokenPayload) allRoles() []string {
	var roles []string
	roles = append(roles, a.Roles...)
	for _, res := range a.ResourceAccess {
		roles = append(roles, res.Roles...)
	}
	return roles
}

type resourceAccess struct {
	Roles []string `json:"roles"`
}

// NewTokenValidator returns a token.Validator that validates tokens against the given jwks.KeySet which should be
// hosted by Keycloak. During validation known Keycloak claims are validated, converted into token.Claims, and returned.
func NewTokenValidator(config *Config, keySet jwks.KeySet) token.Validator {
	return &tokenValidator{
		keySet: keySet,
		expected: jwt.Expected{
			// todo: enable audience checking once we've figured out how to configure KeyCloak
			// Audience: jwt.Audience{config.ClientID},
			Issuer: config.Issuer(),
		},
	}
}

// NewOIDCTokenValidator returns a token.Validator like NewTokenValidator using well known OIDC configuration for available keys.
func NewOIDCTokenValidator(cfg Config) token.Validator {
	issuer := cfg.Issuer()

	var mu sync.RWMutex
	var underlying token.Validator
	cachedValidator := func(ctx context.Context) (token.Validator, error) {
		mu.RLock()
		v := underlying
		mu.RUnlock()

		if v == nil {
			mu.Lock()
			defer mu.Unlock()
			if underlying != nil {
				return underlying, nil
			}

			// todo: during error conditions this fetches every time, make it not do that.
			// This is in the critical path of token validation which is in the critical path of RPCs
			authUrls, err := oidc.FetchConfig(ctx, issuer)
			if err != nil {
				return nil, fmt.Errorf("oidc fetch: %w", err)
			}
			keySet := jwks.NewRemoteKeySet(ctx, authUrls.JWKSURI, jose_utils.ConvertToNativeJose(DefaultPermittedSignatureAlgorithms))
			v = NewTokenValidator(&cfg, keySet)
			underlying = v
		}

		return v, nil
	}

	return token.ValidatorFunc(func(ctx context.Context, token string) (*token.Claims, error) {
		v, err := cachedValidator(ctx)
		if err != nil {
			return nil, err
		}
		return v.ValidateAccessToken(ctx, token)
	})
}

type tokenValidator struct {
	keySet   jwks.KeySet
	expected jwt.Expected
}

func (v *tokenValidator) ValidateAccessToken(ctx context.Context, tokenStr string) (*token.Claims, error) {
	payloadBytes, err := v.keySet.VerifySignature(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	var payload accessTokenPayload
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, err
	}

	err = payload.Claims.Validate(v.expected)
	if err != nil {
		return nil, err
	}

	return &token.Claims{
		SystemRoles: payload.allRoles(),
		IsService:   false,
	}, nil
}
