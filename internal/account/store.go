package account

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/account/queries"
	"github.com/smart-core-os/sc-bos/internal/sqlite"
	"github.com/smart-core-os/sc-bos/internal/util/pass"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

const appID = 0x5C0501

type Store struct {
	db *sqlite.Database
}

func OpenStore(ctx context.Context, path string, logger *zap.Logger) (*Store, error) {
	db, err := sqlite.Open(ctx, path,
		sqlite.WithLogger(logger),
		sqlite.WithApplicationID(appID),
	)
	if err != nil {
		return nil, err
	}

	err = db.Migrate(ctx, schema)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,
	}, nil
}

func NewMemoryStore(logger *zap.Logger) *Store {
	db := sqlite.OpenMemory(
		sqlite.WithLogger(logger),
		sqlite.WithApplicationID(appID),
	)

	err := db.Migrate(context.Background(), schema)
	if err != nil {
		// this can only happen if the migrations are broken
		panic(err)
	}

	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Read(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.ReadTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

func (s *Store) Write(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.WriteTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

type Tx struct {
	*queries.Queries
}

func (tx *Tx) UpdateAccountPassword(ctx context.Context, accountID int64, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	account, err := tx.GetAccount(ctx, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAccountNotFound
	} else if err != nil {
		return err
	}
	if account.Type != accountpb.Account_USER_ACCOUNT.String() {
		return ErrUnexpectedPasswordUpdate
	}

	return tx.UpdateAccountPasswordHash(ctx, queries.UpdateAccountPasswordHashParams{
		AccountID:    accountID,
		PasswordHash: hash,
	})
}

func (tx *Tx) CheckAccountPassword(ctx context.Context, accountID int64, password string) error {
	password = normalisePassword(password)

	details, err := tx.GetAccountDetails(ctx, accountID)
	if err != nil {
		return err
	}
	if details.PasswordHash == nil {
		return ErrIncorrectPassword // no password set
	}
	err = pass.Compare(details.PasswordHash, []byte(password))
	if errors.Is(err, pass.ErrMismatchedHashAndPassword) {
		return ErrIncorrectPassword
	} else if err != nil {
		return err
	}

	return nil
}

func (tx *Tx) CheckClientSecret(ctx context.Context, accountID int64, clientSecret string) error {
	checkHash := hashSecret(clientSecret)
	details, err := tx.GetAccountDetails(ctx, accountID)
	if err != nil {
		return err
	}

	validHashes := make([][]byte, 0, 2)
	if len(details.PrimarySecretHash) != 0 {
		validHashes = append(validHashes, details.PrimarySecretHash)
	}
	// we require an expiry time for the secondary secret, so if that column is null, we treat it as having
	// no secondary secret at all
	if expires := details.SecondarySecretExpireTime; expires.Valid && expires.Time.After(time.Now()) && len(details.SecondarySecretHash) != 0 {
		validHashes = append(validHashes, details.SecondarySecretHash)
	}

	// no need to be constant time, as learning about prefixes of the hash doesn't help you learn the actual secret
	for _, hash := range validHashes {
		if bytes.Equal(checkHash, hash) {
			return nil
		}
	}
	return ErrIncorrectSecret
}

func (tx *Tx) ListRolesAndPermissions(ctx context.Context, params queries.ListRolesAndPermissionsParams) ([]RoleAndPermissions, error) {
	data, err := tx.Queries.ListRolesAndPermissions(ctx, params)
	if err != nil {
		return nil, err
	}

	roleAndPermissions := make([]RoleAndPermissions, len(data))
	for i, d := range data {
		roleAndPermissions[i] = RoleAndPermissions{
			Role:          d.Role,
			PermissionIDs: splitPermissions(d.Permissions),
		}
	}
	return roleAndPermissions, nil
}

type RoleAndPermissions struct {
	Role          queries.Role
	PermissionIDs []string
}

// ListRoleAssignmentsFiltered returns a page of role assignments filtered by the given field and ID.
// If page is nil, the first page is returned, and the total size is calculated.
// Otherwise, the next page is returned and the total size is obtained from the page token.
func (tx *Tx) ListRoleAssignmentsFiltered(ctx context.Context, field roleAssignmentField, filterID int64, page *PageToken, limit int64) (RoleAssignmentsPage, error) {
	var (
		afterID         int64
		totalSize       int64
		roleAssignments []queries.RoleAssignment
		err             error
		calculateSize   = true
	)
	if page != nil {
		totalSize = int64(page.TotalSize)
		afterID = page.LastId
		calculateSize = false
	}

	switch field {
	case roleAssignmentAccountID:
		roleAssignments, err = tx.ListRoleAssignmentsForAccount(ctx, queries.ListRoleAssignmentsForAccountParams{
			AfterID:   afterID,
			Limit:     limit + 1, // fetch one extra to determine if there are more
			AccountID: filterID,
		})
	case roleAssignmentRoleID:
		roleAssignments, err = tx.ListRoleAssignmentsForRole(ctx, queries.ListRoleAssignmentsForRoleParams{
			AfterID: afterID,
			Limit:   limit + 1,
			RoleID:  filterID,
		})
	case roleAssignmentUnfiltered:
		roleAssignments, err = tx.ListRoleAssignments(ctx, queries.ListRoleAssignmentsParams{
			AfterID: afterID,
			Limit:   limit + 1,
		})
	default:
		return RoleAssignmentsPage{}, ErrInvalidFilter
	}
	if err != nil {
		return RoleAssignmentsPage{}, err
	}

	if calculateSize {
		switch field {
		case roleAssignmentAccountID:
			totalSize, err = tx.CountRoleAssignmentsForAccount(ctx, filterID)
		case roleAssignmentRoleID:
			totalSize, err = tx.CountRoleAssignmentsForRole(ctx, filterID)
		case roleAssignmentUnfiltered:
			totalSize, err = tx.CountRoleAssignments(ctx)
		default:
			return RoleAssignmentsPage{}, ErrInvalidFilter
		}
		if err != nil {
			return RoleAssignmentsPage{}, err
		}
	}

	more := int64(len(roleAssignments)) > limit
	var lastID int64
	if more {
		lastID = roleAssignments[limit-1].ID
		roleAssignments = roleAssignments[:limit]
	}
	if totalSize > math.MaxInt32 {
		// cannot represent, so omit
		totalSize = 0
	}
	return RoleAssignmentsPage{
		RoleAssignments: roleAssignments,
		More:            more,
		LastID:          lastID,
		TotalSize:       int32(totalSize),
	}, nil
}

type RoleAssignmentsPage struct {
	RoleAssignments []queries.RoleAssignment
	More            bool
	LastID          int64 // if More is true, contains the last ID in the page
	TotalSize       int32
}

func genSecret() (string, error) {
	secretBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretBytes)
	if err != nil {
		return "", err
	}

	encoding := base64.URLEncoding
	encoded := make([]byte, encoding.EncodedLen(len(secretBytes)))
	encoding.Encode(encoded, secretBytes)

	return string(encoded), nil
}

func hashPassword(password string) ([]byte, error) {
	password = normalisePassword(password)
	if !permitPassword(password) {
		return nil, ErrInvalidPassword
	}
	return pass.Hash([]byte(password))
}

func hashSecret(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}
