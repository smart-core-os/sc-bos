package jose

import "github.com/go-jose/go-jose/v4"

// ConvertToNativeJose converts string Signature Algorithms names into go-jose/v4 SignatureAlgorithm types
func ConvertToNativeJose(algs []string) []jose.SignatureAlgorithm {
	var joseAlgs []jose.SignatureAlgorithm
	for _, alg := range algs {
		joseAlgs = append(joseAlgs, jose.SignatureAlgorithm(alg))
	}

	return joseAlgs
}
