package accesstoken

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	joseUtils "github.com/smart-core-os/sc-bos/internal/util/jose"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

var ErrUnsupportedTokenVersion = errors.New("unsupported token version")

const claimsVersion = 1

type claims struct {
	Version     int                    `json:"v"` // to detect which schema version this token uses
	Name        string                 `json:"name,omitempty"`
	SystemRoles []string               `json:"roles,omitempty"` // Named roles in JSON for back-compat
	Permissions []permissionAssignment `json:"perms,omitempty"`
}

// like token.PermissionAssignment but with a more compact JSON representation
type permissionAssignment struct {
	Permission   permission.ID                         `json:"p"`
	ResourceType accountpb.RoleAssignment_ResourceType `json:"rt,omitempty"` // will serialise as an integer
	Resource     string                                `json:"r,omitempty"`
}

func permissionAssignmentFromToken(pa token.PermissionAssignment) permissionAssignment {
	return permissionAssignment{
		Permission:   pa.Permission,
		ResourceType: accountpb.RoleAssignment_ResourceType(pa.ResourceType),
		Resource:     pa.Resource,
	}
}

func (pa permissionAssignment) Scoped() bool {
	return pa.ResourceType != accountpb.RoleAssignment_RESOURCE_TYPE_UNSPECIFIED
}

func (pa permissionAssignment) ToTokenPermissionAssignment() token.PermissionAssignment {
	return token.PermissionAssignment{
		Permission:   pa.Permission,
		Scoped:       pa.Scoped(),
		ResourceType: token.ResourceType(pa.ResourceType),
		Resource:     pa.Resource,
	}
}

type Source struct {
	Key                 jose.SigningKey
	Issuer              string
	Now                 func() time.Time
	SignatureAlgorithms []string
}

func (ts *Source) GenerateAccessToken(data SecretData, validity time.Duration) (token string, err error) {
	signer, err := jose.NewSigner(ts.Key, nil)
	if err != nil {
		return "", err
	}

	var now time.Time
	if ts.Now != nil {
		now = ts.Now()
	} else {
		now = time.Now()
	}

	expires := now.Add(validity)

	jwtClaims := jwt.Claims{
		Issuer:    ts.Issuer,
		Subject:   data.TenantID,
		Audience:  jwt.Audience{ts.Issuer},
		Expiry:    jwt.NewNumericDate(expires),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	compressedPermissions := make([]permissionAssignment, 0, len(data.Permissions))
	for _, pa := range data.Permissions {
		compressedPermissions = append(compressedPermissions, permissionAssignmentFromToken(pa))
	}
	customClaims := claims{
		Version:     claimsVersion,
		Name:        data.Title,
		Permissions: compressedPermissions,
		SystemRoles: data.SystemRoles,
	}
	return jwt.Signed(signer).
		Claims(jwtClaims).
		Claims(customClaims).
		Serialize()
}

func (ts *Source) ValidateAccessToken(_ context.Context, tokenStr string) (*token.Claims, error) {
	tok, err := jwt.ParseSigned(tokenStr, joseUtils.ConvertToNativeJose(ts.SignatureAlgorithms))
	if err != nil {
		return nil, err
	}
	var jwtClaims jwt.Claims
	var customClaims claims
	err = tok.Claims(ts.Key.Key, &jwtClaims, &customClaims)
	if err != nil {
		return nil, err
	}
	err = jwtClaims.Validate(jwt.Expected{
		AnyAudience: jwt.Audience{ts.Issuer},
		Issuer:      ts.Issuer,
	})
	if err != nil {
		return nil, err
	}
	if customClaims.Version != claimsVersion {
		// token issued using a schema we no longer support
		return nil, ErrUnsupportedTokenVersion
	}
	tokenPermissions := make([]token.PermissionAssignment, 0, len(customClaims.Permissions))
	for _, pa := range customClaims.Permissions {
		tokenPermissions = append(tokenPermissions, pa.ToTokenPermissionAssignment())
	}
	return &token.Claims{
		SystemRoles: customClaims.SystemRoles,
		IsService:   true,
		Permissions: tokenPermissions,
	}, nil
}

// generateKey generates a random ephemeral HS256 signing key.
// Keys generated this way are not persisted and will differ between server instances and restarts,
// meaning tokens issued by one instance will not be accepted by another.
// Use LoadOrGenerateSigningKey to load a persistent key from a file that can be shared across instances.
func generateKey() (jose.SigningKey, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return jose.SigningKey{}, err
	}

	return jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       key,
	}, nil
}

// LoadOrGenerateSigningKey loads a 32-byte HS256 signing key from path.
// The file must contain exactly 64 hex characters (the output of e.g. `openssl rand -hex 32`).
// If the file does not exist, a new key is generated and saved to path with mode 0600.
// The file can be shared between servers so they all validate each other's tokens.
func LoadOrGenerateSigningKey(path string, logger *zap.Logger) (jose.SigningKey, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return jose.SigningKey{}, err
	}
	if err == nil {
		keyBytes, decodeErr := hex.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr != nil {
			return jose.SigningKey{}, fmt.Errorf("decode signing key %q: %w", path, decodeErr)
		}
		if len(keyBytes) != 32 {
			return jose.SigningKey{}, fmt.Errorf("signing key %q must contain exactly 64 hex characters (32 bytes)", path)
		}
		logger.Info("loaded shared token signing key", zap.String("path", path))
		return jose.SigningKey{Algorithm: jose.HS256, Key: keyBytes}, nil
	}

	// file doesn't exist — generate and save
	logger.Warn("token signing key file not found, generating a new key; tokens will not be accepted by other instances",
		zap.String("path", path))
	sk, err := generateKey()
	if err != nil {
		return jose.SigningKey{}, err
	}
	keyBytes, ok := sk.Key.([]byte)
	if !ok {
		return jose.SigningKey{}, fmt.Errorf("key must be []byte for HS256, got %T", sk.Key)
	}
	encoded := []byte(hex.EncodeToString(keyBytes))
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		return jose.SigningKey{}, err
	}
	return sk, nil
}
