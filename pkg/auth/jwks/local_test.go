package jwks

import (
	"context"
	"crypto/x509"
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

// TestLocalKeySet_VerifySignature_keyConfusion attempts the RS256->HS256 algorithm confusion forgery
// described at https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/, signing
// with the published RSA public key as the HMAC secret. HS256 is deliberately permitted here, so what
// blocks the forgery is go-jose binding the algorithm to the key type, not our allow-list.
func TestLocalKeySet_VerifySignature_keyConfusion(t *testing.T) {
	publicKey := testJWK1.Public()
	// the attacker's secret: the public key material, which is all they have
	secret, err := x509.MarshalPKIXPublicKey(publicKey.Key)
	if err != nil {
		t.Fatal(err)
	}

	// kid must name a key in the set, or this fails at key lookup and proves nothing
	opts := (&jose.SignerOptions{}).WithHeader("kid", publicKey.KeyID)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: secret}, opts)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign([]byte("forged"))
	if err != nil {
		t.Fatal(err)
	}
	forged, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{publicKey}}
	localKeySet := NewLocalKeySet(keySet, []jose.SignatureAlgorithm{jose.RS256, jose.HS256})

	_, err = localKeySet.VerifySignature(context.Background(), forged)
	// go-jose collapses the underlying ErrUnsupportedAlgorithm into ErrCryptoFailure
	// (JSONWebSignature.DetachedVerify), so that is the sentinel we see here.
	if !errors.Is(err, jose.ErrCryptoFailure) {
		t.Errorf("forged token: got %v, want ErrCryptoFailure", err)
	}
	// Guards the test itself: a forgery rejected at key lookup would pass any "did it fail?"
	// assertion without the signature ever being checked against the RSA key.
	if errors.Is(err, ErrKeyNotFound) {
		t.Error("forgery was rejected at key lookup, so it never reached signature verification")
	}
}

// TestLocalKeySet_VerifySignature_algorithmNotPermitted checks the allow-list is actually plumbed
// through to jose.ParseSigned, so a token whose algorithm isn't permitted is rejected up front. This
// is the parse-time gate only; the forgery above is what proves key confusion itself is blocked.
func TestLocalKeySet_VerifySignature_algorithmNotPermitted(t *testing.T) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte("0123456789abcdef0123456789abcdef")}, nil)
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

	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{testJWK1.Public()}}
	localKeySet := NewLocalKeySet(keySet, []jose.SignatureAlgorithm{jose.RS256})

	if _, err := localKeySet.VerifySignature(context.Background(), hs256JWS); err == nil {
		t.Error("expected an HS256 token to be rejected when HS256 isn't permitted")
	}
}
