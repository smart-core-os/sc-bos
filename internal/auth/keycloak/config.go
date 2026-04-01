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

// DefaultPermittedSignatureAlgorithms lists the signature algorithms permitted by default.
// TODO: reduce the number of permitted signature algorithms for all keycloak installations
// KeyCloak will select "a reasonable default" cipher suite if none is specified by the installation
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
	string(jose.HS256),
}
