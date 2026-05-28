package download

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

func newKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestHMACSigner_RoundTrip(t *testing.T) {
	s := NewHMACSigner(newKey(t))
	sig, err := s.Sign([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify([]byte("hello"), sig); err != nil {
		t.Fatalf("Verify(valid sig): %v", err)
	}
}

func TestHMACSigner_TamperedData(t *testing.T) {
	s := NewHMACSigner(newKey(t))
	sig, _ := s.Sign([]byte("hello"))
	if err := s.Verify([]byte("hellp"), sig); err == nil {
		t.Fatal("Verify accepted tampered data")
	}
}

func TestHMACSigner_TamperedSignature(t *testing.T) {
	s := NewHMACSigner(newKey(t))
	sig, _ := s.Sign([]byte("hello"))
	sig[0] ^= 0xff
	if err := s.Verify([]byte("hello"), sig); err == nil {
		t.Fatal("Verify accepted tampered signature")
	}
}

func TestHMACSigner_WrongLengthSignature(t *testing.T) {
	s := NewHMACSigner(newKey(t))
	if err := s.Verify([]byte("hello"), []byte{0x00}); err == nil {
		t.Fatal("Verify accepted short signature")
	}
}

func TestHMACSigner_DifferentKeys(t *testing.T) {
	a := NewHMACSigner(newKey(t))
	b := NewHMACSigner(newKey(t))
	sigA, _ := a.Sign([]byte("hello"))
	sigB, _ := b.Sign([]byte("hello"))
	if bytes.Equal(sigA, sigB) {
		t.Fatal("different keys produced equal signatures")
	}
	if err := a.Verify([]byte("hello"), sigB); err == nil {
		t.Fatal("Verify accepted signature from different key")
	}
}

func TestGenerateHMACKey(t *testing.T) {
	key := GenerateHMACKey()
	if len(key) != 32 {
		t.Errorf("len(key) = %d, want 32", len(key))
	}
	if bytes.Equal(key, make([]byte, 32)) {
		t.Errorf("key is all zero — randomness almost certainly broken")
	}
}

func TestLoadHMACKey(t *testing.T) {
	keyBytes := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 8)
	keyHexLower := hex.EncodeToString(keyBytes)
	keyHexUpper := strings.ToUpper(hex.EncodeToString(keyBytes))
	keyHexShort := keyHexLower[:len(keyHexLower)-1]
	keyHexLong := keyHexLower + "de"

	cases := []struct {
		name    string
		input   string
		want    []byte // nil when wantErr
		wantErr bool
	}{
		{name: "valid lowercase hex", input: keyHexLower, want: keyBytes},
		{name: "valid uppercase hex", input: keyHexUpper, want: keyBytes},
		{name: "surrounding whitespace trimmed", input: "  \n\t" + keyHexLower + "\n", want: keyBytes},

		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   \n\t  ", wantErr: true},
		{name: "short input", input: keyHexShort, wantErr: true},
		{name: "long input", input: keyHexLong, wantErr: true},
		{name: "non-hex characters", input: "zzzz" + keyHexLower[4:], wantErr: true},
		{name: "odd hex length", input: keyHexLower[:len(keyHexLower)-1], wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadHMACKey([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("LoadHMACKey(%q): expected error, got nil (key=%x)", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadHMACKey(%q): unexpected error: %v", tc.input, err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Errorf("LoadHMACKey(%q) = %x, want %x", tc.input, got, tc.want)
			}
		})
	}
}
