package download

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errInvalidSignature = errors.New("download: invalid signature")
	errInvalidHMACKey   = errors.New("download: invalid hmac key")
	errTokenFormat      = errors.New("download: malformed token")
	errEnvelope         = errors.New("download: malformed envelope")
	errExpired          = errors.New("download: token expired")
)

// Signer signs arbitrary bytes and verifies signatures over them. The
// download package uses it to sign envelope bytes; implementations choose the
// algorithm.
type Signer interface {
	// Sign returns a signature over data.
	Sign(data []byte) ([]byte, error)
	// Verify returns nil if signature is a valid signature over data, or an
	// error otherwise. Implementations must use constant-time comparison.
	Verify(data, signature []byte) error
}

// keySize is the length of keys produced by GenerateHMACKey.
const keySize = sha256.Size

// minKeySize is the minimum accepted HMAC key length. HMAC-SHA256 accepts keys
// of any length (RFC 2104) — longer keys are reduced internally, shorter ones
// zero-padded — so we only enforce a lower bound for entropy: at least the hash
// output size (32 bytes / 256 bits). This admits previously valid keys, such as
// 64-byte (block-sized) keys, that an exact-length check would wrongly reject.
const minKeySize = keySize

// HMACSigner is a Signer using HMAC-SHA256.
type HMACSigner struct {
	key []byte
}

// NewHMACSigner returns an HMACSigner with the given key. The key must be at
// least 32 bytes of cryptographically random data (e.g. from crypto/rand);
// longer keys are accepted.
//
// Panics if key is too short.
func NewHMACSigner(key []byte) *HMACSigner {
	if len(key) < minKeySize {
		panic(errInvalidHMACKey)
	}
	return &HMACSigner{key: key}
}

func (h *HMACSigner) Sign(data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, h.key)
	mac.Write(data)
	return mac.Sum(nil), nil
}

func (h *HMACSigner) Verify(data, signature []byte) error {
	mac := hmac.New(sha256.New, h.key)
	mac.Write(data)
	if !hmac.Equal(mac.Sum(nil), signature) {
		return errInvalidSignature
	}
	return nil
}

// GenerateHMACKey returns 32 cryptographically random bytes suitable for use
// as an HMAC-SHA256 key.
func GenerateHMACKey() []byte {
	key := make([]byte, keySize)
	_, _ = rand.Read(key) // guaranteed to read all or panic
	return key
}

// LoadHMACKey loads a hexadecimal-encoded HMAC key from a buffer.
//
// The decoded key must be at least 32 bytes; longer keys are accepted, matching
// HMAC-SHA256's support for keys of any length. Whitespace surrounding the hex
// string is ignored.
func LoadHMACKey(buf []byte) ([]byte, error) {
	str := strings.TrimSpace(string(buf))
	key, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	if len(key) < minKeySize {
		return nil, errInvalidHMACKey
	}
	return key, nil
}

const tokenDelim = "."

// signToken builds the wire token: <b64(envelope)>.<b64(signature)>.
func signToken(signer Signer, typ string, payload []byte, expiry time.Time) (string, error) {
	env := &Envelope{
		Type:    typ,
		Payload: payload,
		Expiry:  timestamppb.New(expiry),
	}
	envBytes, err := proto.Marshal(env)
	if err != nil {
		return "", err
	}
	sig, err := signer.Sign(envBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(envBytes) + tokenDelim +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// verifyToken splits, base64-decodes, HMAC-verifies, and only then unmarshals
// the envelope. Callers receive an unmarshalled envelope or a sentinel error.
func verifyToken(signer Signer, token string) (*Envelope, error) {
	encEnv, encSig, ok := strings.Cut(token, tokenDelim)
	if !ok {
		return nil, errTokenFormat
	}
	if strings.Contains(encSig, tokenDelim) {
		return nil, errTokenFormat
	}
	envBytes, err := base64.RawURLEncoding.DecodeString(encEnv)
	if err != nil {
		return nil, errTokenFormat
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(encSig)
	if err != nil {
		return nil, errTokenFormat
	}
	if err := signer.Verify(envBytes, sigBytes); err != nil {
		return nil, errInvalidSignature
	}
	// signature verified; safe to interpret the envelope.
	var env Envelope
	if err := proto.Unmarshal(envBytes, &env); err != nil {
		return nil, errEnvelope
	}
	if env.GetExpiry() == nil {
		return nil, errExpired
	}
	if time.Now().After(env.GetExpiry().AsTime()) {
		return nil, errExpired
	}
	return &env, nil
}
