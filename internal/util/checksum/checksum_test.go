package checksum

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"testing"
)

// Format a raw digest for the wire, then Parse it back and hash a payload to verify it.
func Example() {
	digest := sha256.Sum256([]byte("artefact bytes"))
	wire := Format(SHA256, digest[:])

	c, err := Parse(wire)
	if err != nil {
		panic(err)
	}
	h, _ := c.Algo.NewHash()
	h.Write([]byte("artefact bytes"))
	fmt.Println(c.Algo, bytes.Equal(h.Sum(nil), c.Digest))
	// Output: sha256 true
}

func TestParse_roundTrip(t *testing.T) {
	payload := []byte("the artefact bytes")
	md5Sum := md5.Sum(payload)
	shaSum := sha256.Sum256(payload)

	tests := []struct {
		name     string
		algo     Algorithm
		digest   []byte
		wantSize int
	}{
		{"md5", MD5, md5Sum[:], md5.Size},
		{"sha256", SHA256, shaSum[:], sha256.Size},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Format(tt.algo, tt.digest)
			c, err := Parse(s)
			if err != nil {
				t.Fatalf("Parse(%q): %v", s, err)
			}
			if c.Algo != tt.algo {
				t.Errorf("algo = %q, want %q", c.Algo, tt.algo)
			}
			if !bytes.Equal(c.Digest, tt.digest) {
				t.Errorf("digest mismatch")
			}
			if len(c.Digest) != tt.wantSize {
				t.Errorf("digest len = %d, want %d", len(c.Digest), tt.wantSize)
			}
			if c.String() != s {
				t.Errorf("String() = %q, want %q", c.String(), s)
			}
		})
	}
}

func TestParse_errors(t *testing.T) {
	shaSum := sha256.Sum256([]byte("x"))
	tests := []struct {
		name string
		in   string
	}{
		{"no prefix", "abcdef"},
		{"unknown algo", Format("sha1", shaSum[:])},
		{"bad base64", "sha256:not*base64"},
		{"wrong length for algo", Format(MD5, shaSum[:])}, // 32-byte digest labelled md5
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.in); err == nil {
				t.Errorf("Parse(%q) = nil error, want error", tt.in)
			}
		})
	}
}

func TestAlgorithm_NewHash(t *testing.T) {
	payload := []byte("verify me")
	for _, algo := range []Algorithm{MD5, SHA256} {
		t.Run(string(algo), func(t *testing.T) {
			var want []byte
			switch algo {
			case MD5:
				sum := md5.Sum(payload)
				want = sum[:]
			case SHA256:
				sum := sha256.Sum256(payload)
				want = sum[:]
			}
			c, err := Parse(Format(algo, want))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			h, err := c.Algo.NewHash()
			if err != nil {
				t.Fatalf("NewHash: %v", err)
			}
			h.Write(payload)
			if !bytes.Equal(h.Sum(nil), c.Digest) {
				t.Errorf("hash of payload does not match parsed digest")
			}
		})
	}
}
