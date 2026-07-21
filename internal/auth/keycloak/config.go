package keycloak

import (
	"fmt"

	"github.com/go-jose/go-jose/v4"
)

type Config struct {
	URL      string `json:"url,omitempty"` // Root URL of Keycloak server
	Realm    string `json:"realm,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

func (c *Config) Issuer() string {
	return fmt.Sprintf("%s/realms/%s", c.URL, c.Realm)
}

// DefaultPermittedSignatureAlgorithms lists the JWT signature algorithms permitted when
// verifying tokens issued by Keycloak. Only asymmetric algorithms are permitted: Keycloak
// signs tokens with an asymmetric key (the shipped realm uses RS256) and they are verified
// against its published JWKS public keys. The symmetric HS256 is deliberately excluded —
// permitting it on a path that verifies with public keys opens an RS256->HS256 key-confusion
// forgery vector.
var DefaultPermittedSignatureAlgorithms = []string{
	string(jose.RS256),
	string(jose.RS384),
	string(jose.RS512),
	string(jose.ES256),
	string(jose.ES384),
	string(jose.ES512),
	string(jose.PS256),
	string(jose.PS384),
	string(jose.PS512),
}
