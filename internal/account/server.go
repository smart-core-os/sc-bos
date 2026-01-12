package account

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"regexp"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/account/queries"
	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	"github.com/smart-core-os/sc-bos/internal/sqlite"
	"github.com/smart-core-os/sc-bos/internal/util/pass"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

var (
	ErrDatabase                 = status.Error(codes.Internal, "database internal error")
	ErrAccountNotFound          = status.Error(codes.NotFound, "account not found")
	ErrRoleNotFound             = status.Error(codes.NotFound, "role not found")
	ErrRoleAssignmentNotFound   = status.Error(codes.NotFound, "role assignment not found")
	ErrPermissionNotFound       = status.Error(codes.NotFound, "permission not found")
	ErrInvalidAccountType       = status.Error(codes.InvalidArgument, "invalid account type")
	ErrMissingUserDetails       = status.Error(codes.InvalidArgument, "required user details not supplied")
	ErrIncorrectDetails         = status.Error(codes.InvalidArgument, "wrong details variant supplied for account type")
	ErrMissingUsername          = status.Error(codes.InvalidArgument, "user account requires username")
	ErrMissingDisplayName       = status.Error(codes.InvalidArgument, "account requires display name")
	ErrUnexpectedUsernameUpdate = status.Error(codes.FailedPrecondition, "service account cannot have username")
	ErrUnexpectedSecretRotate   = status.Error(codes.FailedPrecondition, "only service account have client secrets")
	ErrUsernameExists           = status.Error(codes.AlreadyExists, "username already exists")
	ErrRoleAssignmentExists     = status.Error(codes.AlreadyExists, "role assignment already exists")
	ErrRoleDisplayNameExists    = status.Error(codes.AlreadyExists, "role with this display name already exists")
	ErrUnexpectedPasswordCreate = status.Error(codes.InvalidArgument, "only user account can have password")
	ErrUnexpectedPasswordUpdate = status.Error(codes.FailedPrecondition, "only user account can have password")
	ErrInvalidUsername          = status.Error(codes.InvalidArgument, "invalid username")
	ErrInvalidDisplayName       = status.Error(codes.InvalidArgument, "invalid display name")
	ErrInvalidDescription       = status.Error(codes.InvalidArgument, "invalid description")
	ErrInvalidPassword          = status.Error(codes.InvalidArgument, "password does not comply with policy")
	ErrInvalidResourceType      = status.Error(codes.InvalidArgument, "invalid scope resource type")
	ErrInvalidResource          = status.Error(codes.InvalidArgument, "invalid scope resource")
	ErrIncorrectPassword        = status.Error(codes.FailedPrecondition, "incorrect password")
	ErrIncorrectSecret          = status.Error(codes.FailedPrecondition, "incorrect secret")
	ErrInvalidPageToken         = status.Error(codes.InvalidArgument, "invalid page token")
	ErrInvalidFilter            = status.Error(codes.InvalidArgument, "invalid filter")
	ErrRoleInUse                = status.Error(codes.FailedPrecondition, "role is in use")
	ErrRoleProtected            = status.Error(codes.FailedPrecondition, "role is protected and cannot be updated or deleted")
	ErrRoleScopedAssignment     = status.Error(codes.FailedPrecondition, "role cannot be used in a scoped assignment")
	ErrResourceMissing          = status.Error(codes.InvalidArgument, "resource to create/update not supplied")
	ErrGenerateSecret           = status.Error(codes.Internal, "failed to generate secret")
)

type Server struct {
	accountpb.UnimplementedAccountApiServer
	accountpb.UnimplementedAccountInfoServer
	store  *Store
	logger *zap.Logger
}

func NewServer(store *Store, logger *zap.Logger) *Server {
	return &Server{store: store, logger: logger}
}

// GetAccount returns a single account by ID.
func (s *Server) GetAccount(ctx context.Context, req *accountpb.GetAccountRequest) (*accountpb.Account, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrAccountNotFound
	}

	var dbAccount queries.AccountDetail
	err := s.store.Read(ctx, func(tx *Tx) error {
		var err error
		dbAccount, err = tx.GetAccountDetails(ctx, id)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	} else if err != nil {
		return nil, s.processError(err, zap.String("rpc", "GetAccount"), zap.String("id", req.Id))
	}

	return accountToProto(dbAccount, ""), nil
}

func (s *Server) ListAccounts(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
	pageSize := resolvePageSize(req.PageSize)

	res := &accountpb.ListAccountsResponse{
		TotalSize: -1, // sentinel, indicates we need to calculate this
	}
	var afterID int64 = 0
	if req.PageToken != "" {
		// this RPC does not support filtering
		pageToken, err := parsePageToken(req.PageToken, "")
		if err != nil {
			return nil, ErrInvalidPageToken
		}
		afterID = pageToken.LastId
		res.TotalSize = pageToken.TotalSize
	}

	err := s.store.Read(ctx, func(tx *Tx) error {
		page, err := tx.ListAccountDetails(ctx, queries.ListAccountDetailsParams{
			AfterID: afterID,
			Limit:   pageSize + 1, // fetch one extra to determine if there are more
		})
		if err != nil {
			return err
		}

		if res.TotalSize < 0 {
			count, err := tx.CountAccounts(ctx)
			if err != nil {
				return err
			}
			if count > math.MaxInt32 {
				res.TotalSize = 0
			} else {
				res.TotalSize = int32(count)
			}
		}

		if int64(len(page)) > pageSize {
			last := page[pageSize-1] // last element that we are going to send
			res.NextPageToken = encodePageToken(&PageToken{
				LastId:    last.ID,
				TotalSize: res.TotalSize,
			})
			page = page[:pageSize]
		}
		for _, dbAccount := range page {
			res.Accounts = append(res.Accounts, accountToProto(dbAccount, ""))
		}
		return nil
	})
	if err != nil {
		return nil, s.processError(err,
			zap.String("rpc", "ListAccounts"),
			zap.String("pageToken", req.PageToken),
			zap.Int32("pageSize", req.PageSize),
		)
	}

	return res, nil
}

func (s *Server) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.Account, error) {
	account := req.Account
	if account == nil {
		return nil, status.Error(codes.InvalidArgument, "account is required")
	}
	if account.DisplayName == "" {
		return nil, ErrMissingDisplayName
	} else if !validateDisplayName(account.DisplayName) {
		return nil, ErrInvalidDisplayName
	}
	if !validateDescription(account.Description) {
		return nil, ErrInvalidDescription
	}

	var username string
	switch account.Type {
	case accountpb.Account_USER_ACCOUNT:
		if account.Details == nil {
			return nil, ErrMissingUserDetails
		}
		details := account.GetUserDetails()
		if details == nil {
			return nil, ErrIncorrectDetails
		}
		username = details.Username
		if username == "" {
			return nil, ErrMissingUsername
		}
		if !validateUsername(username) {
			return nil, ErrInvalidUsername
		}
	case accountpb.Account_SERVICE_ACCOUNT:
		// allow not providing a details value for service accounts because there are no required fields
		// but still check that no other type of details is provided
		if account.Details != nil && account.GetServiceDetails() == nil {
			return nil, ErrIncorrectDetails
		}
		if req.Password != "" {
			return nil, ErrUnexpectedPasswordCreate
		}
	default:
		return nil, ErrInvalidAccountType
	}

	var detail queries.AccountDetail
	var secret string
	err := s.store.Write(ctx, func(tx *Tx) error {
		var description sql.NullString
		if req.Account.Description != "" {
			description = sql.NullString{Valid: true, String: req.Account.Description}
		}

		dbAccount, err := tx.CreateAccount(ctx, queries.CreateAccountParams{
			DisplayName: account.DisplayName,
			Description: description,
			Type:        account.Type.String(),
		})
		if err != nil {
			return err
		}
		detail = queries.AccountDetail{
			ID:          dbAccount.ID,
			DisplayName: dbAccount.DisplayName,
			Description: dbAccount.Description,
			Type:        dbAccount.Type,
			CreateTime:  dbAccount.CreateTime,
		}

		switch req.Account.Type {
		case accountpb.Account_USER_ACCOUNT:
			var passwordHash []byte
			if req.Password != "" {
				passwordHash, err = hashPassword(req.Password)
				if err != nil {
					return err
				}
			}

			var userAccount queries.UserAccount
			userAccount, err = tx.CreateUserAccount(ctx, queries.CreateUserAccountParams{
				AccountID:    dbAccount.ID,
				Username:     username,
				PasswordHash: passwordHash,
			})
			if sqlite.IsUniqueConstraintError(err) {
				return ErrUsernameExists
			} else if err != nil {
				return err
			}
			detail.Username = sql.NullString{Valid: true, String: userAccount.Username}
			detail.PasswordHash = userAccount.PasswordHash
		case accountpb.Account_SERVICE_ACCOUNT:
			secret, err = genSecret()
			if err != nil {
				s.logger.Error("failed to generate secret on account create", zap.Error(err))
				return ErrGenerateSecret
			}
			hash := hashSecret(secret)

			var serviceAccount queries.ServiceAccount
			serviceAccount, err = tx.CreateServiceAccount(ctx, queries.CreateServiceAccountParams{
				AccountID:         dbAccount.ID,
				PrimarySecretHash: hash,
			})
			if err != nil {
				return err
			}
			detail.PrimarySecretHash = serviceAccount.PrimarySecretHash
		default:
			return ErrInvalidAccountType
		}
		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "CreateAccount"))
	}

	return accountToProto(detail, secret), nil
}

func (s *Server) UpdateAccount(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.Account, error) {
	const (
		fieldDisplayName            = "display_name"
		fieldDescription            = "description"
		fieldUserDetails            = "user_details"
		fieldUserDetailsUsername    = "user_details.username"
		fieldCreateTime             = "create_time"
		fieldServiceDetailsClientID = "service_details.client_id"
	)
	// ignore output-only fields in masks, as per AIP-203
	mask, err := resolveMask(req.Account, req.UpdateMask, fieldCreateTime, fieldServiceDetailsClientID)
	if err != nil {
		return nil, err
	}

	if req.Account == nil {
		return nil, status.Error(codes.InvalidArgument, "account is required")
	}

	id, ok := parseID(req.Account.Id)
	if !ok {
		return nil, ErrAccountNotFound
	}

	var account queries.AccountDetail
	err = s.store.Write(ctx, func(tx *Tx) error {
		var err error
		account, err = tx.GetAccountDetails(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		} else if err != nil {
			return err
		}

		var (
			updateUsername    bool
			updateDisplayName bool
			updateDescription bool
		)
		fields, err := fieldsToUpdate(accountToProto(account, ""), req.Account, mask)
		if err != nil {
			return err
		}
		for _, field := range fields {
			switch field {
			case fieldDisplayName:
				updateDisplayName = true
			case fieldUserDetails, fieldUserDetailsUsername:
				updateUsername = true
			case fieldDescription:
				updateDescription = true
			default:
				return status.Errorf(codes.InvalidArgument, "field %q unsupported for update", field)
			}
		}

		if updateDisplayName {
			if !validateDisplayName(req.Account.DisplayName) {
				return ErrMissingDisplayName
			}
			err = tx.UpdateAccountDisplayName(ctx, queries.UpdateAccountDisplayNameParams{
				ID:          id,
				DisplayName: req.Account.DisplayName,
			})
			if err != nil {
				return err
			}
			account.DisplayName = req.Account.DisplayName
		}

		// only user accounts can have usernames
		usernameAllowed := account.Type == accountpb.Account_USER_ACCOUNT.String()
		if updateUsername && !usernameAllowed {
			return ErrUnexpectedUsernameUpdate
		}

		if updateUsername {
			username := req.Account.GetUserDetails().GetUsername()
			if !validateUsername(username) {
				return ErrInvalidUsername
			}
			err = tx.UpdateAccountUsername(ctx, queries.UpdateAccountUsernameParams{
				AccountID: id,
				Username:  username,
			})
			if sqlite.IsUniqueConstraintError(err) {
				return ErrUsernameExists
			} else if err != nil {
				return err
			}
			account.Username = sql.NullString{Valid: true, String: username}
		}

		if updateDescription {
			if !validateDescription(req.Account.Description) {
				return ErrInvalidResource
			}
			var description sql.NullString
			if req.Account.Description != "" {
				description = sql.NullString{Valid: true, String: req.Account.Description}
			}

			err = tx.UpdateAccountDescription(ctx, queries.UpdateAccountDescriptionParams{
				ID:          id,
				Description: description,
			})
			if err != nil {
				return err
			}
			account.Description = description
		}

		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "UpdateAccount"), zap.String("id", req.Account.Id))
	}

	return accountToProto(account, ""), nil
}

func (s *Server) DeleteAccount(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrAccountNotFound
	}

	var deleted bool
	err := s.store.Write(ctx, func(tx *Tx) error {
		rowsDeleted, err := tx.DeleteAccount(ctx, id)
		if err != nil {
			return err
		}
		deleted = rowsDeleted > 0
		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "DeleteAccount"), zap.String("id", req.Id))
	}
	if !deleted && !req.AllowMissing {
		return nil, ErrAccountNotFound
	}
	return &accountpb.DeleteAccountResponse{}, nil
}

func (s *Server) UpdateAccountPassword(ctx context.Context, req *accountpb.UpdateAccountPasswordRequest) (*accountpb.UpdateAccountPasswordResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if !permitPassword(req.NewPassword) {
		return nil, ErrInvalidPassword
	}

	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrAccountNotFound
	}

	err := s.store.Write(ctx, func(tx *Tx) error {
		if req.OldPassword != "" {
			_, err := tx.GetAccount(ctx, id)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAccountNotFound
			} else if err != nil {
				return err
			}

			err = tx.CheckAccountPassword(ctx, id, req.OldPassword)
			if errors.Is(err, sql.ErrNoRows) {
				// account has no password saved
				return ErrIncorrectPassword
			} else if errors.Is(err, pass.ErrMismatchedHashAndPassword) {
				return ErrIncorrectPassword
			} else if err != nil {
				return err
			}
		}

		return tx.UpdateAccountPassword(ctx, id, req.NewPassword)
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "UpdateAccountPassword"), zap.String("id", req.Id))
	}

	return &accountpb.UpdateAccountPasswordResponse{}, nil
}

func (s *Server) RotateAccountClientSecret(ctx context.Context, req *accountpb.RotateAccountClientSecretRequest) (*accountpb.RotateAccountClientSecretResponse, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrAccountNotFound
	}

	secret, err := genSecret()
	if err != nil {
		s.logger.Error("failed to generate secret for rotation", zap.Error(err))
		return nil, ErrGenerateSecret
	}

	err = s.store.Write(ctx, func(tx *Tx) error {
		account, err := tx.GetAccount(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		} else if err != nil {
			return err
		}
		if account.Type != accountpb.Account_SERVICE_ACCOUNT.String() {
			return ErrUnexpectedSecretRotate
		}

		var secondaryExpireTime sql.NullTime
		if req.PreviousSecretExpireTime != nil {
			secondaryExpireTime = sql.NullTime{Valid: true, Time: req.PreviousSecretExpireTime.AsTime()}
		}
		err = tx.RotateServiceAccountSecret(ctx, queries.RotateServiceAccountSecretParams{
			AccountID:                 id,
			PrimarySecretHash:         hashSecret(secret),
			SecondarySecretExpireTime: secondaryExpireTime,
		})
		return err
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "RotateAccountClientSecret"), zap.String("id", req.Id))
	}

	return &accountpb.RotateAccountClientSecretResponse{ClientSecret: secret}, nil
}

func (s *Server) GetRole(ctx context.Context, req *accountpb.GetRoleRequest) (*accountpb.Role, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrRoleNotFound
	}

	var (
		role        queries.Role
		permissions []string
	)
	err := s.store.Read(ctx, func(tx *Tx) error {
		var err error
		role, err = tx.GetRole(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		} else if err != nil {
			return err
		}
		permissions, err = tx.ListRolePermissions(ctx, id)
		return err
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "GetRole"), zap.String("id", req.Id))
	}

	return roleToProto(role, permissions), nil
}

func (s *Server) ListRoles(ctx context.Context, req *accountpb.ListRolesRequest) (*accountpb.ListRolesResponse, error) {
	pageSize := resolvePageSize(req.PageSize)

	res := &accountpb.ListRolesResponse{
		TotalSize: -1, // sentinel, indicates we need to calculate this
	}
	var afterID int64
	if req.PageToken != "" {
		token, err := parsePageToken(req.PageToken, "")
		if err != nil {
			return nil, ErrInvalidPageToken
		}
		afterID = token.LastId
		res.TotalSize = token.TotalSize
	}

	err := s.store.Read(ctx, func(tx *Tx) error {
		page, err := tx.ListRolesAndPermissions(ctx, queries.ListRolesAndPermissionsParams{
			AfterID: afterID,
			Limit:   pageSize + 1, // fetch one extra to determine if there are more
		})
		if err != nil {
			return err
		}

		if res.TotalSize < 0 {
			count, err := tx.CountRoles(ctx)
			if err != nil {
				return err
			}
			if count > math.MaxInt32 {
				res.TotalSize = 0
			} else {
				res.TotalSize = int32(count)
			}
		}

		if int64(len(page)) > pageSize {
			last := page[pageSize-1] // last element that we are going to send
			res.NextPageToken = encodePageToken(&PageToken{
				LastId:    last.Role.ID,
				TotalSize: res.TotalSize,
			})
			page = page[:pageSize]
		}

		for _, role := range page {
			res.Roles = append(res.Roles, roleToProto(role.Role, role.PermissionIDs))
		}

		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "ListRoles"), zap.String("pageToken", req.PageToken))
	}

	return res, nil
}

func (s *Server) CreateRole(ctx context.Context, req *accountpb.CreateRoleRequest) (*accountpb.Role, error) {
	if req.Role == nil {
		return nil, ErrResourceMissing
	}

	if !validateDisplayName(req.Role.DisplayName) {
		return nil, ErrInvalidDisplayName
	}
	if !validateDescription(req.Role.Description) {
		return nil, ErrInvalidDescription
	}

	var (
		role        queries.Role
		permissions []string
	)
	err := s.store.Write(ctx, func(tx *Tx) error {
		var err error
		params := queries.CreateRoleParams{
			DisplayName: req.Role.DisplayName,
		}
		if req.Role.Description != "" {
			params.Description = sql.NullString{Valid: true, String: req.Role.Description}
		}
		role, err = tx.CreateRole(ctx, params)
		if sqlite.IsUniqueConstraintError(err) {
			return ErrRoleDisplayNameExists
		} else if err != nil {
			return err
		}

		for _, perm := range req.Role.PermissionIds {
			err = tx.AddRolePermission(ctx, queries.AddRolePermissionParams{
				RoleID:     role.ID,
				Permission: perm,
			})
			if err != nil {
				return err
			}
		}

		// re-fetch the permissions, as they have been reordered and deduplicated
		// when added to the database
		permissions, err = tx.ListRolePermissions(ctx, role.ID)
		return err
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "CreateRole"))
	}

	return roleToProto(role, permissions), nil
}

func (s *Server) UpdateRole(ctx context.Context, req *accountpb.UpdateRoleRequest) (*accountpb.Role, error) {
	if req.Role == nil {
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}

	const (
		fieldDisplayName   = "display_name"
		fieldPermissionIDs = "permission_ids"
		fieldDescription   = "description"
	)
	mask, err := resolveMask(req.Role, req.UpdateMask)
	if err != nil {
		return nil, err
	}

	id, ok := parseID(req.Role.Id)
	if !ok {
		return nil, ErrRoleNotFound
	}

	var (
		role        queries.Role
		permissions []string
	)
	err = s.store.Write(ctx, func(tx *Tx) error {
		var err error
		role, err = tx.GetRole(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		} else if err != nil {
			return err
		}
		if role.Protected {
			return ErrRoleProtected
		}
		permissions, err = tx.ListRolePermissions(ctx, id)
		if err != nil {
			return err
		}

		// decide which fields to update
		var (
			updateDisplayName bool
			updatePermissions bool
			updateDescription bool
		)
		fields, err := fieldsToUpdate(roleToProto(role, permissions), req.Role, mask)
		if err != nil {
			return err
		}
		for _, field := range fields {
			switch field {
			case fieldDisplayName:
				updateDisplayName = true
			case fieldPermissionIDs:
				updatePermissions = true
			case fieldDescription:
				updateDescription = true
			default:
				return status.Errorf(codes.InvalidArgument, "field %q unsupported for update", field)
			}
		}

		if updateDisplayName {
			if !validateDisplayName(req.Role.DisplayName) {
				return ErrInvalidDisplayName
			}

			_, err = tx.UpdateRoleDisplayName(ctx, queries.UpdateRoleDisplayNameParams{
				ID:          id,
				DisplayName: req.Role.DisplayName,
			})
			if sqlite.IsUniqueConstraintError(err) {
				return ErrRoleDisplayNameExists
			} else if err != nil {
				return err
			}
			role.DisplayName = req.Role.DisplayName
		}

		if updateDescription {
			if !validateDescription(req.Role.Description) {
				return ErrInvalidDescription
			}

			var value sql.NullString
			if req.Role.Description != "" {
				value = sql.NullString{String: req.Role.Description, Valid: true}
			}
			_, err = tx.UpdateRoleDescription(ctx, queries.UpdateRoleDescriptionParams{
				ID:          id,
				Description: value,
			})
			if err != nil {
				return err
			}
			role.Description = value
		}

		if updatePermissions {
			// clear existing permissions
			_, err = tx.ClearRolePermissions(ctx, id)
			if err != nil {
				return err
			}

			// add new permissions
			for _, perm := range req.Role.PermissionIds {
				err = tx.AddRolePermission(ctx, queries.AddRolePermissionParams{
					RoleID:     id,
					Permission: perm,
				})
				if err != nil {
					return err
				}
			}

			permissions, err = tx.ListRolePermissions(ctx, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "UpdateRole"), zap.String("id", req.Role.Id))
	}

	return roleToProto(role, permissions), nil
}

func (s *Server) DeleteRole(ctx context.Context, req *accountpb.DeleteRoleRequest) (*accountpb.DeleteRoleResponse, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrRoleNotFound
	}

	var deleted bool
	err := s.store.Write(ctx, func(tx *Tx) error {
		rowsDeleted, err := tx.DeleteRole(ctx, id)
		if sqlite.IsForeignKeyError(err) {
			return ErrRoleInUse
		} else if err != nil {
			s.logger.Error("failed to delete role", zap.Error(err), zap.String("id", req.Id))
			return ErrDatabase
		}

		if rowsDeleted > 0 {
			// role was deleted successfully
			deleted = true
			return nil
		}

		// role was not deleted, because either
		// 1. it does not exist, or
		// 2. it is protected and cannot be deleted
		// detect which applies
		_, err = tx.GetRole(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		} else {
			return ErrRoleProtected
		}
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "DeleteRole"), zap.String("id", req.Id))
	}
	if !deleted && !req.AllowMissing {
		return nil, ErrRoleNotFound
	}

	return &accountpb.DeleteRoleResponse{}, nil
}

func (s *Server) GetRoleAssignment(ctx context.Context, req *accountpb.GetRoleAssignmentRequest) (*accountpb.RoleAssignment, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrRoleAssignmentNotFound
	}

	var assignment queries.RoleAssignment
	err := s.store.Read(ctx, func(tx *Tx) error {
		var err error
		assignment, err = tx.GetRoleAssignment(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleAssignmentNotFound
		}
		return err
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "GetRoleAssignment"), zap.String("id", req.Id))
	}

	return roleAssignmentToProto(assignment), nil
}

func (s *Server) ListRoleAssignments(ctx context.Context, req *accountpb.ListRoleAssignmentsRequest) (*accountpb.ListRoleAssignmentsResponse, error) {
	pageSize := resolvePageSize(req.PageSize)

	filterField, filterID, ok := parseRoleAssignmentFilter(req.Filter)
	if !ok {
		return nil, ErrInvalidFilter
	}

	var token *PageToken
	if req.PageToken != "" {
		var err error
		token, err = parsePageToken(req.PageToken, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	var (
		res = &accountpb.ListRoleAssignmentsResponse{}
		err error
	)
	err = s.store.Read(ctx, func(tx *Tx) error {
		page, err := tx.ListRoleAssignmentsFiltered(ctx, filterField, filterID, token, pageSize)
		if err != nil {
			return err
		}
		res.TotalSize = page.TotalSize
		if page.More {
			res.NextPageToken = encodePageToken(&PageToken{
				LastId:    page.LastID,
				TotalSize: page.TotalSize,
				Filter:    req.Filter,
			})
		}

		for _, assignment := range page.RoleAssignments {
			res.RoleAssignments = append(res.RoleAssignments, roleAssignmentToProto(assignment))
		}
		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "ListRoleAssignments"), zap.String("filter", req.Filter))
	}

	return res, nil
}

func (s *Server) CreateRoleAssignment(ctx context.Context, req *accountpb.CreateRoleAssignmentRequest) (*accountpb.RoleAssignment, error) {
	if req.RoleAssignment == nil {
		return nil, ErrResourceMissing
	}

	accountID, ok := parseID(req.RoleAssignment.AccountId)
	if !ok {
		return nil, ErrAccountNotFound
	}
	roleID, ok := parseID(req.RoleAssignment.RoleId)
	if !ok {
		return nil, ErrRoleNotFound
	}

	var (
		scopeType, scopeResource sql.NullString
	)
	if scope := req.RoleAssignment.Scope; scope != nil {
		if !validateResourceType(scope.ResourceType) {
			return nil, ErrInvalidResourceType
		}
		if !validateResource(scope.Resource) {
			return nil, ErrInvalidResource
		}

		scopeType = sql.NullString{Valid: true, String: scope.ResourceType.String()}
		scopeResource = sql.NullString{Valid: true, String: scope.Resource}
	}

	var assignment queries.RoleAssignment
	err := s.store.Write(ctx, func(tx *Tx) error {
		role, err := tx.GetRole(ctx, roleID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		if role.LegacyRole.Valid && scopeType.Valid {
			return ErrRoleScopedAssignment
		}

		assignment, err = tx.CreateRoleAssignment(ctx, queries.CreateRoleAssignmentParams{
			AccountID:     accountID,
			RoleID:        roleID,
			ScopeKind:     scopeType,
			ScopeResource: scopeResource,
		})
		if sqlite.IsUniqueConstraintError(err) {
			return ErrRoleAssignmentExists
		} else if sqlite.IsForeignKeyError(err) {
			// already checked that role exists, so this must be a non-existing account
			return ErrAccountNotFound
		}
		return err
	})
	if err != nil {
		return nil, s.processError(err,
			zap.String("rpc", "CreateRoleAssignment"),
			zap.String("accountId", req.RoleAssignment.AccountId),
			zap.String("roleId", req.RoleAssignment.RoleId),
		)
	}

	return roleAssignmentToProto(assignment), nil
}

func (s *Server) DeleteRoleAssignment(ctx context.Context, req *accountpb.DeleteRoleAssignmentRequest) (*accountpb.DeleteRoleAssignmentResponse, error) {
	id, ok := parseID(req.Id)
	if !ok {
		return nil, ErrRoleAssignmentNotFound
	}

	var deleted bool
	err := s.store.Write(ctx, func(tx *Tx) error {
		rowsDeleted, err := tx.DeleteRoleAssignment(ctx, id)
		if err != nil {
			return err
		}
		deleted = rowsDeleted > 0
		return nil
	})
	if err != nil {
		return nil, s.processError(err, zap.String("rpc", "DeleteRoleAssignment"), zap.String("id", req.Id))
	}
	if !deleted && !req.AllowMissing {
		return nil, ErrRoleAssignmentNotFound
	}

	return &accountpb.DeleteRoleAssignmentResponse{}, nil
}

func (s *Server) GetPermission(_ context.Context, req *accountpb.GetPermissionRequest) (*accountpb.Permission, error) {
	details, ok := permission.GetDetails(permission.ID(req.Id))
	if !ok {
		return nil, ErrPermissionNotFound
	}

	return &accountpb.Permission{
		Id:          string(details.ID),
		DisplayName: details.DisplayName,
		Description: details.Description,
	}, nil
}

func (s *Server) ListPermissions(ctx context.Context, req *accountpb.ListPermissionsRequest) (*accountpb.ListPermissionsResponse, error) {
	// We currently have few permissions, so we can return them all at once without pagination.
	// If the number becomes significantly larger, we should implement pagination.
	if req.PageToken != "" {
		return nil, ErrInvalidPageToken
	}

	allPerms := permission.All()
	converted := make([]*accountpb.Permission, 0, len(allPerms))
	for _, details := range allPerms {
		converted = append(converted, &accountpb.Permission{
			Id:          string(details.ID),
			DisplayName: details.DisplayName,
			Description: details.Description,
		})
	}
	return &accountpb.ListPermissionsResponse{
		Permissions: converted,
	}, nil
}

func (s *Server) GetAccountLimits(ctx context.Context, req *accountpb.GetAccountLimitsRequest) (*accountpb.AccountLimits, error) {
	return &accountpb.AccountLimits{
		Username: &accountpb.AccountLimits_Field{
			MinLength: minUsernameLength,
			MaxLength: maxUsernameLength,
		},
		Password: &accountpb.AccountLimits_Field{
			MinLength: minPasswordLength,
			MaxLength: maxPasswordLength,
		},
		DisplayName: &accountpb.AccountLimits_Field{
			MinLength: minDisplayNameLength,
			MaxLength: maxDisplayNameLength,
		},
		Description: &accountpb.AccountLimits_Field{
			MinLength: minDescriptionLength,
			MaxLength: maxDescriptionLength,
		},
	}, nil
}

func (s *Server) processError(err error, fields ...zap.Field) error {
	logger := s.logger.With(fields...)
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		logger.Debug("request cancelled or timed out", zap.Error(err))
		return err
	case status.Code(err) != codes.Unknown:
		return err
	default:
		s.logger.Error("unexpected account service internal error", zap.Error(err))
		return ErrDatabase
	}
}

func resolvePageSize(pageSize int32) int64 {
	if pageSize == 0 {
		return defaultPageSize
	}
	if pageSize < minPageSize {
		return minPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return int64(pageSize)
}

const (
	minPageSize     = 1
	maxPageSize     = 100
	defaultPageSize = 30
)

var filterRoleAssignmentsRegexp = regexp.MustCompile(`^ *(account_id|role_id) *= *(\d+) *$`)

type roleAssignmentField string

const (
	roleAssignmentAccountID  roleAssignmentField = "account_id"
	roleAssignmentRoleID     roleAssignmentField = "role_id"
	roleAssignmentUnfiltered roleAssignmentField = ""
)

// parses a filter string like:
//   - "account_id=123" - matches role assignment where account_id=123
//   - "role_id=456"    - matches role assignment where role_id=456
//   - ""               - matches all role assignments (the empty string is a valid filter)
func parseRoleAssignmentFilter(filter string) (field roleAssignmentField, id int64, ok bool) {
	if filter == "" {
		return roleAssignmentUnfiltered, 0, true
	}
	m := filterRoleAssignmentsRegexp.FindStringSubmatch(filter)
	if m == nil {
		return "", 0, false
	}
	id, ok = parseID(m[2])
	if !ok {
		return "", 0, false
	}
	return roleAssignmentField(m[1]), id, true
}
