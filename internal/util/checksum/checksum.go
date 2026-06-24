// Package checksum handles type-prefixed content hashes of the form "<algo>:<base64>", the encoding
// Smart Core Connect uses for artefact checksums on the device check-in wire. The digest is the raw
// hash bytes, standard-base64 encoded. md5 and sha256 are supported; the prefix lets the algorithm
// change without a wire break.
package checksum

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"
)

// Algorithm is a supported hash function, the "<algo>" half of a type-prefixed checksum.
type Algorithm string

// Supported algorithms.
const (
	MD5    Algorithm = "md5"
	SHA256 Algorithm = "sha256"
)

// spec records how to construct a hash and the raw digest length it produces.
type spec struct {
	new  func() hash.Hash
	size int
}

var algorithms = map[Algorithm]spec{
	MD5:    {md5.New, md5.Size},
	SHA256: {sha256.New, sha256.Size},
}

// NewHash returns a fresh hash.Hash for the algorithm, or an error if the algorithm is not supported.
func (a Algorithm) NewHash() (hash.Hash, error) {
	s, ok := algorithms[a]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm %q", a)
	}
	return s.new(), nil
}

// Checksum is a computed content hash: an algorithm and the raw digest it produced.
type Checksum struct {
	Algo   Algorithm
	Digest []byte
}

// Parse reads a "<algo>:<base64>" string. It errors on an unknown algorithm, invalid base64, or a
// digest whose length does not match the algorithm.
func Parse(s string) (Checksum, error) {
	algoStr, encoded, ok := strings.Cut(s, ":")
	if !ok {
		return Checksum{}, fmt.Errorf("checksum %q: missing algorithm prefix", s)
	}
	algo := Algorithm(algoStr)
	spec, ok := algorithms[algo]
	if !ok {
		return Checksum{}, fmt.Errorf("checksum %q: unsupported algorithm %q", s, algo)
	}
	digest, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Checksum{}, fmt.Errorf("checksum %q: decode digest: %w", s, err)
	}
	if len(digest) != spec.size {
		return Checksum{}, fmt.Errorf("checksum %q: %s digest must be %d bytes, got %d", s, algo, spec.size, len(digest))
	}
	return Checksum{Algo: algo, Digest: digest}, nil
}

// Format encodes a raw digest as "<algo>:<base64>". The digest is not validated against algo; use it
// only with a digest you produced for that algorithm.
func Format(algo Algorithm, digest []byte) string {
	return string(algo) + ":" + base64.StdEncoding.EncodeToString(digest)
}

// String reformats the checksum as "<algo>:<base64>".
func (c Checksum) String() string {
	return Format(c.Algo, c.Digest)
}
