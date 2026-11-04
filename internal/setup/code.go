package setup

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"
	"strings"
)

// codeAlphabet leaves out characters that are easily confused when read off a screen or typed: 0/O, 1/I/L.
const codeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// codeLen gives 31^8, about 8.5e11 possible codes.
// With guesses limited to about one a second, guessing one is not practical.
const codeLen = 8

var errMalformedCode = errors.New("not a setup code")

// newCode returns a new random setup code in its normalised form.
func newCode() (string, error) {
	var sb strings.Builder
	max := big.NewInt(int64(len(codeAlphabet)))
	for range codeLen {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		sb.WriteByte(codeAlphabet[n.Int64()])
	}
	return sb.String(), nil
}

// normaliseCode returns s as a setup code with the formatting a person might add removed.
// Returns errMalformedCode if s could not be a setup code.
func normaliseCode(s string) (string, error) {
	s = strings.ToUpper(s)
	s = strings.NewReplacer("-", "", " ", "").Replace(s)
	if len(s) != codeLen {
		return "", errMalformedCode
	}
	for _, c := range s {
		if !strings.ContainsRune(codeAlphabet, c) {
			return "", errMalformedCode
		}
	}
	return s, nil
}

// displayCode formats a normalised code for people to read, as two groups of four.
func displayCode(code string) string {
	if len(code) != codeLen {
		return code
	}
	return code[:codeLen/2] + "-" + code[codeLen/2:]
}

func hashCode(code string) []byte {
	sum := sha256.Sum256([]byte(code))
	return sum[:]
}
