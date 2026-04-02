package accesstoken

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"go.uber.org/zap"
)

func TestTokenSource_createAndVerify(t *testing.T) {
	ts := newTestSource(t)
	token, err := ts.GenerateAccessToken(SecretData{TenantID: "Foo"}, 10*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken %v", err)
	}

	_, err = ts.ValidateAccessToken(nil, token)
	if err != nil {
		t.Fatalf("ValidateAccessToken %v", err)
	}
}

func TestLoadOrGenerateSigningKey_generatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	sk, err := LoadOrGenerateSigningKey(path, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Algorithm != jose.HS256 {
		t.Errorf("algorithm = %v, want HS256", sk.Algorithm)
	}
	keyBytes, ok := sk.Key.([]byte)
	if !ok {
		t.Fatalf("key type = %T, want []byte", sk.Key)
	}
	if len(keyBytes) != 32 {
		t.Errorf("key length = %d, want 32", len(keyBytes))
	}

	// file must have been written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("key file not written: %v", err)
	}
	if len(data) != 64 {
		t.Errorf("file length = %d, want 64 hex chars", len(data))
	}
}

func TestLoadOrGenerateSigningKey_persistedFileIsUsed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	// generate once to create the file
	sk1, err := LoadOrGenerateSigningKey(path, logger)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// load again — must return the same key
	sk2, err := LoadOrGenerateSigningKey(path, logger)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	k1 := sk1.Key.([]byte)
	k2 := sk2.Key.([]byte)
	if hex.EncodeToString(k1) != hex.EncodeToString(k2) {
		t.Error("keys differ between calls; expected the persisted key to be reused")
	}
}

func TestLoadOrGenerateSigningKey_loadsExistingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	// write a known key
	known := make([]byte, 32)
	for i := range known {
		known[i] = byte(i)
	}
	encoded := hex.EncodeToString(known)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		t.Fatal(err)
	}

	sk, err := LoadOrGenerateSigningKey(path, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hex.EncodeToString(sk.Key.([]byte)) != encoded {
		t.Error("loaded key does not match the file contents")
	}
}

func TestLoadOrGenerateSigningKey_rejectsBadHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	if err := os.WriteFile(path, []byte("not-valid-hex!!"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOrGenerateSigningKey(path, logger)
	if err == nil {
		t.Fatal("expected error for invalid hex, got nil")
	}
}

func TestLoadOrGenerateSigningKey_rejectsWrongLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	// 16 bytes → 32 hex chars, not 64
	short := make([]byte, 16)
	if err := os.WriteFile(path, []byte(hex.EncodeToString(short)), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOrGenerateSigningKey(path, logger)
	if err == nil {
		t.Fatal("expected error for wrong key length, got nil")
	}
}

func TestLoadOrGenerateSigningKey_trailingNewlineAccepted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	known := make([]byte, 32)
	encoded := hex.EncodeToString(known)
	// openssl rand -hex 32 produces output with a trailing newline
	if err := os.WriteFile(path, []byte(encoded+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	sk, err := LoadOrGenerateSigningKey(path, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hex.EncodeToString(sk.Key.([]byte)) != encoded {
		t.Error("key mismatch after trimming trailing newline")
	}
}

func TestLoadOrGenerateSigningKey_filePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	logger := zap.NewNop()

	if _, err := LoadOrGenerateSigningKey(path, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %04o, want 0600", perm)
	}
}

func newTestSource(t *testing.T) *Source {
	t.Helper()
	key, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}
	return &Source{
		Key:                 key,
		Issuer:              "test",
		Now:                 time.Now,
		SignatureAlgorithms: []string{string(jose.HS256)},
	}
}
