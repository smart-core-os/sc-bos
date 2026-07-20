package keycloak

import (
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
)

// TestDefaultPermittedSignatureAlgorithms_excludesSymmetric guards the security-hardening
// invariant: Keycloak tokens are verified against JWKS public keys, so no symmetric (HS*)
// algorithm may be permitted (it would enable an RS256->HS256 key-confusion forgery), and
// RS256 must remain permitted since that is what the shipped realm signs with.
func TestDefaultPermittedSignatureAlgorithms_excludesSymmetric(t *testing.T) {
	var hasRS256 bool
	for _, alg := range DefaultPermittedSignatureAlgorithms {
		if strings.HasPrefix(alg, "HS") {
			t.Errorf("symmetric algorithm %q must not be permitted for Keycloak token verification", alg)
		}
		if alg == string(jose.RS256) {
			hasRS256 = true
		}
	}
	if !hasRS256 {
		t.Error("RS256 must be permitted (the shipped realm signs with RS256)")
	}
}
