package auth

import (
	"encoding/json"
	"strings"

	"github.com/smart-core-os/sc-bos/pkg/util/slices"
)

type JWTScopes []string

func (s *JWTScopes) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Join(*s, " "))
}

func (s *JWTScopes) UnmarshalJSON(data []byte) error {
	var joined string
	err := json.Unmarshal(data, &joined)
	if err != nil {
		return err
	}
	*s = strings.Fields(joined)
	return nil
}

func (s *JWTScopes) HasScopes(required ...string) bool {
	return slices.ContainsAll(required, *s)
}
