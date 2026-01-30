// Package oidc provides access to remote OpenID Connect configuration.
package oidc

import (
	"context"
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/util/fetch"
)

// Config represents an OpenID Connect configuration document.
// It only contains the properties that we need to function, for now.
type Config struct {
	JWKSURI               string `json:"jwks_uri"`
	TokenEndpoint         string `json:"token_endpoint"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
}

// FetchConfig retrieves OIDC Config from the given issuer URL prefix using the `.well-known/openid-configuration` suffix.
func FetchConfig(ctx context.Context, issuer string) (Config, error) {
	url := fmt.Sprintf("%s/.well-known/openid-configuration", issuer)

	var config Config
	err := fetch.JSON(ctx, url, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
