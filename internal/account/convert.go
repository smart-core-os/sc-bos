package account

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/account/queries"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

func parseID(id string) (int64, bool) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func accountToProto(account queries.AccountDetail, clientSecret string) *accountpb.Account {
	id := formatID(account.ID)
	converted := &accountpb.Account{
		Id:          id,
		DisplayName: account.DisplayName,
		Type:        accountpb.Account_Type(accountpb.Account_Type_value[account.Type]), // default to ACCOUNT_TYPE_UNSPECIFIED
		CreateTime:  timestamppb.New(account.CreateTime),
	}
	switch converted.Type {
	case accountpb.Account_USER_ACCOUNT:
		converted.Details = &accountpb.Account_UserDetails{UserDetails: &accountpb.UserAccount{
			HasPassword: account.PasswordHash != nil,
		}}
		if account.Username.Valid {
			converted.GetUserDetails().Username = account.Username.String
		}
	case accountpb.Account_SERVICE_ACCOUNT:
		converted.Details = &accountpb.Account_ServiceDetails{ServiceDetails: &accountpb.ServiceAccount{
			ClientId:     id,
			ClientSecret: clientSecret,
		}}
		if account.SecondarySecretExpireTime.Valid {
			converted.GetServiceDetails().PreviousSecretExpireTime = timestamppb.New(account.SecondarySecretExpireTime.Time)
		}
	}
	if account.Description.Valid {
		converted.Description = account.Description.String
	}
	return converted
}

func roleToProto(role queries.Role, permissions []string) *accountpb.Role {
	protoRole := &accountpb.Role{
		Id:            formatID(role.ID),
		DisplayName:   role.DisplayName,
		PermissionIds: permissions,
		Protected:     role.Protected,
	}
	if role.Description.Valid {
		protoRole.Description = role.Description.String
	}
	if role.LegacyRole.Valid {
		protoRole.LegacyRoleName = role.LegacyRole.String
	}
	return protoRole
}

func roleAssignmentToProto(assignment queries.RoleAssignment) *accountpb.RoleAssignment {
	ra := &accountpb.RoleAssignment{
		Id:        formatID(assignment.ID),
		AccountId: formatID(assignment.AccountID),
		RoleId:    formatID(assignment.RoleID),
	}
	if assignment.ScopeType.Valid && assignment.ScopeResource.Valid {
		ra.Scope = &accountpb.RoleAssignment_Scope{
			// defaults to RESOURCE_KIND_UNSPECIFIED
			ResourceType: accountpb.RoleAssignment_ResourceType(accountpb.RoleAssignment_ResourceType_value[assignment.ScopeType.String]),
			Resource:     assignment.ScopeResource.String,
		}
	}
	return ra
}

// in SQL queries that return a list of permissions per row, they are joined comma-separated
func splitPermissions(permissions string) []string {
	if permissions == "" {
		return nil
	}
	return strings.Split(permissions, ",")
}
