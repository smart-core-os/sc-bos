package resource

import (
	"encoding/base64"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GenerateUniqueId attempts to find a unique id using rand and using the given function to check existence.
// This will attempt a few times before giving up and returning an error.
func GenerateUniqueId(rng io.Reader, exists func(candidate string) bool) (string, error) {
	tries := 10
	for i := 0; i < tries; i++ {
		r := make([]byte, 6 + i)
		_, _ = rng.Read(r)
		idCandidate := base64.RawURLEncoding.EncodeToString(r)
		if idCandidate != "" && !exists(idCandidate) {
			return idCandidate, nil
		}
	}
	return "", status.Error(codes.Aborted, fmt.Sprintf("id generation attempts exhausted after %v attempts", tries))
}
