package sysconf

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// Tests that resolve order is respected and key parse errors are correctly propagated.
func TestConfig_ResolveDownloadHMACKey(t *testing.T) {
	dir := t.TempDir()
	keyHex := hex.EncodeToString(bytes.Repeat([]byte{0x1}, 32))
	keyPath := writeFile(t, dir, "primary.key", keyHex)
	legacyHex := hex.EncodeToString(bytes.Repeat([]byte{0x2}, 32))
	legacyPath := writeFile(t, dir, "legacy.key", legacyHex)
	garbagePath := writeFile(t, dir, "garbage.key", "not hex")

	t.Run("reads Download.HMACKeyFile", func(t *testing.T) {
		cfg := Config{Download: &Download{HMACKeyFile: keyPath}}
		got, err := cfg.ResolveDownloadHMACKey(zap.NewNop())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want, _ := hex.DecodeString(keyHex)
		if string(got) != string(want) {
			t.Errorf("got %x, want %x", got, want)
		}
	})

	t.Run("falls back to legacy Devices.DownloadHMACKeyFile", func(t *testing.T) {
		cfg := Config{Devices: &Devices{DownloadHMACKeyFile: legacyPath}}
		got, err := cfg.ResolveDownloadHMACKey(zap.NewNop())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want, _ := hex.DecodeString(legacyHex)
		if string(got) != string(want) {
			t.Errorf("got %x, want %x", got, want)
		}
	})

	t.Run("Download.HMACKeyFile wins over legacy field", func(t *testing.T) {
		cfg := Config{
			Download: &Download{HMACKeyFile: keyPath},
			Devices:  &Devices{DownloadHMACKeyFile: legacyPath}, //nolint:staticcheck
		}
		got, err := cfg.ResolveDownloadHMACKey(zap.NewNop())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want, _ := hex.DecodeString(keyHex)
		if string(got) != string(want) {
			t.Errorf("got %x, want %x", got, want)
		}
	})

	t.Run("no configuration falls back to generated key", func(t *testing.T) {
		got, err := Config{}.ResolveDownloadHMACKey(zap.NewNop())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 32 {
			t.Errorf("len(key) = %d, want 32", len(got))
		}
	})

	t.Run("missing file surfaces fs.ErrNotExist", func(t *testing.T) {
		cfg := Config{Download: &Download{HMACKeyFile: filepath.Join(dir, "nope.key")}}
		_, err := cfg.ResolveDownloadHMACKey(zap.NewNop())
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("parse error from LoadHMACKey is surfaced", func(t *testing.T) {
		cfg := Config{Download: &Download{HMACKeyFile: garbagePath}}
		if _, err := cfg.ResolveDownloadHMACKey(zap.NewNop()); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}
