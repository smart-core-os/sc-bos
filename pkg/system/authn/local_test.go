package authn

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/internal/auth/accesstoken"
	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	"github.com/smart-core-os/sc-bos/internal/util/pass"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

func TestLocalUserVerifier_Verify(t *testing.T) {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	accountStore := account.NewMemoryStore(logger)
	accountServer := account.NewServer(accountStore, logger)
	verifier := newLocalUserVerifier(accountStore)

	// Find the auto-created roles by their legacy role names
	systemRoleIDs := make(map[string]string)
	err = accountStore.Read(ctx, func(tx *account.Tx) error {
		for _, roleName := range []string{"admin", "commissioner", "operator", "viewer", "super-admin"} {
			roles, err := tx.ListRolesWithLegacyRole(ctx, sql.NullString{Valid: true, String: roleName})
			if err != nil {
				return err
			}
			if len(roles) > 0 {
				systemRoleIDs[roleName] = strconv.FormatInt(roles[0].ID, 10)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to find role IDs: %v", err)
	}

	// Create custom roles with specific permissions
	readOnlyRole, err := accountServer.CreateRole(ctx, &accountpb.CreateRoleRequest{
		Role: &accountpb.Role{
			DisplayName:   "Read Only Access",
			Description:   "Role with read-only permissions",
			PermissionIds: []string{string(permission.TraitRead)},
		},
	})
	if err != nil {
		t.Fatalf("failed to create read-only role: %v", err)
	}

	writeRole, err := accountServer.CreateRole(ctx, &accountpb.CreateRoleRequest{
		Role: &accountpb.Role{
			DisplayName:   "Write Only Access",
			Description:   "Role with write-only permissions",
			PermissionIds: []string{string(permission.TraitWrite)},
		},
	})
	if err != nil {
		t.Fatalf("failed to create write-only role: %v", err)
	}

	bothRole, err := accountServer.CreateRole(ctx, &accountpb.CreateRoleRequest{
		Role: &accountpb.Role{
			DisplayName:   "Full Access",
			Description:   "Role with full access permissions",
			PermissionIds: []string{string(permission.TraitRead), string(permission.TraitWrite)},
		},
	})
	if err != nil {
		t.Fatalf("failed to create full access role: %v", err)
	}

	// Create test users
	type roleAssignment struct {
		roleID   string
		resource string
		resType  accountpb.RoleAssignment_ResourceType
	}
	type testUser struct {
		username string
		password string
		title    string
		roles    []roleAssignment
	}
	testUsers := []testUser{
		{
			username: "user1",
			password: "Password123456",
			title:    "User 1 - Basic",
			roles: []roleAssignment{
				{roleID: systemRoleIDs["admin"]},
			},
		},
		{
			username: "multi-role-user",
			password: "SecurePass9876",
			title:    "User with Multiple Roles",
			roles: []roleAssignment{
				{roleID: systemRoleIDs["admin"]},
				{roleID: systemRoleIDs["commissioner"]},
				{roleID: systemRoleIDs["operator"]},
			},
		},
		{
			username: "invalid-role-user",
			password: "InvalidRole1234",
			title:    "User with Invalid Role",
			roles: []roleAssignment{
				{roleID: "unknown-role"},
			},
		},
		{
			username: "no-role-user",
			password: "NoRoleUser5678",
			title:    "User with No Roles",
			roles:    []roleAssignment{},
		},
		{
			username: "no-password-user",
			password: "", // Empty password
			title:    "User with No Password",
			roles: []roleAssignment{
				{roleID: systemRoleIDs["viewer"]},
			},
		},
		{
			username: "super-admin",
			password: "SuperAdmin123456!",
			title:    "Super Admin User",
			roles: []roleAssignment{
				{roleID: systemRoleIDs["super-admin"]},
			},
		},
		{
			username: "read-only-user",
			password: "ReadOnlyPass123",
			title:    "User with Read-Only Role",
			roles: []roleAssignment{
				{roleID: readOnlyRole.Id},
			},
		},
		{
			username: "write-only-user",
			password: "WriteOnlyPass123",
			title:    "User with Write-Only Role",
			roles: []roleAssignment{
				{roleID: writeRole.Id},
			},
		},
		{
			username: "both-roles-user",
			password: "FullAccessPass123",
			title:    "User with Full Access Role",
			roles: []roleAssignment{
				{roleID: bothRole.Id},
			},
		},
		{
			username: "scoped-permissions-user",
			password: "ScopedPass123456",
			title:    "User with Scoped Permissions",
			roles: []roleAssignment{
				{
					roleID:   readOnlyRole.Id,
					resource: "lights",
					resType:  accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				},
				{
					roleID:   writeRole.Id,
					resource: "hvac",
					resType:  accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				},
				{
					roleID:   bothRole.Id,
					resource: "doors",
					resType:  accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				},
			},
		},
		{
			username: "mixed-permissions-user",
			password: "MixedPerms123456",
			title:    "User with Mixed Scoped and Global Permissions",
			roles: []roleAssignment{
				{roleID: readOnlyRole.Id},
				{
					roleID:   writeRole.Id,
					resource: "displays",
					resType:  accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				},
			},
		},
		{
			username: "system-role-and-permissions-user",
			password: "SystemRoleAndPerms123",
			title:    "User with System Role and Permissions",
			roles: []roleAssignment{
				{roleID: systemRoleIDs["viewer"]},
				{
					roleID:   writeRole.Id,
					resource: "foo",
					resType:  accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				},
			},
		},
	}

	for _, user := range testUsers {
		created, err := accountServer.CreateAccount(ctx, &accountpb.CreateAccountRequest{
			Account: &accountpb.Account{
				DisplayName: user.title,
				Type:        accountpb.Account_USER_ACCOUNT,
				Details: &accountpb.Account_UserDetails{
					UserDetails: &accountpb.UserAccount{
						Username: user.username,
					},
				},
			},
			Password: user.password,
		})
		if err != nil {
			t.Fatalf("failed to create account %s: %v", user.username, err)
		}

		// Assign roles to the account
		for _, role := range user.roles {
			if role.roleID == "unknown-role" {
				continue // Skip invalid role
			}

			// Create role assignment request
			assignmentReq := &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: created.Id,
					RoleId:    role.roleID,
				},
			}

			// Add scope if provided
			if role.resource != "" {
				assignmentReq.RoleAssignment.Scope = &accountpb.RoleAssignment_Scope{
					ResourceType: role.resType,
					Resource:     role.resource,
				}
			}

			// Assign role
			_, err = accountServer.CreateRoleAssignment(ctx, assignmentReq)
			if err != nil {
				t.Fatalf("failed to assign role %s to account %s: %v", role.roleID, user.username, err)
			}
		}
	}

	type testCase struct {
		username      string
		password      string
		expectedError error
		expect        accesstoken.SecretData
	}
	testCases := map[string]testCase{
		"correct_password_and_roles": {
			username: "user1",
			password: "Password123456",
			expect: accesstoken.SecretData{
				Title:       "User 1 - Basic",
				SystemRoles: []string{"admin"},
			},
		},
		"wrong_password": {
			username:      "user1",
			password:      "wrong-password-123",
			expectedError: accesstoken.ErrInvalidCredentials,
		},
		"non_existent_user": {
			username:      "nonexistent",
			password:      "anything12345678",
			expectedError: accesstoken.ErrInvalidCredentials,
		},
		"multiple_valid_roles": {
			username: "multi-role-user",
			password: "SecurePass9876",
			expect: accesstoken.SecretData{
				Title:       "User with Multiple Roles",
				SystemRoles: []string{"admin", "commissioner", "operator"},
			},
		},
		"no_roles_assigned": {
			username:      "no-role-user",
			password:      "NoRoleUser5678",
			expectedError: accesstoken.ErrNoRolesAssigned,
		},
		"invalid_role_missing": {
			username:      "invalid-role-user",
			password:      "InvalidRole1234",
			expectedError: accesstoken.ErrNoRolesAssigned, // Since the role wasn't imported, user will have no valid roles
		},
		"no_password": {
			username:      "no-password-user",
			password:      "", // Try with empty password
			expectedError: accesstoken.ErrInvalidCredentials,
		},
		"multiple_passwords_second_fails": {
			username:      "multi-password-user",
			password:      "SecondPass123456",
			expectedError: accesstoken.ErrInvalidCredentials,
		},
		"super_admin_role": {
			username: "super-admin",
			password: "SuperAdmin123456!",
			expect: accesstoken.SecretData{
				Title:       "Super Admin User",
				SystemRoles: []string{"super-admin"},
			},
		},
		"read_only_user": {
			username: "read-only-user",
			password: "ReadOnlyPass123",
			expect: accesstoken.SecretData{
				Title:       "User with Read-Only Role",
				SystemRoles: []string{},
				Permissions: []token.PermissionAssignment{
					{Permission: permission.TraitRead},
				},
			},
		},
		"write_only_user": {
			username: "write-only-user",
			password: "WriteOnlyPass123",
			expect: accesstoken.SecretData{
				Title:       "User with Write-Only Role",
				SystemRoles: []string{},
				Permissions: []token.PermissionAssignment{
					{Permission: permission.TraitWrite},
				},
			},
		},
		"both_roles_user": {
			username: "both-roles-user",
			password: "FullAccessPass123",
			expect: accesstoken.SecretData{
				Title:       "User with Full Access Role",
				SystemRoles: []string{},
				Permissions: []token.PermissionAssignment{
					{Permission: permission.TraitRead},
					{Permission: permission.TraitWrite},
				},
			},
		},
		"scoped_permissions_user": {
			username: "scoped-permissions-user",
			password: "ScopedPass123456",
			expect: accesstoken.SecretData{
				Title:       "User with Scoped Permissions",
				SystemRoles: []string{},
				Permissions: []token.PermissionAssignment{
					{
						Permission:   permission.TraitRead,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "doors",
					},
					{
						Permission:   permission.TraitRead,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "lights",
					},
					{
						Permission:   permission.TraitWrite,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "doors",
					},
					{
						Permission:   permission.TraitWrite,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "hvac",
					},
				},
			},
		},
		"mixed_permissions_user": {
			username: "mixed-permissions-user",
			password: "MixedPerms123456",
			expect: accesstoken.SecretData{
				Title:       "User with Mixed Scoped and Global Permissions",
				SystemRoles: []string{},
				Permissions: []token.PermissionAssignment{
					{
						Permission: permission.TraitRead,
					},
					{
						Permission:   permission.TraitWrite,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "displays",
					},
				},
			},
		},
		"system_role_and_permissions": {
			username: "system-role-and-permissions-user",
			password: "SystemRoleAndPerms123",
			expect: accesstoken.SecretData{
				Title:       "User with System Role and Permissions",
				SystemRoles: []string{"viewer"},
				Permissions: []token.PermissionAssignment{
					{
						Permission:   permission.TraitWrite,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "foo",
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := verifier.Verify(ctx, tc.username, tc.password)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error:\n\t%v\ngot:\n\t%v", tc.expectedError, err)
			}
			if tc.expectedError == nil {
				diff := cmp.Diff(tc.expect, result,
					cmpopts.IgnoreFields(accesstoken.SecretData{}, "TenantID"),
					cmpopts.EquateEmpty(),
				)
				if diff != "" {
					t.Errorf("unexpected result (-want +got):\n%s", diff)
				}
				if result.TenantID == "" {
					t.Errorf("expected non-empty TenantID, got empty")
				}
			}
		})
	}
}

func TestLocalServiceVerifier_Verify(t *testing.T) {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	accountStore := account.NewMemoryStore(logger)
	accountServer := account.NewServer(accountStore, logger)
	verifier := newLocalServiceVerifier(accountStore)

	// can't create a role with a system role, need to find the auto-created one
	var adminRoleID string
	err = accountStore.Read(ctx, func(tx *account.Tx) error {
		roles, err := tx.ListRolesWithLegacyRole(ctx, sql.NullString{Valid: true, String: "admin"})
		if err != nil {
			return err
		}
		if len(roles) == 0 {
			return errors.New("no admin role found")
		}
		adminRoleID = strconv.FormatInt(roles[0].ID, 10)
		return nil
	})

	// role to test permissions propagation
	roleWithPermissions, err := accountServer.CreateRole(ctx, &accountpb.CreateRoleRequest{
		Role: &accountpb.Role{
			DisplayName:   "Test Role",
			PermissionIds: []string{string(permission.TraitRead), string(permission.TraitWrite)},
		},
	})
	if err != nil {
		t.Fatalf("failed to create role with permissions: %v", err)
	}

	serviceAccount, err := accountServer.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_SERVICE_ACCOUNT,
			DisplayName: "Test Service Account",
			Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create service account: %v", err)
	}

	// assign both roles to the service account
	_, err = accountServer.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: serviceAccount.Id,
			RoleId:    roleWithPermissions.Id,
			Scope: &accountpb.RoleAssignment_Scope{
				ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX,
				Resource:     "foo",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to assign role %s to service account: %v", adminRoleID, err)
	}
	_, err = accountServer.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: serviceAccount.Id,
			RoleId:    adminRoleID,
		},
	})

	type testCase struct {
		clientID      string
		clientSecret  string
		expectedError error
		expect        accesstoken.SecretData
	}

	testCases := map[string]testCase{
		"valid_service_account": {
			clientID:     serviceAccount.Id,
			clientSecret: serviceAccount.GetServiceDetails().ClientSecret,
			expect: accesstoken.SecretData{
				Title:       serviceAccount.DisplayName,
				TenantID:    serviceAccount.Id,
				SystemRoles: []string{"admin"},
				IsService:   true,
				Permissions: []token.PermissionAssignment{
					{
						Permission:   permission.TraitRead,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "foo",
					},
					{
						Permission:   permission.TraitWrite,
						Scoped:       true,
						ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
						Resource:     "foo",
					},
				},
			},
		},
		"invalid_client_id": {
			clientID:      "9999",
			clientSecret:  serviceAccount.GetServiceDetails().ClientSecret,
			expectedError: accesstoken.ErrInvalidCredentials,
		},
		"invalid_client_secret": {
			clientID:      serviceAccount.Id,
			clientSecret:  "foo",
			expectedError: accesstoken.ErrInvalidCredentials,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := verifier.Verify(ctx, tc.clientID, tc.clientSecret)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("expected error:\n\t%v\ngot:\n\t%v", tc.expectedError, err)
			}
			if tc.expectedError == nil {
				diff := cmp.Diff(tc.expect, result,
					cmpopts.IgnoreFields(accesstoken.SecretData{}, "TenantID"),
				)
				if diff != "" {
					t.Errorf("unexpected result (-want +got):\n%s", diff)
				}
				if result.TenantID == "" {
					t.Errorf("expected non-empty TenantID, got empty")
				}
			}
		})
	}
}

func testHash(t *testing.T, password string) string {
	t.Helper()
	hashed, err := pass.Hash([]byte(password))
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hashed)
}
