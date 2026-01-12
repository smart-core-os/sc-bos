package account

import (
	"context"
	"embed"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ncruces/go-sqlite3"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

//go:embed testdata/*
var testdata embed.FS

func TestMigrations(t *testing.T) {
	type testCase struct {
		dumpFile       string
		expectAccounts []*accountpb.Account
		expectRoles    []*accountpb.Role
		expectAssigns  []*accountpb.RoleAssignment
	}

	cases := map[string]testCase{
		"schema_0001": {
			dumpFile: "testdata/from_schema_0001.sql",
			expectAccounts: []*accountpb.Account{
				{
					Id:          "2",
					DisplayName: "My Service 0",
					Description: "A service with 0 service credentials",
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					CreateTime:  timestamppb.New(time.Date(2025, 3, 19, 12, 20, 0, 0, time.UTC)),
					Details: &accountpb.Account_ServiceDetails{
						ServiceDetails: &accountpb.ServiceAccount{
							ClientId: "2",
						},
					},
				},
				{
					Id:          "3",
					DisplayName: "My Service 1",
					Description: "A service with 1 service credential",
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					CreateTime:  timestamppb.New(time.Date(2025, 3, 19, 12, 21, 0, 0, time.UTC)),
					Details: &accountpb.Account_ServiceDetails{
						ServiceDetails: &accountpb.ServiceAccount{
							ClientId: "3",
						},
					},
				},
				{
					Id:          "4",
					DisplayName: "My Service 2",
					Description: "A service with 2 service credentials",
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					CreateTime:  timestamppb.New(time.Date(2025, 3, 19, 12, 22, 0, 0, time.UTC)),
					Details: &accountpb.Account_ServiceDetails{
						ServiceDetails: &accountpb.ServiceAccount{
							ClientId:                 "4",
							PreviousSecretExpireTime: timestamppb.New(time.Date(2025, 3, 19, 12, 30, 0, 0, time.UTC)),
						},
					},
				},
				{
					Id:          "6",
					DisplayName: "User With Password",
					Type:        accountpb.Account_USER_ACCOUNT,
					CreateTime:  timestamppb.New(time.Date(2025, 3, 19, 12, 23, 0, 0, time.UTC)),
					Details: &accountpb.Account_UserDetails{
						UserDetails: &accountpb.UserAccount{
							Username:    "userpassword",
							HasPassword: true,
						},
					},
				},
				{
					Id:          "7",
					DisplayName: "User Without Password",
					Type:        accountpb.Account_USER_ACCOUNT,
					CreateTime:  timestamppb.New(time.Date(2025, 3, 19, 12, 24, 0, 0, time.UTC)),
					Details: &accountpb.Account_UserDetails{
						UserDetails: &accountpb.UserAccount{
							Username: "usernopassword",
						},
					},
				},
			},
			expectRoles: []*accountpb.Role{
				{
					Id:            "1",
					DisplayName:   "My Role",
					PermissionIds: []string{"account:read", "account:write"},
				},
				{
					Id:             "2",
					DisplayName:    "Admin",
					Description:    "Full system access (built-in role)",
					LegacyRoleName: "admin",
					Protected:      true,
				},
				{
					Id:             "3",
					DisplayName:    "Super Admin",
					Description:    "Full system access (built-in role)",
					LegacyRoleName: "super-admin",
					Protected:      true,
				},
				{
					Id:             "4",
					DisplayName:    "Commissioner",
					Description:    "Alter configurations (built-in role)",
					LegacyRoleName: "commissioner",
					Protected:      true,
				},
				{
					Id:             "5",
					DisplayName:    "Operator",
					Description:    "View data and control devices (built-in role)",
					LegacyRoleName: "operator",
					Protected:      true,
				},
				{
					Id:             "6",
					DisplayName:    "Viewer",
					Description:    "View data (built-in role)",
					LegacyRoleName: "viewer",
					Protected:      true,
				},
			},
			expectAssigns: []*accountpb.RoleAssignment{
				{
					Id:        "1",
					AccountId: "6",
					RoleId:    "1",
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_ZONE,
						Resource:     "foo",
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dump, err := testdata.ReadFile(tc.dumpFile)
			if err != nil {
				t.Fatalf("could not read dump file %q: %v", tc.dumpFile, err)
			}
			dbPath := restoreDump(t, string(dump))

			ctx := context.Background()
			logger := testLogger(t)
			store, err := OpenStore(ctx, dbPath, logger)
			if err != nil {
				t.Fatalf("OpenStore: %v", err)
			}
			server := NewServer(store, logger)

			accounts := listAccounts(t, server)
			roles := listRoles(t, server)
			roleAssignments := listRoleAssignments(t, server)

			diff := cmp.Diff(tc.expectAccounts, accounts, protocmp.Transform())
			if diff != "" {
				t.Errorf("accounts mismatch (-want +got):\n%s", diff)
			}
			diff = cmp.Diff(tc.expectRoles, roles, protocmp.Transform())
			if diff != "" {
				t.Errorf("roles mismatch (-want +got):\n%s", diff)
			}
			diff = cmp.Diff(tc.expectAssigns, roleAssignments, protocmp.Transform())
			if diff != "" {
				t.Errorf("role assignments mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func restoreDump(t *testing.T, dump string) (dbPath string) {
	t.Helper()
	dir := t.TempDir()
	dbPath = filepath.Join(dir, "db.sqlite")

	conn, err := sqlite3.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite3.Open: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("conn.Close: %v", err)
		}
	}()

	err = conn.Exec(dump)
	if err != nil {
		t.Fatalf("conn.Exec: %v", err)
	}
	return dbPath
}

func listAccounts(t *testing.T, server *Server) []*accountpb.Account {
	t.Helper()
	ctx := context.Background()
	var nextPageToken string
	var accounts []*accountpb.Account
	for {
		resp, err := server.ListAccounts(ctx, &accountpb.ListAccountsRequest{PageToken: nextPageToken})
		if err != nil {
			t.Fatalf("ListAccounts: %v", err)
		}
		accounts = append(accounts, resp.Accounts...)
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return accounts
}

func listRoles(t *testing.T, server *Server) []*accountpb.Role {
	t.Helper()
	ctx := context.Background()
	var nextPageToken string
	var roles []*accountpb.Role
	for {
		resp, err := server.ListRoles(ctx, &accountpb.ListRolesRequest{PageToken: nextPageToken})
		if err != nil {
			t.Fatalf("ListRoles: %v", err)
		}
		roles = append(roles, resp.Roles...)
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return roles
}

func listRoleAssignments(t *testing.T, server *Server) []*accountpb.RoleAssignment {
	t.Helper()
	ctx := context.Background()
	var nextPageToken string
	var roleAssignments []*accountpb.RoleAssignment
	for {
		resp, err := server.ListRoleAssignments(ctx, &accountpb.ListRoleAssignmentsRequest{PageToken: nextPageToken})
		if err != nil {
			t.Fatalf("ListRoleAssignments: %v", err)
		}
		roleAssignments = append(roleAssignments, resp.RoleAssignments...)
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return roleAssignments
}
