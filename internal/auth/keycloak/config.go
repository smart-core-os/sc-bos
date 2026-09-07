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

// DefaultPermittedSignatureAlgorithms lists the signature algorithms permitted by default. It stays
// broad because Keycloak selects "a reasonable default" cipher suite when an installation specifies
// none. That breadth is not an algorithm confusion exposure
// (https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/): go-jose picks its
// verifier from the type of the key rather than the token's alg header, so an algorithm the key
// cannot carry is rejected whatever this list permits. See
// jwks.TestLocalKeySet_VerifySignature_keyConfusion.
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
