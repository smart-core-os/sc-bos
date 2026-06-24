package cloud

import (
	"bytes"
	"crypto/sha256"
	"io"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/util/checksum"
)

func TestVerifiedReader(t *testing.T) {
	payload := []byte("some config payload bytes")
	sum := sha256.Sum256(payload)
	good := checksum.Format(checksum.SHA256, sum[:])

	t.Run("matching checksum reads through", func(t *testing.T) {
		vr, err := newVerifiedReader(io.NopCloser(bytes.NewReader(payload)), good)
		if err != nil {
			t.Fatalf("newVerifiedReader: %v", err)
		}
		got, err := io.ReadAll(vr)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("read %q, want %q", got, payload)
		}
	})

	t.Run("mismatched checksum fails the read", func(t *testing.T) {
		wrong := checksum.Format(checksum.SHA256, make([]byte, sha256.Size)) // digest of nothing this payload hashes to
		vr, err := newVerifiedReader(io.NopCloser(bytes.NewReader(payload)), wrong)
		if err != nil {
			t.Fatalf("newVerifiedReader: %v", err)
		}
		_, err = io.ReadAll(vr)
		if err == nil {
			t.Fatal("expected a checksum mismatch error, got nil")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("error = %v, want it to mention checksum mismatch", err)
		}
	})

	t.Run("malformed checksum is rejected", func(t *testing.T) {
		if _, err := newVerifiedReader(io.NopCloser(bytes.NewReader(payload)), "not-a-checksum"); err == nil {
			t.Fatal("expected an error for a malformed checksum, got nil")
		}
	})
}
