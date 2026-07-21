package jwks

import (
	"context"
	"errors"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/go-cmp/cmp"
)

func TestLocalKeySet_VerifySignature(t *testing.T) {
	inputPayload := []byte("TestLocalKeySet_VerifySignature")

	// sign a test message using the key we will use
	sig1 := signJWS(t, testJWK1, inputPayload)
	// sign again using the other key that's not in our key set
	sig2 := signJWS(t, testJWK2, inputPayload)

	// verify the first signature using the JWKS, which should succeed
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{testJWK1.Public()}}
	localKeySet := NewLocalKeySet(jwks, []jose.SignatureAlgorithm{jose.RS256})
	outputPayload, err := localKeySet.VerifySignature(context.Background(), sig1)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(inputPayload, outputPayload) {
		t.Error("payloads different")
	}

	// attempt to verify the second signature using the JWKS, which should fail as that key's not in the set
	_, err = localKeySet.VerifySignature(context.Background(), sig2)
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("verification didn't fail as expected: %s", err.Error())
	}
}

// TestLocalKeySet_VerifySignature_rejectsDisallowedAlgorithm proves the security-hardening
// property behind keycloak.DefaultPermittedSignatureAlgorithms: when only asymmetric
// algorithms are permitted, an HS256-signed token is rejected before key verification, so
// the RS256->HS256 key-confusion forgery vector is closed.
func TestLocalKeySet_VerifySignature_rejectsDisallowedAlgorithm(t *testing.T) {
	hmacKey := []byte("0123456789abcdef0123456789abcdef") // 32 bytes, valid for HS256
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: hmacKey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign([]byte("hs256 payload"))
	if err != nil {
		t.Fatal(err)
	}
	hs256JWS, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	// permit only asymmetric algorithms, as the Keycloak verifier now does
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{testJWK1.Public()}}
	localKeySet := NewLocalKeySet(jwks, []jose.SignatureAlgorithm{jose.RS256, jose.ES256, jose.PS256})

	if _, err := localKeySet.VerifySignature(context.Background(), hs256JWS); err == nil {
		t.Error("expected an HS256-signed token to be rejected when only asymmetric algorithms are permitted")
	}
}
