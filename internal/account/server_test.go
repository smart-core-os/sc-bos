package account

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

func TestServer_CreateAccount(t *testing.T) {
	type testCase struct {
		others []*accountpb.Account // other accounts that should be created before the test account
		req    *accountpb.CreateAccountRequest
		expect *accountpb.Account
		code   codes.Code
	}

	cases := map[string]testCase{
		"user_account_no_password": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user1",
					}},
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
					Username: "user1",
				}},
			},
		},
		"user_account_with_password": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 2",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user2",
					}},
				},
				Password: "user2Password",
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 2",
				Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
					Username:    "user2",
					HasPassword: true,
				}},
			},
		},
		"user_account_short_password": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 3",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user3",
					}},
				},
				Password: "short",
			},
			code: codes.InvalidArgument,
		},
		"user_account_long_password": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 4",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user4",
					}},
				},
				Password: strings.Repeat("a", 101),
			},
			code: codes.InvalidArgument,
		},
		"user_account_short_password_whitespace": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 5",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user5",
					}},
				},
				Password: " short        ",
			},
			code: codes.InvalidArgument,
		},
		"service_account": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service",
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"service_account_password": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service",
				},
				Password: "servicePassword",
			},
			code: codes.InvalidArgument,
		},
		"missing_account_type": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					DisplayName: "Missing Kind",
				},
			},
			code: codes.InvalidArgument,
		},
		"missing_display_name": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type: accountpb.Account_USER_ACCOUNT,
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "foo",
					}},
				},
			},
			code: codes.InvalidArgument,
		},
		"display_name_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: strings.Repeat("a", maxDisplayNameLength),
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: strings.Repeat("a", maxDisplayNameLength),
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"display_name_too_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: strings.Repeat("a", maxDisplayNameLength+1),
				},
			},
			code: codes.InvalidArgument,
		},
		"missing_username_for_user_account": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "Missing Username",
				},
			},
			code: codes.InvalidArgument,
		},
		"username_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: strings.Repeat("a", maxUsernameLength),
					}},
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
					Username: strings.Repeat("a", maxUsernameLength),
				}},
			},
		},
		"username_too_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: strings.Repeat("a", maxUsernameLength+1),
					}},
				},
			},
			code: codes.InvalidArgument,
		},
		"description_short": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service",
					Description: "a description",
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: "a description",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"description_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service",
					Description: strings.Repeat("a", maxDescriptionLength),
				},
			},
			expect: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: strings.Repeat("a", maxDescriptionLength),
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"description_too_long": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service",
					Description: strings.Repeat("a", maxDescriptionLength+1),
				},
			},
			code: codes.InvalidArgument,
		},
		"user_details_supplied_for_service_account": {
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service Account",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "service",
					}},
				},
			},
			code: codes.InvalidArgument,
		},
		"username_conflict": {
			others: []*accountpb.Account{
				{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user1",
					}},
				},
			},
			req: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1A",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
						Username: "user1",
					}},
				},
			},
			code: codes.AlreadyExists,
		},
		"nil": {
			req: &accountpb.CreateAccountRequest{
				Account: nil,
			},
			code: codes.InvalidArgument,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			for _, other := range tc.others {
				_, err := server.CreateAccount(context.Background(), &accountpb.CreateAccountRequest{Account: other})
				if err != nil {
					t.Fatalf("failed to create other account: %v", err)
				}
			}

			res, err := server.CreateAccount(context.Background(), tc.req)
			checkNilIfErrored(t, res, err)
			if status.Code(err) != tc.code {
				t.Errorf("expected error with code %v, got %v", tc.code, err)
			}
			diff := cmp.Diff(tc.expect, res,
				protocmp.Transform(),
				protocmp.IgnoreFields(&accountpb.Account{}, "id", "create_time"),
				protocmp.IgnoreFields(&accountpb.ServiceAccount{}, "client_id", "client_secret"),
			)
			if diff != "" {
				t.Errorf("unexpected provided account value (-want +got):\n%s", diff)
			}

			// also retrieve using GetAccount and check it matches
			if res != nil {
				id := res.Id
				var expect *accountpb.Account
				if tc.expect != nil {
					expect = proto.Clone(tc.expect).(*accountpb.Account)
					expect.Id = id
					expect.CreateTime = res.CreateTime
					service := expect.GetServiceDetails()
					if service != nil {
						service.ClientId = id
					}
				}
				account, err := server.GetAccount(context.Background(), &accountpb.GetAccountRequest{Id: id})
				checkNilIfErrored(t, account, err)
				if err != nil {
					t.Fatalf("failed to get account %q: %v", id, err)
				}
				if res.Type == accountpb.Account_SERVICE_ACCOUNT {
					if res.GetServiceDetails().GetClientSecret() == "" {
						t.Errorf("service account created without client secret")
					}
				}
				diff = cmp.Diff(expect, account,
					protocmp.Transform(),
					protocmp.IgnoreFields(&accountpb.ServiceAccount{}, "client_secret"),
				)
				if diff != "" {
					t.Errorf("unexpected retrieved account value (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// tests ordering and pagination of ListAccounts
func TestServer_ListAccounts(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	createAccount := func(ty accountpb.Account_Type, username, displayName string) (*accountpb.Account, string) {
		t.Helper()
		account := &accountpb.Account{
			Type:        ty,
			DisplayName: displayName,
		}
		switch ty {
		case accountpb.Account_USER_ACCOUNT:
			account.Details = &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: username}}
		case accountpb.Account_SERVICE_ACCOUNT:
			account.Details = &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}}
		default:
			t.Fatalf("wrong type %v", ty)
		}
		res, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
			Account: account,
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to create account: %v", err)
		}
		account.Id = res.Id
		if service := account.GetServiceDetails(); service != nil {
			service.ClientId = res.GetServiceDetails().ClientId
		}
		return account, res.Id
	}

	// we assume that accounts are returned in creation order
	var expected []*accountpb.Account
	const numAccounts = 200
	for i := range numAccounts {
		username := fmt.Sprintf("account-%03d", i)
		displayName := fmt.Sprintf("Account %d", i)

		var account *accountpb.Account
		if i%2 == 0 {
			// make it a user account
			account, _ = createAccount(accountpb.Account_USER_ACCOUNT, username, displayName)
		} else {
			// make it a service account
			account, _ = createAccount(accountpb.Account_SERVICE_ACCOUNT, "", displayName)
		}
		expected = append(expected, account)
	}

	const pageSize = 42
	var nextPageToken string
	var got [][]*accountpb.Account
	for {
		t.Logf("fetching page with token %q", nextPageToken)
		res, err := server.ListAccounts(ctx, &accountpb.ListAccountsRequest{
			PageToken: nextPageToken,
			PageSize:  pageSize,
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to list accounts: %v", err)
		}

		if res.TotalSize != numAccounts {
			t.Errorf("expected total size %d, got %d", numAccounts, res.TotalSize)
		}

		size := len(res.Accounts)
		if size < pageSize && res.NextPageToken != "" {
			t.Errorf("fewer results (%d) returned than expected (%d), but got a page token", size, pageSize)
		}

		got = append(got, res.Accounts)

		nextPageToken = res.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	comparePages(t, pageSize, expected, got, "create_time")

}

func TestServer_UpdateAccount(t *testing.T) {
	ctx := context.Background()
	type testCase struct {
		others   []*accountpb.Account // other accounts that should be created before the test account
		initial  *accountpb.Account   // initial account to create
		update   *accountpb.UpdateAccountRequest
		expected *accountpb.Account
		code     codes.Code
	}
	cases := map[string]testCase{
		"empty update": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
		},
		"kind_change_prohibited": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Type: accountpb.Account_SERVICE_ACCOUNT,
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			code: codes.InvalidArgument,
		},
		"same_kind_allowed": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Type: accountpb.Account_USER_ACCOUNT,
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
		},
		"update_display_name": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					DisplayName: "Service MODIFIED",
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service MODIFIED",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"update_username": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1-modified"}},
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1-modified"}},
			},
		},
		"update_username_service_account": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "username"}},
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
			code: codes.FailedPrecondition,
		},
		"update_display_name_empty": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					DisplayName: "",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
			code: codes.InvalidArgument,
		},
		"update_username_empty_user": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: ""}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"user_details.username"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			code: codes.InvalidArgument,
		},
		"update_username_empty_service": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: ""}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"user_details.username"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"update_description_implicit": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Description: "Description",
				},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: "Description",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"update_description_explicit": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: "Before",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Description: "After",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: "After",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"update_description_empty": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Description: "Before",
			},
			update: &accountpb.UpdateAccountRequest{
				Account:    &accountpb.Account{},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"invalid_update_mask": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account:    &accountpb.Account{},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"foo"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			code: codes.InvalidArgument,
		},
		"wildcard_update_mask": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				Description: "Description",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1 MODIFIED",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1-modified"}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"*"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1 MODIFIED",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1-modified"}},
			},
		},
		"wildcard_update_mask_zero": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1 MODIFIED",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: ""}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"*"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "User 1", // not changed, because updates are all-or-nothing
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
			},
			code: codes.InvalidArgument, // because username is required, and we have tried to clear it
		},
		"wildcard_update_mask_service": {
			initial: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service",
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_SERVICE_ACCOUNT,
					DisplayName: "Service MODIFIED",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"*"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: "Service MODIFIED",
				Details:     &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{}},
			},
		},
		"username_conflict": {
			others: []*accountpb.Account{
				{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "Foo",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "foo"}},
				},
			},
			initial: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "Bar",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "bar"}},
			},
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "foo"}},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"user_details.username"}},
			},
			expected: &accountpb.Account{
				Type:        accountpb.Account_USER_ACCOUNT,
				DisplayName: "Bar",
				Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "bar"}},
			},
			code: codes.AlreadyExists,
		},
		"not_exist": {
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Id:      "123",
					Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "bar"}},
				},
			},
			code: codes.NotFound,
		},
		"invalid_id": {
			update: &accountpb.UpdateAccountRequest{
				Account: &accountpb.Account{
					Id:          "invalid",
					DisplayName: "Barbar",
				},
			},
			code: codes.NotFound,
		},
		"nil": {
			update: &accountpb.UpdateAccountRequest{
				Account: nil,
			},
			code: codes.InvalidArgument,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			for _, other := range tc.others {
				_, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{Account: other})
				if err != nil {
					t.Fatalf("failed to create other account: %v", err)
				}
			}

			var (
				account  *accountpb.Account
				expected *accountpb.Account
			)
			update := proto.Clone(tc.update).(*accountpb.UpdateAccountRequest)
			if tc.initial != nil {
				var err error
				account, err = server.CreateAccount(ctx, &accountpb.CreateAccountRequest{Account: tc.initial})
				checkNilIfErrored(t, account, err)
				if err != nil {
					t.Fatalf("failed to create account: %v", err)
				}
				// inject ID, which is now known, into update and expected
				update.Account.Id = account.Id
			}
			id := update.GetAccount().GetId()
			if tc.expected != nil {
				expected = proto.Clone(tc.expected).(*accountpb.Account)
				expected.Id = id
				expected.CreateTime = account.CreateTime
				if service := expected.GetServiceDetails(); service != nil {
					service.ClientId = id
				}
			}

			updated, err := server.UpdateAccount(ctx, update)
			checkNilIfErrored(t, updated, err)
			if status.Code(err) != tc.code {
				t.Errorf("expected error with code %v, got %v", tc.code, err)
			}
			if updated != nil {
				diff := cmp.Diff(expected, updated, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected updated account value (-want +got):\n%s", diff)
				}
			}

			if tc.expected != nil {
				// fetch again to check that the update was persisted
				account, err = server.GetAccount(ctx, &accountpb.GetAccountRequest{Id: id})
				checkNilIfErrored(t, account, err)
				if err != nil {
					t.Fatalf("failed to get account: %v", err)
				}
				diff := cmp.Diff(expected, account, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected retrieved account value (-want +got):\n%s", diff)
				}
			}
		})
	}
}

var validUsernames = []string{
	"foo",
	"foo.bar",
	"foo-bar",
	"foo_bar",
	"foo123",
	"foo@example.com",
	strings.Repeat("a", maxUsernameLength),
}

var invalidUsernames = []string{
	"ab",
	"foo bar",
	"foo\nbar",
	"foo😀bar",
	"foo/bar",
	"foo\\bar",
	"foo:bar",
}

func TestServer_CreateAccount_Usernames(t *testing.T) {
	template := &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "User",
			Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{}},
		},
	}

	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)
	for _, username := range validUsernames {
		req := proto.Clone(template).(*accountpb.CreateAccountRequest)
		req.Account.GetUserDetails().Username = username
		_, err := server.CreateAccount(ctx, req)
		if err != nil {
			t.Errorf("valid username %q failed: %v", username, err)
		}
	}
	for _, username := range invalidUsernames {
		req := proto.Clone(template).(*accountpb.CreateAccountRequest)
		req.Account.GetUserDetails().Username = username
		_, err := server.CreateAccount(ctx, req)
		if !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("invalid username %q returned %v", username, err)
		}
	}
}

func TestServer_UpdateAccount_Usernames(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	account, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "User",
			Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user"}},
		},
	})
	if err != nil {
		t.Fatalf("failed to set up test account: %v", err)
	}

	template := &accountpb.UpdateAccountRequest{
		Account: &accountpb.Account{
			Id:      account.Id,
			Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"user_details.username"}},
	}
	for _, username := range validUsernames {
		req := proto.Clone(template).(*accountpb.UpdateAccountRequest)
		req.Account.GetUserDetails().Username = username
		_, err := server.UpdateAccount(ctx, req)
		if err != nil {
			t.Errorf("valid username %q failed: %v", username, err)
		}
	}
	for _, username := range invalidUsernames {
		req := proto.Clone(template).(*accountpb.UpdateAccountRequest)
		req.Account.GetUserDetails().Username = username
		_, err := server.UpdateAccount(ctx, req)
		if !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("invalid username %q returned %v", username, err)
		}
	}
}

func TestServer_DeleteAccount(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	createAccount := func(ty accountpb.Account_Type, username, displayName, password string) *accountpb.Account {
		t.Helper()
		account := &accountpb.Account{
			Type:        ty,
			DisplayName: displayName,
		}
		if ty == accountpb.Account_USER_ACCOUNT {
			account.Details = &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
				Username: username,
			}}
		}
		res, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
			Account:  account,
			Password: password,
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to create account: %v", err)
		}
		return res
	}
	user := createAccount(accountpb.Account_USER_ACCOUNT, "user1", "User 1", "user1Password")
	service := createAccount(accountpb.Account_SERVICE_ACCOUNT, "", "Service", "")

	// assign a role to the accounts
	// to check that role assignments
	//   - do not prevent deletion of the account
	//   - are deleted when the account is deleted
	role, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
		Role: &accountpb.Role{
			DisplayName:   "Foo Role",
			PermissionIds: []string{"foo"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	ra1, err := server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: user.Id,
			RoleId:    role.Id,
		},
	})
	if err != nil {
		t.Fatalf("failed to create role assignment: %v", err)
	}
	ra2, err := server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: service.Id,
			RoleId:    role.Id,
		},
	})

	// delete the accounts
	_, err = server.DeleteAccount(ctx, &accountpb.DeleteAccountRequest{Id: user.Id})
	if err != nil {
		t.Errorf("failed to delete user account: %v", err)
	}
	_, err = server.DeleteAccount(ctx, &accountpb.DeleteAccountRequest{Id: service.Id})
	if err != nil {
		t.Errorf("failed to delete service account: %v", err)
	}
	// deleting an account again should fail
	_, err = server.DeleteAccount(ctx, &accountpb.DeleteAccountRequest{Id: user.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error when deleting user account twice, got %v", err)
	}
	// deleting an account again with allow_missing should succeed
	_, err = server.DeleteAccount(ctx, &accountpb.DeleteAccountRequest{Id: user.Id, AllowMissing: true})
	if err != nil {
		t.Errorf("failed to delete user account with allow_missing: %v", err)
	}

	// check that the accounts are actually gone
	_, err = server.GetAccount(ctx, &accountpb.GetAccountRequest{Id: user.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for user account, got %v", err)
	}
	_, err = server.GetAccount(ctx, &accountpb.GetAccountRequest{Id: service.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for service account, got %v", err)
	}

	// check that the role assignments are gone
	_, err = server.GetRoleAssignment(ctx, &accountpb.GetRoleAssignmentRequest{Id: ra1.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for role assignment, got %v", err)
	}
	_, err = server.GetRoleAssignment(ctx, &accountpb.GetRoleAssignmentRequest{Id: ra2.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for role assignment, got %v", err)
	}
}

// tests that uniqueness of usernames is enforced
func TestServer_Account_Username(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	_, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "User 1",
			Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create account1: %v", err)
	}
	user2, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "User 2",
			Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user2"}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create account2: %v", err)
	}

	// try to create account that collides with user1
	_, err = server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "User 1A",
			Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
		},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists error, got %v", err)
	}

	// try to change user2 to user1
	_, err = server.UpdateAccount(ctx, &accountpb.UpdateAccountRequest{
		Account: &accountpb.Account{
			Id:      user2.Id,
			Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
		},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists error, got %v", err)
	}
}

func TestServer_UpdateAccountPassword(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		create           *accountpb.CreateAccountRequest
		update           *accountpb.UpdateAccountPasswordRequest
		validPassword    string   // password to check after update
		invalidPasswords []string // passwords to check for invalidity
		code             codes.Code
	}

	cases := map[string]testCase{
		"add_password": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "user1Password",
			},
			validPassword: "user1Password",
		},
		"change_password": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
				Password: "thepassword1",
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "thepassword2",
			},
			validPassword:    "thepassword2",
			invalidPasswords: []string{"thepassword1"},
		},
		"change_password_valid_old_password": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
				Password: "thepassword1",
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				OldPassword: "thepassword1",
				NewPassword: "thepassword2",
			},
			validPassword:    "thepassword2",
			invalidPasswords: []string{"thepassword1"},
		},
		"change_password_invalid_old_password": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
				Password: "thepassword1",
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				OldPassword: "wrongPassword",
				NewPassword: "thepassword2",
			},
			validPassword:    "thepassword1",
			invalidPasswords: []string{"thepassword2", "wrongPassword"},
			code:             codes.FailedPrecondition,
		},
		"add_password_old_password_supplied": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				OldPassword: "thepassword1",
				NewPassword: "thepassword2",
			},
			invalidPasswords: []string{"thepassword1", "thepassword2"},
			code:             codes.FailedPrecondition,
		},
		"account_id_not_found": {
			update: &accountpb.UpdateAccountPasswordRequest{
				Id:          "12345",
				NewPassword: "thepassword",
			},
			code: codes.NotFound,
		},
		"account_id_invalid": {
			update: &accountpb.UpdateAccountPasswordRequest{
				Id:          "invalid",
				NewPassword: "thepassword",
			},
			code: codes.NotFound,
		},
		"account_id_empty": {
			update: &accountpb.UpdateAccountPasswordRequest{
				Id:          "",
				NewPassword: "thepassword",
			},
			code: codes.InvalidArgument,
		},
		"add_password_empty": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "",
			},
			code: codes.InvalidArgument,
		},
		"add_password_too_short": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "123456789",
			},
			code: codes.InvalidArgument,
		},
		"add_password_long": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: strings.Repeat("a", maxPasswordLength),
			},
		},
		"add_password_too_long": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: strings.Repeat("a", maxPasswordLength+1),
			},
			code: codes.InvalidArgument,
		},
		"password_ignores_leading_whitespace": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "  thepassword",
			},
			validPassword: "thepassword",
		},
		"password_ignores_trailing_whitespace": {
			create: &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User 1",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user1"}},
				},
			},
			update: &accountpb.UpdateAccountPasswordRequest{
				NewPassword: "thepassword  ",
			},
			validPassword: "thepassword",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			var account *accountpb.Account
			var err error
			if tc.create != nil {
				account, err = server.CreateAccount(ctx, tc.create)
				if err != nil {
					t.Fatalf("failed to create account: %v", err)
				}
			}

			req := proto.Clone(tc.update).(*accountpb.UpdateAccountPasswordRequest)
			if account != nil {
				req.Id = account.Id
			}
			_, err = server.UpdateAccountPassword(ctx, req)
			if status.Code(err) != tc.code {
				t.Errorf("expected error with code %v, got %v", tc.code, err)
			}

			check := func(password string) error {
				return store.Read(ctx, func(tx *Tx) error {
					id, ok := parseID(account.Id)
					if !ok {
						t.Fatalf("failed to parse account ID %q", account.Id)
					}
					return tx.CheckAccountPassword(ctx, id, password)
				})
			}

			if tc.validPassword != "" {
				err = check(tc.validPassword)
				if err != nil {
					t.Errorf("password check for %q failed: %v", tc.validPassword, err)
				}
			}

			for _, password := range tc.invalidPasswords {
				err = check(password)
				if err == nil {
					t.Errorf("expected error for password %q, got nil", password)
				}
			}

		})
	}

}

func TestServer_RotateAccountClientSecret(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	// create a service account
	account, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_SERVICE_ACCOUNT,
			DisplayName: "Service",
		},
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	secret1 := account.GetServiceDetails().GetClientSecret()
	id, ok := parseID(account.Id)
	if !ok {
		t.Fatalf("failed to parse account ID %q", account.Id)
	}

	check := func(secret string, wantOK bool) {
		err := store.Read(ctx, func(tx *Tx) error {
			err := tx.CheckClientSecret(ctx, id, secret)
			if wantOK {
				if errors.Is(err, ErrIncorrectSecret) {
					t.Errorf("expected secret %q to be valid, got error %v", secret, err)
				} else if err != nil {
					return err
				}
			} else {
				if !errors.Is(err, ErrIncorrectSecret) {
					t.Errorf("expected secret %q to be invalid, got error %v", secret, err)
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error checking secret: %v", err)
		}
	}

	check(secret1, true)

	// rotate the secret, keeping the old one valid for a while
	res, err := server.RotateAccountClientSecret(ctx, &accountpb.RotateAccountClientSecretRequest{
		Id:                       account.Id,
		PreviousSecretExpireTime: timestamppb.New(future),
	})
	if err != nil {
		t.Fatalf("failed to rotate secret: %v", err)
	}
	secret2 := res.ClientSecret
	check(secret1, true)
	check(secret2, true)

	// rotate again, this time supplying a time that is already expired
	res, err = server.RotateAccountClientSecret(ctx, &accountpb.RotateAccountClientSecretRequest{
		Id:                       account.Id,
		PreviousSecretExpireTime: timestamppb.New(past),
	})
	if err != nil {
		t.Fatalf("failed to rotate secret: %v", err)
	}
	secret3 := res.ClientSecret
	check(secret1, false)
	check(secret2, false)
	check(secret3, true)

	// rotate again, with no expiry time
	res, err = server.RotateAccountClientSecret(ctx, &accountpb.RotateAccountClientSecretRequest{
		Id: account.Id,
	})
	if err != nil {
		t.Fatalf("failed to rotate secret: %v", err)
	}
	secret4 := res.ClientSecret
	check(secret1, false)
	check(secret2, false)
	check(secret3, false)
	check(secret4, true)
}

func TestServer_Role(t *testing.T) {
	// tests a sequence of role operations, checking that role lifecycles are handled correctly

	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	// collect the system-created roles
	res, err := server.ListRoles(ctx, &accountpb.ListRolesRequest{
		PageSize: 10, // shouldn't be more than this
	})
	checkNilIfErrored(t, res, err)
	if err != nil {
		t.Fatalf("failed to list roles: %v", err)
	}
	if len(res.Roles) >= 10 {
		t.Fatalf("expected less than 10 system roles, got %d", len(res.Roles))
	}
	roles := res.Roles
	numSystemRoles := int32(len(roles))

	const numRoles = 200
	const numPermissions = 10

	const description = "A role for testing"
	t.Log("CreateRole:")
	for i := range numRoles {
		displayName := fmt.Sprintf("Role %d", i)
		numPermissions := i % numPermissions // test roles with different numbers of permissions
		role, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
			Role: &accountpb.Role{
				DisplayName: displayName,
				// supply the permissions shuffled to check they are returned in order instead
				PermissionIds: shuffledPermissions(numPermissions),
				Description:   description,
			},
		})
		if err != nil {
			t.Fatalf("failed to create role %q: %v", displayName, err)
		}
		expect := &accountpb.Role{
			Id:            role.Id,
			DisplayName:   displayName,
			PermissionIds: orderedPermissions(numPermissions),
			Description:   description,
		}
		diff := cmp.Diff(expect, role, protocmp.Transform())
		if diff != "" {
			t.Errorf("unexpected role value (-want +got):\n%s", diff)
		}

		roles = append(roles, role)
	}

	checkAll := func(expect []*accountpb.Role) {
		var pages [][]*accountpb.Role
		const pageSize = 42
		var nextPageToken string
		for {
			res, err := server.ListRoles(ctx, &accountpb.ListRolesRequest{
				PageToken: nextPageToken,
				PageSize:  pageSize,
			})
			checkNilIfErrored(t, res, err)
			if err != nil {
				t.Fatalf("failed to list roles: %v", err)
			}
			t.Logf("fetched page with token %q, returned %d results", nextPageToken, len(res.Roles))
			if res.TotalSize != numRoles+numSystemRoles {
				t.Errorf("expected total size %d, got %d", numRoles, res.TotalSize)
			}

			if res.NextPageToken != "" && len(res.Roles) < pageSize {
				t.Errorf("fewer results (%d) returned than expected (%d), but got a page token", len(res.Roles), pageSize)
			}

			pages = append(pages, res.Roles)
			nextPageToken = res.NextPageToken
			if nextPageToken == "" {
				break
			}
		}
		comparePages(t, pageSize, roles, pages)
	}

	checkAll(roles)

	// test that roles can be updated
	t.Log("UpdateRole:")
	role := roles[len(roles)-1]                                           // pick the last role created, as this will definitely not be a protected system role
	role.PermissionIds = append(role.PermissionIds, "000-new-permission") // should go at the beginning
	role.DisplayName += " MODIFIED"
	updated, err := server.UpdateRole(ctx, &accountpb.UpdateRoleRequest{
		Role: role,
	})
	slices.Sort(role.PermissionIds)
	checkNilIfErrored(t, role, err)
	if err != nil {
		t.Fatalf("failed to update role: %v", err)
	}
	diff := cmp.Diff(role, updated, protocmp.Transform())
	if diff != "" {
		t.Errorf("unexpected updated role value (-want +got):\n%s", diff)
	}
	// test that update is persisted
	updated, err = server.GetRole(ctx, &accountpb.GetRoleRequest{Id: role.Id})
	checkNilIfErrored(t, updated, err)
	if err != nil {
		t.Fatalf("failed to get updated role: %v", err)
	}
	diff = cmp.Diff(role, updated, protocmp.Transform())
	if diff != "" {
		t.Errorf("unexpected retrieved role value (-want +got):\n%s", diff)
	}

	// test that a role can't be deleted if it is assigned
	t.Log("DeleteRole:")
	account, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_SERVICE_ACCOUNT,
			DisplayName: "foo account",
		},
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	assignment, err := server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: account.Id,
			RoleId:    role.Id,
		},
	})
	if err != nil {
		t.Fatalf("failed to create role assignment: %v", err)
	}
	_, err = server.DeleteRole(ctx, &accountpb.DeleteRoleRequest{Id: role.Id})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition error when deleting role with assignments, got %v", err)
	}

	// delete the assignment and try again
	_, err = server.DeleteRoleAssignment(ctx, &accountpb.DeleteRoleAssignmentRequest{Id: assignment.Id})
	if err != nil {
		t.Fatalf("failed to delete role assignment: %v", err)
	}

	// test that roles can be deleted
	_, err = server.DeleteRole(ctx, &accountpb.DeleteRoleRequest{Id: role.Id})
	if err != nil {
		t.Fatalf("failed to delete role: %v", err)
	}

	// check that the role is actually gone
	_, err = server.GetRole(ctx, &accountpb.GetRoleRequest{Id: role.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for role, got %v", err)
	}
	// deleting it again should fail
	_, err = server.DeleteRole(ctx, &accountpb.DeleteRoleRequest{Id: role.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for role, got %v", err)
	}
	// deleting with allow_missing should succeed
	_, err = server.DeleteRole(ctx, &accountpb.DeleteRoleRequest{Id: role.Id, AllowMissing: true})
	if err != nil {
		t.Errorf("failed to delete role with allow_missing: %v", err)
	}
}

func TestServer_CreateRole(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		existing []*accountpb.Role
		role     *accountpb.Role
		err      error
	}

	cases := map[string]testCase{
		"missing_display_name": {
			role: &accountpb.Role{},
			err:  ErrInvalidDisplayName,
		},
		"display_name_too_long": {
			role: &accountpb.Role{DisplayName: strings.Repeat("a", maxDisplayNameLength+1)},
			err:  ErrInvalidDisplayName,
		},
		"long_display_name": {
			role: &accountpb.Role{DisplayName: strings.Repeat("a", maxDisplayNameLength)},
		},
		"short_display_name": {
			role: &accountpb.Role{DisplayName: "Role"},
		},
		"short_description": {
			role: &accountpb.Role{
				DisplayName: "Role",
				Description: "Role Description",
			},
		},
		"description_too_long": {
			role: &accountpb.Role{
				DisplayName: "Role",
				Description: strings.Repeat("a", maxDescriptionLength+1),
			},
			err: ErrInvalidDescription,
		},
		"long_description": {
			role: &accountpb.Role{
				DisplayName: "Role",
				Description: strings.Repeat("a", maxDescriptionLength),
			},
		},
		"single_permission": {
			role: &accountpb.Role{
				DisplayName:   "Role",
				PermissionIds: []string{"foo"},
			},
		},
		"multiple_permissions_ordered": {
			role: &accountpb.Role{
				DisplayName:   "Role",
				PermissionIds: []string{"bar", "baz", "foo"},
			},
		},
		"multiple_permissions_unordered": {
			role: &accountpb.Role{
				DisplayName:   "Role",
				PermissionIds: []string{"foo", "bar", "baz"},
			},
		},
		"duplicate_permissions": {
			role: &accountpb.Role{
				DisplayName:   "Role",
				PermissionIds: []string{"foo", "bar", "foo"},
			},
		},
		"nil": {
			role: nil,
			err:  ErrResourceMissing,
		},
		"display_name_conflict": {
			existing: []*accountpb.Role{
				{DisplayName: "Role 1"},
			},
			role: &accountpb.Role{DisplayName: "Role 1"},
			err:  ErrRoleDisplayNameExists,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			for _, existing := range tc.existing {
				_, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{Role: existing})
				if err != nil {
					t.Fatalf("failed to create existing role: %v", err)
				}
			}

			created, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{Role: tc.role})
			if !errors.Is(err, tc.err) {
				t.Errorf("expected error %v, got %v", tc.err, err)
			}

			if created != nil {
				expect := proto.Clone(tc.role).(*accountpb.Role)
				expect.Id = created.Id
				// normalise the permission IDs
				slices.Sort(expect.PermissionIds)
				expect.PermissionIds = slices.Compact(expect.PermissionIds)

				diff := cmp.Diff(expect, created, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected created role value (-want +got):\n%s", diff)
				}

				fetched, err := server.GetRole(ctx, &accountpb.GetRoleRequest{Id: created.Id})
				if err != nil {
					t.Errorf("failed to fetch created role: %v", err)
				}

				diff = cmp.Diff(expect, fetched, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected fetched role value (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestServer_UpdateRole(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		initial  *accountpb.Role
		others   []*accountpb.Role // other roles to create before the update
		update   *accountpb.UpdateRoleRequest
		expected *accountpb.Role
		code     codes.Code
	}
	cases := map[string]testCase{
		"empty update": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"bar", "foo"},
			},
		},
		"update_display_name_implicit": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					DisplayName: "Role 1 MODIFIED",
				},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1 MODIFIED",
				PermissionIds: []string{"bar", "foo"},
			},
		},
		"update_display_name_explicit": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					DisplayName:   "Role 1 MODIFIED",
					PermissionIds: []string{"foo2", "bar2"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1 MODIFIED",
				PermissionIds: []string{"bar", "foo"},
			},
		},
		"update_display_name_clear": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
			},
			update: &accountpb.UpdateRoleRequest{
				Role:       &accountpb.Role{},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
			},
			code: codes.InvalidArgument,
		},
		"update_display_name_too_long": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					DisplayName: strings.Repeat("a", maxDisplayNameLength+1),
				},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
			},
			code: codes.InvalidArgument,
		},
		"update_permissions_implicit": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					PermissionIds: []string{"foo2", "bar2"},
				},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"bar2", "foo2"},
			},
		},
		"update_permissions_explicit": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					DisplayName:   "Role 1 MODIFIED",
					PermissionIds: []string{"foo2", "bar2"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"permission_ids"}},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"bar2", "foo2"},
			},
		},
		"update_permissions_empty": {
			initial: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: []string{"foo", "bar"},
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					PermissionIds: nil,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"permission_ids"}},
			},
			expected: &accountpb.Role{
				DisplayName:   "Role 1",
				PermissionIds: nil,
			},
		},
		"update_description_implicit": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					Description: "A role for testing",
				},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
				Description: "A role for testing",
			},
		},
		"update_description_explicit": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{
					Description: "A role for testing",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
				Description: "A role for testing",
			},
		},
		"update_description_clear": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
				Description: "A role for testing",
			},
			update: &accountpb.UpdateRoleRequest{
				Role:       &accountpb.Role{},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
			},
		},
		"update_description_invalid": {
			initial: &accountpb.Role{
				DisplayName: "Role 1",
			},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{Description: strings.Repeat("a", maxDescriptionLength+1)},
			},
			expected: &accountpb.Role{
				DisplayName: "Role 1",
			},
			code: codes.InvalidArgument,
		},
		"nil": {
			initial:  &accountpb.Role{DisplayName: "Role 1"},
			update:   &accountpb.UpdateRoleRequest{Role: nil},
			expected: &accountpb.Role{DisplayName: "Role 1"},
			code:     codes.InvalidArgument,
		},
		"display_name_conflict": {
			initial: &accountpb.Role{DisplayName: "Role 1"},
			others:  []*accountpb.Role{{DisplayName: "Role 2"}},
			update: &accountpb.UpdateRoleRequest{
				Role: &accountpb.Role{DisplayName: "Role 2"},
			},
			expected: &accountpb.Role{DisplayName: "Role 1"},
			code:     codes.AlreadyExists,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			role, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
				Role: tc.initial,
			})
			checkNilIfErrored(t, role, err)
			if err != nil {
				t.Fatalf("failed to create role: %v", err)
			}

			for _, other := range tc.others {
				_, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
					Role: other,
				})
				if err != nil {
					t.Fatalf("failed to create other role: %v", err)
				}
			}

			// inject ID, which is now known, into update and expected
			id := role.Id
			update := proto.Clone(tc.update).(*accountpb.UpdateRoleRequest)
			if update.Role != nil {
				update.Role.Id = id
			}
			expected := proto.Clone(tc.expected).(*accountpb.Role)
			expected.Id = id

			updated, err := server.UpdateRole(ctx, update)
			checkNilIfErrored(t, updated, err)
			if status.Code(err) != tc.code {
				t.Errorf("expected error with code %v, got %v", tc.code, err)
			}

			if updated != nil {
				diff := cmp.Diff(expected, updated, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected updated role value (-want +got):\n%s", diff)
				}
			}

			// fetch again to check that the update was persisted
			role, err = server.GetRole(ctx, &accountpb.GetRoleRequest{Id: id})
			checkNilIfErrored(t, role, err)
			if err != nil {
				t.Fatalf("failed to get role: %v", err)
			}
			diff := cmp.Diff(expected, role, protocmp.Transform())
			if diff != "" {
				t.Errorf("unexpected retrieved role value (-want +got):\n%s", diff)
			}
		})
	}
}

// tests that protected roles cannot be updated or deleted
func TestRole_Protected(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	findLegacyRole := func(legacyRole string) *accountpb.Role {
		t.Helper()
		res, err := server.ListRoles(ctx, &accountpb.ListRolesRequest{
			PageSize: 100,
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to list roles: %v", err)
		}
		for _, role := range res.Roles {
			if role.Protected && role.LegacyRoleName == legacyRole {
				return role
			}
		}
		return nil
	}
	adminRole := findLegacyRole("admin")
	if adminRole == nil {
		t.Fatal("no admin role found")
	}

	res, err := server.UpdateRole(ctx, &accountpb.UpdateRoleRequest{
		Role: &accountpb.Role{
			Id:            adminRole.Id,
			DisplayName:   "foo",
			Description:   "bar",
			PermissionIds: []string{"baz"},
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition error when updating role with no ID, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil response when updating role with no ID, got %v", res)
	}

	_, err = server.DeleteRole(ctx, &accountpb.DeleteRoleRequest{Id: adminRole.Id})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition error when deleting protected role, got %v", err)
	}
}

func comparePages[T messageWithID](t *testing.T, pageSize int32, expect []T, gotPages [][]T, ignoreFields ...protoreflect.Name) {
	var zero T
	t.Helper()
	for i, page := range gotPages {
		var allIDs []string
		for _, a := range page {
			allIDs = append(allIDs, a.GetId())
		}
		t.Logf("page %d contains %d items: %s", i, len(page), strings.Join(allIDs, ", "))
		if len(page) > int(pageSize) {
			t.Errorf("page %d has more than %d items: %d", i, pageSize, len(page))
		}
		if len(page) > len(expect) {
			t.Errorf("page %d has more items (%d) than remaining expected items (%d)", i, len(page), len(expect))
			return
		}

		expectPage := expect[:len(page)]
		expect = expect[len(page):]

		diff := cmp.Diff(expectPage, page,
			protocmp.Transform(),
			protocmp.IgnoreFields(zero, ignoreFields...),
		)
		if diff != "" {
			t.Errorf("unexpected page %d contents (-want +got):\n%s", i, diff)
		}
	}

	if len(expect) > 0 {
		t.Errorf("expected %d more items than received", len(expect))
	}
}

func TestServer_RoleAssignments(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	createAccount := func(displayName string) *accountpb.Account {
		t.Helper()
		res, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
			Account: &accountpb.Account{
				Type:        accountpb.Account_SERVICE_ACCOUNT,
				DisplayName: displayName,
			},
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to create account: %v", err)
		}
		return res
	}
	createRole := func(displayName string, permissions ...string) *accountpb.Role {
		t.Helper()
		res, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
			Role: &accountpb.Role{
				DisplayName:   displayName,
				PermissionIds: permissions,
			},
		})
		checkNilIfErrored(t, res, err)
		if err != nil {
			t.Fatalf("failed to create role: %v", err)
		}
		return res
	}

	var accounts []*accountpb.Account
	var roles []*accountpb.Role
	const numAccounts = 50
	const numRoles = 50

	for i := range numAccounts {
		accounts = append(accounts, createAccount(fmt.Sprintf("Account %d", i)))
	}
	for i := range numRoles {
		roles = append(roles, createRole(fmt.Sprintf("Role %d", i)))
	}

	// create role assignments randomly
	var assignments []*accountpb.RoleAssignment
	for _, account := range accounts {
		for _, role := range roles {
			// don't assign all roles to all accounts, just some
			if rand.IntN(2) == 0 {
				continue
			}

			assignment, err := server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: account.Id,
					RoleId:    role.Id,
				},
			})
			checkNilIfErrored(t, assignment, err)
			if err != nil {
				t.Errorf("failed to create role assignment between account=%s and role=%s: %v", account.Id, role.Id, err)
			}

			assignments = append(assignments, assignment)
		}
	}

	type filter struct {
		name   string
		filter string
		expect []*accountpb.RoleAssignment
	}
	var filters []filter
	// no filter
	filters = append(filters, filter{
		name:   "no filter",
		filter: "",
		expect: assignments,
	})
	{
		// filter by account
		account := accounts[rand.IntN(len(accounts))]
		var expect []*accountpb.RoleAssignment
		for _, a := range assignments {
			if a.AccountId == account.Id {
				expect = append(expect, a)
			}
		}
		filters = append(filters, filter{
			name:   "account",
			filter: fmt.Sprintf("account_id = %s", account.Id),
			expect: expect,
		})
	}
	{
		// filter by role
		role := roles[rand.IntN(len(roles))]
		var expect []*accountpb.RoleAssignment
		for _, a := range assignments {
			if a.RoleId == role.Id {
				expect = append(expect, a)
			}
		}
		filters = append(filters, filter{
			name:   "role",
			filter: fmt.Sprintf("role_id = %s", role.Id),
			expect: expect,
		})
	}

	for _, f := range filters {
		t.Run(f.name, func(t *testing.T) {
			var got [][]*accountpb.RoleAssignment
			const pageSize = 15
			var nextPageToken string
			for {
				res, err := server.ListRoleAssignments(ctx, &accountpb.ListRoleAssignmentsRequest{
					Filter:    f.filter,
					PageToken: nextPageToken,
					PageSize:  pageSize,
				})
				checkNilIfErrored(t, res, err)
				if err != nil {
					t.Errorf("failed to list role assignments with filter %q: %v", f.filter, err)
				}
				t.Logf("fetched page with token %q, returned %d results", nextPageToken, len(res.RoleAssignments))

				if int(res.TotalSize) != len(f.expect) {
					t.Errorf("expected total size %d, got %d", len(f.expect), res.TotalSize)
				}

				if res.NextPageToken != "" && len(res.RoleAssignments) < pageSize {
					t.Errorf("fewer results (%d) returned than expected (%d), but got a page token", len(res.RoleAssignments), pageSize)
				}

				got = append(got, res.RoleAssignments)
				nextPageToken = res.NextPageToken
				if nextPageToken == "" {
					break
				}
			}

			comparePages(t, pageSize, f.expect, got)
		})
	}

	// delete a role assignment, check it's gone
	_, err := server.DeleteRoleAssignment(ctx, &accountpb.DeleteRoleAssignmentRequest{
		Id: assignments[0].Id,
	})
	if err != nil {
		t.Fatalf("failed to delete role assignment: %v", err)
	}
	// check that the role assignment is actually gone
	_, err = server.GetRoleAssignment(ctx, &accountpb.GetRoleAssignmentRequest{Id: assignments[0].Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for get role assignment, got %v", err)
	}
	// deleting it again should fail
	_, err = server.DeleteRoleAssignment(ctx, &accountpb.DeleteRoleAssignmentRequest{Id: assignments[0].Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound error for delete role assignment, got %v", err)
	}
	// deleting with allow_missing should succeed
	_, err = server.DeleteRoleAssignment(ctx, &accountpb.DeleteRoleAssignmentRequest{Id: assignments[0].Id, AllowMissing: true})
	if err != nil {
		t.Errorf("failed to delete role assignment with allow_missing: %v", err)
	}
}

func TestServer_CreateRoleAssignment(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		existing []*accountpb.RoleAssignment
		req      *accountpb.CreateRoleAssignmentRequest
		code     codes.Code
	}

	// will be replaced with IDs created during the test run
	const (
		accountPlaceholder = "ACCOUNT PLACEHOLDER"
		rolePlaceholder    = "ROLE PLACEHOLDER"
	)

	cases := map[string]testCase{
		"role_does_not_exist": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    "999",
				},
			},
			code: codes.NotFound,
		},
		"account_does_not_exist": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: "999",
					RoleId:    rolePlaceholder,
				},
			},
			code: codes.NotFound,
		},
		"role_id_invalid": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    "invalid",
				},
			},
			code: codes.NotFound,
		},
		"account_id_invalid": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: "invalid",
					RoleId:    rolePlaceholder,
				},
			},
			code: codes.NotFound,
		},
		"scope_provided": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE,
						Resource:     "named-resource",
					},
				},
			},
			code: codes.OK,
		},
		"scope_not_provided": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
				},
			},
			code: codes.OK,
		},
		"scope_invalid_resource_type": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: 500,
						Resource:     "invalid resource type",
					},
				},
			},
			code: codes.InvalidArgument,
		},
		"scope_empty_resource_string": {
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE,
						Resource:     "",
					},
				},
			},
			code: codes.InvalidArgument,
		},
		"scoped_and_unscoped": {
			existing: []*accountpb.RoleAssignment{
				{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
				},
			},
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NODE,
						Resource:     "node1",
					},
				},
			},
			code: codes.OK,
		},
		"different_scope_types": {
			existing: []*accountpb.RoleAssignment{
				{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE,
						Resource:     "name1",
					},
				},
			},
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NODE,
						Resource:     "node1",
					},
				},
			},
			code: codes.OK,
		},
		"different_scope_resources": {
			existing: []*accountpb.RoleAssignment{
				{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NODE,
						Resource:     "node1",
					},
				},
			},
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NODE,
						Resource:     "node2",
					},
				},
			},
			code: codes.OK,
		},
		"conflict_unscoped": {
			existing: []*accountpb.RoleAssignment{
				{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
				},
			},
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
				},
			},
			code: codes.AlreadyExists,
		},
		"conflict_scoped": {
			existing: []*accountpb.RoleAssignment{
				{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE,
						Resource:     "resource1",
					},
				},
			},
			req: &accountpb.CreateRoleAssignmentRequest{
				RoleAssignment: &accountpb.RoleAssignment{
					AccountId: accountPlaceholder,
					RoleId:    rolePlaceholder,
					Scope: &accountpb.RoleAssignment_Scope{
						ResourceType: accountpb.RoleAssignment_NAMED_RESOURCE,
						Resource:     "resource1",
					},
				},
			},
			code: codes.AlreadyExists,
		},
		"nil": {
			req:  &accountpb.CreateRoleAssignmentRequest{},
			code: codes.InvalidArgument,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logger := testLogger(t)
			store := NewMemoryStore(logger)
			server := NewServer(store, logger)

			// Create a valid account and role for positive test cases
			account, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
				Account: &accountpb.Account{
					Type:        accountpb.Account_USER_ACCOUNT,
					DisplayName: "User",
					Details:     &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{Username: "user"}},
				},
			})
			if err != nil {
				t.Fatalf("failed to create account: %v", err)
			}
			role, err := server.CreateRole(ctx, &accountpb.CreateRoleRequest{
				Role: &accountpb.Role{
					DisplayName:   "Role",
					PermissionIds: []string{"foo"},
				},
			})
			if err != nil {
				t.Fatalf("failed to create role: %v", err)
			}

			substitute := func(ra *accountpb.RoleAssignment) *accountpb.RoleAssignment {
				if ra == nil {
					return nil
				}
				// Inject valid IDs into the request if needed
				cloned := proto.Clone(ra).(*accountpb.RoleAssignment)
				if cloned.RoleId == rolePlaceholder {
					cloned.RoleId = role.Id
				}
				if cloned.AccountId == accountPlaceholder {
					cloned.AccountId = account.Id
				}
				return cloned
			}

			// Create any existing role assignments
			for _, existing := range tc.existing {
				existing = substitute(existing)

				_, err = server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
					RoleAssignment: existing,
				})
				if err != nil {
					t.Fatalf("failed to create existing role assignment: %v", err)
				}
			}

			req := proto.Clone(tc.req).(*accountpb.CreateRoleAssignmentRequest)
			req.RoleAssignment = substitute(req.RoleAssignment)

			created, err := server.CreateRoleAssignment(ctx, req)
			if status.Code(err) != tc.code {
				t.Errorf("expected error code %v, got %v", tc.code, err)
			}

			if created != nil {
				expected := proto.Clone(req.RoleAssignment).(*accountpb.RoleAssignment)
				expected.Id = created.Id

				diff := cmp.Diff(expected, created, protocmp.Transform())
				if diff != "" {
					t.Errorf("unexpected created role assignment value (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// check that a role assignment with a scope is forbidden if the role is a legacy role
func TestServer_CreateRoleAssignment_ScopeWithLegacyRole(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)
	store := NewMemoryStore(logger)
	server := NewServer(store, logger)

	// find the admin role, which is a legacy role
	var adminRole *accountpb.Role
	res, err := server.ListRoles(ctx, &accountpb.ListRolesRequest{
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("failed to list roles: %v", err)
	}
	for _, role := range res.Roles {
		if role.LegacyRoleName == "admin" {
			adminRole = role
			break
		}
	}
	if adminRole == nil {
		t.Fatal("no admin role found")
	}

	// create a user account
	account, err := server.CreateAccount(ctx, &accountpb.CreateAccountRequest{
		Account: &accountpb.Account{
			Type:        accountpb.Account_USER_ACCOUNT,
			DisplayName: "Test Account",
			Details: &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
				Username: "test",
			}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// try to create a role assignment with a scope - should fail
	_, err = server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: account.Id,
			RoleId:    adminRole.Id,
			Scope: &accountpb.RoleAssignment_Scope{
				ResourceType: accountpb.RoleAssignment_ZONE,
				Resource:     "foo",
			},
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition error when creating role assignment with scope for legacy role, got %v", err)
	}

	// try to create a role assignment without a scope - should succeed
	_, err = server.CreateRoleAssignment(ctx, &accountpb.CreateRoleAssignmentRequest{
		RoleAssignment: &accountpb.RoleAssignment{
			AccountId: account.Id,
			RoleId:    adminRole.Id,
		},
	})
	if err != nil {
		t.Errorf("failed to create role assignment without scope for legacy role: %v", err)
	}
}

type messageWithID interface {
	proto.Message
	GetId() string
}

func checkNilIfErrored[V any](t *testing.T, v *V, err error) {
	t.Helper()
	if err != nil && v != nil {
		t.Errorf("expected nil return because of error %v, found value %v", err, v)
	}
	if err == nil && v == nil {
		t.Error("expected non-nil return but got nil")
	}
}

func shuffledPermissions(n int) []string {
	source := rand.Perm(n)
	perms := make([]string, n)
	for i, p := range source {
		perms[i] = fmt.Sprintf("perm-%02d", p)
	}
	return perms
}

func orderedPermissions(n int) []string {
	perms := make([]string, n)
	for i := range n {
		perms[i] = fmt.Sprintf("perm-%02d", i)
	}
	return perms
}

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return logger
}
