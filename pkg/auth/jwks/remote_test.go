package jwks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/go-cmp/cmp"
)

func TestRemoteKeySet_VerifySignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			testJWK1.Public(),
		}}
		data, err := json.Marshal(jwks)
		if err != nil {
			t.Error(err)
			panic(err)
		}
		_, err = writer.Write(data)
		if err != nil {
			t.Error(err)
			panic(err)
		}
	}))
	defer server.Close()

	remoteKeySet := NewRemoteKeySet(context.Background(), server.URL, []jose.SignatureAlgorithm{jose.RS256})
	// sign a test message using the key we will use
	inputPayload := []byte("TestRemoteKeySet_VerifySignature")
	sig := signJWS(t, testJWK1, inputPayload)

	outputPayload, err := remoteKeySet.VerifySignature(context.Background(), sig)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(inputPayload, outputPayload) {
		t.Error("payload mismatch")
	}
}
