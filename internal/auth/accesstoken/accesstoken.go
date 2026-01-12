package accesstoken

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	jose_utils "github.com/smart-core-os/sc-bos/internal/util/jose"
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
	tok, err := jwt.ParseSigned(tokenStr, jose_utils.ConvertToNativeJose(ts.SignatureAlgorithms))
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
